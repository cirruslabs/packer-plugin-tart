package tart

import (
	"context"
	"fmt"
	"packer-plugin-tart/builder/tart/recoverypartition"
	"packer-plugin-tart/builder/tart/statekey"
	"strconv"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepDiskFilePrepare struct{}

func (s *stepDiskFilePrepare) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Inspecting machine disk image...")

	diskImagePath := PathInTartHome("vms", config.VMName, "disk.img")

	if config.DiskSizeGb > 0 {
		vmInfo, err := TartVMInfo(ctx, nil, config.VMName)
		if err != nil {
			state.Put("error", fmt.Errorf("Failed to retrieve VM's information: %w", err))

			return multistep.ActionHalt
		}

		_, err = TartExec(ctx, ui, "set", "--disk-size", strconv.Itoa(int(config.DiskSizeGb)), config.VMName)
		if err != nil {
			state.Put("error", fmt.Errorf("Failed to resize a VM: %w", err))

			return multistep.ActionHalt
		}

		// Deleting or relocating the recovery partition below marks the disk as changed
		// only when it actually touches the partition table, which doesn't happen when
		// there's no recovery partition to begin with (e.g. it was deleted in an earlier
		// build). Mark the disk as changed here too, so that the main partition is grown
		// into the newly added space.
		if shouldClaimNewSpace(vmInfo.Disk, config.DiskSizeGb, config.RecoveryPartition) {
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

// Cleanup stops the VM.
func (s *stepDiskFilePrepare) Cleanup(state multistep.StateBag) {
}

// shouldClaimNewSpace reports whether the main partition should be grown after the disk
// was resized from currentSizeGb to newSizeGb.
//
// With "keep", the recovery partition stays at the end of the disk and blocks
// the main partition from growing, so there's nothing to claim.
func shouldClaimNewSpace(currentSizeGb int64, newSizeGb uint16, recoveryPartition string) bool {
	if recoveryPartition == "keep" {
		return false
	}

	return int64(newSizeGb) > currentSizeGb
}
