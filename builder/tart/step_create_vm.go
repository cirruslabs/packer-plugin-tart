package tart

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateVM struct {
	vmName string
}

func (s *stepCreateVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {

	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Cloning virtual machine...")

	if _, err := TartExec("clone", config.VMBaseName, config.VMName); err != nil {
		err := fmt.Errorf("Error cloning VM: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *stepCreateVM) Cleanup(state multistep.StateBag) {
}
