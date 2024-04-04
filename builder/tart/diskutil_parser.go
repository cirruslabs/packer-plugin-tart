package tart

import (
	"fmt"
	"howett.net/plist"
)

const expectedPartitionContent = "Apple_APFS"

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
// makes sure there's only one disk on the system and returns its
// name and the name of a single APFS partition or errors otherwise.
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
		for _, partition := range diskWithPartitions.Partitions {
			if partition.Content != expectedPartitionContent {
				continue
			}

			candidates = append(candidates, Candidate{
				DiskName:      diskWithPartitions.DeviceIdentifier,
				PartitionName: partition.DeviceIdentifier,
			})
		}
	}

	if len(candidates) == 0 {
		return "", "", fmt.Errorf("found no disks on which the partition's \"Content\" "+
			"is %q, make sure that the macOS is installed", expectedPartitionContent)
	}

	if len(candidates) > 1 {
		return "", "", fmt.Errorf("found more than one disk on which the partition's \"Content\" "+
			"is %q, please only mount a single disk that contains APFS partitions otherwise it's hard "+
			"to tell on which disk the macOS is installed", expectedPartitionContent)
	}

	return candidates[0].DiskName, candidates[0].PartitionName, nil
}
