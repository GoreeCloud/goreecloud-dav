# GoreeCloud DAV — Repository Specifications

## Authority

The canonical GoreeCloud project record is the Google Drive document **Project Specification — DAV** under `GoreeCloud/Projects`.

This repository specification records the source-level implementation contract. It must not be used to inflate planned capabilities into implemented status.

## Identity

- Product: **GoreeCloud DAV**
- Short name: **DAV**
- Repository: `GoreeCloud/goreecloud-dav`
- Development model: original native GoreeCloud software
- Primary language: Go
- License: AGPL-3.0-only
- Lifecycle: Active Development
- Service class: infrastructure/interoperability
- Suite member: no

## Architectural role

GoreeCloud DAV is the external standards interoperability boundary for CalDAV, CardDAV, and the WebDAV functionality those protocols require.

It is not GoreeCloud's general synchronization authority and does not replace GoreeCloud Sync.

Native application data ownership is expected to move behind versioned GoreeCloud service APIs and GoreeCloud Mesh as those contracts become available.

## Current source implementation

The current source implements:

1. Loopback-only default HTTP service and graceful shutdown.
2. `GET /healthz`.
3. `GET /readyz`.
4. `GET /api/v1/status` with explicit non-conformance/migration-required platform status.
5. Development authentication provider abstraction.
6. Loopback-only credentialless development mode and optional development HTTP Basic authentication with storage-compatible development principal validation.
7. RFC 6764 `.well-known` CalDAV/CardDAV redirects and conservative `OPTIONS` method discovery without RFC compliance tokens.
8. `PROPFIND` Depth 0/1 with empty-body / `allprop`, `propname`, and explicit property selection for the supported live-property set.
9. Separate 404 `propstat` handling for unavailable requested properties, bounded PROPFIND bodies, and `DAV:propfind-finite-depth` rejection when omitted/infinite depth would require unsupported infinite traversal.
10. Principal, calendar-home, and address-book-home discovery paths.
11. `MKCALENDAR` for calendar collections.
12. `MKCOL` for address-book collections.
13. `GET`/`HEAD` resources.
14. `PUT` of baseline-validated `.ics` and `.vcf` resources.
15. `DELETE` resources and empty development collections.
16. Deterministic SHA-256 ETags.
17. `If-Match` and `If-None-Match` PUT preconditions at the current HTTP/storage foundation boundary.
18. Calendar-query requiring a CalDAV filter, with CalDAV Depth handling and a safe component-existence filter subset.
19. Addressbook-query requiring a CardDAV filter and Depth header, with a safe property-existence `anyof` / `allof` subset.
20. Calendar-multiget and addressbook-multiget REPORT handling.
21. REPORT namespace validation and authenticated collection-scoped `DAV:href` validation.
22. Same-authority absolute multiget href support, cross-authority rejection, and duplicate-href collapse.
23. One multistatus response per requested multiget resource href, including explicit 404 status responses for missing resources.
24. REPORT property projection that returns supported live properties and calendar/vCard payloads only when requested; report-specific data is not implicitly returned by a propertyless request or `DAV:allprop`.
25. Explicit failure for unsupported richer filters and partial `calendar-data` / `address-data` projection instead of silently broadening result sets or disclosing more data than requested.
26. Filesystem persistence with temporary-file + fsync + rename publication.
27. Restricted path segments, storage-root containment, and fail-closed symlink-path rejection.
28. Bounded resource, PROPFIND, and REPORT bodies.
29. GoreeCloud Platform Contract schema v0.2 manifest with current development/nonconformant integration status and explicit blockers.
30. Immutable-pinned reusable Platform Contract validation.
31. Go tests, vet/build validation, repository-document validation, PR concurrency control, and CI configuration.

The foundation intentionally emits **no `DAV` compliance class/token today**. RFC 4918 class `1` is withheld because this source does not yet satisfy all applicable RFC 4918 MUST-level requirements, including `PROPPATCH`, `COPY`, `MOVE`, and complete WebDAV property behavior. The CalDAV `calendar-access` and CardDAV `addressbook` tokens are likewise withheld until their applicable requirements and interoperability qualification are complete.

## Explicit non-claims

The source does not yet establish:

