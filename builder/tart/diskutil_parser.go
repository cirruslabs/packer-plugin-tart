package tart

import (
	"fmt"
	"howett.net/plist"
)

const expectedLastPartitionContent = "Apple_APFS"

type DiskUtilOutput struct {
	AllDisksAndPartitions []DiskWithPartitions `plist:"AllDisksAndPartitions"`
}

type DiskWithPartitions struct {
	DeviceIdentifier string      `plist:"DeviceIdentifier"`
	Partitions       []Partition `plist:"Partitions"`
}

type Partition struct {
	DeviceIdentifier string `plist:"DeviceIdentifier"`
	Content          string `plist:"Content"`
}

// ParseDiskUtilPlistOutput parses "diskutil list -plist" output,
// makes sure there's only one disk on the system and returns
// its name and the name of the last partition, additionally
// validating that the last partition is not a recovery one
// (which we should've deleted for the disk expansion to work).
func ParseDiskUtilPlistOutput(input []byte) (string, string, error) {
	var diskUtilOutput DiskUtilOutput

	_, err := plist.Unmarshal(input, &diskUtilOutput)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse \"diskutil list -plist\" output: %w", err)
	}

	type Candidate struct {
		DiskName      string
		PartitionName string
	}

	var candidates []Candidate

	for _, diskWithPartitions := range diskUtilOutput.AllDisksAndPartitions {
		// Skip disks without partitions
		if len(diskWithPartitions.Partitions) == 0 {
			continue
		}

		// Add a candidate if this disk's last partition is expectedLastPartitionContent
		lastPartition := diskWithPartitions.Partitions[len(diskWithPartitions.Partitions)-1]

		if lastPartition.Content != expectedLastPartitionContent {
			continue
		}

		candidates = append(candidates, Candidate{
			DiskName:      diskWithPartitions.DeviceIdentifier,
			PartitionName: lastPartition.DeviceIdentifier,
		})
	}

	if len(candidates) == 0 {
		return "", "", fmt.Errorf("found no disks on which the last partition's \"Content\" "+
			"is %q, make sure that the macOS is installed", expectedLastPartitionContent)
	}

	if len(candidates) > 1 {
		return "", "", fmt.Errorf("found more than one disk on which the last partition's \"Content\" "+
			"is %q, please only mount a single disk that contains APFS partitions otherwise it's hard "+
			"to tell on which disk the macOS is installed", expectedLastPartitionContent)
	}

	return candidates[0].DiskName, candidates[0].PartitionName, nil
}
