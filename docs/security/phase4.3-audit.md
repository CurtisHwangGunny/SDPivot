# Phase 4.3 Security Audit

Date: 2026-07-26

## Scope

- Hardcoded keys, passwords, and tokens in tracked files
- HTTP response security headers and server timeout hardening
- PostgreSQL row-level security (RLS) policies
- Docker Compose and active host port bindings

## Remediated

- Removed executable weak credentials from `.env.example` and replaced them with explicit change-me placeholders.
- Replaced the SDPivot hardcoded fallback JWT signing secret with a process-local cryptographically random secret.
- Stopped returning recovered panic values to HTTP clients.
- Added baseline `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`, and `X-Frame-Options` headers to Go HTTP responses.
- Added HTTP header-read and idle timeouts plus a header-size limit without imposing a write timeout on SSE responses.
- Disabled credentialed CORS for the main API's wildcard origin policy.
- Bound direct application and optional infrastructure Compose ports to loopback by default. Public exposure now requires an explicit `APP_BIND`, `INFRA_BIND`, or `DEV_BIND` override.

## Secret Scan

No provider-issued credentials or private-key blocks were identified in tracked files. Matches were limited to documentation examples, test fixtures, environment-variable references, and development defaults.

Ignored local environment files contain deployment credentials and must remain untracked. Operators should rotate them if they have been shared or used outside a disposable environment.

## PostgreSQL RLS

The SDPivot migrations define tenant policies with both `USING` and `WITH CHECK`, but most protected tables do not use `FORCE ROW LEVEL SECURITY`. The deployment currently runs migrations and application queries as the same database role, making that role the table owner and allowing owner bypass.

This cannot be safely corrected by adding `FORCE ROW LEVEL SECURITY` alone. Ops-admin handlers currently depend on owner bypass for cross-tenant queries, and their `SET LOCAL row_security = off` calls are not transaction-scoped to the queries they precede. A complete correction requires:

1. A non-owner application role for tenant-scoped traffic.
2. A separate narrowly privileged role or controlled database functions for operations-admin queries.
3. `FORCE ROW LEVEL SECURITY` on every tenant-protected table.
4. Removal of connection-pool-dependent `SET LOCAL row_security = off` calls.

Until that role split is implemented, RLS is defense-in-depth rather than an enforceable boundary against application-role mistakes.

## Port Audit

The repository Compose defaults now bind direct app and optional infrastructure ports to `127.0.0.1`, except the primary frontend ingress, which remains intentionally public.

At audit time, existing containers on the host still exposed PostgreSQL `5432`, the app `8080`, SDPivot `8082`, and frontend ports `81` and `3099` on all interfaces. Compose file changes do not reconfigure already-running containers; those stacks must be recreated to adopt the loopback bindings.

Host SSH `22`, HTTP `80`, and additional non-repository listeners were also present. Firewall and reverse-proxy policy remain host-operator responsibilities.
