# GoreeCloud DAV Security

## Status

GoreeCloud DAV is under active development and is **not production-approved**.

The current authentication provider is explicitly a development boundary. It is not GoreeCloud Identity integration and must not be treated as a production security control.

## Development safety defaults

Current source:

- listens on `127.0.0.1:5232` by default;
- refuses a non-loopback listener without configured development credentials;
- accepts credentialless development DAV access only from a loopback remote address;
- bounds resource request bodies;
- bounds REPORT bodies;
- restricts user-controlled storage path segments;
- resolves the configured filesystem root before use;
- rejects symlinked storage path components and symlink entries before DAV reads/writes;
- prevents accepted storage paths from escaping the configured data root;
- writes resources atomically;
- generates deterministic ETags;
- supports conditional PUT protections;
- validates multiget DAV hrefs against the authenticated principal and requested collection;
- avoids logging DAV resource bodies;
- emits basic browser-oriented defensive response headers.

The symlink checks are defense in depth against local filesystem tampering. They are not a substitute for host permissions, application authorization, GoreeCloud Identity, or Wardveil Security.

## Production gaps

Before production or external exposure, the project still requires applicable:

- GoreeCloud Identity authentication and authorization;
- Wardveil Security integration and review;
- Privacy Shield governance;
- TLS/reverse-proxy deployment validation;
- rate limiting/abuse controls;
- credential lifecycle controls;
- audit/security event contracts;
- backup and restore validation through Everkeep;
- threat modeling;
- client interoperability/security testing;
- production configuration and secret handling;
- exact-release security acceptance.

## Sensitive information

Never commit:

- passwords;
- bearer tokens;
- private keys;
- session secrets;
- real contact/calendar datasets;
- production configuration containing credentials;
- exported private GoreeCloud account data.

## Reporting

Use the repository's private vulnerability-reporting mechanism when available. Do not place undisclosed vulnerabilities or sensitive reproduction data in a public issue.
