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
		vmInfo, err := TartVMInfo(ctx, ui, config.VMName)
		if err != nil {
			state.Put("error", fmt.Errorf("Failed to retrieve VM's information: %w", err))

			return multistep.ActionHalt
		}

		_, err = TartExec(ctx, ui, "set", "--disk-size", strconv.Itoa(int(config.DiskSizeGb)), config.VMName)
		if err != nil {
			state.Put("error", fmt.Errorf("Failed to resize a VM: %w", err))

			return multistep.ActionHalt
		}

		if int64(config.DiskSizeGb) != vmInfo.Disk {
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
