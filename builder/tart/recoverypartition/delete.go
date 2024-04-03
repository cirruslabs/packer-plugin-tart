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
	disk, err := diskfs.Open(diskImagePath)
	if err != nil {
		return fmt.Errorf("failed to open the disk image: %w", err)
	}

	ui.Say("Getting partition table...")

	partitionTable, err := disk.GetPartitionTable()
	if err != nil {
		if err.Error() == "unknown disk partition type" {
			// Disk may not be initialized with a partition table yet
			return nil
		}

		return fmt.Errorf("failed to get the partition table: %w", err)
	}

	gptTable := partitionTable.(*gpt.Table)

	for i, partition := range gptTable.Partitions {
		if partition.Name != Name {
			continue
		}

		ui.Say("Found recovery partition. Let's remove it to save space...")

		// there are max 128 partitions and we probably on the third one
		// the rest are just empty structs so let's reuse them
		gptTable.Partitions[i] = gptTable.Partitions[len(gptTable.Partitions)-1]

		if err = disk.Partition(gptTable); err != nil {
			return fmt.Errorf("failed to write the new partition table: %w", err)
		}

		ui.Say("Successfully updated partitions...")

		state.Put(statekey.DiskChanged, true)

		return nil
	}

	ui.Say("No recovery partition was found, assuming that it was already deleted.")

	return nil
}
