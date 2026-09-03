# GoreeCloud DAV Features

This file distinguishes implemented source behavior from planned work.

## Implemented in the foundation

- Native Go DAV service.
- Loopback-only development default.
- Health, readiness, and implementation-status endpoints.
- Development authentication provider boundary.
- RFC 6764 `.well-known` CalDAV/CardDAV redirects.
- Conservative DAV `OPTIONS` advertising only currently substantiated WebDAV compliance.
- Principal/calendar/address-book discovery.
- `PROPFIND` Depth 0/1 baseline properties.
- Calendar collection creation with `MKCALENDAR`.
- Address-book collection creation with `MKCOL`.
- iCalendar and vCard resource storage.
- `GET` and `HEAD`.
- `PUT` with resource-size limits and baseline format validation.
- `DELETE` for resources and empty collections.
- SHA-256 ETags.
- `If-Match` and `If-None-Match`.
- Baseline calendar-query and calendar-multiget REPORTs.
- Baseline addressbook-query and addressbook-multiget REPORTs.
- Atomic filesystem resource writes.
- Restricted path segments and storage-root containment.
- Automated tests and CI definition.

## Planned / not yet implemented

- Full WebDAV property model and property mutation.
- Full CalDAV/CardDAV filtering semantics.
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
