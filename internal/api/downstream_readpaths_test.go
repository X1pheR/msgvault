package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.kenn.io/msgvault/internal/config"
	"go.kenn.io/msgvault/internal/deletion"
	"go.kenn.io/msgvault/internal/search"
	"go.kenn.io/msgvault/internal/store"
)

func newCLIHandlerTestServer(st *mockStore) *Server {
	return NewServer(&config.Config{Server: config.ServerConfig{APIPort: 8080}}, st, nil, testLogger())
}

func TestHandleCLIRunBackupSubcommandAdmission(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		args    []string
		allowed bool
	}{
		{"backup create allowed", []string{"backup", "create"}, true},
		{"backup init rejected", []string{"backup", "init"}, false},
		{"backup verify rejected", []string{"backup", "verify"}, false},
		{"backup with no subcommand rejected", []string{"backup"}, false},
		{"backup unknown subcommand rejected", []string{"backup", "restore"}, false},
		{"import-discord-observations allowed", []string{"import-discord-observations", "/imports/discord/sample.jsonl", "--source", "700"}, true},
		{"logs still allowed", []string{"logs"}, true},
		{"gc allowed", []string{"gc", "--yes"}, true},
		{"import-maildir allowed", []string{"import-maildir", "--identifier", "me@example.test", "dir"}, true},
		{"import-eml allowed", []string{"import-eml", "--identifier", "me@example.test", "dir"}, true},
		{"remove-account still allowed", []string{"remove-account", "alice@example.com", "--yes"}, true},
		{"purge excluded media dry-run allowed", []string{"purge-excluded-media", "--dry-run"}, true},
		{"pack-attachments allowed", []string{"pack-attachments"}, true},
		{"repair-dates apply allowed", []string{"repair-dates", "--apply"}, true},
		{"repair-labels apply allowed", []string{"repair-labels", "--apply"}, true},
		{"repair list IDs apply allowed", []string{"repair-list-ids", "--apply"}, true},
		{"repair-senders apply allowed", []string{"repair-senders", "--apply"}, true},
		{"repack-attachments allowed", []string{"repack-attachments"}, true},
		{"add-discord allowed", []string{"add-discord"}, true},
		{"sync-discord allowed", []string{"sync-discord", "113456789012345678"}, true},
		{"export-discord allowed", []string{"export-discord", "113456789012345678"}, true},
		{"export-messages allowed", []string{"export-messages", "--start", "2026-07-20T00:00:00Z", "--end", "2026-07-21T00:00:00Z"}, true},
		{"backfill-discord-media allowed", []string{"backfill-discord-media", "113456789012345678"}, true},
		{"unpack-attachments rejected", []string{"unpack-attachments"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			st := &mockStore{
				runFunc: func(_ context.Context, _ CLIRunRequest, emit func(CLIRunEvent) error) error {
					return emit(CLIRunEvent{Type: cliStreamEventTypeComplete})
				},
			}
			srv := newCLIHandlerTestServer(st)

			body, err := json.Marshal(CLIRunRequest{Args: tc.args})
			require.NoError(err, "marshal request")
			req := httptest.NewRequest(http.MethodPost, "/api/v1/cli/run", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			srv.Router().ServeHTTP(resp, req)

			if tc.allowed {
				assert.Equal(http.StatusOK, resp.Code, "status: %s", resp.Body.String())
			} else {
				assert.Equal(http.StatusBadRequest, resp.Code, "status: %s", resp.Body.String())
				assert.Contains(resp.Body.String(), "command_not_allowed", "error body")
			}
		})
	}
}

func TestOpenAPIDocumentUsesAPISchemaVersion(t *testing.T) {
	t.Parallel()
	doc := OpenAPIDocument()

	require.NotNil(t, doc.Info, "openapi info")
	assert.Equal(t, APISchemaVersion, doc.Info.Version, "info.version tracks API schema, not binary version")
	assert.NotEmpty(t, doc.Paths, "paths")
}

func TestMeetingIntelligenceOpenAPIContract(t *testing.T) {
	t.Parallel()
	requirements := require.New(t)
	assertions := assert.New(t)
	assertions.Equal("3.1.0", APISchemaVersion)
	doc := OpenAPIDocument()
	for path, operationID := range map[string]string{
		"/api/v1/meetings/context": "getMeetingContext",
		"/api/v1/meetings/actions": "listMeetingActionItems",
		"/api/v1/meetings/metrics": "getMeetingMetrics",
	} {
		operation := doc.Paths[path]
		requirements.NotNil(operation, path)
		requirements.NotNil(operation.Post, path)
		assertions.Equal(operationID, operation.Post.OperationID)
		assertions.Contains(operation.Post.Responses, "413")
		assertions.Contains(operation.Post.Responses, "415")
	}

	schemas := doc.Components.Schemas.Map()
	contextRequest := schemas["MeetingContextRequest"]
	requirements.NotNil(contextRequest)
	requirements.Len(contextRequest.OneOf, 2,
		"context requires exactly one direct IDs or Explore selection")
	assertions.False(contextRequest.Properties["message_ids"].Nullable,
		"omitted and explicit [] are distinct; null is not a third meaning")
	actionsRequest := schemas["MeetingActionsRequest"]
	requirements.NotNil(actionsRequest)
	requirements.NotNil(actionsRequest.Not, "direct and Explore scopes are mutually exclusive")
	metricsRequest := schemas["MeetingMetricsRequest"]
	requirements.NotNil(metricsRequest)
	requirements.NotNil(metricsRequest.Not, "direct and Explore scopes are mutually exclusive")
	scopeRequest := schemas["MeetingScopeRequest"]
	requirements.NotNil(scopeRequest)
	assertions.False(scopeRequest.Properties["message_ids"].Nullable)
	requirements.Len(scopeRequest.AllOf, 3,
		"singular references exclude each other and the exact plural participant IDs")

	meetingImport := schemas["Meeting"]
	requirements.NotNil(meetingImport)
	requirements.NotNil(meetingImport.Properties["action_items"])
	assertions.False(meetingImport.Properties["action_items"].Nullable,
		"omitted means unavailable while [] means source-reported empty")
	requirements.NotNil(meetingImport.Properties["action_items"].MaxItems)
	assertions.Equal(1000, *meetingImport.Properties["action_items"].MaxItems)
	actionImport := schemas["MeetingActionItem"]
	requirements.NotNil(actionImport)
	assertions.Empty(actionImport.Properties["status"].Enum,
		"source status remains an unrestricted bounded string")
	assertions.Empty(actionImport.Properties["due_date"].Format,
		"source due dates remain bounded text")
	requirements.NotNil(actionImport.Properties["status"].MaxLength)
	assertions.Equal(128, *actionImport.Properties["status"].MaxLength)
	requirements.NotNil(actionImport.Properties["due_date"].MaxLength)
	assertions.Equal(256, *actionImport.Properties["due_date"].MaxLength)
}

