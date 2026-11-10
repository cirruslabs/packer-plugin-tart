package tart

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldClaimNewSpace(t *testing.T) {
	// Disk grown with the recovery partition deleted (the default) or relocated
	require.True(t, shouldClaimNewSpace(50, 500, ""))
	require.True(t, shouldClaimNewSpace(50, 500, "delete"))
	require.True(t, shouldClaimNewSpace(50, 500, "relocate"))

	// Disk size unchanged
	require.False(t, shouldClaimNewSpace(50, 50, ""))
	require.False(t, shouldClaimNewSpace(50, 50, "relocate"))

	// The kept recovery partition blocks the main partition from growing
	require.False(t, shouldClaimNewSpace(50, 500, "keep"))
}
