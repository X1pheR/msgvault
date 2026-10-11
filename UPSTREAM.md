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
      "scope": "Mirrored Discord source provenance and sanitized read-only message provenance",
      "local_commits": [
        "16ef0bd",
        "ee363a4"
      ],
      "checked_on": "2026-10-10",
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
      "scope": "Remaining Go, documentation Python and runtime PCRE2 dependency hardening; excludes gRPC MV-P08 and web dependencies MV-P09",
      "local_commits": [
        "da4f187",
        "e7b84d7",
        "f063ec7",
        "02c9d30"
      ],
      "checked_on": "2026-10-10",
      "issue": null,
      "pr": null
    },
    {
      "id": "MV-P06",
      "kind": "upstream",
      "scope": "Release and CI contract adaptation with trusted cancellation and verifier parity",
      "local_commits": [
        "1cf1134",
        "43516c0",
        "453154d",
        "d9f6209",
        "1c7011b"
      ],
      "checked_on": "2026-10-10",
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