func TestMessageDetailPublishesSafeSourceProvenance(t *testing.T) {
	doc := OpenAPIDocument()
	schema := doc.Components.Schemas.Map()["MessageDetail"]
	require.NotNil(t, schema, "MessageDetail schema")
	require.NotNil(t, schema.Properties["source_conversation_id"], "source_conversation_id")
	provenance := schema.Properties["source_provenance"]
	require.NotNil(t, provenance, "source_provenance")
	require.NotEmpty(t, provenance.Ref, "source_provenance must reference a typed schema")

	provenanceSchema := doc.Components.Schemas.SchemaFromRef(provenance.Ref)
	require.NotNil(t, provenanceSchema, "SourceProvenanceResponse schema")
	keys := make([]string, 0, len(provenanceSchema.Properties))
	for key := range provenanceSchema.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	assert.ElementsMatch(t,
		[]string{"kind", "repository", "repository_url", "commit", "archive_paths", "author_identity", "lifecycle_authority"},
		keys,
	)
	assert.NotContains(t, schema.Required, "source_conversation_id")
	assert.NotContains(t, schema.Required, "source_provenance")
}

// testLogger returns a logger for tests that discards output.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// mockStore implements MessageStore for tests.
type mockStore struct {
	stats            *StoreStats
	messages         []APIMessage
	total            int64
	needsFTSBackfill bool
	// needsFTSBackfillQuick is the cheap tail-check answer; independent of
	// the full probe so tests can exercise their divergence.
	needsFTSBackfillQuick bool
	// needsFTSBackfillFunc overrides the needsFTSBackfill field when set, so
	// tests can block inside the probe or vary its answer per call.
	needsFTSBackfillFunc     func() bool
	backfillFTSFunc          func(func(done, total int64)) (int64, error)
	rebuildFTSFunc           func(func(done, total int64)) (int64, error)
	buildCacheFunc           func(context.Context, bool, func(CLICacheBuildEvent) error) error
	syncFunc                 func(context.Context, CLISyncRequest, func(CLISyncEvent) error) error
	verifyFunc               func(context.Context, CLIVerifyRequest, func(CLIVerifyEvent) error) error
	repairFunc               func(context.Context, func(CLIRepairEncodingEvent) error) error
	runFunc                  func(context.Context, CLIRunRequest, func(CLIRunEvent) error) error
	repairMessageFunc        func(context.Context, CLIRepairMessageRequest, func(CLIRepairMessageEvent) error) error
	planCalendarFunc         func(context.Context, CLIAddCalendarPlanRequest) (CLIAddCalendarPlanResponse, error)
	planEmbedsFunc           func(context.Context, CLIEmbeddingsPlanRequest) (CLIEmbeddingsPlanResponse, error)
	planDeleteFunc           func(context.Context, CLIDeleteStagedPlanRequest) (CLIDeleteStagedPlanResponse, error)
	planDedupFunc            func(context.Context, CLIDeduplicatePlanRequest) (CLIDeduplicatePlanResponse, error)
	saveManifestFunc         func(context.Context, *deletion.Manifest) error
	documentSearchFunc       func(context.Context, store.DocumentSearchRequest) (store.DocumentSearchResponse, error)
	documentCurrentScopeFunc func(context.Context) (string, []string, error)
	documentStatusFunc       func(context.Context, string, string, []string, []string) (store.DocumentIndexStatus, error)
	documentRebuildFunc      func(context.Context, string, string) (store.DocumentExtractionRebuild, error)
	documentRemainingFunc    func(
		context.Context, store.DocumentExtractionRebuild, []string, []string,
	) (int64, error)
	documentReconcileFunc func(context.Context) error
	personContextFunc     func(context.Context, int64) (*store.Person, error)

	// Error injection for the context-aware read paths, used to verify
	// handlers map context deadline/cancellation to a structured 503.
	statsErr              error
	getStatsFunc          func()
	getStatsContextFunc   func(context.Context) error
	getScopedStatsFunc    func([]int64)
	getScopedStatsCtxFunc func(context.Context, []int64) error
	summariesErr          error

	// Call counts so tests can assert that bulk hydration paths use
	// GetMessagesSummariesByIDs (one round-trip) instead of looping
	// GetMessage (per-hit N+1).
	getMessageCalls                atomic.Int32
	getSummariesByIDsCalls         atomic.Int32
	getSummariesByIDsLastIDs       []int64
	searchMessagesCalls            atomic.Int32
	searchMessagesQueryCalls       atomic.Int32
	searchMessagesQueryLast        *search.Query
	searchMessagesQueryFunc        func(*search.Query, int, int) ([]APIMessage, int64, error)
	searchMessagesQueryLimits      []int
	searchMessagesQueryTransferred int
	needsFTSBackfillCalls          atomic.Int32

	sourcesByLookup    map[string][]*store.Source
	sourcesByLookupErr error
	collections        map[string]*store.CollectionWithSources
}

