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
)

type stepDiskFilePrepare struct{}

func (s *stepDiskFilePrepare) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Inspecting machine disk image...")

	diskImagePath := PathInTartHome("vms", config.VMName, "disk.img")

	if config.DiskSizeGb > 0 {
		sizeChanged, err := growDisk(config.DiskSizeGb, diskImagePath)

		if err != nil {
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		if sizeChanged {
			state.Put(statekey.DiskChanged, true)
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

func growDisk(diskSizeGb uint16, diskImagePath string) (bool, error) {
	desiredSizeInBytes := int64(diskSizeGb) * 1_000_000_000

	diskImageStat, err := os.Stat(diskImagePath)

	if err != nil {
		return false, err
	}

	if diskImageStat.Size() > desiredSizeInBytes {
		return false, errors.New("Image disk is larger then desired! Only disk size increasing is supported! Can't shrink the disk ATM. :-(")
	}

	if diskImageStat.Size() == desiredSizeInBytes {
		return false, nil
	}

	return true, os.Truncate(diskImagePath, desiredSizeInBytes)
}

// Cleanup stops the VM.
func (s *stepDiskFilePrepare) Cleanup(state multistep.StateBag) {
}
