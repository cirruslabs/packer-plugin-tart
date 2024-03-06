package tart

import (
	"context"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCleanVM struct{}

func (s *stepCleanVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	return multistep.ActionContinue
}

func (s *stepCleanVM) Cleanup(state multistep.StateBag) {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	if cancelled || halted {
		ui.Say("Cleaning up cloned virtual machine...")
		cmdArgs := []string{"delete", config.VMName}
		_, _ = TartExec(context.Background(), cmdArgs...)
	}
}
