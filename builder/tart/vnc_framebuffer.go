package tart

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"sync"

	"github.com/mitchellh/go-vnc"
)

type vncFrameBuffer struct {
	mu   sync.RWMutex
	rgba *image.RGBA
}

func newVNCFrameBuffer(width, height int) *vncFrameBuffer {
	return &vncFrameBuffer{
		rgba: image.NewRGBA(image.Rect(0, 0, width, height)),
	}
}

func (f *vncFrameBuffer) applyUpdate(rectangles []vnc.Rectangle, width, height uint16) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, rect := range rectangles {
		switch encoding := rect.Enc.(type) {
		case *DesktopSizePseudoEncoding:
			f.rgba = image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
			fmt.Fprintf(os.Stderr, "🖥️ New desktop size is %dx%d, resized framebuffer\n", width, height)
		case *vnc.RawEncoding:
			expectedColors := int(rect.Width) * int(rect.Height)
			if len(encoding.Colors) != expectedColors {
				return fmt.Errorf("raw frame rectangle %dx%d contains %d colors, expected %d",
					rect.Width, rect.Height, len(encoding.Colors), expectedColors)
			}

			for i, c := range encoding.Colors {
				x, y := i%int(rect.Width), i/int(rect.Width)
				r, g, b := uint8(c.R), uint8(c.G), uint8(c.B)
				f.rgba.Set(int(rect.X)+x, int(rect.Y)+y, color.RGBA{r, g, b, 255})
			}
		default:
			return fmt.Errorf("frame had unknown encoding %T", encoding)
		}
	}

	return nil
}

func (f *vncFrameBuffer) snapshot() *image.RGBA {
	f.mu.RLock()
	defer f.mu.RUnlock()

	frame := image.NewRGBA(f.rgba.Bounds())
	copy(frame.Pix, f.rgba.Pix)

	return frame
}
