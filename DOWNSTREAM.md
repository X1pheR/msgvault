# Downstream maintenance

## Identity and provenance

This repository is the X1pheR-maintained downstream of `kenn-io/msgvault`.

- Upstream repository: <https://github.com/kenn-io/msgvault>
- Current upstream baseline: `v0.19.3`
- Baseline commit: `e90bcdc5eeaeaf387ff68872601d0c382c184a9b`
- Downstream repository: <https://github.com/X1pheR/msgvault>
- First public downstream release: `v0.19.3-x1pher.3`
- Upstream convergence issue: <https://github.com/kenn-io/msgvault/issues/884>
- License: upstream MIT license retained unchanged

`upstream.json` is the machine-readable copy of this baseline. Upstream remains the product origin and primary source of general msgvault behavior. This downstream owns only the source-level delta and its regression evidence.

## Owned delta

The downstream source adds the bounded local Discord observation path already represented by the downstream tests:

- versioned JSONL `container`, complete `message`, and explicit `delete` observations;
- `source_type=discord_local` isolation while archived messages keep Discord message/content semantics;
- deterministic guild and account-scoped source binding with fail-closed mismatch checks;
- normal msgvault persistence, FTS, reply, attachment-metadata, tombstone, and export behavior rather than direct database side writes;
- explicit read-only MCP registration that omits stateful/export mutation tools;
- daemon CLI admission for `import-discord-observations`, preserving one archive writer;
- transactional observed research history and explicit local read-only history retrieval;
- terminal local deletion provenance with atomic current/raw/prior-body scrub and stale replay prevention.

It does not add Discord credential acquisition, user-token/selfbot support, native history pagination, a local-observation sync cursor, an ingest daemon, a second archive or attachment-binary acquisition. External acquisition configuration and backup scheduling remain deployment-owned.

## Versioning

Downstream release tags use:

`v<upstream-version>-x1pher.<revision>`

Rules:

1. The upstream semantic version remains visible in every downstream release.
2. A downstream-only change on the same upstream base increments `<revision>`.
3. Moving to a new upstream release creates a new downstream version on that base after compatibility and regression acceptance.
4. Accepted release tags are immutable; never rewrite an accepted release to follow upstream.
5. The first public release is `v0.19.3-x1pher.3`; the `x1pher.N` suffix is the maintained downstream release identity.

The inherited upstream Docker publication workflow derives its registry from `github.repository`. Versioned downstream images therefore publish under `ghcr.io/x1pher/msgvault`. For `v0.19.3-x1pher.3`, the semver image tag is `ghcr.io/x1pher/msgvault:0.19.3-x1pher.3`. Production consumers must pin an exact downstream version tag. Moving tags such as `latest` are never production selectors; the resolved image digest is release/provenance evidence.

## Local verification

Run the downstream acceptance entrypoint before a release or upstream-base change:

```bash
./scripts/verify-downstream.sh
```

The verifier checks formatting, vets the touched package surfaces, runs every downstream-owned SDD regression plus the modified daemon-admission regression with msgvault's required fts5 sqlite_vec build tags, builds the real CLI, and smoke-checks the downstream commands. The inherited upstream CI/release workflows remain responsible for the broader upstream test and packaging suites.

A release candidate must also build from this repository's upstream-owned `Dockerfile` and pass the packaged CLI smoke checks. Do not copy Docker build logic into a consumer repository.

## Upstream tracking

After the first release, use two remotes:

```text
origin   https://github.com/X1pheR/msgvault.git
upstream https://github.com/kenn-io/msgvault.git
```

Keep downstream `main` as upstream history plus ordinary downstream source commits. Do not maintain a generated patch as the source of truth.

For an upstream upgrade:

