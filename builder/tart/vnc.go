package tart

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/mitchellh/go-vnc"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"compress/zlib"
	"encoding/binary"

	"image"
	"image/color"
	"image/png"
	"io/ioutil"

	"github.com/progrium/darwinkit/macos/foundation"
	"github.com/progrium/darwinkit/macos/vision"
	"github.com/progrium/darwinkit/objc"
	"unsafe"
)

var vncRegexp = regexp.MustCompile("vnc://.*:(.*)@(.*):([0-9]{1,5})")

type customDriver struct {
	vncClient            *vnc.ClientConn
	serverMessageChannel chan vnc.ServerMessage
	config               *Config
	vncDriver            bootcommand.BCDriver
	keyInterval          time.Duration
	waitString           strings.Builder
	ctx                  context.Context
}

func newCustomDriver(vncClient *vnc.ClientConn,
	serverMessageChannel chan vnc.ServerMessage,
	config *Config,
	ctx context.Context) *customDriver {

	// Resolve key interval manually so we can accurately report it back
	keyInterval := bootcommand.PackerKeyDefault
	if delay, err := time.ParseDuration(os.Getenv(bootcommand.PackerKeyEnv)); err == nil {
		keyInterval = delay
	}
	if config.BootKeyInterval > time.Duration(0) {
		keyInterval = config.BootKeyInterval
	}

	return &customDriver{
		vncClient:            vncClient,
		serverMessageChannel: serverMessageChannel,
		config:               config,
		vncDriver:            bootcommand.NewVNCDriver(vncClient, keyInterval),
		keyInterval:          keyInterval,
		ctx:                  ctx,
	}
}

func (d *customDriver) KeyInterval() time.Duration {
	return d.keyInterval
}

const WaitForStringStart uint32 = 0xE000
const WaitForStringEnd uint32 = 0xE0001

var debugFrameIndex int = 0

func (d *customDriver) SendKey(key rune, action bootcommand.KeyAction) error {
	switch uint32(key) {
	case WaitForStringStart:
		d.waitString.Grow(1)
		return nil
	case WaitForStringEnd:
		waitString := d.waitString.String()
		d.waitString.Reset()
		log.Printf("Waiting for '%s'...", waitString)

		waitRegex, err := regexp.Compile(".*" + waitString + ".*")
	    if err != nil {
	        return err
	    }

		for {
			frame, err := d.WaitForFramebufferUpdate()
			if err != nil {
				return err
			}
			if len(frame.Rectangles) == 0 {
				return fmt.Errorf("Frame update did not have any rectangles")
			}

			w, h := int(d.vncClient.FrameBufferWidth), int(d.vncClient.FrameBufferHeight)
			img := image.NewRGBA(image.Rect(0, 0, w, h))

			for _, rect := range frame.Rectangles {
				switch enc := rect.Enc.(type) {
				case *DesktopSizePseudoEncoding:
					continue
				case *ZRLEEncoding:
					// FIXME: Feed pixel data directly to image, instead of going via Colors
					for i, c := range enc.Colors {
						x, y := i%w, i/w
						r, g, b := uint8(c.R), uint8(c.G), uint8(c.B)

						img.Set(int(rect.X)+x, int(rect.Y)+y, color.RGBA{r, g, b, 255})
					}
				default:
					return fmt.Errorf("VNC frame had unknown encoding %s", enc)
				}
			}

			// FIXME: Pass image data directly to Vision without PNG encoding first
			pngData := bytes.Buffer{}
			pngEncoder := png.Encoder{CompressionLevel: png.NoCompression}
			if err := pngEncoder.Encode(io.Writer(&pngData), img); err != nil {
				return fmt.Errorf("Failed to encode frame as PNG: %s", err)
			}

			if d.config.PackerDebug {
				ioutil.WriteFile(fmt.Sprintf("vnc-frame-%02d.png", debugFrameIndex), pngData.Bytes(), 0600)
				debugFrameIndex++
			}

			var foundText = false
			log.Printf("Looking for '%s'...", waitString)

			objc.WithAutoreleasePool(func() {
				imageRequest := vision.NewImageRequestHandlerWithDataOptions(pngData.Bytes(), nil)
				textRecognizer := vision.NewRecognizeTextRequest()

				imageRequest.PerformRequestsError([]vision.IRequest{textRecognizer}, unsafe.Pointer(&err))
				if err != nil {
					err = fmt.Errorf("Failed to perform image recognition request")
					return
				}

				// Manually check that we have results, as otherwise we panic when converting
				// an empty array to a slice.
				results := objc.Call[foundation.Array](textRecognizer, objc.Sel("results"))
				if results.Count() == 0 {
					return
				}

				for _, observation := range textRecognizer.Results() {
					textObservation := vision.RecognizedTextObservationFrom(observation.Ptr())
					for _, candidate := range textObservation.TopCandidates(1) {
						log.Printf("Observed '%s' with confidence %g", candidate.String(), candidate.Confidence())
						if waitRegex.MatchString(candidate.String()) {
							log.Printf("Found match for '%s'", waitString)
							foundText = true
							return
						}
					}
				}
			})

			if err != nil {
				return err
			}

			if foundText {
				break
			}
		}

		return nil
	default:
		if d.waitString.Cap() > 0 {
			d.waitString.WriteRune(key)
			return nil
		} else {
			return d.vncDriver.SendKey(key, action)
		}
	}
}

