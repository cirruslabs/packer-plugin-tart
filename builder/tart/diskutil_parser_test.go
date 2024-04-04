package tart_test

import (
	"github.com/stretchr/testify/require"
	"packer-plugin-tart/builder/tart"
	"testing"
)

func TestSingleDisk(t *testing.T) {
	plistBytes := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>AllDisks</key>
	<array>
		<string>disk0</string>
		<string>disk0s1</string>
		<string>disk0s2</string>
		<string>disk0s3</string>
	</array>
	<key>AllDisksAndPartitions</key>
	<array>
		<dict>
			<key>Content</key>
			<string>GUID_partition_scheme</string>
			<key>DeviceIdentifier</key>
			<string>disk0</string>
			<key>OSInternal</key>
			<false/>
			<key>Partitions</key>
			<array>
				<dict>
					<key>Content</key>
					<string>Apple_APFS_ISC</string>
					<key>DeviceIdentifier</key>
					<string>disk0s1</string>
					<key>DiskUUID</key>
					<string>024D2AE5-891F-4244-81EC-182B88D1AA0B</string>
					<key>Size</key>
					<integer>524288000</integer>
				</dict>
				<dict>
					<key>Content</key>
					<string>Apple_APFS</string>
					<key>DeviceIdentifier</key>
					<string>disk0s2</string>
					<key>DiskUUID</key>
					<string>430F1409-D91A-47F4-8418-0876B14AA807</string>
					<key>Size</key>
					<integer>494384795648</integer>
				</dict>
			</array>
			<key>Size</key>
			<integer>500277792768</integer>
		</dict>
	</array>
	<key>VolumesFromDisks</key>
	<array/>
	<key>WholeDisks</key>
	<array>
		<string>disk0</string>
	</array>
</dict>
</plist>
`

	diskName, partitionName, err := tart.ParseDiskUtilPlistOutput([]byte(plistBytes))
	require.NoError(t, err)
	require.Equal(t, "disk0", diskName)
	require.Equal(t, "disk0s2", partitionName)
}

func TestSingleDiskWithRecoveryPartition(t *testing.T) {
	plistBytes := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>AllDisks</key>
	<array>
		<string>disk0</string>
		<string>disk0s1</string>
		<string>disk0s2</string>
		<string>disk0s3</string>
	</array>
	<key>AllDisksAndPartitions</key>
	<array>
		<dict>
			<key>Content</key>
			<string>GUID_partition_scheme</string>
			<key>DeviceIdentifier</key>
			<string>disk0</string>
			<key>OSInternal</key>
			<false/>
			<key>Partitions</key>
			<array>
				<dict>
					<key>Content</key>
					<string>Apple_APFS_ISC</string>
					<key>DeviceIdentifier</key>
					<string>disk0s1</string>
					<key>DiskUUID</key>
					<string>22A4F5CC-CB5B-4EF1-B4F5-EBE9F79C0EE6</string>
					<key>Size</key>
					<integer>524288000</integer>
				</dict>
				<dict>
					<key>Content</key>
					<string>Apple_APFS</string>
					<key>DeviceIdentifier</key>
					<string>disk0s2</string>
					<key>DiskUUID</key>
					<string>E59D951F-E0B0-42B5-BE26-DF3611AD673C</string>
					<key>Size</key>
					<integer>44107001856</integer>
				</dict>
				<dict>
					<key>Content</key>
					<string>Apple_APFS_Recovery</string>
					<key>DeviceIdentifier</key>
					<string>disk0s3</string>
					<key>DiskUUID</key>
					<string>7684137C-F165-45C4-948B-681012742DAC</string>
					<key>Size</key>
					<integer>5368672256</integer>
				</dict>
			</array>
			<key>Size</key>
			<integer>50000000000</integer>
		</dict>
	</array>
	<key>VolumesFromDisks</key>
	<array/>
	<key>WholeDisks</key>
	<array>
		<string>disk0</string>
	</array>
</dict>
</plist>
`

	diskName, partitionName, err := tart.ParseDiskUtilPlistOutput([]byte(plistBytes))
	require.NoError(t, err)
	require.Equal(t, "disk0", diskName)
	require.Equal(t, "disk0s2", partitionName)
}

func TestMultipleDisks(t *testing.T) {
	plistBytes := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>AllDisks</key>
	<array>
		<string>disk0</string>
		<string>disk0s1</string>
		<string>disk0s2</string>
		<string>disk2</string>
	</array>
	<key>AllDisksAndPartitions</key>
	<array>
		<dict>
			<key>Content</key>
			<string>GUID_partition_scheme</string>
			<key>DeviceIdentifier</key>
			<string>disk0</string>
			<key>OSInternal</key>
			<false/>
			<key>Partitions</key>
			<array>
				<dict>
					<key>Content</key>
					<string>Apple_APFS_ISC</string>
					<key>DeviceIdentifier</key>
					<string>disk0s1</string>
					<key>DiskUUID</key>
					<string>D2B79297-879E-4461-8DA2-EEA50EA7319A</string>
					<key>Size</key>
					<integer>524288000</integer>
				</dict>
				<dict>
					<key>Content</key>
					<string>Apple_APFS</string>
					<key>DeviceIdentifier</key>
					<string>disk0s2</string>
					<key>DiskUUID</key>
					<string>D5BA624D-182F-40D0-8248-D08508A8D1B3</string>
					<key>Size</key>
					<integer>89475674112</integer>
				</dict>
			</array>
			<key>Size</key>
			<integer>90000000000</integer>
		</dict>
		<dict>
			<key>Content</key>
			<string></string>
			<key>DeviceIdentifier</key>
			<string>disk2</string>
			<key>OSInternal</key>
			<false/>
			<key>Size</key>
			<integer>107164426240</integer>
		</dict>
	</array>
	<key>VolumesFromDisks</key>
	<array/>
	<key>WholeDisks</key>
	<array>
		<string>disk0</string>
		<string>disk2</string>
	</array>
</dict>
</plist>
`

	diskName, partitionName, err := tart.ParseDiskUtilPlistOutput([]byte(plistBytes))
	require.NoError(t, err)
	require.Equal(t, "disk0", diskName)
	require.Equal(t, "disk0s2", partitionName)
}