func (m *mockStore) ReconcileDocumentOccurrences(ctx context.Context) error {
	if m.documentReconcileFunc == nil {
		return nil
	}
	return m.documentReconcileFunc(ctx)
}

func (m *mockStore) SearchDocuments(
	ctx context.Context,
	request store.DocumentSearchRequest,
) (store.DocumentSearchResponse, error) {
	if m.documentSearchFunc == nil {
		return store.DocumentSearchResponse{}, nil
	}
	return m.documentSearchFunc(ctx, request)
}

func (m *mockStore) GetPersonContext(ctx context.Context, id int64) (*store.Person, error) {
	if m.personContextFunc == nil {
		return nil, store.ErrPersonNotFound
	}
	return m.personContextFunc(ctx, id)
}

func (m *mockStore) GetDocumentIndexStatusForScope(
	ctx context.Context,
	profileID string,
	extractionInputKey string,
	allowedMediaTypes []string,
	allowedMessageTypes []string,
) (store.DocumentIndexStatus, error) {
	if m.documentStatusFunc == nil {
		return store.DocumentIndexStatus{}, nil
	}
	return m.documentStatusFunc(
		ctx, profileID, extractionInputKey, allowedMediaTypes, allowedMessageTypes,
	)
}

func (m *mockStore) GetCurrentDocumentIndexStatusScope(
	ctx context.Context,
) (string, []string, error) {
	if m.documentCurrentScopeFunc == nil {
		return "", nil, store.ErrDocumentIndexStatusScopeUnavailable
	}
	return m.documentCurrentScopeFunc(ctx)
}

func (m *mockStore) GetActiveDocumentExtractionRebuild(
	ctx context.Context,
	profileID string,
	extractionInputKey string,
) (store.DocumentExtractionRebuild, error) {
	if m.documentRebuildFunc == nil {
		return store.DocumentExtractionRebuild{}, store.ErrDocumentExtractionRebuildMissing
	}
	return m.documentRebuildFunc(ctx, profileID, extractionInputKey)
}

func (m *mockStore) CountIncompleteDocumentExtractionRebuild(
	ctx context.Context,
	rebuild store.DocumentExtractionRebuild,
	allowedMediaTypes []string,
	allowedMessageTypes []string,
) (int64, error) {
	if m.documentRemainingFunc == nil {
		return 0, nil
	}
	return m.documentRemainingFunc(ctx, rebuild, allowedMediaTypes, allowedMessageTypes)
}

func (m *mockStore) GetStats() (*StoreStats, error) {
	if m.getStatsFunc != nil {
		m.getStatsFunc()
		return &StoreStats{}, nil
	}
	if m.stats == nil {
		return &StoreStats{}, nil
	}
	return m.stats, nil
}

func (m *mockStore) ListMessages(offset, limit int) ([]APIMessage, int64, error) {
	return m.messages, m.total, nil
}

func (m *mockStore) GetMessage(id int64) (*APIMessage, error) {
	m.getMessageCalls.Add(1)
	for _, msg := range m.messages {
		if msg.ID == id {
			return &msg, nil
		}
	}
	return nil, store.ErrMessageNotFound
}

func (m *mockStore) GetMessagesSummariesByIDs(ids []int64) ([]APIMessage, error) {
	m.getSummariesByIDsCalls.Add(1)
	m.getSummariesByIDsLastIDs = append([]int64(nil), ids...)
	byID := make(map[int64]APIMessage, len(m.messages))
	for _, msg := range m.messages {
		byID[msg.ID] = msg
	}
	out := make([]APIMessage, 0, len(ids))
	for _, id := range ids {
		if msg, ok := byID[id]; ok {
			out = append(out, msg)
		}
	}
	return out, nil
}

// The Context variants make mockStore satisfy CtxMessageStore, so handler
// tests exercise the same context-aware read path used in production.
func (m *mockStore) GetStatsContext(ctx context.Context) (*StoreStats, error) {
	if m.getStatsContextFunc != nil {
		return nil, m.getStatsContextFunc(ctx)
	}
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return m.GetStats()
}

func (m *mockStore) ListMessagesContext(_ context.Context, offset, limit int) ([]APIMessage, int64, error) {
	return m.ListMessages(offset, limit)
}

func (m *mockStore) GetMessageContext(_ context.Context, id int64) (*APIMessage, error) {
	return m.GetMessage(id)
}

func (m *mockStore) GetMessagesSummariesByIDsContext(_ context.Context, ids []int64) ([]APIMessage, error) {
	if m.summariesErr != nil {
		return nil, m.summariesErr
	}
	return m.GetMessagesSummariesByIDs(ids)
}

func (m *mockStore) SearchMessages(query string, offset, limit int) ([]APIMessage, int64, error) {
	m.searchMessagesCalls.Add(1)
	return m.messages, m.total, nil
}

func (m *mockStore) SearchMessagesContext(_ context.Context, query string, offset, limit int) ([]APIMessage, int64, error) {
	return m.SearchMessages(query, offset, limit)
}

func (m *mockStore) SearchMessagesQueryContext(ctx context.Context, q *search.Query, offset, limit int) ([]APIMessage, int64, error) {
	return m.SearchMessagesQuery(q, offset, limit)
}

func (m *mockStore) SearchMessageIDsQueryContext(_ context.Context, q *search.Query, limit int) ([]int64, int64, error) {
	messages, total, err := m.SearchMessagesQuery(q, 0, limit)
	ids := make([]int64, 0, min(len(messages), limit))
	for _, message := range messages[:min(len(messages), limit)] {
		ids = append(ids, message.ID)
	}
	return ids, total, err
}

