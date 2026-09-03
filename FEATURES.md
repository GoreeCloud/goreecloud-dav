# GoreeCloud DAV Features

This file distinguishes implemented source behavior from planned work.

## Implemented in the foundation

- Native Go DAV service.
- Loopback-only development default.
- Health, readiness, and implementation-status endpoints.
- Development authentication provider boundary with filesystem-compatible development principal validation.
- RFC 6764 `.well-known` CalDAV/CardDAV redirects.
- Conservative `OPTIONS` method discovery with RFC 4918, CalDAV, and CardDAV compliance tokens withheld until their applicable MUST-level requirements are implemented and qualified.
- Principal/calendar/address-book discovery.
- `PROPFIND` Depth 0/1 with empty-body / `allprop`, `propname`, and explicit `prop` request handling.
- Separate 404 `propstat` responses for unavailable requested properties.
- Bounded `PROPFIND` bodies and conservative `DAV:propfind-finite-depth` rejection when omitted or infinite depth would require unsupported infinite traversal.
- Calendar collection creation with `MKCALENDAR`.
- Address-book collection creation with `MKCOL`.
- iCalendar and vCard resource storage.
- `GET` and `HEAD`.
- `PUT` with resource-size limits and baseline format validation.
- `DELETE` for resources and empty collections.
- SHA-256 ETags.
- `If-Match` and `If-None-Match` PUT preconditions at the current HTTP/storage foundation boundary.
- Calendar-query with required filter parsing, CalDAV Depth handling, and a safe component-existence filter subset.
- Addressbook-query with required filter/Depth handling and a safe property-existence `anyof` / `allof` subset.
- Calendar-multiget and addressbook-multiget REPORT handling.
- REPORT namespace and collection-scoped `DAV:href` validation.
- Same-authority absolute multiget href support and rejection of cross-authority absolute hrefs.
- Duplicate multiget href collapse so one requested resource href appears once in a multistatus response.
- Per-requested-resource multiget responses, including explicit 404 status responses for missing resources.
- REPORT property projection that returns calendar/vCard payloads only when `calendar-data` / `address-data` are explicitly requested.
- Separate 404 report `propstat` responses for unknown requested properties.
- Explicit failure for richer unsupported query filters and partial calendar/address-data projections instead of silently broadening results or returning excess data.
- Atomic filesystem file publication with temporary-file + fsync + rename.
- Restricted path segments, storage-root containment, and fail-closed rejection of symlinked storage path components.
- GoreeCloud Platform Contract v0.2 declaration with truthful development/nonconformant platform-system status.
- Immutable-pinned reusable Platform Contract validation.
- Automated tests and CI, including PR-level concurrency cancellation for obsolete heads.

## Planned / not yet implemented

- Full RFC 4918 WebDAV class-1 behavior and qualification.
- `PROPPATCH` and complete live/dead WebDAV property behavior.
- `COPY` and `MOVE` WebDAV behavior.
- Atomic compare-and-write enforcement for conditional resource mutations; the current HTTP precondition check and filesystem publication are not yet one storage transaction.
- Complete CalDAV filtering, including recurrence-aware and time-range semantics.
- Complete CardDAV filtering, including text, parameter, and negative-match semantics.
- Complete selective / partial `calendar-data` and `address-data` projection semantics.
- WebDAV ACL.
- RFC 6578 sync tokens and incremental change journal.
- Scheduling/invitation extensions.
- Native Calendar, Contacts, and Tasks datastore adapters.
- Production GoreeCloud Identity authentication/authorization.
- Privacy Shield data-governance enforcement.
- Wardveil Security trust/security integration.
- Everkeep backup/restore integration.
- GoreeCloud Manager operational integration.
- GoreeCloud Mesh discovery/events/capability contracts.
- Client compatibility qualification.
- Performance and scale qualification.
- Production packaging and deployment acceptance.
- Stable release qualification.
- Glaze UI administration only if a human-facing web interface is later introduced.
