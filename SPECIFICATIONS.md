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
6. Loopback-only credentialless development mode and optional development HTTP Basic authentication.
7. RFC 6764 `.well-known` CalDAV/CardDAV redirects and conservative `OPTIONS` behavior.
8. `PROPFIND` Depth 0/1 for a supported baseline property set.
9. Principal, calendar-home, and address-book-home discovery paths.
10. `MKCALENDAR` for calendar collections.
11. `MKCOL` for address-book collections.
12. `GET`/`HEAD` resources.
13. `PUT` of baseline-validated `.ics` and `.vcf` resources.
14. `DELETE` resources and empty development collections.
15. Deterministic SHA-256 ETags.
16. `If-Match` and `If-None-Match` write preconditions.
17. Baseline calendar/address-book query and multiget REPORTs.
18. Filesystem persistence with atomic file publication.
19. Restricted path segments and storage-root containment.
20. Bounded resource and REPORT bodies.
21. Go tests, vet/build validation, repository-document validation, and CI configuration.

The implementation intentionally advertises only `DAV: 1` today. It does not emit `calendar-access` or `addressbook` compliance tokens until the relevant MUST-level requirements are implemented and interoperability-qualified.

## Explicit non-claims

The source does not yet establish:

- full WebDAV RFC 4918 conformance;
- full CalDAV RFC 4791 conformance;
- full CardDAV RFC 6352 conformance;
- complete WebDAV property mutation/dead-property behavior;
- complete WebDAV ACL;
- RFC 6578 sync-token/change-journal behavior;
- scheduling extensions;
- complete CalDAV filter semantics;
- complete CardDAV filter semantics;
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

The filesystem adapter stores only validated path segments under the configured root, creates private directories, publishes resource updates atomically, and derives ETags from resource bytes.

Storage is behind an interface and is replaceable.

## Development authentication

The repository contains a deliberately bounded development provider.

- Credentialless mode is accepted only for loopback requests and resolves to principal `local`.
- Optional HTTP Basic credentials resolve to the configured development username.
- Non-loopback listeners are refused unless development credentials are configured.

This is test/development behavior only. Production identity/authorization is intended to use GoreeCloud Identity and applicable Wardveil/Privacy Shield contracts.

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

A passing build is source evidence, not production acceptance.
