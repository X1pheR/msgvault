#!/usr/bin/env bash
set -euo pipefail

verify_memory_mb="${MSGVAULT_VERIFY_MEMORY_MB:-2048}"
if [[ ! "$verify_memory_mb" =~ ^[0-9]{3,4}$ ]] ||
  ((10#$verify_memory_mb < 512 || 10#$verify_memory_mb > 2048)); then
  echo "MSGVAULT_VERIFY_MEMORY_MB must be an integer from 512 to 2048" >&2
  exit 2
fi

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
toolchain="golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195"

mod_cache="msgvault-downstream-go-mod-cache"
build_cache="msgvault-downstream-go-build-cache"
docker volume create "$mod_cache" >/dev/null
docker volume create "$build_cache" >/dev/null

verify_state=$(mktemp -d /tmp/msgvault-downstream-verify.XXXXXX)
cleanup_verify() {
  local cid
  if [[ -s "$verify_state/container.cid" ]]; then
    cid=$(<"$verify_state/container.cid")
    if [[ "$cid" =~ ^[0-9a-f]{64}$ ]]; then
      docker inspect --format 'Verifier OOMKilled={{.State.OOMKilled}} ExitCode={{.State.ExitCode}}' "$cid" || true
      docker rm -f "$cid" >/dev/null 2>&1 || true
    fi
  fi
  rm -rf -- "$verify_state"
}
trap cleanup_verify EXIT
printf 'Verifier container receipt: %s\n' "$verify_state/container.cid"

docker run --cidfile "$verify_state/container.cid" \
  --cpus=2 --memory="${verify_memory_mb}m" --memory-swap="${verify_memory_mb}m" --pids-limit=128 --env GOMAXPROCS=2 \
  --mount "type=bind,src=$root,dst=/src,readonly" \
  --mount "type=volume,src=$mod_cache,dst=/go/pkg/mod" \
  --mount "type=volume,src=$build_cache,dst=/root/.cache/go-build" \
  --workdir /src \
  "$toolchain" \
  bash -c '
    set -euo pipefail
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq --no-install-recommends gcc g++ make git libsqlite3-dev >/dev/null

    go_files=(
      cmd/msgvault/cmd/import_discord_observations.go
      cmd/msgvault/cmd/import_discord_observations_test.go
      cmd/msgvault/cmd/mcp.go
      cmd/msgvault/cmd/mcp_read_only_test.go
      internal/api/cli_handlers.go
      internal/api/handlers.go
      internal/api/handlers_test.go
      internal/api/downstream_readpaths_test.go
      internal/api/openapi.go
      internal/api/openapi_test.go
      internal/api/source_provenance.go
      internal/daemonclient/engine_adapter.go
      internal/daemonclient/store_adapter.go
      internal/discord/mapping.go
      internal/discord/observation_recovery_test.go
      internal/discord/observation_import.go
      internal/discord/observation_import_test.go
      internal/mcp/discord_local_read_only_test.go
      internal/mcp/handlers.go
      internal/mcp/server.go
      internal/query/models.go
      internal/query/shared.go
      internal/sourceprovenance/source.go
      internal/sourceprovenance/source_test.go
      internal/store/api.go
      internal/store/discord_local_message_export_test.go
      internal/store/message_export.go
      internal/store/discord_local_lifecycle.go
      internal/store/messages.go
      internal/discord/importer.go
      internal/discord/observation_lifecycle_test.go
      cmd/msgvault/cmd/discord_local_history.go
      cmd/msgvault/cmd/discord_local_history_test.go
    )

    unformatted=$(gofmt -l "${go_files[@]}")
    if [[ -n "$unformatted" ]]; then
      printf "gofmt required for:\n%s\n" "$unformatted" >&2
      exit 1
    fi

    packages=(
      ./internal/discord
      ./cmd/msgvault/cmd
      ./internal/api
      ./internal/mcp
      ./internal/store
    )

    go vet -tags "fts5 sqlite_vec" "${packages[@]}"

    # Bound compiler/linker memory only; test processes retain default runtime GC settings.
    go test -toolexec="env GOGC=30 GOMEMLIMIT=768MiB" -count=1 -tags "fts5 sqlite_vec" ./internal/discord -run "^(TestSDDDLE001LocalObservationSourceTypeAndMessageType|TestSDDDLE002003SourceIdentifierAndScopeValidation|TestImportObservationsArchivesLocalDiscordFactsWithoutProviderAPI|TestSDDDLE005AccountScopedDMAndGroupDMConversationTypes|TestImportObservationsInterruptedReplayConverges|TestImportObservationsRejectsMalformedInput|TestSDDDLE003004RejectsMissingOrMismatchedEmbeddedSourceIdentity)$"
    go test -toolexec="env GOGC=30 GOMEMLIMIT=768MiB" -count=1 -tags "fts5 sqlite_vec" ./cmd/msgvault/cmd -run "^(TestImportDiscordObservationsRoutesThroughDaemonCLIRunner|TestSDDDLE001NativeDiscordResolverExcludesLocalObservationSources|TestSDDDLE009ApplyMCPReadOnlyDropsStatefulCapability|TestSDDDLE009MCPCommandExposesReadOnlyFlag)$"
    go test -toolexec="env GOGC=30 GOMEMLIMIT=768MiB" -count=1 -tags "fts5 sqlite_vec" ./internal/sourceprovenance
    # Compile all native API production sources and the unchanged selected fixtures.
    # Full-package vet above still checks every upstream test file.
    mapfile -t api_sources < <(go list -tags "fts5 sqlite_vec" -f "{{range .GoFiles}}{{println .}}{{end}}{{range .CgoFiles}}{{println .}}{{end}}" ./internal/api)
    test "${#api_sources[@]}" -gt 0
    api_files=()
    for source in "${api_sources[@]}"; do
      api_files+=("internal/api/$source")
    done
    api_files+=(internal/api/downstream_readpaths_test.go)
    go test -toolexec="env GOGC=30 GOMEMLIMIT=768MiB" -count=1 -tags "fts5 sqlite_vec" "${api_files[@]}" -run "^Test"
    go test -toolexec="env GOGC=30 GOMEMLIMIT=768MiB" -count=1 -tags "fts5 sqlite_vec" ./internal/daemonclient -run "^TestEngineGetMessage"
    go test -toolexec="env GOGC=30 GOMEMLIMIT=768MiB" -count=1 -tags "fts5 sqlite_vec" ./internal/mcp -run "^TestSDDDLE009ReadOnlyServeOptionsOmitStatefulTools$"
    go test -toolexec="env GOGC=30 GOMEMLIMIT=768MiB" -count=1 -tags "fts5 sqlite_vec" ./internal/store -run "^TestSDDDLE013DiscordLocalExportKeepsDiscordParentAndAuthorSemantics$"

    go test -toolexec="env GOGC=30 GOMEMLIMIT=768MiB" -count=1 -tags "fts5 sqlite_vec" ./internal/discord -run "^TestSDDDLE(017019|019|020|022)"
    go test -toolexec="env GOGC=30 GOMEMLIMIT=768MiB" -count=1 -tags "fts5 sqlite_vec" ./cmd/msgvault/cmd -run "^TestSDDDLE019"
    build_dir=$(mktemp -d /tmp/msgvault-downstream-build.XXXXXX)
    trap "rm -rf \"$build_dir\"" EXIT
    CGO_ENABLED=1 go build \
      -tags "fts5 sqlite_vec" \
      -trimpath \
      -buildvcs=false \
      -o "$build_dir/msgvault" \
      ./cmd/msgvault

    "$build_dir/msgvault" discord-message-history --help >/dev/null
    "$build_dir/msgvault" import-discord-observations --help >/dev/null
    "$build_dir/msgvault" mcp --help | grep -- "--read-only" >/dev/null
  '

echo "msgvault downstream verification passed"
