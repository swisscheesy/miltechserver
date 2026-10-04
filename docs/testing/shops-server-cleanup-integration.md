# Shops server local integration into cleanup

Date: 2026-10-04. The user authorized merging the reviewed server remediation into the current `cleanup` branch. No client source change, named database operation, push, deployment or capability activation is included. Automatic tagged Jet generation remains at startup; generated sources are not hand-edited.

Base: `518ef4e828abeb64c203e567287b730799222da7`. Candidate: `codex/shops-server-release-remediation-20261003`. Reviewed source SHA-256: `843b5f28f2b970d4254a8c4a30f83cbf5b53f93932722be37ff42fef50511f49`, 753 selected inputs. All 32 tracked PMCS generated files match the base. Current final review records are archived in [review evidence](../reviews/2026-10-04-shops-server-remediation/README.md); original checkpoints/logs are preserved in the remediation worktree.

The two existing dirty primary CodeGraph files are unrelated and excluded. Canonical tagged-Jet output from the reviewed full disposable candidate will be carried byte-for-byte into the merged checkout for build inputs, with source/PMCS equality checks. This is generated output transfer, not manual model editing or generation against an existing named target. The integration matrix regenerates tagged models in its protected disposable fixtures after planned database actions. The worktree-only publication guard remains unchanged.

Merge and fresh merged-tree verification are pending in this checkpoint. The final verification record will replace this paragraph after execution.

The [client API handoff](shops-server-client-api-changes.md) enumerates endpoint changes, reasons, client requirements and JSON examples without claiming a client implementation review. Outstanding design/code and operator actions are listed there and in [release gates](shops-server-remediation-release.md). Local test success does not close those gates.