func (m *mockStore) SearchMessagesQuery(q *search.Query, offset, limit int) ([]APIMessage, int64, error) {
	m.searchMessagesQueryCalls.Add(1)
	m.searchMessagesQueryLimits = append(m.searchMessagesQueryLimits, limit)
	if q != nil {
		cp := *q
		cp.AccountIDs = append([]int64(nil), q.AccountIDs...)
		cp.TextTerms = append([]string(nil), q.TextTerms...)
		m.searchMessagesQueryLast = &cp
	} else {
		m.searchMessagesQueryLast = nil
	}
	if m.searchMessagesQueryFunc != nil {
		messages, total, err := m.searchMessagesQueryFunc(q, offset, limit)
		m.searchMessagesQueryTransferred += len(messages)
		return messages, total, err
	}
	m.searchMessagesQueryTransferred += len(m.messages)
	return m.messages, m.total, nil
}

func (m *mockStore) GetStatsForScope(sourceIDs []int64) (*store.Stats, error) {
	if m.getScopedStatsFunc != nil {
		m.getScopedStatsFunc(sourceIDs)
		return &store.Stats{}, nil
	}
	if m.stats == nil {
		return &store.Stats{}, nil
	}
	return m.stats, nil
}

func (m *mockStore) GetStatsForScopeContext(
	ctx context.Context,
	sourceIDs []int64,
) (*store.Stats, error) {
	if m.getScopedStatsCtxFunc != nil {
		return nil, m.getScopedStatsCtxFunc(ctx, sourceIDs)
	}
	return m.GetStatsForScope(sourceIDs)
}

func (m *mockStore) GetSourcesByIdentifierOrDisplayName(input string) ([]*store.Source, error) {
	if m.sourcesByLookupErr != nil {
		return nil, m.sourcesByLookupErr
	}
	if m.sourcesByLookup != nil {
		return m.sourcesByLookup[input], nil
	}
	return nil, nil
}

func (m *mockStore) GetSourcesByTypeAndAccount(string, string) ([]*store.Source, error) {
	return nil, nil
}

func (m *mockStore) GetCollectionByName(name string) (*store.CollectionWithSources, error) {
	if m.collections != nil {
		if coll, ok := m.collections[name]; ok {
			return coll, nil
		}
	}
	return nil, store.ErrCollectionNotFound
}

func (m *mockStore) ListCollections() ([]*store.CollectionWithSources, error) {
	return nil, nil
}

func (m *mockStore) CreateCollection(
	string,
	string,
	[]int64,
) (*store.Collection, error) {
	return &store.Collection{}, nil
}

func (m *mockStore) AddSourcesToCollection(string, []int64) error {
	return nil
}

func (m *mockStore) RemoveSourcesFromCollection(string, []int64) error {
	return nil
}

func (m *mockStore) DeleteCollection(string) error {
	return nil
}

func (m *mockStore) UpdateSourceDisplayName(int64, string) error {
	return nil
}

func (m *mockStore) ListSources(string) ([]*store.Source, error) {
	return nil, nil
}

func (m *mockStore) GetSourceByID(int64) (*store.Source, error) {
	return nil, store.ErrSourceNotFound
}

func (m *mockStore) ListAccountIdentities(int64) ([]store.AccountIdentity, error) {
	return nil, nil
}

func (m *mockStore) AddAccountIdentity(int64, string, string) error {
	return nil
}

func (m *mockStore) RemoveAccountIdentity(int64, string) (int64, error) {
	return 0, nil
}

func (m *mockStore) CountMessagesForSource(int64) (int64, error) {
	return 0, nil
}

func (m *mockStore) CountSourceDeletedMessages(...int64) (int64, error) {
	return 0, nil
}

func (m *mockStore) NeedsFTSBackfill() bool {
	m.needsFTSBackfillCalls.Add(1)
	if m.needsFTSBackfillFunc != nil {
		return m.needsFTSBackfillFunc()
	}
	return m.needsFTSBackfill
}

func (m *mockStore) NeedsFTSBackfillQuick() bool {
	return m.needsFTSBackfillQuick
}

func (m *mockStore) BackfillFTS(progress func(done, total int64)) (int64, error) {
	if m.backfillFTSFunc != nil {
		return m.backfillFTSFunc(progress)
	}
	return 0, nil
}

func (m *mockStore) RebuildFTS(progress func(done, total int64)) (int64, error) {
	if m.rebuildFTSFunc != nil {
		return m.rebuildFTSFunc(progress)
	}
	return 0, nil
}

func (m *mockStore) BuildCLICache(
	ctx context.Context,
	fullRebuild bool,
	emit func(CLICacheBuildEvent) error,
) error {
	if m.buildCacheFunc != nil {
		return m.buildCacheFunc(ctx, fullRebuild, emit)
	}
	return nil
}

func (m *mockStore) RunCLISync(
	ctx context.Context,
	req CLISyncRequest,
	emit func(CLISyncEvent) error,
) error {
	if m.syncFunc != nil {
		return m.syncFunc(ctx, req, emit)
	}
	return nil
}

func (m *mockStore) RunCLIVerify(
	ctx context.Context,
	req CLIVerifyRequest,
	emit func(CLIVerifyEvent) error,
) error {
	if m.verifyFunc != nil {
		return m.verifyFunc(ctx, req, emit)
	}
	return nil
}

func (m *mockStore) RunCLIRepairEncoding(
	ctx context.Context,
	emit func(CLIRepairEncodingEvent) error,
) error {
	if m.repairFunc != nil {
		return m.repairFunc(ctx, emit)
	}
	return nil
}

func (m *mockStore) RunCLICommand(
	ctx context.Context,
	req CLIRunRequest,
	emit func(CLIRunEvent) error,
) error {
	if m.runFunc != nil {
		return m.runFunc(ctx, req, emit)
	}
	return nil
}

func (m *mockStore) RunCLIRepairMessage(
	ctx context.Context,
	req CLIRepairMessageRequest,
	emit func(CLIRepairMessageEvent) error,
) error {
	if m.repairMessageFunc != nil {
		return m.repairMessageFunc(ctx, req, emit)
	}
	return nil
}

