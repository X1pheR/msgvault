package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSDDDLE009ReadOnlyServeOptionsOmitStatefulTools(t *testing.T) {
	tools := newMCPServer(ServeOptions{ReadOnly: true}).ListTools()

	assert.Contains(t, tools, ToolSearchMetadata)
	assert.Contains(t, tools, ToolGetMessage)
	assert.Contains(t, tools, ToolListMessages)

	assert.NotContains(t, tools, ToolExportAttachment)
	assert.NotContains(t, tools, ToolStageDeletion)
}
