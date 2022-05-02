package tart

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepRun struct {
	vmName string
}

func (s *stepRun) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Starting the virtual machine...")
	cmd := exec.Command("tart", "run", "--no-graphics", config.VMName)
	writer := uiWriter{ui: ui}
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Start(); err != nil {
		err = fmt.Errorf("Error starting VM: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	state.Put("tart-cmd", cmd)

	ui.Say("Successfully started the virtual machine...")

	return multistep.ActionContinue
}

type uiWriter struct {
	ui packersdk.Ui
}

func (u uiWriter) Write(p []byte) (n int, err error) {
	u.ui.Say(string(p))
	return len(p), nil
}

// Cleanup stops the VM.
func (s *stepRun) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packersdk.Ui)

	communicator := state.Get("communicator").(packersdk.Communicator)

	if communicator == nil {
		return
	}

	ui.Say("Gracefully shutting down the VM...")
	shutdownCmd := packersdk.RemoteCmd{
		Command: "sudo shutdown -h now",
	}

	err := shutdownCmd.RunWithUi(context.Background(), communicator, ui)

	if err != nil {
		ui.Say("Failed to gracefully shutdown VM...")
		ui.Error(err.Error())
	}

	cmd := state.Get("tart-cmd").(*exec.Cmd)

	if cmd != nil {
		ui.Say("Waiting for the tart process to exit...")
		_, _ = cmd.Process.Wait()
	}
}