func (m *mockStore) PlanCLIAddCalendar(
	ctx context.Context,
	req CLIAddCalendarPlanRequest,
) (CLIAddCalendarPlanResponse, error) {
	if m.planCalendarFunc != nil {
		return m.planCalendarFunc(ctx, req)
	}
	return CLIAddCalendarPlanResponse{}, nil
}

func (m *mockStore) PlanCLIEmbeddings(
	ctx context.Context,
	req CLIEmbeddingsPlanRequest,
) (CLIEmbeddingsPlanResponse, error) {
	if m.planEmbedsFunc != nil {
		return m.planEmbedsFunc(ctx, req)
	}
	return CLIEmbeddingsPlanResponse{}, nil
}

func (m *mockStore) PlanCLIDeleteStaged(
	ctx context.Context,
	req CLIDeleteStagedPlanRequest,
) (CLIDeleteStagedPlanResponse, error) {
	if m.planDeleteFunc != nil {
		return m.planDeleteFunc(ctx, req)
	}
	return CLIDeleteStagedPlanResponse{}, nil
}

func (m *mockStore) PlanCLIDeduplicate(
	ctx context.Context,
	req CLIDeduplicatePlanRequest,
) (CLIDeduplicatePlanResponse, error) {
	if m.planDedupFunc != nil {
		return m.planDedupFunc(ctx, req)
	}
	return CLIDeduplicatePlanResponse{}, nil
}

func (m *mockStore) SaveCLIDeletionManifest(ctx context.Context, manifest *deletion.Manifest) error {
	if m.saveManifestFunc != nil {
		return m.saveManifestFunc(ctx, manifest)
	}
	return nil
}

func TestOpenAPISchemaVersionSavedViewRun(t *testing.T) {
	t.Parallel()
	assertions := assert.New(t)
	requirements := require.New(t)
	assertions.Equal("3.1.0", APISchemaVersion)
	doc := OpenAPIDocument()
	run := doc.Paths["/api/v1/saved-views/{id}/run"]
	requirements.NotNil(run, "Saved View run path")
	requirements.NotNil(run.Post, "Saved View run operation")
	assertions.Equal("runSavedView", run.Post.OperationID)

	schemas := doc.Components.Schemas.Map()
	filter := schemas["SavedViewFilter"]
	requirements.NotNil(filter)
	assertions.Equal(enumValues(store.SavedViewFilterFields()), filter.Properties["field"].Enum,
		"the Saved View schema publishes the store vocabulary")
	sort := schemas["SavedViewSort"]
	requirements.NotNil(sort)
	assertions.Equal([]any{"desc"}, sort.Properties["direction"].Enum)
}

func TestDeletionSubsetSchemaVersion(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "3.1.0", APISchemaVersion)
}

func TestSearchTimingFieldsUseAdditiveSchemaVersion(t *testing.T) {
	t.Parallel()
	assertions := assert.New(t)
	requirements := require.New(t)
	assertions.Equal("3.1.0", APISchemaVersion)

	for _, document := range []*huma.OpenAPI{OpenAPIDocument(), openAPIClientDocument()} {
		schemas := document.Components.Schemas.Map()
		hybrid := schemas["HybridSearchResponse"]
		requirements.NotNil(hybrid)
		assertions.Contains(hybrid.Required, "took_ms")
		assertions.Contains(hybrid.Required, "timings")

		timings := schemas["HybridSearchTimings"]
		requirements.NotNil(timings)
		assertions.ElementsMatch(
			[]string{"query_embedding_ms", "retrieval_ms", "hydration_ms"},
			timings.Required,
		)
	}
}

func TestOpenAPISchemaVersionPersonBrief(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "3.1.0", APISchemaVersion)
}

func TestOpenAPISeparatesParticipantAnalyticsFromDurablePeople(t *testing.T) {
	t.Parallel()
	assertions := assert.New(t)
	doc := OpenAPIDocument()

	assertions.Equal("3.1.0", APISchemaVersion)
	for _, path := range []string{
		"/api/v1/participants/search",
		"/api/v1/participants/{id}",
		"/api/v1/participants/{id}/summary",
		"/api/v1/participants/{id}/timeline",
		"/api/v1/participants/{id}/files/search",
		"/api/v1/people/{id}/files/search",
		"/api/v1/people",
		"/api/v1/people/{id}",
		"/api/v1/people/{id}/profile",
		"/api/v1/people/{id}/contact-state",
	} {
		assertions.Contains(doc.Paths, path)
	}
	assertions.NotContains(doc.Paths, "/api/v1/persons")
	assertions.NotContains(doc.Paths, "/api/v1/persons/{id}")
	assertions.NotNil(doc.Paths["/api/v1/people/search"])
	assertions.Nil(doc.Paths["/api/v1/people/{id}/summary"])
}

func TestAnalyticsCacheReadinessUsesAdditiveSchemaVersion(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "3.1.0", APISchemaVersion)
}

func TestPersonFilesUseAdditiveSchemaVersion(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "3.1.0", APISchemaVersion)
}

func TestOrganizationCreateOpenAPIDocumentsLocationHeader(t *testing.T) {
	t.Parallel()
	requirements := require.New(t)
	assert.Equal(t, "3.1.0", APISchemaVersion,
		"document and person-file search preserve the organization and employment contract")
	for _, document := range []*huma.OpenAPI{
		OpenAPIDocument(),
		openAPIClientDocument(),
	} {
		operation := document.Paths[organizationsPath].Post
		requirements.NotNil(operation)
		response := operation.Responses[httpStatusKey(http.StatusCreated)]
		requirements.NotNil(response)
		requirements.Contains(response.Headers, "Location")
		requirements.Equal(huma.TypeString, response.Headers["Location"].Schema.Type)
	}
}

