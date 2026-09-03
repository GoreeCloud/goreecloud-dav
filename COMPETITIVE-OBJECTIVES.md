# GoreeCloud DAV Competitive Objectives

## Benchmark

Radicale is a useful public benchmark for lightweight CalDAV/CardDAV interoperability and simple self-hosting.

GoreeCloud DAV is **not** intended to copy Radicale source, architecture, branding, product identity, or user interface. It is an original GoreeCloud implementation of public interoperability standards.

## Objectives

GoreeCloud DAV should eventually:

- remain lightweight enough for small self-hosted deployments;
- interoperate reliably with standards-compliant CalDAV and CardDAV clients;
- provide predictable collection and resource behavior;
- provide strong conditional-update and conflict protections;
- integrate with GoreeCloud Identity instead of owning a permanent parallel account system;
- integrate privacy decisions through Privacy Shield;
- integrate security/trust behavior through Wardveil Security;
- expose recoverability through Everkeep rather than calling synchronization backup;
- expose operational state through GoreeCloud Manager;
- use GoreeCloud Mesh for approved first-party coordination;
- keep native GoreeCloud application APIs richer than the external DAV compatibility surface;
- provide portable data and migration paths;
- remain independently maintainable without dependence on another DAV product's release cycle.

## Current competitive status

The current repository is a foundation milestone only.

It implements a useful subset of DAV storage/discovery operations but has **not** demonstrated full protocol conformance, mature client compatibility, production security integration, production recovery integration, or feature parity with established DAV servers.

## Legal and provenance boundary

Competitive work may use public standards, public behavior, public documentation, and lawfully usable foundational libraries.

It must not copy protected branding, proprietary assets, confidential information, incompatible source code, or an upstream product architecture as GoreeCloud's permanent application model.
