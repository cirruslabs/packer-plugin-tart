package tart

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"sync"
	"time"

	"github.com/mitchellh/go-vnc"
)

var debugVNC = os.Getenv("PACKER_TART_DEBUG_VNC_FRAMEBUFFER_UPDATES") != ""

type vncSession struct {
	vncClient            *vnc.ClientConn
	serverMessageChannel chan vnc.ServerMessage
	frameBuffer          *vncFrameBuffer
	updateMu             sync.Mutex
}

func newVNCSession(vncClient *vnc.ClientConn, serverMessageChannel chan vnc.ServerMessage) *vncSession {
	return &vncSession{
		vncClient:            vncClient,
		serverMessageChannel: serverMessageChannel,
		frameBuffer: newVNCFrameBuffer(
			int(vncClient.FrameBufferWidth),
			int(vncClient.FrameBufferHeight),
		),
	}
}

func (s *vncSession) Snapshot() *image.RGBA {
	return s.frameBuffer.snapshot()
}

func (s *vncSession) WaitForFramebufferUpdate(ctx context.Context) error {
	return s.waitForFramebufferUpdate(ctx, true)
}

func (s *vncSession) WaitForFullFramebufferUpdate(ctx context.Context) error {
	return s.waitForFramebufferUpdate(ctx, false)
}

func (s *vncSession) waitForFramebufferUpdate(ctx context.Context, initialIncremental bool) error {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()

	incremental := initialIncremental

	for {
		w, h := s.vncClient.FrameBufferWidth, s.vncClient.FrameBufferHeight
		fmt.Fprintf(os.Stderr, "📡 Requesting %s frame buffer update for %dx%d\n",
			map[bool]string{true: "incremental", false: "full"}[incremental], w, h)

		if err := s.vncClient.FramebufferUpdateRequest(incremental, 0, 0, w, h); err != nil {
			return err
		}

		select {
		case msg, ok := <-s.serverMessageChannel:
			if !ok {
				return fmt.Errorf("VNC server message channel closed")
			}

			framebufferUpdateMessage, ok := msg.(*vnc.FramebufferUpdateMessage)
			if !ok {
				// Ignore messages we didn't ask for.
				fmt.Fprintln(os.Stderr, "⚠️ Ignoring unknown message type", msg.Type(), msg)
				continue
			}

			if len(framebufferUpdateMessage.Rectangles) == 0 {
				return fmt.Errorf("⚠️ Frame update did not have any rectangles")
			}
			fmt.Fprintf(os.Stderr, "🖼️ New framebuffer update with %d rectangles\n",
				len(framebufferUpdateMessage.Rectangles))

			if err := s.frameBuffer.applyUpdate(framebufferUpdateMessage.Rectangles,
				s.vncClient.FrameBufferWidth, s.vncClient.FrameBufferHeight); err != nil {
				return err
			}

			if debugVNC {
				writeDebugVNCFrame(s.Snapshot())
			}

			return nil
		case <-time.After(30 * time.Second):
			fmt.Fprintf(os.Stderr, "⏱️ Framebuffer update timed out after 30s. ")
			// The built-in VNC server in Virtualization.framework will sometimes
			// fail to deliver a framebuffer update, even though the VM view shows
			// new content.
			if incremental {
				// As a first step, we try a full update, which according to
				// RFC 6143 7.5.3 should result in the server sending the entire
				// contents of the specified area as soon as possible.
				fmt.Fprintf(os.Stderr, "Switching to full update\n")
				incremental = false
			} else {
				// However even full updates may in some cases fail to trigger
				// an update from the VZ VNC server. As a second step, we move
				// the mouse, which should result in an update (as long as the
				// VM shows a local cursor).
				fmt.Fprintf(os.Stderr, "Moving mouse to trigger update\n")
				if err := s.vncClient.PointerEvent(0, w-1, 0); err != nil {
					return err
				}
				time.Sleep(1 * time.Second)
				if err := s.vncClient.PointerEvent(0, 0, 0); err != nil {
					return err
				}
			}
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func writeDebugVNCFrame(frame *image.RGBA) {
	file, err := os.Create("framebuffer.png")
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Failed to create framebuffer file: %v\n", err)
		return
	}
	defer file.Close()

	if err := png.Encode(file, frame); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Failed to encode framebuffer: %v\n", err)
	}
}
