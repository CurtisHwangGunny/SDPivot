# SDPivot OP Changelog: Phase 0-5

This document records the SDPivot OP changes delivered on top of the SaaS v4.2 product baseline through Phase 5.3.

## Scope and baseline

- Product baseline: the SaaS v4.2-aligned SDPivot application, including the auth, document, organization, knowledge-space, Q&A, and AI-writing behavior introduced by `4803a86e`.
- Phase 0 handoff: `cd5f622d` (`main`, 2026-07-22), which includes the Phase 0 product rename, OP build, migration, deployment, validation, and hardening work.
- Changelog head before this file: `dc701647` (`feature/op-phase0`, 2026-07-26).
- Detailed branch delta after the Phase 0 handoff: `cd5f622d..dc701647`.
- The repository does not contain a dedicated Git tag named SaaS v4.2. The commits above are therefore the auditable baseline references for this changelog.

## Phase 0: OP product and deployment baseline

### Product identity and routing

- Renamed the active product and backend surfaces from SmartKnora to SDPivot/SDP while retaining required legacy compatibility identifiers.
- Established `cmd/sdp-server` and `frontend/sdpivot` as the OP backend and frontend build targets.
- Standardized the canonical product API prefix as `/api/v1/sdp`, with the legacy `/api/v1/smartknora` prefix retained where compatibility is required.
- Corrected OP frontend routing so product APIs reach the correct backend and operations-only routes are excluded from OP frontend artifacts.
- Added route and artifact tests, including rejection of symlinked files and symlinked artifact roots.

### Deployment and runtime

- Added the OP Docker Compose baseline, environment example, deployment launcher, migration orchestrator, deployment validator, and smoke tests under `deploy/`.
- Hardened startup readiness, health checks, shutdown behavior, resource limits, build serialization, transient-load handling, and final PostgreSQL readiness checks.
- Added a dedicated frontend ingress network and restricted direct frontend bindings.
- Secured deployment launcher lifecycle and lock handling.
- Added UTF-8-safe artifact manifest validation.
- Preserved readable immutable runtime assets while tightening application asset permissions.
- Added a safe test environment template with placeholders; deployment credentials and runtime artifacts remain ignored.

### Database migration safety

- Added the OP PostgreSQL bootstrap baseline and migration-state detection for both existing and greenfield databases.
- Removed legacy tenant and operations-admin RLS bypass paths and localized tenant fallback context handling.
- Added savepoint recovery for tenant-context fallback failures.
- Neutralized legacy operations credentials with re-entrant cleanup.
- Added schema compatibility, indexed-schema, migration runtime-contract, and audit-contract validation.
- Hardened the audit schema migration for Core/Bootstrap variants, PostgreSQL 17, and BIGSERIAL fingerprint differences.
- Added unit, integration, shell, and deployment tests around bootstrap and audit migration behavior.

### Phase 0 reference range

- Initial Phase 0 work is included through `cd5f622d`.
- Follow-up hardening after the handoff spans `6f2bf734..9c198218`.

## Phase 1: SaaS surface reduction and operations cleanup

### SaaS and tenant behavior removal

- Removed self-service tenant creation and tenant switching from the standard frontend.
- Removed tenant-switch request overrides, related client state, and obsolete tenant selector utilities.
- Removed the sensitive-word filter path from chat, authentication, and operations modules.
- Simplified user JWT/authentication behavior to match the OP product surface.
- Removed obsolete operations module routes and UI elements that were no longer part of the OP scope.

### Operations administration

- Streamlined operations administration APIs and frontend panels.
- Removed the billing subscription UI and added migration `000015_remove_billing_subscription`.
- Refined the user model for OP account status and administration behavior.
- Updated the shared-database architecture documentation to reflect the active SDPivot design.

### SMS configuration

- Added an SMS provider abstraction with configuration validation and tests.
- Added PostgreSQL migration `000016_sms_provider_config` for SMS provider settings.

### Phase 1 reference commits

- `1b36e3de` - remove the sensitive-word and self-service SaaS paths.
- `29445b63` - operations cleanup, SMS provider support, billing removal, and user-model refinement.

## Phase 2: Administration, configuration, usage, and writing

### User and department administration

- Added operations user-management APIs and expanded the operations user-management UI.
- Added department types, repository, service, handler, routing, and CRUD support.
- Added batch user import with validation, per-row results, and tests.
- Added access-role definitions and role-aware middleware behavior.
- Added department schema support to PostgreSQL-compatible versioned migration `000064` and SQLite initialization.

### Model configuration

- Added system model-configuration APIs and the platform model-configuration page.
- Added create, read, update, delete, validation, and enablement behavior for model configurations.
- Added multilingual UI strings and settings navigation.
- Added versioned migration `000065_model_configs` and matching SQLite schema support.

### System configuration

- Added backend APIs for system configuration with validation, audit-aware updates, and tests.
- Added the `SystemAdminSettings` frontend and API client support for platform settings.
- Consolidated the new model and system administration pages into the settings navigation.

### Security and usage administration

- Added password policy helpers, IP whitelist parsing, and sensitive-value masking utilities.
- Added login-failure/lockout and token-related user fields and service behavior.
- Added operations usage-statistics aggregation and tests.
- Added audit coverage for privileged operations-admin actions.

### AI writing

- Completed AI-writing list, detail, update, delete, generate, and export API behavior.
- Added validation and test coverage for writing operations.
- Connected writing-related system settings to the administration frontend.
- Added server-generated PDF, DOCX, and Markdown draft exports with sanitized download filenames.

### Phase 2 reference commits

