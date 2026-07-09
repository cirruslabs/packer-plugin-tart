package tart

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type vncRecorder struct {
	session  *vncSession
	dir      string
	interval time.Duration
	now      func() time.Time
	hasLast  bool
	lastHash [sha256.Size]byte
}

func newVNCRecorder(session *vncSession, dir string, interval time.Duration) *vncRecorder {
	return &vncRecorder{
		session:  session,
		dir:      dir,
		interval: interval,
		now:      time.Now,
	}
}

func (r *vncRecorder) Prepare() error {
	if err := os.MkdirAll(r.dir, 0755); err != nil {
		return fmt.Errorf("failed to create VNC recording directory %q: %w", r.dir, err)
	}

	if err := clearDirectory(r.dir); err != nil {
		return fmt.Errorf("failed to clear VNC recording directory %q: %w", r.dir, err)
	}

	return nil
}

func clearDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

func (r *vncRecorder) Run(ctx context.Context) error {
	if err := r.capture(ctx, false); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return err
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.capture(ctx, true); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
}

func (r *vncRecorder) capture(ctx context.Context, incremental bool) error {
	var err error
	if incremental {
		err = r.session.WaitForFramebufferUpdate(ctx)
	} else {
		err = r.session.WaitForFullFramebufferUpdate(ctx)
	}
	if err != nil {
		return err
	}

	return r.writeIfChanged(r.session.Snapshot())
}

func (r *vncRecorder) writeIfChanged(frame *image.RGBA) error {
	hash := hashFrame(frame)
	if r.hasLast && hash == r.lastHash {
		return nil
	}

	path := filepath.Join(r.dir, r.now().Format("20060102-150405.000000000")+".png")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create VNC snapshot %q: %w", path, err)
	}

	if err := png.Encode(file, frame); err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to encode VNC snapshot %q: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close VNC snapshot %q: %w", path, err)
	}

	r.lastHash = hash
	r.hasLast = true

	return nil
}

func hashFrame(frame *image.RGBA) [sha256.Size]byte {
	hasher := sha256.New()
	bounds := frame.Bounds()

	var buf [8]byte
	writeInt := func(v int) {
		binary.LittleEndian.PutUint64(buf[:], uint64(v))
		_, _ = hasher.Write(buf[:])
	}

	writeInt(bounds.Min.X)
	writeInt(bounds.Min.Y)
	writeInt(bounds.Max.X)
	writeInt(bounds.Max.Y)
	writeInt(frame.Stride)
	_, _ = hasher.Write(frame.Pix)

	var sum [sha256.Size]byte
	copy(sum[:], hasher.Sum(nil))

	return sum
}

type vncRecorderHandle struct {
	cancel context.CancelFunc
	done   chan struct{}

	mu  sync.Mutex
	err error
}

func startVNCRecorder(ctx context.Context, recorder *vncRecorder) *vncRecorderHandle {
	recorderCtx, cancel := context.WithCancel(ctx)
	handle := &vncRecorderHandle{
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go func() {
		handle.setErr(recorder.Run(recorderCtx))
		close(handle.done)
	}()

	return handle
}

func (h *vncRecorderHandle) Stop() error {
	h.cancel()
	<-h.done

	return h.Err()
}

func (h *vncRecorderHandle) Err() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.err
}

func (h *vncRecorderHandle) setErr(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.err = err
}
