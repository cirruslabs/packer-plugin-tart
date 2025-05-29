package tart

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework CoreGraphics -framework Vision

#import <stdlib.h>
#import <stdio.h>

#import <CoreGraphics/CoreGraphics.h>
#import <Vision/Vision.h>

bool recognizeTextInFramebuffer(const char* text, void* framebuffer, int width, int height)
{
	@autoreleasepool {
		// Prepare regular expression for needle
		NSError *error = nil;
		NSRegularExpression *regex = [NSRegularExpression regularExpressionWithPattern:
			[NSString stringWithUTF8String:text] options:NSRegularExpressionCaseInsensitive
				error:&error];
		if (error) {
			fprintf(stderr, "⚠️ Failed to create search string regex: %s\n",
				error.localizedDescription.UTF8String);
			return false;
		}

		// Create CGImage wrapper around framebuffer pixel data
		CGDataProviderRef provider = CGDataProviderCreateWithData(
			NULL, framebuffer, width * height * 4, NULL);
		CGColorSpaceRef colorSpace = CGColorSpaceCreateDeviceRGB();
		CGImageRef image = CGImageCreate(width, height, 8, 32, width * 4,
			colorSpace, (CGBitmapInfo)kCGImageAlphaPremultipliedLast, provider,
			NULL, false, kCGRenderingIntentDefault);
		CGColorSpaceRelease(colorSpace);
		CGDataProviderRelease(provider);

		// Recognize text in the framebuffer
		VNRecognizeTextRequest *textRecognizer = [[VNRecognizeTextRequest alloc] init];
		textRecognizer.recognitionLevel = VNRequestTextRecognitionLevelAccurate;
		VNImageRequestHandler *imageRequest = [[VNImageRequestHandler alloc]
			initWithCGImage:image options:@{}];
		CGImageRelease(image);

		BOOL ret = [imageRequest performRequests:@[textRecognizer] error:&error];
		if (error || !ret) {
			fprintf(stderr, "⚠️ Failed to perform image recognition request: %s\n",
				error.localizedDescription.UTF8String);
			return false;
		}

		// Then search for the needle
		for (VNRecognizedTextObservation *observation in textRecognizer.results) {
			for (VNRecognizedText *candidate in [observation topCandidates:1]) {
				fprintf(stderr, "💬 Observed '%s' with confidence %f\n",
					candidate.string.UTF8String, candidate.confidence);
				NSRange range = NSMakeRange(0, candidate.string.length);
				if ([regex matchesInString:candidate.string options:0 range:range].count > 0)
					return true;
			}
		}
	}

	return false;
}
*/
import "C"

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/mitchellh/go-vnc"
	"io"
	"os"
	"strings"
	"time"

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

const WaitForStringStart uint32 = 0xE000
const WaitForStringEnd uint32 = 0xE0001

func (d *customDriver) SendKey(key rune, action bootcommand.KeyAction) error {
	switch uint32(key) {
	case WaitForStringStart:
		d.waitString.Grow(1)
		return nil
	case WaitForStringEnd:
		waitString := d.waitString.String()
		d.waitString.Reset()

		waitStringCStr := C.CString(waitString)
		defer C.free(unsafe.Pointer(waitStringCStr))

		for {
			fmt.Fprintf(os.Stderr, "🔎 Looking for '%s'...\n", waitString)
			if C.recognizeTextInFramebuffer(waitStringCStr,
				unsafe.Pointer(&d.frameBuffer.Pix[0]),
				C.int(d.frameBuffer.Bounds().Dx()),
				C.int(d.frameBuffer.Bounds().Dy())) {
				break
			}

			if err := d.WaitForFramebufferUpdate(); err != nil {
				return err
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