func operationBodySchema(
	t *testing.T, document *huma.OpenAPI, operation *huma.Operation,
) *huma.Schema {
	t.Helper()
	requirements := require.New(t)
	requirements.NotNil(operation)
	requirements.NotNil(operation.RequestBody)
	schema := operation.RequestBody.Content["application/json"].Schema
	requirements.NotNil(schema)
	if schema.Ref == "" {
		return schema
	}
	const prefix = "#/components/schemas/"
	requirements.True(strings.HasPrefix(schema.Ref, prefix), schema.Ref)
	resolved := document.Components.Schemas.Map()[strings.TrimPrefix(schema.Ref, prefix)]
	requirements.NotNil(resolved)
	return resolved
}

func TestOpenAPIPersonAttributeContract(t *testing.T) {
	t.Parallel()
	requirements := require.New(t)
	assertions := assert.New(t)
	assertions.Equal("3.1.0", APISchemaVersion,
		"activity, identity match review, document search, and person files preserve the structured profile contract")

	doc := OpenAPIDocument()
	definitions := doc.Paths["/api/v1/attribute-definitions"]
	requirements.NotNil(definitions, "attribute definitions path")
	assertions.NotNil(definitions.Get, "attribute definitions list operation")
	assertions.NotNil(definitions.Post, "attribute definitions create operation")
	definition := doc.Paths["/api/v1/attribute-definitions/{id}"]
	requirements.NotNil(definition, "attribute definition path")
	assertions.NotNil(definition.Patch, "attribute definition patch operation")
	assertions.NotNil(definition.Delete, "attribute definition delete operation")
	values := doc.Paths["/api/v1/people/{id}/attributes"]
	requirements.NotNil(values, "person attributes path")
	assertions.NotNil(values.Get, "person attributes list operation")
	value := doc.Paths["/api/v1/people/{id}/attributes/{slug}"]
	requirements.NotNil(value, "person attribute value path")
	assertions.NotNil(value.Put, "person attribute set operation")
	assertions.NotNil(value.Delete, "person attribute clear operation")

	createSchema := operationBodySchema(t, doc, definitions.Post)
	assertions.Contains(createSchema.Properties, "slug")
	assertions.NotContains(createSchema.Required, "slug",
		"the server generates a slug when the client omits it")
}

func TestOpenAPIOrganizationProfilePutDocumentsLimits(t *testing.T) {
	t.Parallel()
	assertions := assert.New(t)
	requirements := require.New(t)
	assertions.Equal("3.1.0", APISchemaVersion,
		"organization profile write limits advance the published contract")
	doc := OpenAPIDocument()
	path := doc.Paths["/api/v1/organizations/{id}/profile"]
	requirements.NotNil(path)
	requirements.NotNil(path.Put)
	assertions.Contains(path.Put.Description, "200")

	response := path.Put.Responses[httpStatusKey(http.StatusRequestEntityTooLarge)]
	requirements.NotNil(response)
	media := response.Content["application/json"]
	requirements.NotNil(media)
	requirements.NotNil(media.Schema)
	assertions.Equal("#/components/schemas/ErrorResponse", media.Schema.Ref)
}

func TestOpenAPIPersonProfileMediaContentContract(t *testing.T) {
	t.Parallel()
	requirements := require.New(t)
	assertions := assert.New(t)

	assertions.Equal("3.1.0", APISchemaVersion,
		"activity, identity match review, document search, and person files preserve the raw profile media contract")
	doc := OpenAPIDocument()
	path := doc.Paths["/api/v1/people/{id}/profile/media/{media_id}/content"]
	requirements.NotNil(path)
	requirements.NotNil(path.Get)
	assertions.Equal("getPersonProfileMediaContent", path.Get.OperationID)
	requirements.Len(path.Get.Security, 1)
	_, secured := path.Get.Security[0]["apiKey"]
	assertions.True(secured)
	requirements.Len(path.Get.Parameters, 2)
	assertions.Equal("id", path.Get.Parameters[0].Name)
	assertions.Equal("media_id", path.Get.Parameters[1].Name)
	response := path.Get.Responses["200"]
	requirements.NotNil(response)
	binary := response.Content["*/*"]
	requirements.NotNil(binary)
	requirements.NotNil(binary.Schema)
	assertions.Equal("binary", binary.Schema.Format)
	for _, status := range []string{"400", "401", "404", "500", "503"} {
		assertions.NotNil(path.Get.Responses[status], status)
	}
}

func TestOpenAPIIdentityMatchReviewContract(t *testing.T) {
	t.Parallel()
	requirements := require.New(t)
	assertions := assert.New(t)

	assertions.Equal("3.1.0", APISchemaVersion,
		"document and person-file search preserve the identity match review contract")

	doc := OpenAPIDocument()
	list := doc.Paths["/api/v1/identity/match-candidates"]
	requirements.NotNil(list, "identity match candidate list path")
	requirements.NotNil(list.Get, "identity match candidate list operation")
	assertions.Equal("listIdentityMatchCandidates", list.Get.OperationID)

	get := doc.Paths["/api/v1/identity/match-candidates/{id}"]
	requirements.NotNil(get)
	requirements.NotNil(get.Get)
	assertions.Equal("getIdentityMatchCandidate", get.Get.OperationID)
	for path, operationID := range map[string]string{
		"/api/v1/identity/match-candidates/{id}/review/accept": "reviewAcceptIdentityMatchCandidate",
		"/api/v1/identity/match-candidates/{id}/review/reject": "reviewRejectIdentityMatchCandidate",
	} {
		item := doc.Paths[path]
		requirements.NotNil(item, path)
		requirements.NotNil(item.Post, path)
		assertions.Equal(operationID, item.Post.OperationID, path)
		requirements.NotNil(item.Post.RequestBody, path)
		assertions.True(item.Post.RequestBody.Required, path)
	}

	// Keep the source-identity route that shipped
	// before identity match review.
	sourceIdentities := doc.Paths["/api/v1/sources/{source_id}/identities"]
	requirements.NotNil(sourceIdentities, "source identity path")
	requirements.NotNil(sourceIdentities.Get, "source identity list operation")
	assertions.Equal("listSourceIdentities", sourceIdentities.Get.OperationID)
}

