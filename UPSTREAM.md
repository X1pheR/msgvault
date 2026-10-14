# Upstream convergence

- Upstream: https://github.com/kenn-io/msgvault
- Downstream: https://github.com/X1pheR/msgvault
- Current documented comparison baseline: v0.19.3

## Discord offline observations: existing upstream issue

Local MV-P01 source commit cade6dc adds a local Discord observation
importer with associated tests. The existing upstream
[kenn-io/msgvault issue #884](https://github.com/kenn-io/msgvault/issues/884),
authored by X1pheR, requests a bounded local/offline importer with
snapshot/upsert and explicit delete semantics, without user-token
or selfbot acquisition. The existing public comment documents a
synthetic additive importer prototype with tests on release v0.19.3
and upstream main. The maintainer has not yet accepted source identity
for DM/group-DM observations and no upstream PR exists.

MV-P01 remains nonconformant. Do not treat the issue alone as an
upstream contribution, or submit the entire downstream product
patch with unrelated MCP/product-specific changes. Resolve
source identity with the maintainer, then prepare an upstream-native
tested PR or a precise operator-approved blocked disposition.

## gRPC upstream convergence — 2026-10-11

Upstream issue #1218 is closed by merged PR #1220. The PR head is
`ed991028884a148888778c87db39fae6caa7678b`; upstream main contains
merge commit `6268a0d6`. GitHub read-back confirmed the exact go.mod and
go.sum update to gRPC-Go v1.83.2 and the explicit Closes #1218 link.
MV-P08 separates this dependency correction from the remaining MV-P05
Go, frontend, documentation and runtime dependency changes.

The latest observed official release is v0.21.0 (2026-10-02), before
the PR merge on 2026-10-11. No official release adoption, downstream
patch removal, product release or production change is established.
Retain the downstream dependency until a suitable immutable release
passes the normal product acceptance gates.

## Machine-readable patch register

This is a source-history-based initial logical patch inventory and must be reconciled against an exact upstream tree, current GitHub issue/PR state and the canonical product contract before it can be called exhaustive or compliant. The missing upstream links intentionally fail the current SDLC policy. Product-specific operational changes should move out of the fork when possible. No divergence ADR is approved, and these records do not grandfather one.

<!-- SDLC-UPSTREAM-REGISTER:1 -->
```json
{
  "schema_version": 1,
  "upstream": "kenn-io/msgvault",
  "baseline": "v0.19.3",
  "patches": [
    {
      "id": "MV-P01",
      "kind": "upstream",
      "scope": "Local Discord observation ingestion in maintained source",
      "local_commits": [
        "cade6dc"
      ],
      "checked_on": "2026-10-10",
      "issue": "https://github.com/kenn-io/msgvault/issues/884",
      "pr": null
    },
    {
      "id": "MV-P02",
      "kind": "upstream",
      "scope": "Preserve observed Discord history and protect retained observations against deletes",
      "local_commits": [
        "8a3dc67"
      ],
      "checked_on": "2026-10-10",
      "issue": null,
      "pr": null
    },
    {
      "id": "MV-P03",
      "kind": "upstream",
      "scope": "Mirrored Discord source capture and provenance persistence; safe message-read projection split into MV-P11",
      "local_commits": [
        "16ef0bd",
        "ee363a4"
      ],
      "checked_on": "2026-10-14",
      "issue": null,
      "pr": null
    },
    {
      "id": "MV-P04",
      "kind": "upstream",
      "scope": "Downstream browser/reference verification and portable documentation capture",
      "local_commits": [
        "728d65c",
        "62c07a6",
        "7d4830e",
        "1f99861"
      ],
      "checked_on": "2026-10-10",
      "issue": null,
      "pr": null
    },
    {
      "id": "MV-P05",
      "kind": "upstream",
      "scope": "Remaining Go dependency and toolchain hardening; excludes gRPC MV-P08, web dependencies MV-P09, documentation dependency MV-P10 and runtime PCRE2 MV-P12",
      "local_commits": [
        "e7b84d7",
        "f063ec7",
        "02c9d30"
      ],
      "checked_on": "2026-10-14",
      "issue": null,
      "pr": null
    },
    {
      "id": "MV-P06",
      "kind": "upstream",
      "scope": "Remaining release and CI contract adaptation with trusted cancellation and verifier parity; portable fixture hydration split into MV-P13",
      "local_commits": [
        "1cf1134",
        "43516c0",
        "453154d",
        "d9f6209",
        "1c7011b"
      ],
      "checked_on": "2026-10-14",
      "issue": null,
      "pr": null
    },
    {
      "id": "MV-P07",
      "kind": "upstream",
      "scope": "Product provenance and maintained downstream identity metadata",
      "local_commits": [
        "51124c8",
        "ccc1666"
      ],
      "checked_on": "2026-10-10",
      "issue": null,
      "pr": null
    },
    {
      "id": "MV-P08",
      "kind": "upstream",
      "scope": "gRPC-Go 1.83.2 dependency correction accepted upstream",
      "local_commits": [
        "da4f187",
        "e7b84d7"
      ],
      "checked_on": "2026-10-11",
      "issue": "https://github.com/kenn-io/msgvault/issues/1218",
      "pr": "https://github.com/kenn-io/msgvault/pull/1220",
      "pr_state": "merged"
    },
    {
      "id": "MV-P09",
      "kind": "upstream",
      "scope": "Pinned web dependencies DOMPurify 3.4.13, PostCSS 8.5.23 and Vitest 4.1.11",
      "local_commits": [
        "da4f187"
      ],
      "checked_on": "2026-10-11",
      "issue": "https://github.com/kenn-io/msgvault/issues/1221",
      "pr": "https://github.com/kenn-io/msgvault/pull/1222",
      "pr_state": "draft"
    },
    {
      "id": "MV-P10",
      "kind": "upstream",
      "scope": "Documentation pymdown-extensions lock update to 11.0.1 with unchanged Zensical constraints",
      "local_commits": [
        "f063ec7",
        "02c9d30"
      ],
      "checked_on": "2026-10-13",
      "issue": "https://github.com/kenn-io/msgvault/issues/1264",
      "pr": "https://github.com/kenn-io/msgvault/pull/1265",
      "pr_state": "draft"
    },
    {
      "id": "MV-P12",
      "kind": "upstream",
      "scope": "Explicit refresh of the existing runtime PCRE2 library from the current Debian security candidate",
      "local_commits": [
        "02c9d30"
      ],
      "checked_on": "2026-10-14",
      "issue": "https://github.com/kenn-io/msgvault/issues/1269",
      "pr": "https://github.com/kenn-io/msgvault/pull/1270",
      "pr_state": "draft"
    },
    {
      "id": "MV-P11",
      "kind": "upstream",
      "scope": "Allowlisted optional source provenance in SQLite/query, HTTP/MCP and daemon message-read paths with generated API/client parity",
      "local_commits": [
        "16ef0bd",
        "ee363a4"
      ],
      "checked_on": "2026-10-14",
      "issue": "https://github.com/kenn-io/msgvault/issues/1268",
      "pr": "https://github.com/kenn-io/msgvault/pull/1275",
      "pr_state": "open",
      "pr_head_sha": "0c390700f3673bf63951f96f1a7099ed1648957a"
    },
    {
      "id": "MV-P13",
      "kind": "upstream",
      "scope": "Hydrate the locked upstream documentation fixture independently of fork origin while preserving ancestry and review/digest checks",
      "local_commits": [
        "d9f6209"
      ],
      "checked_on": "2026-10-14",
      "issue": "https://github.com/kenn-io/msgvault/issues/1276",
      "pr": "https://github.com/kenn-io/msgvault/pull/1277",
      "pr_state": "open",
      "pr_head_sha": "4902082172c8bedff3339740a20f23bc7171e624"
    }
  ]
}
```

## Verified web dependency contribution (2026-10-11)

MV-P09 is separated from the mixed dependency history in da4f187 so the
remaining Go/docs/PCRE2 work is not falsely covered. Existing upstream
issue #1221 is linked by draft PR #1222 with exact head
fbed48cf7562a01b17a48b281753bb98abebd698. The contribution uses the
existing genuine GitHub fork X1pheR/msgvault-1; X1pheR/msgvault is the
independent maintained product and is not in the upstream fork network.
The diff changes only web/package.json and web/bun.lock on official main
6268a0d6. GitHub readback verifies exact fork, branch, issue link and SHA.

All 155 web unit-test files (2483 tests), generated-client checks, Svelte
checks, kit-ui checks and the production web build passed on Node24/Bun1.3.14.
Successful suites were retained across bounded runs; two large-payload
sanitizer/inline-image cases required a 30-second timeout on the 2-CPU host
and then passed unchanged. No product assertions were weakened. The diff
was scrubbed for private downstream names/paths before publication.
Exact file hashes and per-suite receipts are in
/srv/hypershell/reports/sdlc09-20261011/msgvault-web-1221-receipt.json.

The PR remains draft, unmerged and unreleased. No maintained product
source baseline, release or production runtime was advanced. MV-P05
remains blocked until the remaining dependency scope converges.

## Upstream-first exit

For each patch, first check whether the official current upstream source or an existing issue and PR already cover its behavior. Do not open duplicate contributions. Preserve one logical patch for an iterative fix. Publish only a tested upstream-compatible contribution after the repository's established acceptance gates. Once the exact patch merges, require inclusion in an immutable official release and candidate regression evidence before removing the downstream delta. An approved intentional divergence requires a separate operator decision and accepted exact-scope product ADR. End fork maintenance only after release adoption, desired state, rollback readiness, live acceptance and explicit retirement gates. No automatic production or database migration is authorized by this register.

## Documentation dependency contribution — 2026-10-13

MV-P10 separates the retained documentation dependency from the still-blocked Go/PCRE2 remainder of MV-P05. Official issue #1264 is closed by the proposed draft PR #1265, exact contribution head `be488ada58099b2450ebfadf808c809c54c5b588` on official main `3c63f3a979b589c60e154f4b9ec6fdad272f3de3`. Only three lock-entry lines change pymdown-extensions 10.21.3 to 11.0.1 and matching artifact metadata; Zensical remains 0.0.45. `uv lock --check`, `uv sync --frozen` and canonical `make docs-check` passed, including the actual Zensical/site build, built-site validator, validator self-tests and redirect check. Disposable verifier uses uv/Python3.12 image digest `sha256:73f4c257f96061b7c38e1a14416971f41d980fd3cd2837e5e1671140dc4affb9`. Independent static review passed. Evidence: `/srv/hypershell/reports/sdlc09-20261011/msgvault-docs-1264-acceptance.log`, run `run-20261013T224442923031Z-025f2f1fbcbb`; managed publication `run-20261013T224621727034Z-e6d5cdfcf988`; official PR readback confirmed exact SHA, open/draft and Closes #1264. No upstream merge, release adoption or product deployment is claimed.


## Unapproved distribution disposition proposal — MV-P07

This is a proposal for an operator decision, not an approved ADR or a conformance exception. The register row remains unlinked and nonconformant. Exact source candidate: `30287b8d91b80eb2def9f6261e4f073d1b5926bd`; mapped historical commits: `51124c8`, `ccc1666`. Initial/import commits can contain baseline and unrelated application files; approval must not cover the entire mapped commit merely because it appears in this row.

Proposed narrow scope: Only downstream identity/provenance documentation, hypershell.json/upstream.json and the product-owned image-build wrapper. MV-P01 through MV-P06 and all importer, retention, API and dependency behavior are excluded.

Requested decision: retain this precisely bounded product-owned distribution adapter, or move/remove it through the existing source/release/deployment owners when the upstream artifact supplies an equivalent accepted distribution. Our package names, registry targets and operator release controls do not constitute a useful upstream feature request; do not manufacture an issue/PR asking upstream to adopt downstream ownership. Any reusable application or verification improvement discovered in this history still needs its own tested upstream-compatible contribution.

Before an approved disposition can clear a gate, the operator-owned product ADR must identify the exact retained scope, rationale, owner, acceptance evidence and review/exit trigger. Proposed review trigger: the next suitable official upstream release or2026-11-14, whichever comes first. Exit requires equivalent released functionality and distribution, accepted source/runtime/consumer parity, rollback readiness and the existing explicit production/retirement acceptance. This proposal authorizes none of those transitions and approves no missing contribution.

The existing shared repository/release lifecycle invokes the single canonical SDLC checker before managed tag/publication/recovery transitions. Direct public workflow integration remains incomplete: the checker source is private and repository-scoped public GitHub tokens cannot read it. No production credentials, privileged public runner, second checker or extra service was introduced.


## Runtime PCRE2 contribution — 2026-10-14

MV-P12 separates only the explicit runtime PCRE2 installation from the Go toolchain portion of historical commit `02c9d30`. Upstream issue [#1269](https://github.com/kenn-io/msgvault/issues/1269) and draft PR [#1270](https://github.com/kenn-io/msgvault/pull/1270) are read-back verified at exact head `8e585662fbf0f956c26ec9807662d3c15dd8126d`, from the actual contribution fork `X1pheR/msgvault-1`, against official source `3c63f3a979b589c60e154f4b9ec6fdad272f3de3`. The PR closes the exact issue and changes only one Dockerfile runtime dependency line.

The pinned official Debian base contains PCRE2 `10.42-1`. The unchanged runtime package transaction retains it; refreshed APT metadata exposes `10.42-1+deb12u2` from Bookworm Security. This actual package-layer comparison failed before implementation and passed after explicit installation. All existing runtime packages remain installed and a real PCRE2-backed grep operation passes. Independent source/security review passed. Evidence: `msgvault-runtime-pcre2-current-red.log` / `run-20261014T004917970393Z-d02c31c655ee`, `msgvault-runtime-pcre2-green.log` / `run-20261014T005042728552Z-74a7b2befb0b`; accepted package-layer image `sha256:d887fd7785526ad7a50fe71d50fae667b77a5be367b71b279a267326a11aca9a`; publication `run-20261014T005257965948Z-5941533cf526`. This evidence covers the changed native runtime package layer; it does not claim a complete release-image build or production adoption. An earlier metadata-free baseline comparison cannot prove candidate currency and is not accepted RED evidence.

MV-P05 still owns the independent retained Go dependency/toolchain scope. Official v0.21.0 source already uses Go 1.27.1 and newer x/crypto, x/net, x/sys and x/tools versions; this is source comparison, not accepted downstream adoption or patch retirement. No duplicate contribution for already supported dependency versions is justified. The PCRE2 pair does not waive that remaining source convergence or any other patch group. There are now eleven retained groups, four complete active/merged issue/PR pairs and seven nonconformant groups in this product. No downstream release or production state changed.


## Verified safe read-only source provenance contribution — 2026-10-14

MV-P11 separates only sanitized message-read projection from MV-P03 mirror capture. Issue #1268 and draft PR #1275 are verified open at exact head 0c390700f3673bf63951f96f1a7099ed1648957a in the genuine contribution fork X1pheR/msgvault-1, with Closes #1268. The contribution source base is 3c63f3a979b589c60e154f4b9ec6fdad272f3de3; live PR main base at publication is 0b0b8c851599a3d1f1e11e5c68a074c04f2a87c3. MV-P03 still owns source capture and remains unlinked; this narrower pair does not cover importer identity, retention or capture.

The parser publishes exactly kind, repository, repository_url, commit, archive_paths, author_identity and lifecycle_authority from an existing mirror_provenance object, plus optional source_conversation_id. Unknown, malformed, empty and unknown-only objects are omitted. Store/query, HTTP/MCP and daemon adapters expose the same optional typed projection and copied archive paths. It reads no source files or network and grants no authorization. OpenAPI3.15 and generated client/schema parity are accepted. Production/docs and all fixture changes pass independent review.

Accepted native evidence: full go vet005925-8a8d; docs235829-e49cb; parser/store/query/MCP/daemon/client/codegenfix003501-242b and corrected DuckDB003914-cdfc; affected MCP lint/helper/test033248-5db4; parser/query/store lint/helper/tests035215-28cb; API selected-production-file/new-fixture native helper, pinned76-linter/plugin and three realSQLite/engine/copy regressions040107-c59dd0d82a46. Final API fixture SHA256 c96b3cedfd622946201bc56001b5d17b12e6d3de0ef479343456e9db8457b20a. Canonical regeneration040305-256141e10fd2 is byte-identical to staged artifacts; commit040426-025e9a1edd8d and post-commit make api-check040441-ed56bcda272b pass. Managed source modes restored040507-9d2c7; public added-line scrub passes; exact managed publication040522-bd309e1ae02d and typed PR head/link readback pass. All named logs remain in the existing SDLC-09 report directory.

The legacy API test package contains62,917 unrelated test lines; compiling it exceeds bounded native worker memory, including1700MiB013920-86c2 with cgroup OOM. Full make lint-ci helper030120-1e6e also exceeds memory/deadline; broad source-only analysis030503-63c5 ended137 without final cgroup proof. These are failures, not full-green claims. Exact changed-scope native defaultlinter/helper and executable owning regressions are accepted; no rules/assertions were disabled. This remains a draft upstream contribution, not downstream release or production acceptance.

Current product register: twelve retained logical groups, five complete active/merged issue/PR pairs, seven structurally nonconformant groups. The extra logical read slice preserves all retained capture scope. No downstream patch removal, own release or production adoption is claimed.


## Verified upstream fixture hydration — 2026-10-14

MV-P13 separates the portable hydration correction from MV-P06's mixed historical d9f6209 CI/toolchain work. Issue #1276 and draft PR #1277 are verified open at exact head 4902082172c8bedff3339740a20f23bc7171e624, true fork X1pheR/msgvault-1, main base and Closes #1276. Only the controlled fixture fetch changes from origin to the official kenn-io/msgvault URL and separate upstream tracking ref. Locked branch/commit ancestry, manual-review validation, archive/manifest SHA256 and local/offline behavior are preserved. MV-P04 reviewed browser-reference/capture workflow and the remaining MV-P06 release/verifier scope remain unpaired.

A real local-Git regression uses existing reviewed synthetic fixture builders, an empty fork origin and an originless checkout. Network is excluded with file-only protocols and local URL rewriting; digest, branch ref and unchanged origin checks remain strict. Valid RED041015-57f262b57620 failed only missing fixture ref/missingorigin; GREEN041043-9fa33be714fa passed both. Independent source/test review identified ambient fixture-lock isolation; separate fixing context adds the supported --lock flag only, fixtureSHA c7fd1b3dd467be8ce305484627e21b92dd54c321cd4bf6585b75bd5568e27ef0, sourceSHA c278863ca299d43d2a2ed877b77902ec1b4265b39b58b46db07f39ff59420d0f. Incremental review passes. Exact canonical docs-fixture-test Python recipe passes31tests and a separate invalid-ambient-lock regression041226-d3e93d4a5fd7; Bash syntax/whitespace pass. Host make is absent, so failed wrapper041114-ea29 is retained as failure and the unchanged single-command Makefile recipe was run directly. No new checker or verifier was added.

Managed normalization041244-36486; public added-line boundary and verifiedcommit041304-e2ee0f9ad878; exact managedpublication041313-016cb03e6d02 and typedGitHubhead/link pass. Logs msgvault-fixture-origin-1276-*.log are in the existing report directory. Current register has13retained logicalgroups,6complete active/merged pairs and7nonconformant groups. No downstream adoption/release/production change is claimed.
