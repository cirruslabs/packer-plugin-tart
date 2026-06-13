package tart

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrepareVNCRecordingDefaultsInterval(t *testing.T) {
	var builder Builder

	_, _, err := builder.Prepare(map[string]interface{}{
		"communicator":      "none",
		"vnc_recording_dir": "frames",
	})
	require.NoError(t, err)
	require.Equal(t, time.Second, builder.config.VNCRecordingInterval)
}

func TestPrepareVNCRecordingUsesConfiguredInterval(t *testing.T) {
	var builder Builder

	_, _, err := builder.Prepare(map[string]interface{}{
		"communicator":           "none",
		"vnc_recording_dir":      "frames",
		"vnc_recording_interval": "250ms",
	})
	require.NoError(t, err)
	require.Equal(t, 250*time.Millisecond, builder.config.VNCRecordingInterval)
}

func TestPrepareVNCRecordingRejectsDisableVNC(t *testing.T) {
	var builder Builder

	_, _, err := builder.Prepare(map[string]interface{}{
		"communicator":      "none",
		"disable_vnc":       true,
		"vnc_recording_dir": "frames",
	})
	require.ErrorContains(t, err, "vnc_recording_dir requires VNC")
}

func TestPrepareVNCRecordingRejectsNegativeInterval(t *testing.T) {
	var builder Builder

	_, _, err := builder.Prepare(map[string]interface{}{
		"communicator":           "none",
		"vnc_recording_dir":      "frames",
		"vnc_recording_interval": "-1s",
	})
	require.ErrorContains(t, err, "vnc_recording_interval must be greater than 0")
}
