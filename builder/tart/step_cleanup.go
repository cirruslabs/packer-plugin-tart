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
	ui := state.Get("ui").(packersdk.Ui)

	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)

	// Only cleanup on cancellation
	if !cancelled && !halted {
		return
	}

	// Only cleanup when the VM was created
	vmName, ok := state.Get("vm_name").(string)
	if !ok {
		return
	}

	ui.Say("Cleaning up virtual machine...")
	cmdArgs := []string{"delete", vmName}
	_, _ = TartExec(context.Background(), ui, cmdArgs...)
}
