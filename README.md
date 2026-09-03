# GoreeCloud DAV

GoreeCloud DAV is the native GoreeCloud CalDAV and CardDAV interoperability service.

It is original GoreeCloud software, not a Radicale fork, wrapper, or rebrand. DAV is designed as an external standards compatibility boundary while GoreeCloud applications use native APIs and GoreeCloud Mesh contracts internally.

## Status

**Active Development — foundation milestone.**

The current source provides a real, testable development foundation. It is **not** a Stable release, production-approved service, complete RFC implementation, or evidence of finished GoreeCloud platform integration.

Implemented now:

- Go HTTP service with loopback-only default.
- Health, readiness, and implementation-status endpoints.
- Development authentication provider boundary.
- Conservative `OPTIONS` behavior that advertises only currently substantiated WebDAV compliance.
- RFC 6764 `/.well-known/caldav` and `/.well-known/carddav` redirects plus DAV principal, calendar-home, and address-book-home discovery.
- `PROPFIND` with supported properties at Depth 0 and 1.
- `MKCALENDAR` for calendar collections.
- `MKCOL` for address-book collections.
- `GET`, `HEAD`, `PUT`, and `DELETE` for supported resources.
- SHA-256 ETags and `If-Match` / `If-None-Match` write protection.
- Baseline `calendar-query`, `calendar-multiget`, `addressbook-query`, and `addressbook-multiget` REPORT handling.
- Atomic filesystem persistence.
- Request-size and path-segment validation.
- Automated tests and CI definition.

See [FEATURES.md](FEATURES.md) for the implemented/planned split and [SPECIFICATIONS.md](SPECIFICATIONS.md) for conformance boundaries.

## Product boundary

GoreeCloud DAV is for CalDAV/CardDAV/WebDAV interoperability.

It does **not** replace GoreeCloud Sync. GoreeCloud Sync remains the authority for approved file/folder replication, Nearby transfer, and temporary sharing.

Long-term:

```text
External DAV clients
        |
 CalDAV / CardDAV
        |
  GoreeCloud DAV
        |
 Native APIs / GoreeCloud Mesh
   |        |        |
Calendar  Contacts  Tasks
```

The foundation uses a filesystem store so protocol behavior can be developed and tested independently. Native GoreeCloud datastore adapters are planned; the filesystem is not declared to be the permanent application source of truth.

## Development use

Requirements:

- Go 1.23+
- No external Go module dependencies are required by the current foundation.

Run:

```sh
go run ./cmd/goreecloud-dav
```

The default listener is `127.0.0.1:5232` and the default data directory is `./data`.

With no configured development credentials, only loopback requests are accepted and the development principal is `local`.

Example URLs:

```text
http://127.0.0.1:5232/dav/principals/local/
http://127.0.0.1:5232/dav/calendars/local/
http://127.0.0.1:5232/dav/addressbooks/local/
```

To require development HTTP Basic authentication, set both:

```text
GOREECLOUD_DAV_USERNAME
GOREECLOUD_DAV_PASSWORD
```

The service refuses a non-loopback listener unless both development credentials are configured. **Development Basic authentication is transitional and is not GoreeCloud Identity integration or production authentication.**

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `GOREECLOUD_DAV_LISTEN` | `127.0.0.1:5232` | Development HTTP listener |
| `GOREECLOUD_DAV_DATA_DIR` | `./data` | Filesystem storage root |
| `GOREECLOUD_DAV_USERNAME` | empty | Optional development Basic Auth user/principal |
| `GOREECLOUD_DAV_PASSWORD` | empty | Optional development Basic Auth password |
| `GOREECLOUD_DAV_MAX_BODY_BYTES` | `10485760` | Maximum DAV resource request body |

Do not commit real credentials.

## Validation

```sh
sh ./scripts/check-repository-docs.sh
test -z "$(gofmt -l .)"
go test ./...
go vet ./...
go build ./cmd/goreecloud-dav
```

## GoreeCloud platform systems

Current source status is intentionally conservative:

- **GoreeCloud Identity:** Applicable — Migration Required.
- **Privacy Shield:** Applicable — Migration Required.
- **Wardveil Security:** Applicable — Migration Required.
- **Everkeep:** Applicable — Migration Required.
- **GoreeCloud Manager:** Applicable — Migration Required.
- **GoreeCloud Mesh:** Applicable — Migration Required.
- **Glaze UI:** Not Applicable — Justified for the current headless service. It becomes mandatory if a user-facing/admin interface is introduced.

Status endpoints do not convert these into conformance claims.

## Standards targets

The project targets applicable portions of WebDAV, CalDAV, CardDAV, iCalendar, vCard, WebDAV ACL, WebDAV Sync, and DAV service discovery. The foundation intentionally does **not** advertise the `calendar-access` or `addressbook` DAV compliance tokens because those tokens imply protocol requirements the project has not yet fully qualified. Compatibility must be demonstrated through tests and client interoperability evidence before stronger claims are made.

## License

GoreeCloud DAV is licensed under **GNU AGPL-3.0-only**. Third-party dependencies, if introduced, retain their applicable licenses.

See [LICENSE](LICENSE).
