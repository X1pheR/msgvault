package api

import "go.kenn.io/msgvault/internal/sourceprovenance"

// SourceProvenanceResponse is the safe, deliberately small provenance subset
// exposed by read-only message-detail endpoints.
type SourceProvenanceResponse struct {
	Kind               string   `json:"kind,omitempty"`
	Repository         string   `json:"repository,omitempty"`
	RepositoryURL      string   `json:"repository_url,omitempty"`
	Commit             string   `json:"commit,omitempty"`
	ArchivePaths       []string `json:"archive_paths,omitempty"`
	AuthorIdentity     string   `json:"author_identity,omitempty"`
	LifecycleAuthority string   `json:"lifecycle_authority,omitempty"`
}

func sourceProvenanceResponse(p *sourceprovenance.Source) *SourceProvenanceResponse {
	if p == nil {
		return nil
	}
	return &SourceProvenanceResponse{
		Kind: p.Kind, Repository: p.Repository, RepositoryURL: p.RepositoryURL,
		Commit: p.Commit, ArchivePaths: append([]string(nil), p.ArchivePaths...),
		AuthorIdentity: p.AuthorIdentity, LifecycleAuthority: p.LifecycleAuthority,
	}
}
