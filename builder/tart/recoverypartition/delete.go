package recoverypartition

import (
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"packer-plugin-tart/builder/tart/statekey"
)

func Delete(diskImagePath string, ui packer.Ui, state multistep.StateBag) error {
	// Open the disk image and read its partition table
	disk, err := diskfs.Open(diskImagePath)
	if err != nil {
		return fmt.Errorf("failed to open the disk image: %w", err)
	}

	ui.Say("Getting partition table...")

	partitionTable, err := disk.GetPartitionTable()
	if err != nil {
		// Disk may not be initialized with a partition table yet
		// when running on a freshly created Linux VMs, for example
		if err.Error() == "unknown disk partition type" {
			return nil
		}

		return fmt.Errorf("failed to get the partition table: %w", err)
	}

	gptTable := partitionTable.(*gpt.Table)

	recoveryPartitionIdx := -1

	for idx, partition := range gptTable.Partitions {
		if partition.Name != Name {
			continue
		}

		if recoveryPartitionIdx != -1 {
			return fmt.Errorf("found a recovery partition at GPT entry %d, but there's another recovery "+
				"partition at GPT entry %d, refusing to proceed", idx+1, recoveryPartitionIdx+1)
		}

		recoveryPartitionIdx = idx
	}

	if recoveryPartitionIdx == -1 {
		ui.Say("No recovery partition was found, assuming that it was already deleted.")

		return nil
	}

	if recoveryPartitionIdx != len(gptTable.Partitions)-1 {
		return fmt.Errorf("found a recovery partition at GPT entry %d, but it's "+
			"not the last partition on the disk, refusing to proceed", recoveryPartitionIdx+1)
	}

	ui.Say(fmt.Sprintf("Found a recovery partition at GPT entry %d, let's remove it "+
		"to save space and allow for resizing the main partition...", recoveryPartitionIdx+1))

	gptTable.Partitions = gptTable.Partitions[:recoveryPartitionIdx]

	if err = disk.Partition(gptTable); err != nil {
		return fmt.Errorf("failed to write the new partition table: %w", err)
	}

	ui.Say("Successfully updated partitions!")

	state.Put(statekey.DiskChanged, true)

	return nil
}
