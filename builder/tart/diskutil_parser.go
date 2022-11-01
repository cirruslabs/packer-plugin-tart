package tart

import (
	"fmt"
	"howett.net/plist"
)

const expectedLastPartitionContent = "Apple_APFS"

func ParseDiskUtilPlistOutput(input []byte) (string, string, error) {
	unmarshalledInput := map[string]interface{}{}

	_, err := plist.Unmarshal(input, &unmarshalledInput)
	if err != nil {
		return "", "", err
	}

	allDisksAndPartitions, ok := unmarshalledInput["AllDisksAndPartitions"].([]interface{})
	if !ok {
		return "", "", fmt.Errorf("\"AllDisksAndPartitions\" value doesn't seem to be a dictionary")
	}

	if len(allDisksAndPartitions) != 1 {
		return "", "", fmt.Errorf("there are more than one physical disk present on the system")
	}

	disk, ok := allDisksAndPartitions[0].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("first disk entry doesn't seem to be a dictionary")
	}

	diskDeviceIdentifier, ok := disk["DeviceIdentifier"].(string)
	if !ok {
		return "", "", fmt.Errorf("first disk's \"DeviceIdentifier\" doesn't seem to be a string")
	}

	partitions, ok := disk["Partitions"].([]interface{})
	if !ok {
		return "", "", fmt.Errorf("first disk's \"Partitions\" doesn't seem to be a list")
	}

	lastPartitionRaw := partitions[len(partitions)-1]
	lastPartition, ok := lastPartitionRaw.(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("last partition entry doesn't seem to be a map")
	}

	lastPartitionContent, ok := lastPartition["Content"].(string)
	if !ok {
		return "", "", fmt.Errorf("last partition's \"Content\" doesn't seem to be a string")
	}
	if lastPartitionContent != expectedLastPartitionContent {
		return "", "", fmt.Errorf("last partition's \"Content\" should be %q, got %q",
			expectedLastPartitionContent, lastPartitionContent)
	}

	lastPartitionDeviceIdentifier, ok := lastPartition["DeviceIdentifier"].(string)
	if !ok {
		return "", "", fmt.Errorf("last partition's \"DeviceIdentifier\" doesn't seem to be a string")
	}

	return diskDeviceIdentifier, lastPartitionDeviceIdentifier, nil
}
