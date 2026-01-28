package tart

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework CoreGraphics -framework Vision

#import <stdlib.h>
#import "vnc.mm"
*/
import "C"

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/mitchellh/go-vnc"

	"image"
	"image/color"

	"unsafe"
)

type customDriver struct {
	vncClient            *vnc.ClientConn
	serverMessageChannel chan vnc.ServerMessage
	config               *Config
	vncDriver            bootcommand.BCDriver
	keyInterval          time.Duration
	ctx                  context.Context
	frameBuffer          *image.RGBA
	waitString           strings.Builder
	clickString          strings.Builder
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

	w, h := int(vncClient.FrameBufferWidth), int(vncClient.FrameBufferHeight)

	d := &customDriver{
		vncClient:            vncClient,
		serverMessageChannel: serverMessageChannel,
		config:               config,
		vncDriver:            bootcommand.NewVNCDriver(vncClient, keyInterval),
		keyInterval:          keyInterval,
		ctx:                  ctx,
		frameBuffer:          image.NewRGBA(image.Rect(0, 0, w, h)),
	}

	return d
}

func (d *customDriver) KeyInterval() time.Duration {
	return d.keyInterval
}

const (
	WaitForStringStart rune = 0xE0000
	WaitForStringEnd   rune = 0xE0001

	ClickStringStart rune = 0xE0002
	ClickStringEnd   rune = 0xE0003

	LeftCommand  rune = 0xFFE9
	RightCommand rune = 0xFFEA
	LeftOption   rune = 0xFFE7
	RightOption  rune = 0xFFE8
)

func (d *customDriver) SendKey(key rune, action bootcommand.KeyAction) error {
	switch key {
	case WaitForStringStart:
		d.waitString.Grow(1)
	case WaitForStringEnd:
		waitString := d.waitString.String()
		d.waitString.Reset()

		for {
			fmt.Fprintf(os.Stderr, "🔎 Looking for '%s'...\n", waitString)
			if FindTextCoordinates(d.frameBuffer, waitString) != nil {
				break
			}

			if err := d.WaitForFramebufferUpdate(); err != nil {
				return err
			}
		}
	case ClickStringStart:
		d.clickString.Grow(1)
	case ClickStringEnd:
		clickString := d.clickString.String()
		d.clickString.Reset()

		var rectangle *image.Rectangle

		for {
			fmt.Fprintf(os.Stderr, "🔎 Looking for '%s'...\n", clickString)

			rectangle = FindTextCoordinates(d.frameBuffer, clickString)
			if rectangle != nil {
				break
			}

			if err := d.WaitForFramebufferUpdate(); err != nil {
				return err
			}
		}

		centerX := (rectangle.Min.X + rectangle.Max.X) / 2
		centerY := (rectangle.Min.Y + rectangle.Max.Y) / 2

		fmt.Fprintf(os.Stderr, "🖱️ Clicking at '%s's center (%d, %d) ...\n",
			clickString, centerX, centerY)

		if err := d.vncClient.PointerEvent(vnc.ButtonLeft, uint16(centerX), uint16(centerY)); err != nil {
			return err
		}
	default:
		switch {
		case d.waitString.Cap() > 0:
			d.waitString.WriteRune(key)
			return nil
		case d.clickString.Cap() > 0:
			d.clickString.WriteRune(key)
			return nil
		default:
			return d.vncDriver.SendKey(key, action)
		}
	}

	return nil
}

func (d *customDriver) SendSpecial(special string, action bootcommand.KeyAction) error {
	return d.vncDriver.SendSpecial(special, action)
}

func (d *customDriver) Flush() error {
	return d.vncDriver.Flush()
}

func (d *customDriver) WaitForFramebufferUpdate() error {

	for {
		w, h := d.vncClient.FrameBufferWidth, d.vncClient.FrameBufferHeight
		fmt.Fprintf(os.Stderr, "📡 Requesting frame buffer update for %dx%d\n", w, h)

		if err := d.vncClient.FramebufferUpdateRequest(true, 0, 0, w, h); err != nil {
			return err
		}

		select {
		case msg := <-d.serverMessageChannel:
			if framebufferUpdateMessage, ok := msg.(*vnc.FramebufferUpdateMessage); ok {
				if len(framebufferUpdateMessage.Rectangles) == 0 {
					return fmt.Errorf("⚠️ Frame update did not have any rectangles")
				}

				for _, rect := range framebufferUpdateMessage.Rectangles {
					switch encoding := rect.Enc.(type) {
					case *DesktopSizePseudoEncoding:
						w, h := int(d.vncClient.FrameBufferWidth), int(d.vncClient.FrameBufferHeight)
						d.frameBuffer = image.NewRGBA(image.Rect(0, 0, w, h))
						fmt.Fprintf(os.Stderr, "🖥️ New desktop size is %dx%d, resized framebuffer\n", w, h)
						continue
					case *vnc.RawEncoding:
						for i, c := range encoding.Colors {
							x, y := i%int(rect.Width), i/int(rect.Width)
							r, g, b := uint8(c.R), uint8(c.G), uint8(c.B)
							d.frameBuffer.Set(int(rect.X)+x, int(rect.Y)+y, color.RGBA{r, g, b, 255})
						}
					default:
						return fmt.Errorf("⚠️ Frame had unknown encoding %s", encoding)
					}
				}
				return nil
			} else {
				// Ignore messages we didn't ask for
				fmt.Fprintln(os.Stderr, "⚠️ Ignoring unknown message type", msg.Type(), msg)
				continue
			}
		case <-d.ctx.Done():
			return d.ctx.Err()
		}
	}
}

type DesktopSizePseudoEncoding struct{}

func (*DesktopSizePseudoEncoding) Read(c *vnc.ClientConn, rect *vnc.Rectangle, r io.Reader) (vnc.Encoding, error) {
	c.FrameBufferWidth = rect.Width
	c.FrameBufferHeight = rect.Height
	return &DesktopSizePseudoEncoding{}, nil
}

func (*DesktopSizePseudoEncoding) Type() int32 {
	return -223 // RFC 6143 7.8.2
}

func FindTextCoordinates(rgba *image.RGBA, s string) *image.Rectangle {
	sC := C.CString(s)
	defer C.free(unsafe.Pointer(sC))

	rectangleC := (*C.struct_Rectangle)(C.calloc(1, C.size_t(unsafe.Sizeof(C.struct_Rectangle{}))))
	defer C.free(unsafe.Pointer(rectangleC))

	ok := C.recognizeTextInFramebuffer(sC, unsafe.Pointer(&rgba.Pix[0]),
		C.int(rgba.Bounds().Dx()), C.int(rgba.Bounds().Dy()), rectangleC)

	if ok {
		width, height := float64(rgba.Bounds().Max.X), float64(rgba.Bounds().Max.Y)

		return &image.Rectangle{
			Min: image.Point{
				X: int(float64(rectangleC.MinX) * width),
				Y: int(float64(rectangleC.MinY) * height),
			},
			Max: image.Point{
				X: int(float64(rectangleC.MaxX) * width),
				Y: int(float64(rectangleC.MaxY) * height),
			},
		}
	}

	return nil
}
