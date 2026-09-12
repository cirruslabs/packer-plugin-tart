package tart

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVNCRecorderWritesOnlyChangedFrames(t *testing.T) {
	dir := t.TempDir()
	frame := image.NewRGBA(image.Rect(0, 0, 1, 1))
	frame.SetRGBA(0, 0, color.RGBA{R: 1, A: 255})

	times := []time.Time{
		time.Date(2026, 6, 12, 10, 0, 0, 1, time.UTC),
		time.Date(2026, 6, 12, 10, 0, 0, 2, time.UTC),
	}
	timeIndex := 0
	recorder := &vncRecorder{
		dir: dir,
		now: func() time.Time {
			t := times[timeIndex]
			timeIndex++
			return t
		},
	}

	require.NoError(t, recorder.writeIfChanged(frame))
	require.NoError(t, recorder.writeIfChanged(frame))

	frame.SetRGBA(0, 0, color.RGBA{G: 1, A: 255})
	require.NoError(t, recorder.writeIfChanged(frame))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "20260612-100000.000000001.png", entries[0].Name())
	require.Equal(t, "20260612-100000.000000002.png", entries[1].Name())
}

func TestVNCRecorderPrepareClearsExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "old.png"), []byte("old"), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "nested"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested", "old.png"), []byte("old"), 0644))

	recorder := &vncRecorder{dir: dir}
	require.NoError(t, recorder.Prepare())

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestHashFrameIncludesDimensions(t *testing.T) {
	oneByTwo := image.NewRGBA(image.Rect(0, 0, 1, 2))
	twoByOne := image.NewRGBA(image.Rect(0, 0, 2, 1))

	require.NotEqual(t, hashFrame(oneByTwo), hashFrame(twoByOne))
}