- `3bea7b35` - user management, departments, batch import, and role system.
- `2c4797f3`, `2617461b` - model-configuration backend and frontend.
- `49a1283a` - system-configuration backend.
- `e0ab2ab5` - usage statistics and security utilities.
- `63c7d5da` - writing API completion and system-settings frontend.

## Phase 3: Tags, API tokens, CLI QA, and backup

### Tag dictionary and document tags

- Added tag dictionary, tag-rule, and document-tag domain models and APIs.
- Added tag-system schema migration `000066_tag_system` and SQLite equivalents.
- Added automatic document tagging during knowledge post-processing.
- Added tag confidence persistence through migration `000067_document_tag_confidence`.
- Added knowledge list filtering by tag IDs and returned tag metadata with knowledge records.
- Added repository, service, handler, and integration tests for tag assignment and filtering.
- Added tag display and filter controls to knowledge-base and document-detail UI surfaces.

### API tokens and CLI authentication

- Added persisted API tokens through migration `000068_api_tokens` and SQLite equivalents.
- Added token creation/verification service behavior and authentication endpoint support.
- Updated CLI login, logout, token, and chat flows for the OP authentication contract.
- Added CLI command tests, dry-run coverage updates, root-command QA, and client auth support.

### Backup and restore foundation

- Added backup types, repository/service interfaces, service implementation, handlers, and routes.
- Added backup-record persistence through migration `000069_backup_records` and SQLite equivalents.
- Connected backup schedule settings to the system-administration configuration surface and added extensive handler tests.

### Phase 3 reference commits

- `4b6239cb` - tag models, dictionary APIs, and schema.
- `15bf4058` - automatic tagging and tag-filter APIs.
- `a3b955a3` - API tokens and CLI QA.
- `f36455ce` - tag UI, backup foundation, and integration cleanup.

## Phase 4: Security, reliability, and performance validation

### Security hardening

- Replaced executable weak example credentials with explicit change-required placeholders.
- Replaced the SDPivot fallback JWT signing secret with a process-local cryptographically random value.
- Added HTTP security headers: `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy`, and `X-Frame-Options`.
- Added header-read and idle timeouts plus a request-header size limit while preserving SSE streaming behavior.
- Stopped exposing recovered panic details to clients.
- Disabled credentialed wildcard CORS behavior.
- Added IP whitelist middleware and tests.
- Bound direct application and optional infrastructure ports to loopback by default; explicit bind variables are required for broader exposure.
- Recorded the detailed findings and residual RLS risk in `docs/security/phase4.3-audit.md`.

### Reliability and compatibility testing

- Added authentication, JWT expiry, login lockout, token, middleware, security-header, recovery, and route tests.
- Added department, tag, system administration, system configuration, backup, and operations authentication test suites.
- Added a platform smoke script covering configuration, administration, tags, backup, and security behavior.

### Performance validation

- Added knowledge-tag repository and automatic-tagging benchmarks.
- Added token and route performance tests.
- Added environment and Compose tuning needed by the Phase 4 validation profile.

### Known security limitation

- PostgreSQL RLS remains defense-in-depth because the application and migration paths use an owner-capable database role. Enforceable tenant isolation still requires a non-owner application role, a separate privileged operations path, `FORCE ROW LEVEL SECURITY`, and removal of connection-pool-dependent row-security overrides.

### Phase 4 reference commit

- `6e6aaf77` - security audit remediation, platform smoke coverage, and performance benchmarks.

## Phase 5: Operational documentation

### Administrator guide

- Added `ADMIN_GUIDE.md` covering authorization scopes, deployment, administrator bootstrap, secrets, registration, users, departments, model and system configuration, security controls, backups, audit/usage data, monitoring, upgrades, and troubleshooting.

### User manual

- Added `USER_MANUAL.md` covering login and registration, knowledge spaces, document workflows, AI Q&A, AI writing, search, feedback, session management, roles, and common troubleshooting.

### Phase 5 reference commits

- `71ed1b35` - administrator guide.
- `dc701647` - user manual.

## Schema changes after the Phase 0 handoff

| Migration | Change |
| --- | --- |
| `000015_remove_billing_subscription` | Removes the retired billing-subscription surface. |
| `000016_sms_provider_config` | Adds SMS provider configuration. |
| `000064_departments` | Adds department hierarchy and metadata. |
| `000065_model_configs` | Adds system model configurations. |
| `000066_tag_system` | Adds tag dictionary, rules, and document-tag relationships. |
| `000067_document_tag_confidence` | Adds confidence metadata to document tags. |
| `000068_api_tokens` | Adds persisted API tokens. |
| `000069_backup_records` | Adds backup execution records. |

SQLite initialization and incremental migrations were updated for the applicable cross-database features.

## Compatibility and operator notes

- Existing v4.2 auth, knowledge-space, Q&A, and writing data remain the functional starting point; OP phases narrow or extend those surfaces rather than replacing the core RAG platform.
- Legacy naming remains in selected migration names, database objects, and compatibility routes for upgrade safety.
- Operations administration, platform System Admin, and tenant/workspace roles are separate authorization scopes.
- Existing containers must be recreated to adopt changed Compose port bindings and network policy.
- No default administrator credentials are provided. Operators must replace all `CHANGE_ME` values and bootstrap administrators through the supported procedures.
- Local deployment environment files may contain credentials and must remain untracked.

## Verification delivered across Phase 0-5

- OP artifact, route, deployment, migration, PostgreSQL 17, and audit-fingerprint tests.
- Go unit and handler tests for departments, users, tokens, system configuration, tags, backups, middleware, and security behavior.
- CLI authentication and command-contract tests.
- Platform smoke coverage in `deploy/tests/test-phase4-platform-smoke.sh`.
- Tagging, repository, token, and router performance benchmarks.
- Security audit and operator/user documentation matching the implemented repository state.