func TestOpenAPIMeetingImportContract(t *testing.T) {
	t.Parallel()
	requirements := require.New(t)
	assertions := assert.New(t)

	// Pinned so that anyone bumping the schema version has to come here and
	// confirm the meeting-import contract below still holds. Meeting import
	// shipped in 1.33.0; the feed added in 1.34.0, the attributes added in
	// 1.35.0, source-scoped identities added in 1.36.0, and structured profiles
	// added in 1.37.0, raw profile media added in 1.38.0, typed temporal
	// person relationships added in 1.39.0, organizations and employments
	// added in 1.40.0, identity match review added in 1.41.0, and dated activity
	// routes added in 1.42.0, cache-readiness responses added in 1.43.0,
	// document search added in 1.44.0, participant/people separation added in
	// 2.0.0, tracking added in 2.1.0, and participant-scoped files added in
	// 2.5.0. Person search in 2.6.0, structured filters in 2.7.0, CardDAV routes
	// in 2.8.0, person merge/split operations in 2.9.0, and relationship
	// calendars in 2.10.0, person fact diagnostics in 2.11.0, lexical deletion
	// scope in 2.12.0, Directory people and deduplicate planning in 2.13.0,
	// CardDAV status and run history plus List-ID filtering in 2.14.0, Gmail
	// repair in 2.15.0, complete TUI search and statistics contracts plus
	// historical import jobs in 2.16.0, collection source scopes in 2.17.0,
	// deletion subset counts in 2.18.0, Operations in 2.19.0, person briefs
	// in 2.20.0, and Saved View execution in 2.21.0 did not touch it.
	assertions.Equal("3.1.0", APISchemaVersion, "meeting import remains in the current schema")

	doc := OpenAPIDocument()
	path := doc.Paths["/api/v1/import/meeting"]
	requirements.NotNil(path, "meeting import path")
	op := path.Post
	requirements.NotNil(op, "meeting import operation")
	assertions.Equal("importMeeting", op.OperationID)
	requirements.Len(op.Security, 1, "API-key security requirement")
	_, secured := op.Security[0]["apiKey"]
	assertions.True(secured, "apiKey security requirement")

	requirements.NotNil(op.RequestBody, "request body")
	assertions.True(op.RequestBody.Required, "request body is required")
	requestMedia := op.RequestBody.Content["application/json"]
	requirements.NotNil(requestMedia, "JSON request media type")
	requirements.NotNil(requestMedia.Schema, "request schema")
	assertions.Equal("#/components/schemas/MeetingImportRequest", requestMedia.Schema.Ref)

	schemas := doc.Components.Schemas.Map()
	requestSchema := schemas["MeetingImportRequest"]
	requirements.NotNil(requestSchema, "request component")
	requestAdditionalProperties, ok := requestSchema.AdditionalProperties.(bool)
	requirements.True(ok, "request additionalProperties is boolean")
	assertions.False(requestAdditionalProperties, "request rejects unknown fields")
	assertions.ElementsMatch([]string{"source", "meeting"}, requestSchema.Required)

	for _, name := range []string{"Source", "Meeting", "MeetingPerson", "TranscriptSegment"} {
		schema := schemas[name]
		requirements.NotNil(schema, "%s component", name)
		additionalProperties, ok := schema.AdditionalProperties.(bool)
		requirements.True(ok, "%s additionalProperties is boolean", name)
		assertions.False(additionalProperties, "%s rejects unknown fields", name)
	}
	assertions.ElementsMatch(
		[]string{"external_id", "started_at"},
		schemas["Meeting"].Required,
	)
	source := schemas["Source"]
	requirements.NotNil(source.Properties["identifier"].MaxLength)
	assertions.Equal(128, *source.Properties["identifier"].MaxLength)
	requirements.NotNil(source.Properties["display_name"].MaxLength)
	assertions.Equal(256, *source.Properties["display_name"].MaxLength)
	assertions.Equal("email", source.Properties["account_email"].Format)

	meeting := schemas["Meeting"]
	requirements.NotNil(meeting.Properties["external_id"].MaxLength)
	assertions.Equal(256, *meeting.Properties["external_id"].MaxLength)
	requirements.NotNil(meeting.Properties["title"].MaxLength)
	assertions.Equal(4096, *meeting.Properties["title"].MaxLength)
	assertions.Equal("date-time", meeting.Properties["started_at"].Format)
	assertions.Equal("date-time", meeting.Properties["ended_at"].Format)
	requirements.Len(meeting.AllOf, 2, "meeting cross-field constraints")
	requirements.Len(meeting.AllOf[0].AnyOf, 4, "meeting requires a non-empty summary or transcript")
	assertions.ElementsMatch(
		[]string{"summary_markdown", "summary_text", "transcript", "transcript_segments"},
		[]string{
			meeting.AllOf[0].AnyOf[0].Required[0],
			meeting.AllOf[0].AnyOf[1].Required[0],
			meeting.AllOf[0].AnyOf[2].Required[0],
			meeting.AllOf[0].AnyOf[3].Required[0],
		},
	)
	for idx, property := range []string{"summary_markdown", "summary_text", "transcript"} {
		requirements.NotNil(meeting.AllOf[0].AnyOf[idx].Properties[property].MinLength)
		assertions.Equal(1, *meeting.AllOf[0].AnyOf[idx].Properties[property].MinLength)
	}
	requirements.NotNil(meeting.AllOf[0].AnyOf[3].Properties["transcript_segments"].MinItems)
	assertions.Equal(1, *meeting.AllOf[0].AnyOf[3].Properties["transcript_segments"].MinItems)
	requirements.NotNil(meeting.AllOf[1].Not, "plain and segmented transcripts are mutually exclusive")
	assertions.ElementsMatch(
		[]string{"transcript", "transcript_segments"},
		meeting.AllOf[1].Not.Required,
	)

	assertions.Equal("email", schemas["MeetingPerson"].Properties["email"].Format)
	person := schemas["MeetingPerson"]
	assertions.NotContains(person.Required, "email", "an attendee may be identified by phone alone")
	requirements.Len(person.AnyOf, 2, "attendee requires an email or a phone")
	assertions.ElementsMatch([]string{"email", "phone"},
		[]string{person.AnyOf[0].Required[0], person.AnyOf[1].Required[0]})
	requirements.NotNil(person.Properties["id"].MaxLength)
	assertions.Equal(200, *person.Properties["id"].MaxLength)
	requirements.NotNil(person.Properties["phone"].MaxLength)
	offset := schemas["TranscriptSegment"].Properties["offset_seconds"]
	requirements.NotNil(offset.Minimum)
	assertions.Zero(*offset.Minimum)

	metadata := schemas["Meeting"].Properties["metadata"]
	requirements.NotNil(metadata, "metadata schema")
	_, extensible := metadata.AdditionalProperties.(*huma.Schema)
	assertions.True(extensible, "metadata accepts provider-specific values")

	responseSchema := schemas["MeetingImportResponse"]
	requirements.NotNil(responseSchema, "meeting import response component")
	assertions.Equal([]any{"created", "updated"}, responseSchema.Properties["status"].Enum)

	for _, status := range []string{"200", "201"} {
		response := op.Responses[status]
		requirements.NotNil(response, "response %s", status)
		media := response.Content["application/json"]
		requirements.NotNil(media, "response %s JSON media type", status)
		requirements.NotNil(media.Schema, "response %s schema", status)
		assertions.Equal("#/components/schemas/MeetingImportResponse", media.Schema.Ref)
	}
}

