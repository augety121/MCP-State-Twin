# Phase 7: v1.0 Stability and Maintainer Contract

- **Target:** `v1.0.0`
- **Entry:** at least one production-quality consumer of the supported local
  profile and release evidence across multiple preview releases

## Required stable contracts

- TwinSpec schema and compatibility policy;
- canonical JSON/digest contract;
- Evidence and TwinBundle formats;
- supported storage migrations;
- CLI and control/data-plane API policy;
- ResourceProfile and HostProfile versioning;
- deprecation and removal windows;
- supported Go/platform matrix;
- security response and disclosure process;
- release signing, checksums, SBOM and provenance;
- maintainer/contributor governance and regression policy.

## Release conditions

Every stable claim must have current evidence from the exact commit. Breaking
format or semantic changes require an accepted ADR, migration guidance and a
major version. A v1 release is defined by narrow reliable contracts, not by
shipping every roadmap item.

## Explicitly independent future work

Cloud multi-tenancy, distributed storage, arbitrary native Twin plugins,
marketplaces, general A2A orchestration and production service mirroring remain
separate RFCs. They are not implied by v1.0.
