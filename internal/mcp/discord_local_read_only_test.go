package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSDDDLE009ReadOnlyServeOptionsOmitStatefulTools(t *testing.T) {
	assert := assert.New(t)
	tools := newMCPServer(ServeOptions{ReadOnly: true}).ListTools()

	assert.Contains(tools, ToolSearchMetadata)
	assert.Contains(tools, ToolGetMessage)
	assert.Contains(tools, ToolListMessages)

	assert.NotContains(tools, ToolExportAttachment)
	assert.NotContains(tools, ToolStageDeletion)
}
