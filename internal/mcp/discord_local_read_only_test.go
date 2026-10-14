package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.kenn.io/msgvault/internal/query/querytest"
)

func TestSDDDLE009ReadOnlyServeOptionsOmitStatefulTools(t *testing.T) {
	assert := assert.New(t)
	opts := ServeOptions{Engine: &querytest.MockEngine{}}
	writable := toolsByName(t, rawListTools(t, opts, true))
	assert.Contains(writable, ToolExportAttachment)
	assert.Contains(writable, ToolStageDeletion)

	opts.ReadOnly = true
	tools := toolsByName(t, rawListTools(t, opts, true))

	assert.Contains(tools, ToolSearchMetadata)
	assert.Contains(tools, ToolGetMessage)
	assert.Contains(tools, ToolListMessages)

	assert.NotContains(tools, ToolExportAttachment)
	assert.NotContains(tools, ToolStageDeletion)
}
