package sourceprovenance

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromMessageMetadataProjectsOnlyMirrorProvenance(t *testing.T) {
	raw := []byte(`{
		"discord_message_type": 0,
		"author_display_name": "private display",
		"mirror_provenance": {
			"kind": "public_git_mirror",
			"repository": "example/public-archive",
			"repository_url": "https://example.invalid/public-archive",
			"commit": "0123456789abcdef",
			"archive_paths": ["threads/example.json"],
			"author_identity": "public_archive_identity",
			"lifecycle_authority": "append_only_snapshot_no_edit_delete_authority"
		}
	}`)

	got := FromMessageMetadata(raw)
	require.NotNil(t, got)
	assert.Equal(t, "public_git_mirror", got.Kind)
	assert.Equal(t, "example/public-archive", got.Repository)
	assert.Equal(t, "https://example.invalid/public-archive", got.RepositoryURL)
	assert.Equal(t, "0123456789abcdef", got.Commit)
	assert.Equal(t, []string{"threads/example.json"}, got.ArchivePaths)
	assert.Equal(t, "public_archive_identity", got.AuthorIdentity)
	assert.Equal(t, "append_only_snapshot_no_edit_delete_authority", got.LifecycleAuthority)
}

func TestFromMessageMetadataDoesNotExposeUnrelatedProviderMetadata(t *testing.T) {
	assert.Nil(t, FromMessageMetadata([]byte(`{"author_display_name":"private display","guild_nickname":"private nickname"}`)))
	assert.Nil(t, FromMessageMetadata([]byte(`{not-json`)))
	assert.Nil(t, FromMessageMetadata(nil))
}
