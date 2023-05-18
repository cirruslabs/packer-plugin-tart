package tart

import (
	"context"
	"errors"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"os"
)

type stepDiskFilePrepare struct{}

func (s *stepDiskFilePrepare) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	ui.Say("Inspecting machine disk image...")

	diskImagePath := PathInTartHome("vms", config.VMName, "disk.img")

	if config.DiskSizeGb > 0 {
		err := growDisk(config.DiskSizeGb, diskImagePath)

		if err != nil {
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		state.Put("disk-changed", true)
	}

	disk, err := diskfs.Open(diskImagePath)

	if err != nil {
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say("Getting partition table...")
	partitionTable, err := disk.GetPartitionTable()

	if err != nil {
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	gptTable := partitionTable.(*gpt.Table)

	for i, partition := range gptTable.Partitions {
		if partition.Name != "RecoveryOSContainer" {
			continue
		}
		ui.Say("Found recovery partition. Let's remove it to save space...")
		// there are max 128 partitions and we probably on the third one
		// the rest are just empty structs so let's reuse them
		gptTable.Partitions[i] = gptTable.Partitions[len(gptTable.Partitions)-1]
		err = disk.Partition(gptTable)
		if err != nil {
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		ui.Say("Successfully updated partitions...")
		state.Put("disk-changed", true)
		break
	}

	return multistep.ActionContinue
}

func growDisk(diskSizeGb uint16, diskImagePath string) error {
	desiredSizeInBytes := int64(diskSizeGb) * 1_000_000_000

	diskImageStat, err := os.Stat(diskImagePath)

	if err != nil {
		return err
	}

	if diskImageStat.Size() > desiredSizeInBytes {
		return errors.New("Image disk is larger then desired! Only disk size increasing is supported! Can't shrink the disk ATM. :-(")
	}

	return os.Truncate(diskImagePath, desiredSizeInBytes)
}

// Cleanup stops the VM.
func (s *stepDiskFilePrepare) Cleanup(state multistep.StateBag) {
}
