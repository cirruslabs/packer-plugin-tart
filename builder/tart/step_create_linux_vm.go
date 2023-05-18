package tart

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"packer-plugin-tart/builder/tart/tartcmd"
	"strconv"
	"time"
)

type stepCreateLinuxVM struct{}

func (s *stepCreateLinuxVM) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Creating virtual machine...")

	createArguments := []string{
		"create", "--linux",
	}

	if config.DiskSizeGb > 0 {
		createArguments = append(createArguments, "--disk-size", strconv.Itoa(int(config.DiskSizeGb)))
	}

	createArguments = append(createArguments, config.VMName)

	if _, err := tartcmd.Sync(ctx, createArguments...); err != nil {
		err := fmt.Errorf("Failed to create a VM: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())

		return multistep.ActionHalt
	}

	if runInstaller(ctx, state) != multistep.ActionContinue {
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

func (s *stepCreateLinuxVM) Cleanup(state multistep.StateBag) {
	// nothing to clean up
}

func runInstaller(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Starting the virtual machine for installation...")
	runArgs := []string{"run", config.VMName}
	if config.Headless {
		runArgs = append(runArgs, "--no-graphics")
	} else {
		runArgs = append(runArgs, "--graphics")
	}
	if config.Rosetta != "" {
		runArgs = append(runArgs, fmt.Sprintf("--rosetta=%s", config.Rosetta))
	}
	if !config.DisableVNC {
		runArgs = append(runArgs, "--vnc-experimental")
	}
	for _, iso := range config.FromISO {
		runArgs = append(runArgs, fmt.Sprintf("--disk=%s:ro", iso))
	}

	// Prevent the Tart from opening the Screen Sharing
	// window connected to the VNC server we're starting
	var env []string
	if !config.DisableVNC {
		env = append(env, "CI=true")
	}

	tartCmdHandle := tartcmd.Async(ctx, runArgs, env)

	if !config.DisableVNC {
		if !typeBootCommandOverVNC(tartCmdHandle.Ctx(), state, config, ui, tartCmdHandle) {
			return multistep.ActionHalt
		}
	}

	ui.Say("Waiting for the \"tart run\" installation process to shutdown the VM...")

	<-tartCmdHandle.Ctx().Done()

	if err := tartCmdHandle.Err(); err != nil {
		ui.Error(err.Error())
		state.Put("error", err)

		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}
