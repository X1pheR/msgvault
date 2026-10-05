// Package sourceprovenance projects a deliberately small, provider-neutral
// provenance subset from private per-message source metadata.
package sourceprovenance

import "encoding/json"

// Source describes externally imported source evidence without exposing the
// rest of a provider's private message metadata.
type Source struct {
	Kind               string   `json:"kind,omitempty"`
	Repository         string   `json:"repository,omitempty"`
	RepositoryURL      string   `json:"repository_url,omitempty"`
	Commit             string   `json:"commit,omitempty"`
	ArchivePaths       []string `json:"archive_paths,omitempty"`
	AuthorIdentity     string   `json:"author_identity,omitempty"`
	LifecycleAuthority string   `json:"lifecycle_authority,omitempty"`
}

// FromMessageMetadata returns only an explicitly stored mirror provenance
// object. Unrelated provider metadata and malformed input remain private.
func FromMessageMetadata(raw []byte) *Source {
	if len(raw) == 0 {
		return nil
	}
	var envelope struct {
		MirrorProvenance *Source `json:"mirror_provenance"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || envelope.MirrorProvenance == nil {
		return nil
	}
	p := envelope.MirrorProvenance
	if p.Kind == "" && p.Repository == "" && p.Commit == "" && len(p.ArchivePaths) == 0 {
		return nil
	}
	return p
}
