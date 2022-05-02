package tart

import (
	"context"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepResize struct {
	vmName string
}

func (s *stepResize) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)

	_, diskChanged := state.GetOk("disk-changed")

	if !diskChanged {
		return multistep.ActionContinue
	}

	ui.Say("Let's SSH in and claim the new space for the disk...")
	communicator := state.Get("communicator").(packersdk.Communicator)

	ui.Say("Freeing space...")
	repairCmd := packersdk.RemoteCmd{
		Command: "yes | diskutil repairDisk disk0",
	}

	err := repairCmd.RunWithUi(ctx, communicator, ui)

	if err != nil {
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	resizeCmd := packersdk.RemoteCmd{
		Command: "diskutil apfs resizeContainer disk0s2 0",
	}

	ui.Say("Resizing the partition...")
	err = resizeCmd.RunWithUi(ctx, communicator, ui)

	if err != nil {
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// Cleanup stops the VM.
func (s *stepResize) Cleanup(state multistep.StateBag) {
}