- WebDAV RFC 4918 class-1 conformance;
- full CalDAV RFC 4791 conformance;
- full CardDAV RFC 6352 conformance;
- `PROPPATCH` and complete WebDAV live/dead property mutation behavior;
- `COPY` and `MOVE` WebDAV behavior;
- atomic compare-and-write conditional mutation across the storage boundary;
- complete CalDAV filter semantics;
- complete CardDAV filter semantics;
- complete partial calendar/address-data projection semantics;
- complete WebDAV ACL;
- RFC 6578 sync-token/change-journal behavior;
- scheduling extensions;
- production authentication;
- production GoreeCloud Identity integration;
- Privacy Shield conformance;
- Wardveil Security conformance;
- Everkeep integration/conformance;
- GoreeCloud Manager integration/conformance;
- GoreeCloud Mesh participation/conformance;
- production TLS or public exposure approval;
- broad client compatibility;
- production packaging/deployment;
- Stable qualification.

## DAV URL model

```text
/dav/
/dav/principals/{principal}/
/dav/calendars/{principal}/
/dav/calendars/{principal}/{collection}/
/dav/calendars/{principal}/{collection}/{resource}.ics
/dav/addressbooks/{principal}/
/dav/addressbooks/{principal}/{collection}/
/dav/addressbooks/{principal}/{collection}/{resource}.vcf
```

Authenticated principal identity must match the principal represented in a DAV path.

## Storage model

The current storage interface separates:

- principals;
- calendar homes;
- address-book homes;
- collections;
- calendar resources;
- contact resources.

The filesystem adapter stores only validated path segments under the configured root, resolves the configured root before use, rejects symlinked storage path components, creates private directories, publishes resource updates atomically at the file-publication level, and derives ETags from resource bytes.

File publication atomicity must not be confused with conditional-write atomicity. The current HTTP handler evaluates `If-Match` / `If-None-Match` before invoking the filesystem write, so compare-and-write enforcement is still a required storage-contract improvement for concurrent writers.

Storage is behind an interface and is replaceable.

## Development authentication

The repository contains a deliberately bounded development provider.

- Credentialless mode is accepted only for loopback requests and resolves to principal `local`.
- Optional HTTP Basic credentials resolve to the configured development username.
- Development usernames are validated against the current filesystem-safe principal identifier rules so authentication cannot succeed with an identifier the development store cannot represent.
- Non-loopback listeners are refused unless development credentials are configured.

This is test/development behavior only. Production identity/authorization is intended to use GoreeCloud Identity and applicable Wardveil/Privacy Shield contracts. Production Identity subject mapping must not be constrained by the development filesystem naming rule.

## Platform Contract

The repository contains `goreecloud.platform.yaml` using GoreeCloud Platform Contract schema version `0.2`.

The declaration currently identifies GoreeCloud DAV as:

- a `service`;
- lifecycle `development`;
- version `unversioned-development`;
- Linux as the currently CI-supported platform;
- `/healthz`, `/readyz`, and `/api/v1/status` as implemented service interfaces;
- GoreeCloud Identity, Privacy Shield, Wardveil Security, Everkeep, GoreeCloud Manager, and GoreeCloud Mesh as applicable but migration-required;
- Glaze UI as not-applicable-justified for the current headless service;
- continuity obligations for backup, restore, export, and portability;
- overall `nonconformant` with explicit blockers and no fabricated acceptance/release evidence.

The reusable validation workflow is pinned to an immutable central commit. Passing manifest validation proves schema/contract consistency for that revision; it does not establish platform conformance, production acceptance, or Stable qualification.

## Platform-system applicability

| System | Applicability | Current source status |
| --- | --- | --- |
| GoreeCloud Identity | Applicable | Migration Required |
| Privacy Shield | Applicable | Migration Required |
| Wardveil Security | Applicable | Migration Required |
| Everkeep | Applicable | Migration Required |
| GoreeCloud Manager | Applicable | Migration Required |
| GoreeCloud Mesh | Applicable | Migration Required |
| Glaze UI | Not Applicable for current headless service | Justified; becomes applicable if a UI is introduced |

## Release rule

This repository remains Active Development until applicable implementation, interoperability, security, privacy, recovery, platform integration, deployment, and exact-release validation gates are satisfied.

A passing build or Platform Contract validation is source/manifest evidence, not production acceptance or review approval.
