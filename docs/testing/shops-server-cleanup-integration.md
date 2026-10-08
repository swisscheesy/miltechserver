# Shops server local integration into cleanup

Date: 2026-10-04. The user authorized merging the reviewed server remediation into the current `cleanup` branch. No client source change, named database operation, push, deployment or capability activation is included. Automatic tagged Jet generation remains at startup; generated sources are not hand-edited.

Final documentation checkpoint: the merge was reconfirmed through Git as `Already up to date`; the reviewed branch is fully included in `cleanup`. At the user's instruction, server tests/builds/database checks were not rerun in this final pass. The results below are earlier recorded evidence. See the [final server change report](shops-server-refactor-final-report.md) and finalized [client endpoint handoff](shops-server-client-api-changes.md).

Base: `518ef4e828abeb64c203e567287b730799222da7`. Candidate: `codex/shops-server-release-remediation-20261003`. Reviewed source SHA-256: `843b5f28f2b970d4254a8c4a30f83cbf5b53f93932722be37ff42fef50511f49`, 753 selected inputs. All 32 tracked PMCS generated files match the base. Current final review records are archived in [review evidence](../reviews/2026-10-04-shops-server-remediation/README.md); original checkpoints/logs are preserved in the remediation worktree.

The two existing dirty primary CodeGraph files are unrelated and excluded; no overwrite/reset was performed. Canonical tagged-Jet output from the reviewed full disposable candidate was carried byte-for-byte into the merged checkout: all 261 public generated files matched, all 753 selected source inputs matched, and all 32 tracked PMCS outputs matched HEAD. This is generated output transfer, not manual model editing or generation against an existing named target. The integration matrix regenerates tagged models in protected disposable fixtures after planned database actions. The worktree-only publication guard remains unchanged. Generated lock/provenance files are untracked local artifacts, not committed sources.

`cleanup` was fast-forwarded locally from `518ef4e` to **`2e58cd8522f424a88193814530dbf190d0fad794`**. A subsequent docs-only commit records the completed integration. Nothing was pushed. The source digest remains the reviewed value above.

## Fresh merged-tree verification

Exact commands, working directory, independent exits/times, test names and log hashes are retained in [verification JSON](shops-server-cleanup-verification.json). Logs and command records are preserved in the remediation worktree's `.superpowers/sdd/2026-10-03-shops-server-release-remediation/cleanup-integration/` evidence directory. The recorder unsets inherited test selectors, disables module network access, uses local Go 1.23.3 and the warm verification cache. Only the marker-protected loopback disposable wrapper executes database suites; dotenv-backed unrelated integration packages were not run.

| Check | Final result |
| --- | --- |
| Safe host `go test -json -count=1 ./api/... ./bootstrap/... ./tools/... ./internal/... .` | Exit 0; 619 top-level / 1308 including subtests PASS; three inherited PSMag database skips |
| Shops/equipment/shared middleware unit race suite | Exit 0; 147 / 437 PASS; no skips |
| `go build ./...` and scoped `go vet` | Both exit 0 |
| Full protected integration, notification/sync/remediation migration matrices, release runner and all performance fixtures | Final exit 0; 303 / 1002 PASS; no skips; tagged regeneration after planned database actions |
| Protected physical concurrency/race selection | Exit 0; 34 / 132 PASS; no skips; exact selection in JSON |
| Python isolation / guarded runner / publication / container static tests | 11 / 13 / 4 / 3 PASS |
| Client handoff JSON examples | All 73 parse as valid JSON |

The publication test is intentionally tied to the approved linked worktree. It was run there against the same committed source bytes; the guard was not weakened to publish into arbitrary checkouts. All application checks above ran from the merged primary source (the protected wrapper compiles its exact disposable source copy).

**First full matrix was not green:** exit 1, `TestCleanupCascadeAndFinalMember/reply`, reported one active transaction in its database-wide `pg_stat_activity` assertion. The case then passed 20 repeats; 200 repeats with temporary pool/activity diagnostics and a full diagnostic sequence passed. The temporary test changes were restored exactly before the final full matrix/race run. The failure was not reproduced with a diagnostic observation, so its cause is **unresolved/intermittent**; no assertion was weakened and no production fix is claimed. Final rerun success does not prove test reliability.

**Docker acceptance remains incomplete:** the local daemon now answered; context and image builds progressed. The merged scanner recursively expanded embedded Go TAR fixtures, including a 60 GB logical sparse member. It ran for 1111.451 seconds at roughly 3 GB RSS/one CPU before the controller terminated only its identified scanner child; the cleanup trap ran and the invocation recorded exit 143. This is not a container/runtime PASS. Real SDK/PostgreSQL startup and real credential mounts remain unexecuted.

An additional scanner repair and seven regression tests were developed and narrowly re-reviewed in the isolated worktree. The original malformed-header/traversal cases were RED and the repaired cases GREEN; the saved images scanned in 0.326 seconds with zero literal sentinels. **Those two additional files/changes are uncommitted and excluded from `cleanup` following the user's scope correction.** They are available for a separate tooling change; no complete repaired Docker runtime run is claimed.

## Why shell/Python files were included

The merged remediation contains **10 script changes (8 added, 2 modified)**. They are verification/migration tooling, not the Shops application implementation:

| Purpose | Files |
| --- | --- |
| Guarded release migration and disposable rehearsal | `apply-shops-remediation-migrations.sh`, `verify-shops-remediation-migrations.sh`, existing `test-shops-isolated.sh` |
| Tests of migration/isolation refusal behavior | `apply-shops-remediation-migrations_test.py`, existing `test-shops-isolated_test.py` |
| Exact reviewed-source and tagged generated-output publication checks | `shops_candidate_inputs.py`, `publish-shops-candidate-generated.py`, `test-publish-shops-candidate-generated.py` |
| Synthetic container packaging checks | `test-server-container-inputs.sh`, `test-server-container-inputs-static.py` |

All are under `scripts/`. Adding them did not apply an existing-target migration or run them at normal application startup. The additional unmerged scanner repair is `scripts/test-server-container-inputs.sh` plus new `scripts/test-server-container-archive-scan.py` in the isolated worktree. Further harness work stopped when the user questioned that scope expansion.

The [client API handoff](shops-server-client-api-changes.md) enumerates endpoint changes, reasons, client requirements and JSON examples without claiming a client implementation review. Outstanding design/code and operator actions are listed there and in [release gates](shops-server-remediation-release.md). Local test success does not close those gates.
