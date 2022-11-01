package tart

import (
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"io"
)

type QuietUi struct {
	BaseUi packer.Ui
}

func (ui QuietUi) Ask(s string) (string, error) {
	return ui.BaseUi.Ask(s)
}

func (ui QuietUi) Say(s string) {
	// do nothing
}

func (ui QuietUi) Message(s string) {
	// do nothing
}

func (ui QuietUi) Error(s string) {
	ui.BaseUi.Error(s)
}

func (ui QuietUi) Machine(s string, s2 ...string) {
	ui.BaseUi.Machine(s, s2...)
}

func (ui QuietUi) TrackProgress(src string, currentSize, totalSize int64, stream io.ReadCloser) (body io.ReadCloser) {
	return ui.BaseUi.TrackProgress(src, currentSize, totalSize, stream)
}