func (d *customDriver) SendSpecial(special string, action bootcommand.KeyAction) error {
	return d.vncDriver.SendSpecial(special, action)
}

func (d *customDriver) Flush() error {
	return d.vncDriver.Flush()
}

func (d *customDriver) WaitForFramebufferUpdate() (*vnc.FramebufferUpdateMessage, error) {
	for {
		w, h := d.vncClient.FrameBufferWidth, d.vncClient.FrameBufferHeight
		log.Printf("Requesting frame buffer update for %dx%d", w, h)
		if err := d.vncClient.FramebufferUpdateRequest(false, 0, 0, w, h); err != nil {
			return nil, err
		}

		select {
		case msg := <-d.serverMessageChannel:
			log.Println("Received message type", msg.Type(), msg)
			if msg.Type() == 0 {
				return msg.(*vnc.FramebufferUpdateMessage), nil
			} else {
				continue
			}
		case <-time.After(time.Second):
			continue // Retry request
		case <-d.ctx.Done():
			return nil, d.ctx.Err()
		}
	}
}

type DesktopSizePseudoEncoding struct{}

func (*DesktopSizePseudoEncoding) Read(c *vnc.ClientConn, rect *vnc.Rectangle, r io.Reader) (vnc.Encoding, error) {
	c.FrameBufferWidth = rect.Width
	c.FrameBufferHeight = rect.Height
	log.Printf("New desktop size is %dx%d", rect.Width, rect.Height)
	return &DesktopSizePseudoEncoding{}, nil
}

func (*DesktopSizePseudoEncoding) Type() int32 {
	return -223 // RFC 6143 7.8.2
}

type ZRLEEncoding struct {
	Colors []vnc.Color
	reader *io.ReadCloser
}

func (ze *ZRLEEncoding) Read(c *vnc.ClientConn, rect *vnc.Rectangle, r io.Reader) (vnc.Encoding, error) {
	// Skip length
	if err := binary.Read(r, binary.BigEndian, new(uint32)); err != nil {
		return nil, err
	}

	if ze.reader == nil {
		// A single zlib stream is used for each RFB protocol connection,
		// so we must re-use the zlib reader between each decode, as we
		// can only read the zlib header once.
		if reader, err := zlib.NewReader(r); err != nil {
			return nil, err
		} else {
			ze.reader = &reader
		}
	}

	if rawEnc, err := (&vnc.RawEncoding{}).Read(c, rect, *ze.reader); err != nil {
		return nil, err
	} else {
		return &ZRLEEncoding{Colors: rawEnc.(*vnc.RawEncoding).Colors}, nil
	}
}

func (*ZRLEEncoding) Type() int32 {
	return 6 // RFC 6143 7.7.6
}