func TestPersonBriefOpenAPIContract(t *testing.T) {
	t.Parallel()
	assertions := assert.New(t)
	requirements := require.New(t)
	document := OpenAPIDocument()

	brief := document.Paths["/api/v1/people/{id}/brief"]
	requirements.NotNil(brief)
	requirements.NotNil(brief.Get)
	assertions.Equal("getPersonBrief", brief.Get.OperationID)

	versions := document.Paths["/api/v1/people/{id}/brief/versions"]
	requirements.NotNil(versions)
	requirements.NotNil(versions.Get)
	assertions.Equal("listPersonBriefVersions", versions.Get.OperationID)

	reject := document.Paths["/api/v1/people/{id}/brief/reject"]
	requirements.NotNil(reject)
	requirements.NotNil(reject.Post)
	assertions.Equal("rejectPersonBrief", reject.Post.OperationID)

	generate := document.Paths["/api/v1/people/{id}/brief/generate"]
	requirements.NotNil(generate)
	requirements.NotNil(generate.Post)
	assertions.Equal("generatePersonBrief", generate.Post.OperationID)

	enrollment := document.Paths["/api/v1/people/{id}/brief-enrollment"]
	requirements.NotNil(enrollment)
	requirements.NotNil(enrollment.Get)
	requirements.NotNil(enrollment.Put)
	assertions.Equal("getPersonBriefEnrollment", enrollment.Get.OperationID)
	assertions.Equal("setPersonBriefEnrollment", enrollment.Put.OperationID)

	assertions.Equal("3.1.0", APISchemaVersion)

	// The handlers accept an absent reason and an absent track, so the schema
	// generated clients are built from must not demand either.
	schemas := document.Components.Schemas.Map()
	rejectBody := schemas["RejectPersonBriefRequest"]
	requirements.NotNil(rejectBody)
	assertions.Contains(rejectBody.Properties, "reason")
	assertions.NotContains(rejectBody.Required, "reason")
	enrollmentBody := schemas["PutPersonBriefEnrollmentRequest"]
	requirements.NotNil(enrollmentBody)
	assertions.Equal([]string{"enrolled"}, enrollmentBody.Required)
	assertions.Contains(enrollmentBody.Properties, "track")

	// The join key is part of the contract generated clients are built from.
	briefSchema := schemas["PersonBrief"]
	requirements.NotNil(briefSchema)
	assertions.Contains(briefSchema.Properties, "renderer_policy")
	sentenceSchema := schemas["PersonBriefSentence"]
	requirements.NotNil(sentenceSchema)
	assertions.Contains(sentenceSchema.Properties, "evidence_ordinals")
}

func TestRelationshipCalendarOpenAPIContract(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)
	assert.Equal("3.1.0", APISchemaVersion)
	document := OpenAPIDocument()
	path := document.Paths["/api/v1/relationships/{id}/calendar"]
	require.NotNil(path)
	require.NotNil(path.Post)
	assert.Equal("getRelationshipCalendar", path.Post.OperationID)
	require.Len(path.Post.Parameters, 1)
	assert.Equal("id", path.Post.Parameters[0].Name)
	assert.Equal("path", path.Post.Parameters[0].In)
	assert.True(path.Post.Parameters[0].Required)
	assert.Contains(path.Post.Responses, "200")
	assert.Contains(path.Post.Responses, "404")

	person := document.Components.Schemas.Map()["PersonSummary"]
	require.NotNil(person)
	for _, property := range []string{
		"current_relationship_temperature", "peak_relationship_temperature", "peak_relationship_year",
	} {
		assert.Contains(person.Properties, property)
	}
	day := document.Components.Schemas.Map()["RelationshipCalendarDay"]
	require.NotNil(day)
	require.NotNil(day.Properties["level"])
	assert.ElementsMatch([]any{
		"NONE", "FIRST_QUARTILE", "SECOND_QUARTILE", "THIRD_QUARTILE", "FOURTH_QUARTILE",
	}, day.Properties["level"].Enum)
}
