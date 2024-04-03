package recoverypartition

import (
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"packer-plugin-tart/builder/tart/statekey"
)

func Delete(diskImagePath string, ui packer.Ui, state multistep.StateBag) multistep.StepAction {
	disk, err := diskfs.Open(diskImagePath)
	if err != nil {
		ui.Error(err.Error())

		return multistep.ActionHalt
	}

	ui.Say("Getting partition table...")

	partitionTable, err := disk.GetPartitionTable()
	if err != nil {
		if err.Error() == "unknown disk partition type" {
			// Disk may not be initialized with a partition table yet
			return multistep.ActionContinue
		}

		ui.Error(err.Error())

		return multistep.ActionHalt
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
		err = disk.Partition(gptTable)
		if err != nil {
			ui.Error(err.Error())

			return multistep.ActionHalt
		}

		ui.Say("Successfully updated partitions...")

		state.Put(statekey.DiskChanged, true)

		return multistep.ActionContinue
	}

	ui.Say("No recovery partition was found, assuming that it was already deleted.")

	return multistep.ActionContinue
}
