package recoverypartition_test

import (
	"bufio"
	"bytes"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"packer-plugin-tart/builder/tart/recoverypartition"
	"path/filepath"
	"testing"
)

func TestRelocate(t *testing.T) {
	// A disk size that results in a 3-sector hole after relocating the recovery partition
	const diskSizeBytes = 40_960

	// Partition constants
	const sectorSizeBytes = 512
	const partitionSizeSectors = 5
	const partitionSizeBytes = partitionSizeSectors * sectorSizeBytes

	// Hole constants
	const holeSizeSectors = 3
	const holeSizeBytes = holeSizeSectors * sectorSizeBytes

	// Create our disk
	diskPath := filepath.Join(t.TempDir(), "disk.img")

	diskFile, err := os.Create(diskPath)
	require.NoError(t, err)
	require.NoError(t, diskFile.Truncate(diskSizeBytes))
	require.NoError(t, diskFile.Close())

	// Partition our disk as GPT with a macOS recovery partition
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

	// Write partition contents
	firstPartitionContents := bytes.Repeat([]byte{'A'}, partitionSizeBytes)
	writeFile(t, diskPath, firstPartition.GetStart(), firstPartition.GetSize(), firstPartitionContents)

	secondPartitionContents := bytes.Repeat([]byte{'B'}, partitionSizeBytes)
	writeFile(t, diskPath, secondPartition.GetStart(), secondPartition.GetSize(), secondPartitionContents)

	// Relocate the recovery partition to the end of disk
	require.NoError(t, recoverypartition.Relocate(diskPath, packer.TestUi(t), &multistep.BasicStateBag{}))

	// Validate partition entries
	newDisk, err := diskfs.Open(diskPath)
	require.NoError(t, err)

	newDiskPartitionTable, err := newDisk.GetPartitionTable()
	require.NoError(t, err)

	newDiskPartitions := newDiskPartitionTable.(*gpt.Table).Partitions

	require.Len(t, newDiskPartitions, 2)
	require.Equal(t, firstPartitionContents, readFile(t, diskPath, newDiskPartitions[0].GetStart(),
		newDiskPartitions[0].GetSize()))
	require.Equal(t, secondPartitionContents, readFile(t, diskPath, newDiskPartitions[1].GetStart(),
		newDiskPartitions[1].GetSize()))

	// Ensure that the disk size hasn't changed
	diskFileInfo, err := os.Stat(diskPath)
	require.NoError(t, err)
	require.EqualValues(t, diskSizeBytes, diskFileInfo.Size())

	diskFile, err = os.Open(diskPath)
	require.NoError(t, err)

	// Validate the disk contents, piece by piece
	diskFileReader := bufio.NewReader(diskFile)

	// Discard the protective MBR (1 sector), primary GPT header (1 sector)
	// and GPT partition entries (32 sectors)[1].
	//
	// [1]: https://commons.wikimedia.org/wiki/File:GUID_Partition_Table_Scheme.svg
	const primaryTableSize = 34 * sectorSizeBytes

	_, err = diskFileReader.Discard(primaryTableSize)
	require.NoError(t, err)

	// Validate the first partition contents
	actualFirstPartitionContents, err := io.ReadAll(io.LimitReader(diskFileReader, partitionSizeBytes))
	require.NoError(t, err)
	require.Equal(t, firstPartitionContents, actualFirstPartitionContents)

	// Validate the hole that gets formed after we relocate the recovery partition
	//
	// This hole still contains the bytes of the recovery partition.
	expectedHoleContents := bytes.Repeat([]byte{'B'}, holeSizeBytes)

	actualHoleContents, err := io.ReadAll(io.LimitReader(diskFileReader, holeSizeBytes))
	require.NoError(t, err)

	require.Equal(t, expectedHoleContents, actualHoleContents)

	// Validate the second partition contents
	actualSecondPartitionContents, err := io.ReadAll(io.LimitReader(diskFileReader, partitionSizeBytes))
	require.NoError(t, err)
	require.Equal(t, secondPartitionContents, actualSecondPartitionContents)

	// Validate the first two partition entries in the secondary (backup) table
	//
	// Note that we can't simply marshal the gpt.AppleAPFS into bytes here
	// because GPT uses mixed-endianess marshalling, and go-diskfs does not
	// expose this function.
	const gptEntrySizeBytes = 128

	expectedUUID := []byte{0xEF, 0x57, 0x34, 0x7C, 0x00, 0x00, 0xAA, 0x11, 0xAA, 0x11, 0x00, 0x30, 0x65, 0x43, 0xEC, 0xAC}

	secondaryTableFirstEntry, err := io.ReadAll(io.LimitReader(diskFileReader, gptEntrySizeBytes))
	require.NoError(t, err)
	require.Equal(t, expectedUUID, secondaryTableFirstEntry[:16])

	secondaryTableSecondEntry, err := io.ReadAll(io.LimitReader(diskFileReader, gptEntrySizeBytes))
	require.NoError(t, err)
	require.Equal(t, expectedUUID, secondaryTableSecondEntry[:16])

	const secondaryTableSize = 33 * sectorSizeBytes
	_, err = diskFileReader.Discard(secondaryTableSize - (2 * gptEntrySizeBytes))
	require.NoError(t, err)

	// Ensure that we've reached end of the disk
	_, err = diskFileReader.ReadByte()
	require.ErrorIs(t, io.EOF, err)
}

func readFile(t *testing.T, diskPath string, off int64, n int64) []byte {
	diskFile, err := os.Open(diskPath)
	require.NoError(t, err)
	defer diskFile.Close()

	result, err := io.ReadAll(io.NewSectionReader(diskFile, off, n))
	require.NoError(t, err)

	return result
}

func writeFile(t *testing.T, diskPath string, off int64, n int64, contents []byte) {
	diskFile, err := os.OpenFile(diskPath, os.O_RDWR, 0600)
	require.NoError(t, err)

	written, err := diskFile.WriteAt(contents, off)
	require.NoError(t, err)
	require.EqualValues(t, n, written)

	require.NoError(t, diskFile.Close())
}
