package tart

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCloneVM struct{}

func (s *stepCloneVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	var commonArgs []string

	if config.AllowInsecure {
		commonArgs = append(commonArgs, "--insecure")
	}

	if config.PullConcurrency > 0 {
		commonArgs = append(commonArgs, "--concurrency", fmt.Sprintf("%d", config.PullConcurrency))
	}

	if config.AlwaysPull {
		ui.Say("Pulling virtual machine...")

		cmdArgs := []string{"pull", config.VMBaseName}
		cmdArgs = append(cmdArgs, commonArgs...)

		if _, err := TartExec(ctx, ui, cmdArgs...); err != nil {
			err := fmt.Errorf("Error pulling VM: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}
	}

	ui.Say("Cloning virtual machine...")

	cmdArgs := []string{"clone", config.VMBaseName, config.VMName}
	cmdArgs = append(cmdArgs, commonArgs...)

	if _, err := TartExec(ctx, ui, cmdArgs...); err != nil {
		err := fmt.Errorf("Error cloning VM: %s", err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	state.Put("vm_name", config.VMName)

	return multistep.ActionContinue
}

func (s *stepCloneVM) Cleanup(state multistep.StateBag) {
	// nothing to clean up
}
