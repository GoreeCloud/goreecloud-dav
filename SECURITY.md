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
- prevents storage paths from escaping the configured data root;
- writes resources atomically;
- generates deterministic ETags;
- supports conditional write protections;
- avoids logging DAV resource bodies;
- emits basic browser-oriented defensive response headers.

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
