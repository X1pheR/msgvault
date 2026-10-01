#!/usr/bin/env bash
set -euo pipefail

root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
toolchain="golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36"

mod_cache="msgvault-downstream-go-mod-cache"
build_cache="msgvault-downstream-go-build-cache"
docker volume create "$mod_cache" >/dev/null
docker volume create "$build_cache" >/dev/null

docker run --rm \
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
      internal/api/handlers_test.go
      internal/discord/observation_import.go
      internal/discord/observation_import_test.go
      internal/mcp/discord_local_read_only_test.go
      internal/mcp/server.go
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

    go test -count=1 -tags "fts5 sqlite_vec" ./internal/discord -run "^(TestSDDDLE001LocalObservationSourceTypeAndMessageType|TestSDDDLE002003SourceIdentifierAndScopeValidation|TestImportObservationsArchivesLocalDiscordFactsWithoutProviderAPI|TestSDDDLE005AccountScopedDMAndGroupDMConversationTypes|TestImportObservationsInterruptedReplayConverges|TestImportObservationsRejectsMalformedInput|TestSDDDLE003004RejectsMissingOrMismatchedEmbeddedSourceIdentity)$"
    go test -count=1 -tags "fts5 sqlite_vec" ./cmd/msgvault/cmd -run "^(TestImportDiscordObservationsRoutesThroughDaemonCLIRunner|TestSDDDLE001NativeDiscordResolverExcludesLocalObservationSources|TestSDDDLE009ApplyMCPReadOnlyDropsStatefulCapability|TestSDDDLE009MCPCommandExposesReadOnlyFlag)$"
    go test -count=1 -tags "fts5 sqlite_vec" ./internal/api -run "^TestHandleCLIRunBackupSubcommandAdmission$"
    go test -count=1 -tags "fts5 sqlite_vec" ./internal/mcp -run "^TestSDDDLE009ReadOnlyServeOptionsOmitStatefulTools$"
    go test -count=1 -tags "fts5 sqlite_vec" ./internal/store -run "^TestSDDDLE013DiscordLocalExportKeepsDiscordParentAndAuthorSemantics$"

    go test -count=1 -tags "fts5 sqlite_vec" ./internal/discord -run "^TestSDDDLE(017019|019|020|022)"
    go test -count=1 -tags "fts5 sqlite_vec" ./cmd/msgvault/cmd -run "^TestSDDDLE019"
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
