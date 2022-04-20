package tart

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepStop struct {
	vmName string
}

func (s *stepStop) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {

	ui := state.Get("ui").(packersdk.Ui)

	cmd := state.Get("tart-cmd").(*exec.Cmd)

	if err := cmd.Process.Kill(); err != nil {
		ui.Error(fmt.Sprintf("Error stopping VM: %s", err))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// Cleanup stops the VM.
func (s *stepStop) Cleanup(state multistep.StateBag) {
}
