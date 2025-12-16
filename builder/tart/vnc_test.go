package tart

import (
	imagepkg "image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindTextCoordinates(t *testing.T) {
	imageFile, err := os.Open(filepath.Join("testdata", "select-your-country-or-region.png"))
	require.Nil(t, err)
	defer imageFile.Close()

	image, err := png.Decode(imageFile)
	require.Nil(t, err)

	imageBounds := image.Bounds()
	rgba := imagepkg.NewRGBA(imageBounds)
	draw.Draw(rgba, imageBounds, image, imageBounds.Min, draw.Src)

	rectangle := FindTextCoordinates(rgba, "Select")
	require.NotNil(t, rectangle)

	// Ensure that resulting coordinates differ no more than 5 pixels
	// from the coordinates observed during writing this test
	const maxAllowedDelta = 5
	require.InDelta(t, 683, rectangle.Min.X, maxAllowedDelta)
	require.InDelta(t, 659, rectangle.Min.Y, maxAllowedDelta)
	require.InDelta(t, 1175, rectangle.Max.X, maxAllowedDelta)
	require.InDelta(t, 699, rectangle.Max.Y, maxAllowedDelta)
}
