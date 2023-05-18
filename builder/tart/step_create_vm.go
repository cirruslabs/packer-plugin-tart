package tart

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"strconv"
	"time"
)

type stepCreateVM struct{}

func (s *stepCreateVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Creating virtual machine...")

	createArguments := []string{
		"create", "--from-ipsw", config.FromIPSW,
	}

	if config.DiskSizeGb > 0 {
		createArguments = append(createArguments, "--disk-size", strconv.Itoa(int(config.DiskSizeGb)))
	}

	createArguments = append(createArguments, config.VMName)

	if _, err := TartExec(ctx, createArguments...); err != nil {
		err := fmt.Errorf("Failed to create a VM: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return multistep.ActionHalt
	}

	if config.CreateGraceTime != 0 {
		message := fmt.Sprintf("Waiting %v to let the Virtualization.Framework's installation process "+
			"to finish correctly...", config.CreateGraceTime)
		ui.Say(message)
		time.Sleep(config.CreateGraceTime)
	}

	return multistep.ActionContinue
}

func (s *stepCreateVM) Cleanup(state multistep.StateBag) {
	// nothing to clean up
}
