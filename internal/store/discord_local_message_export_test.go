package store_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.kenn.io/msgvault/internal/store"
	"go.kenn.io/msgvault/internal/testutil"
)

func TestSDDDLE013DiscordLocalExportKeepsDiscordParentAndAuthorSemantics(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	st := testutil.NewTestStore(t)
	source, err := st.GetOrCreateSource("discord_local", "700")
	require.NoError(err)

	conversationID := insertMessageExportConversation(
		t,
		st,
		source.ID,
		"301",
		"forum post",
		"thread",
		`{"parent_channel_id":"300","discord_channel_type":11}`,
	)

	sentAt := time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)
	messageID := insertMessageExportMessage(
		t,
		st,
		source.ID,
		conversationID,
		"900",
		"discord",
		sentAt,
		"discord local export body",
		"",
		0,
		false,
		false,
	)
	require.NoError(st.SetMessageMetadata(messageID, sql.NullString{
		String: `{"author_display_name":"Alice"}`,
		Valid:  true,
	}))

	sink := &collectingMessageExportSink{}
	counts, err := st.ExportMessages(context.Background(), store.MessageExportFilter{
		Start:     sentAt.Add(-time.Minute),
		End:       sentAt.Add(time.Minute),
		SourceIDs: []int64{source.ID},
	}, sink)
	require.NoError(err)
	assert.Equal(store.MessageExportCounts{Sources: 1, Conversations: 1, Messages: 1}, counts)

	require.Len(sink.conversations, 1)
	assert.Equal(store.MessageExportConversationThread, sink.conversations[0].ConversationType)
	require.NotNil(sink.conversations[0].ParentID)
	assert.Equal("300", *sink.conversations[0].ParentID)

	require.Len(sink.messages, 1)
	require.NotNil(sink.messages[0].Author)
	assert.Equal("Alice", sink.messages[0].Author.DisplayName)
}
