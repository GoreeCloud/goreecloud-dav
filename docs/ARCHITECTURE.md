# GoreeCloud DAV Architecture

## Role

GoreeCloud DAV is a protocol gateway and interoperability service.

```text
CalDAV/CardDAV clients
        |
        v
+---------------------+
|   GoreeCloud DAV    |
| HTTP / DAV engine   |
| auth boundary       |
| storage boundary    |
+----------+----------+
           |
           | future versioned native contracts
           v
  GoreeCloud Mesh / APIs
     |       |       |
 Calendar Contacts Tasks
```

## Current source layers

### `cmd/goreecloud-dav`

Process startup, configuration loading, HTTP server limits, signal handling, and graceful shutdown.

### `internal/config`

Environment configuration and the fail-safe listener rule. A wildcard/non-loopback listener is rejected when development credentials are absent.

### `internal/auth`

Authentication abstraction.

The current `DevelopmentProvider` is deliberately transitional. It supports:

- credentialless loopback development principal `local`; or
- configured development HTTP Basic credentials.

Production authority belongs to GoreeCloud Identity and applicable security/privacy contracts.

### `internal/dav`

HTTP/DAV request routing and baseline protocol behavior:

- health/readiness/status;
- authenticated DAV request boundary;
- URL/principal isolation;
- RFC 6764 well-known redirects;
- `OPTIONS` method discovery with RFC compliance tokens withheld;
- PROPFIND;
- MKCALENDAR/MKCOL;
- GET/HEAD/PUT/DELETE;
- baseline query and multiget REPORT behavior;
- report namespace and in-scope href validation;
- per-requested-resource multiget status responses;
- DAV multistatus XML;
- ETag PUT preconditions;
- body limits and resource validation.

### `internal/storage`

Replaceable storage interface and filesystem adapter.

The filesystem adapter:

- restricts path segments;
- resolves and confines paths to the configured root;
- rejects symlinked storage path components and symlink entries;
- uses private directories/files;
- uses temporary-file + fsync + rename publication;
- derives ETags from SHA-256 of stored resource bytes.

The symlink checks are defense in depth for local filesystem tampering; they do not convert the development filesystem adapter into a production authorization boundary.

## Data ownership

Filesystem persistence is a development-stage implementation, not a declaration that GoreeCloud Calendar/Contacts/Tasks must use DAV files as their permanent source of truth.

Native application adapters should eventually translate between DAV resources and authoritative application/service contracts.

## Platform integration boundaries

- GoreeCloud Identity: actor and authorization authority.
- Privacy Shield: permitted information use.
- Wardveil Security: trust/security enforcement.
- Everkeep: recovery and continuity.
- GoreeCloud Manager: operational control and health.
- GoreeCloud Mesh: first-party service discovery/coordination.
- Glaze UI: not applicable to the current headless service; mandatory for any future human-facing interface.

## Non-goals of the foundation

The foundation intentionally does not claim RFC 4918 class-1 conformance, CalDAV/CardDAV compliance-token qualification, production security, production deployment, or Stable status.
