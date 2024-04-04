package recoverypartition_test

import (
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/stretchr/testify/require"
	"os"
	"packer-plugin-tart/builder/tart/recoverypartition"
	"path/filepath"
	"testing"
)

func TestDelete(t *testing.T) {
	// Create a disk
	const diskSizeBytes = 1 * 1024 * 1024

	diskPath := filepath.Join(t.TempDir(), "disk.img")

	diskFile, err := os.Create(diskPath)
	require.NoError(t, err)
	require.NoError(t, diskFile.Truncate(diskSizeBytes))
	require.NoError(t, diskFile.Close())

	// Partition our disk as GPT with a macOS recovery partition
	const sectorSizeBytes = 512
	const partitionSizeSectors = 5
	const partitionSizeBytes = partitionSizeSectors * sectorSizeBytes

	firstPartition := &gpt.Partition{
		Start: 34,
		Size:  partitionSizeBytes,
		Type:  gpt.AppleAPFS,
		Name:  "Doesn't matter",
	}
	secondPartition := &gpt.Partition{
		Start: 34 + partitionSizeSectors,
		Size:  partitionSizeBytes,
		Type:  gpt.AppleAPFS,
		Name:  recoverypartition.Name,
	}
	gptTable := &gpt.Table{
		LogicalSectorSize:  sectorSizeBytes,
		PhysicalSectorSize: sectorSizeBytes,
		Partitions: []*gpt.Partition{
			firstPartition,
			secondPartition,
		},
	}
	oldDisk, err := diskfs.Open(diskPath)
	require.NoError(t, err)
	require.NoError(t, oldDisk.Partition(gptTable))

	// Delete the recovery partition
	require.NoError(t, recoverypartition.Delete(diskPath, packer.TestUi(t), &multistep.BasicStateBag{}))

	// Ensure that the recovery partition was deleted
	disk, err := diskfs.Open(diskPath)
	require.NoError(t, err)

	partitionTable, err := disk.GetPartitionTable()
	require.NoError(t, err)

	partitions := partitionTable.(*gpt.Table).Partitions
	require.Len(t, partitions, 1)
	require.Equal(t, "Doesn't matter", partitions[0].Name)
}
