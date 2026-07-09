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
	"image"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/mitchellh/go-vnc"

	"unsafe"
)

type customDriver struct {
	vncClient   *vnc.ClientConn
	session     *vncSession
	vncDriver   bootcommand.BCDriver
	keyInterval time.Duration
	ctx         context.Context
	waitString  strings.Builder
	clickString strings.Builder
}

func newCustomDriver(session *vncSession, config *Config, ctx context.Context) *customDriver {
	// Resolve key interval manually so we can accurately report it back
	keyInterval := bootcommand.PackerKeyDefault
	if delay, err := time.ParseDuration(os.Getenv(bootcommand.PackerKeyEnv)); err == nil {
		keyInterval = delay
	}
	if config.BootKeyInterval > time.Duration(0) {
		keyInterval = config.BootKeyInterval
	}

	d := &customDriver{
		vncClient:   session.vncClient,
		session:     session,
		vncDriver:   bootcommand.NewVNCDriver(session.vncClient, keyInterval),
		keyInterval: keyInterval,
		ctx:         ctx,
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
			if FindTextCoordinates(d.session.Snapshot(), waitString) != nil {
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

			rectangle = FindTextCoordinates(d.session.Snapshot(), clickString)
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
		time.Sleep(100 * time.Millisecond)
		if err := d.vncClient.PointerEvent(0, uint16(centerX), uint16(centerY)); err != nil {
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
	return d.session.WaitForFramebufferUpdate(d.ctx)
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
