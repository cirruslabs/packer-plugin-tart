package tart

import (
	"bytes"
	"context"
	"fmt"
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

	communicator := state.Get("communicator").(packersdk.Communicator)
	if communicator == nil {
		return multistep.ActionContinue
	}

	ui.Say("Let's SSH in and claim the new space for the disk...")

	// Determine the disk and a partition to act on
	listCmd := packersdk.RemoteCmd{
		Command: "diskutil list -plist physical",
	}

	buf := bytes.NewBufferString("")
	listCmd.Stdout = buf

	err := listCmd.RunWithUi(ctx, communicator, &QuietUi{BaseUi: ui})
	if err != nil {
		ui.Error(err.Error())

		return multistep.ActionHalt
	}

	diskName, partitionName, err := ParseDiskUtilPlistOutput(buf.Bytes())
	if err != nil {
		ui.Error(fmt.Sprintf("failed to parse \"diskutil list -plist physical\" output: %v", err))

		return multistep.ActionHalt
	}

	ui.Say("Freeing space...")
	repairCmd := packersdk.RemoteCmd{
		Command: fmt.Sprintf("yes | diskutil repairDisk %s", diskName),
	}

	err = repairCmd.RunWithUi(ctx, communicator, ui)

	if err != nil {
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	resizeCmd := packersdk.RemoteCmd{
		Command: fmt.Sprintf("diskutil apfs resizeContainer %s 0", partitionName),
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
