package tart

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"os"
	"packer-plugin-tart/builder/tart/recoverypartition"
	"packer-plugin-tart/builder/tart/statekey"
	"strconv"
)

type stepDiskFilePrepare struct{}

func (s *stepDiskFilePrepare) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Inspecting machine disk image...")

	diskImagePath := PathInTartHome("vms", config.VMName, "disk.img")

	if config.DiskSizeGb > 0 {
		// Skip disk resizing for ASIF disks - they should be resized using diskutil
		resizeArguments := []string{"set", "--disk-size", strconv.Itoa(int(config.DiskSizeGb)), config.VMName}
		if _, err := TartExec(ctx, ui, resizeArguments...); err != nil {
			err := fmt.Errorf("Failed to resize a VM: %s", err)
			state.Put("error", err)
			return multistep.ActionHalt
		}
	}

	switch config.RecoveryPartition {
	case "":
		fallthrough
	case "delete":
		if err := recoverypartition.Delete(diskImagePath, ui, state); err != nil {
			ui.Error(fmt.Sprintf("Failed to delete the recovery partition: %v", err))

			return multistep.ActionHalt
		}
	case "keep":
		// do nothing
	case "relocate":
		if err := recoverypartition.Relocate(diskImagePath, ui, state); err != nil {
			ui.Error(fmt.Sprintf("Failed to relocate the recovery partition: %v", err))

			return multistep.ActionHalt
		}
	default:
		ui.Error(fmt.Sprintf("Unsupported \"recovery_partition\" value: %q", config.RecoveryPartition))

		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

// Cleanup stops the VM.
func (s *stepDiskFilePrepare) Cleanup(state multistep.StateBag) {
}
