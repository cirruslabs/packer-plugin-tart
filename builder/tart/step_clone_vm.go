package tart

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"packer-plugin-tart/builder/tart/tartcmd"
)

type stepCloneVM struct{}

func (s *stepCloneVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Cloning virtual machine...")

	if _, err := tartcmd.Sync(ctx, "clone", config.VMBaseName, config.VMName); err != nil {
		err := fmt.Errorf("Error cloning VM: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *stepCloneVM) Cleanup(state multistep.StateBag) {
	// nothing to clean up
}