1. `git fetch upstream --tags`
2. Create a disposable upgrade branch/worktree from the accepted downstream `main`.
3. Rebase the downstream commit range onto the candidate upstream release, for example `git rebase --onto v0.20.0 v0.19.3`.
4. Resolve only real source/layout conflicts; do not weaken tests to make the rebase pass.
5. Run `./scripts/verify-downstream.sh`.
6. Build and smoke-test the candidate image with the new downstream version.
7. Measure current upstream `main` separately when useful; compatibility with `main` is evidence, not a production pin.
8. Promote only after the new upstream base and downstream behavior are both accepted.

If upstream implements an equivalent seam, first characterize the upstream behavior against the downstream regression tests. Remove redundant downstream source only after equivalence is proven.

## Release flow

1. Accept the exact local source tree and downstream verifier.
2. At release transition, create the Git commit from that exact accepted tree and tag the accepted downstream version.
3. Publish only that accepted commit/tag to `X1pheR/msgvault`.
4. Let inherited repository workflows re-run upstream build/test/release checks and publish versioned artifacts/images.
5. Record the release commit and resolved image digest.
6. Change deployment configuration separately to consume the exact accepted versioned image.
7. Verify the live consumer before removing its previous rollback artifact.

No upstream pull request is created automatically. Issue `kenn-io/msgvault#884` remains the alignment/convergence route; any upstream PR is a separate maintainer-alignment decision.

## Lifecycle candidate security and recovery

The lifecycle candidate retains upstream v0.19.3 while updating the indirect golang.org/x/mod dependency to v0.40.0 after the source scanner reported CVE-2026-56864 and CVE-2026-56865 at v0.38.0. The Go module graph also advances x/tools to v0.49.0 and x/telemetry to the selected August 2026 revision. Regression and packaged-build gates must pass for the exact candidate. The optional inherited screenshot Dockerfile has an explicit, expiring non-production disposition in .trivyignore.yaml; production remains non-root.

The maintained verifier includes passive archive backup/recovery, schema/logical parity, observed history/tombstones and read-only restored queries. It uses no provider client. Production backup/recovery acceptance remains the deployment owner's responsibility and must not start acquisition during restore.

Packaged-runtime acceptance additionally identified inherited Go 1.26.5 standard-library and Debian PCRE2 fixes. The candidate build/verifier pins official Go 1.26.6 by manifest digest and explicitly installs the current Bookworm PCRE2 security package during runtime assembly. Image scanning and exact packaged acceptance must pass before publication.

The immutable v0.19.3-x1pher.4 release exposed a packaging mismatch: standalone release assets derive Go from go.mod, which still declared 1.26.5. The corrective release advances the module minimum to 1.26.6, keeping standalone assets, CI, the verifier and Docker packaging on the accepted patched toolchain. The earlier release/tag is not rewritten.

The corrective candidate aligns the module and Nix toolchains with the already accepted Go 1.26.6 runtime and refreshes the Nix vendor hash for the selected dependency graph. Documentation fixtures retain upstream authority, the locked commit and both content digests. The inherited low-memory regression accepts a legacy query that fits on a given platform only after exact row-count validation; the production streaming build still must produce all one million rows under 96MB.

Documentation evidence also fetches both upstream-owned asset branches explicitly and selects the existing local-asset hydration mode. This preserves expected-asset, document build and pixel-comparison checks without publishing duplicate downstream asset branches.

Owned lifecycle tests follow the inherited local testify-helper lint policy. The inherited marked-CLI cancellation fixture now supplies the existing explicit test API key, so it exercises the trusted marked request policy rather than racing a five-millisecond deadline on an untrusted request. No production timeout or authorization policy is relaxed.

The upstream static branch later refreshed its screenshots for 0.20 (`cba6c00`), beyond the accepted v0.19.3 product baseline. Downstream evidence pins the original static asset revision `60b0a07ee9c0b8fc4341b41a3a60c65a731e31c4` and generated asset revision `dedad6fe03b2b6d533011e970921ab303a951c80`; existing pixel tolerances remain unchanged and compatibility must be proved by the evidence workflow.