func TypeBootCommandOverVNC(
	ctx context.Context,
	state multistep.StateBag,
	config *Config,
	ui packersdk.Ui,
	tartRunStdout *bytes.Buffer,
) bool {
	ui.Say("Typing boot commands over VNC...")

	if config.HTTPDir != "" || len(config.HTTPContent) != 0 {
		ui.Say("Detecting host IP...")

		hostIP, err := detectHostIP(ctx, config)
		if err != nil {
			err := fmt.Errorf("Failed to detect the host IP address: %v", err)
			state.Put("error", err)
			ui.Error(err.Error())

			return false
		}

		ui.Say(fmt.Sprintf("Host IP is assumed to be %s", hostIP))
		state.Put("http_ip", hostIP)

		// Should be already filled by the Packer's commonsteps.StepHTTPServer
		httpPort := state.Get("http_port").(int)

		config.ctx.Data = &bootCommandTemplateData{
			HTTPIP:   hostIP,
			HTTPPort: httpPort,
		}
	}

	ui.Say("Waiting for VNC server credentials from Tart...")

	vncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var vncPassword string
	var vncHost string
	var vncPort string

	for {
		matches := vncRegexp.FindStringSubmatch(tartRunStdout.String())
		if len(matches) == 1+vncRegexp.NumSubexp() {
			vncPassword = matches[1]
			vncHost = matches[2]
			vncPort = matches[3]

			break
		}

		select {
		case <-vncCtx.Done():
			return false
		case <-time.After(time.Second):
			// continue
		}
	}

	ui.Say("Retrieved VNC credentials, connecting...")
	ui.Message(fmt.Sprintf(
		"If you want to view the screen of the VM, connect via VNC with the password \"%s\" to\n"+
			"vnc://%s:%s", vncPassword, vncHost, vncPort))

	dialer := net.Dialer{}
	netConn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%s", vncHost, vncPort))
	if err != nil {
		err := fmt.Errorf("Failed to connect to Tart's VNC server: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}
	defer netConn.Close()

	serverMessageChannel := make(chan vnc.ServerMessage)
	vncClient, err := vnc.Client(netConn, &vnc.ClientConfig{
		Auth: []vnc.ClientAuth{
			&vnc.PasswordAuth{Password: vncPassword},
		},
		ServerMessageCh: serverMessageChannel,
	})
	if err != nil {
		err := fmt.Errorf("Failed to connect to Tart's VNC server: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}
	defer vncClient.Close()

	ui.Say("Connected to VNC server!")

	vncClient.SetEncodings([]vnc.Encoding{
		&ZRLEEncoding{},
		&DesktopSizePseudoEncoding{},
	})

	vncDriver := newCustomDriver(vncClient, serverMessageChannel, config, ctx)

	if config.VNCConfig.BootWait > 0 {
		message := fmt.Sprintf("Waiting %v after the VM has booted...", config.VNCConfig.BootWait)
		ui.Say(message)
		time.Sleep(config.VNCConfig.BootWait)
	} else {
		ui.Say(fmt.Sprintf("Waiting for first frame..."))
		for i := 0; i < 2; i++ {
			if _, err := vncDriver.WaitForFramebufferUpdate(); err != nil {
				state.Put("error", err)
				ui.Error(err.Error())
				return false
			}
		}
	}

	command, err := interpolate.Render(config.VNCConfig.FlatBootCommand(), &config.ctx)
	if err != nil {
		err := fmt.Errorf("Failed to render the boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}

	stringWaitRegex := regexp.MustCompile(`<wait\s*"([^"]+)">`)
	command = stringWaitRegex.ReplaceAllString(command,
		fmt.Sprintf(`%c${1}%c`, WaitForStringStart, WaitForStringEnd))

	seq, err := bootcommand.GenerateExpressionSequence(command)
	if err != nil {
		err := fmt.Errorf("Failed to parse the boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}

	message := fmt.Sprintf("Typing commands with key interval %v...", vncDriver.KeyInterval())
	ui.Say(message)

	if err := seq.Do(ctx, vncDriver); err != nil {
		err := fmt.Errorf("Failed to run the boot command: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return false
	}

	ui.Say("Done typing commands!")

	return true
}
