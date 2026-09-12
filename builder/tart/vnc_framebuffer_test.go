package tart

import (
	"image"
	"image/color"
	"testing"

	"github.com/mitchellh/go-vnc"
	"github.com/stretchr/testify/require"
)

func TestVNCFrameBufferAppliesRawUpdates(t *testing.T) {
	frameBuffer := newVNCFrameBuffer(2, 2)

	err := frameBuffer.applyUpdate([]vnc.Rectangle{
		{
			X:      0,
			Y:      1,
			Width:  2,
			Height: 1,
			Enc: &vnc.RawEncoding{Colors: []vnc.Color{
				{R: 255, G: 0, B: 0},
				{R: 0, G: 255, B: 0},
			}},
		},
	}, 2, 2)
	require.NoError(t, err)

	snapshot := frameBuffer.snapshot()
	require.Equal(t, color.RGBA{R: 255, A: 255}, snapshot.RGBAAt(0, 1))
	require.Equal(t, color.RGBA{G: 255, A: 255}, snapshot.RGBAAt(1, 1))
}

func TestVNCFrameBufferSnapshotIsACopy(t *testing.T) {
	frameBuffer := newVNCFrameBuffer(1, 1)
	err := frameBuffer.applyUpdate([]vnc.Rectangle{
		{
			Width:  1,
			Height: 1,
			Enc: &vnc.RawEncoding{Colors: []vnc.Color{
				{R: 10, G: 20, B: 30},
			}},
		},
	}, 1, 1)
	require.NoError(t, err)

	snapshot := frameBuffer.snapshot()
	snapshot.SetRGBA(0, 0, color.RGBA{})

	require.Equal(t, color.RGBA{R: 10, G: 20, B: 30, A: 255}, frameBuffer.snapshot().RGBAAt(0, 0))
}

func TestVNCFrameBufferResizesOnDesktopSizeUpdate(t *testing.T) {
	frameBuffer := newVNCFrameBuffer(1, 1)

	err := frameBuffer.applyUpdate([]vnc.Rectangle{
		{Enc: &DesktopSizePseudoEncoding{}},
	}, 3, 4)
	require.NoError(t, err)

	require.Equal(t, image.Rect(0, 0, 3, 4), frameBuffer.snapshot().Bounds())
}

func TestVNCFrameBufferRejectsMalformedRawUpdate(t *testing.T) {
	frameBuffer := newVNCFrameBuffer(2, 2)

	err := frameBuffer.applyUpdate([]vnc.Rectangle{
		{
			Width:  2,
			Height: 2,
			Enc:    &vnc.RawEncoding{Colors: []vnc.Color{{}}},
		},
	}, 2, 2)
	require.ErrorContains(t, err, "contains 1 colors, expected 4")
}
