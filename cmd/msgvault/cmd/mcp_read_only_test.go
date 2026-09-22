package cmd

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.kenn.io/msgvault/internal/deletion"
	mcpserver "go.kenn.io/msgvault/internal/mcp"
)

type testDeletionManifestSaver struct{}

func (testDeletionManifestSaver) SaveManifest(_ context.Context, _ *deletion.Manifest) error {
	return nil
}

func TestSDDDLE009ApplyMCPReadOnlyDropsStatefulCapability(t *testing.T) {
	opts := mcpserver.ServeOptions{
		ManifestSaver: testDeletionManifestSaver{},
	}
	applyMCPReadOnly(&opts, true)

	assert.True(t, opts.ReadOnly)
	assert.Nil(t, opts.ManifestSaver)
}

func TestSDDDLE009MCPCommandExposesReadOnlyFlag(t *testing.T) {
	require.NotNil(t, mcpCmd.Flags().Lookup("read-only"))
}
