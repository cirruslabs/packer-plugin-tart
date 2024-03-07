package tart

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"strconv"
)

type stepSetVM struct{}

func (s *stepSetVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	needToConfig := config.CpuCount > 0 || config.MemoryGb > 0 || config.Display != ""
	if !needToConfig {
		return multistep.ActionContinue
	}

	ui.Say("Updating virtual machine resources...")

	setArguments := []string{"set", config.VMName}
	if config.CpuCount > 0 {
		setArguments = append(setArguments, "--cpu", strconv.FormatUint(uint64(config.CpuCount), 10))
	}
	if config.MemoryGb > 0 {
		setArguments = append(setArguments, "--memory", strconv.FormatUint(uint64(config.MemoryGb)*1024, 10))
	}
	if config.Display != "" {
		setArguments = append(setArguments, "--display", config.Display)
	}

	if _, err := TartExec(ctx, ui, setArguments...); err != nil {
		err := fmt.Errorf("Error updating VM: %s", err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *stepSetVM) Cleanup(state multistep.StateBag) {
	// nothing to clean up
}
