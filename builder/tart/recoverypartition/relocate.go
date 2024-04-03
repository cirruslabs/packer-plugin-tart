package recoverypartition

import (
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/samber/lo"
	"io"
	"os"
	"packer-plugin-tart/builder/tart/statekey"
)

func Relocate(diskImagePath string, ui packer.Ui, state multistep.StateBag) error {
	// Open the disk image and read its partition table
	disk, err := diskfs.Open(diskImagePath)
	if err != nil {
		return fmt.Errorf("failed to open the disk image: %w", err)
	}

	partitionTable, err := disk.GetPartitionTable()
	if err != nil {
		return fmt.Errorf("failed to get the partition table: %w", err)
	}

	// We only support relocating a recovery partition on a GPT table
	gptTable, ok := partitionTable.(*gpt.Table)
	if !ok {
		return fmt.Errorf("expected a \"gpt\" partition table, got %q", partitionTable.Type())
	}

	// Find the recovery partition
	recoveryPartition, recoveryPartitionIndex, ok := lo.FindIndexOf(gptTable.Partitions, func(partition *gpt.Partition) bool {
		return partition.Name == Name
	})
	if !ok {
		ui.Say("Nothing to relocate: no recovery partition found.")

		return nil
	}

	// We only support relocating the recovery partition if it's the last partition on disk
	if (recoveryPartitionIndex + 1) != len(gptTable.Partitions) {
		return fmt.Errorf("cannot relocate the recovery partition since it's not the last partition " +
			"on disk")
	}

	// Determine the last sector available for partitions, which is normally the (total LBA - 34) sector[1]
	//
	// [1]: https://commons.wikimedia.org/wiki/File:GUID_Partition_Table_Scheme.svg
	lastSectorAvailableForPartitions := uint64((disk.Size / disk.LogicalBlocksize) - 34)

	// Perhaps the recovery partition already resides at the last sector available for partitions?
	if recoveryPartition.End >= lastSectorAvailableForPartitions {
		ui.Say("Nothing to relocate: recovery partition already ends at the last sector available to partitions.")

		return nil
	}

	ui.Say("Dumping recovery partition contents...")

	tmpFile, err := os.CreateTemp("", "")
	if err != nil {
		return fmt.Errorf("failed to create a temporary file for storing "+
			"the recovery partition contents: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := dumpPartition(diskImagePath, tmpFile.Name(), recoveryPartition.GetStart(), recoveryPartition.GetSize()); err != nil {
		return fmt.Errorf("failed to dump the recovery partition contents: %w", err)
	}

	ui.Say("Re-partitioning the disk to adjust the recovery partition bounds...")

	recoveryPartitionSizeInSectors := recoveryPartition.End - recoveryPartition.Start

	recoveryPartition.Start = lastSectorAvailableForPartitions - recoveryPartitionSizeInSectors
	recoveryPartition.End = lastSectorAvailableForPartitions

	// Re-partition the disk with the new recovery partition bounds
	if err := disk.Partition(gptTable); err != nil {
		return fmt.Errorf("failed to write the new partition table: %w", err)
	}

	ui.Say("Restoring recovery partition contents...")

	if err := restorePartition(diskImagePath, tmpFile.Name(), recoveryPartition.GetStart(), recoveryPartition.GetSize()); err != nil {
		return fmt.Errorf("failed to restore the recovery partition contents: %w", err)
	}

	state.Put(statekey.DiskChanged, true)

	return nil
}

func dumpPartition(diskFilePath string, partitionFilePath string, off int64, n int64) error {
	diskFile, err := os.Open(diskFilePath)
	if err != nil {
		return err
	}
	defer diskFile.Close()

	partitionFile, err := os.Create(partitionFilePath)
	if err != nil {
		return err
	}

	partitionContentsReader := io.NewSectionReader(diskFile, off, n)

	if _, err := io.Copy(partitionFile, partitionContentsReader); err != nil {
		return err
	}

	return nil
}

func restorePartition(diskFilePath string, partitionFilePath string, off int64, n int64) error {
	diskFile, err := os.OpenFile(diskFilePath, os.O_RDWR, 0600)
	if err != nil {
		return err
	}

	contentsFile, err := os.Open(partitionFilePath)
	if err != nil {
		return err
	}
	defer contentsFile.Close()

	partitionContentsWriter := io.NewOffsetWriter(diskFile, off)

	if _, err := io.CopyN(partitionContentsWriter, contentsFile, n); err != nil {
		return err
	}

	return diskFile.Close()
}
