package tart

import (
	"bytes"
	"context"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"os/exec"
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

	if _, err := TartExec(ctx, ui, createArguments...); err != nil {
		err := fmt.Errorf("Failed to create a VM: %s", err)
		state.Put("error", err)
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
	cmd := exec.CommandContext(ctx, TartCommand(), runArgs...)
	stdout := bytes.NewBufferString("")
	cmd.Stdout = stdout
	cmd.Stderr = uiWriter{ui: ui}

	// Prevent the Tart from opening the Screen Sharing
	// window connected to the VNC server we're starting
	if !config.DisableVNC {
		cmd.Env = cmd.Environ()
		cmd.Env = append(cmd.Env, "CI=true")
	}

	if err := cmd.Start(); err != nil {
		err = fmt.Errorf("Error starting VM: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	defer func() {
		ui.Say("Waiting for the install process to shutdown the VM...")
		_, _ = cmd.Process.Wait()
	}()

	if !config.DisableVNC {
		if !typeBootCommandOverVNC(ctx, state, config, ui, stdout) {
			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}
