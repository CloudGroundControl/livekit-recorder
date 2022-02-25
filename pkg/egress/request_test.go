package egress

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetOutputDescriptionForRootFile(t *testing.T) {
	output := "test"
	description := getOutputDescription(output)
	require.Equal(t, output, description.RemoteID)
	require.Equal(t, description.RemoteID, description.LocalID)
}

func TestGetOutputDescriptionForFileInDirectory(t *testing.T) {
	output := "livekit/test"
	description := getOutputDescription(output)
	require.Equal(t, output, description.RemoteID)

	// Assert that local ID, while not equal to remote ID, should be contained
	// in remote ID (minus the directory slashes)
	require.NotEqual(t, description.RemoteID, description.LocalID)
	require.Contains(t, description.RemoteID, description.LocalID)
}
