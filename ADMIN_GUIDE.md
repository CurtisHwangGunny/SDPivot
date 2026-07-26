# WeKnora Administration Guide

This guide covers administration of the standard WeKnora application and the
optional SDPivot operations surface. It is written for operators of a
self-hosted deployment and reflects the features currently implemented in this
repository.

> **No default administrator credentials exist.** Create the initial account,
> promote it through the supported bootstrap procedure, replace every
> `CHANGE_ME` secret, and close public registration when appropriate.

## 1. Administration scopes

WeKnora has separate authorization scopes. A high role in one scope does not
automatically grant access in another.

| Scope | Roles | Typical responsibilities |
| --- | --- | --- |
| Tenant/workspace | `owner`, `admin`, `contributor`, `viewer` | Members, tenant models, integrations, knowledge resources, tenant audit log |
| Platform | `SystemAdmin` flag on a user | Global settings, security policy, platform model configurations, platform audit log, backups |
| SDPivot operations | `super_admin` and related product roles | SDPivot users, enterprises, operational models, audit and usage data |

The main settings route is:

```text
/platform/settings
```

Platform sections are shown only to a System Admin. Legacy
`/platform/system/*` links redirect to the consolidated settings panel.

### Tenant role matrix

| Role | Effective scope |
| --- | --- |
| Owner | Full tenant control, including membership, tenant metadata, API keys, and tenant deletion |
| Admin | Tenant infrastructure, models, integrations, and creator-or-admin resource management |
| Contributor | Create and manage owned knowledge bases and agents; read tenant resources |
| Viewer | Read-only tenant access and permitted agent use |

Tenant RBAC is enabled by default. Keep this enabled in production:

```dotenv
WEKNORA_TENANT_ENABLE_RBAC=true
```

Setting it to `false` changes tenant role checks to observation/fail-open mode.
It does not disable the independent System Admin checks.

## 2. Initial deployment and administrator bootstrap

### Standard Docker Compose deployment

```bash
git clone https://github.com/Tencent/WeKnora.git
cd WeKnora
cp .env.example .env
# Replace all CHANGE_ME values and review the deployment configuration.
docker compose up -d
```

Default endpoints are:

| Service | Default endpoint |
| --- | --- |
| Web UI | `http://localhost` |
| Backend API | `http://127.0.0.1:8080/api/v1` |
| Health check | `http://127.0.0.1:8080/health` |
| Swagger, non-release mode only | `http://127.0.0.1:8080/swagger/index.html` |

Inspect startup and migration health with:

```bash
docker compose ps
docker compose logs -f app docreader postgres
curl -fsS http://127.0.0.1:8080/health
```

At minimum, replace and securely retain these values:

```dotenv
DB_PASSWORD=<strong-database-password>
REDIS_PASSWORD=<strong-redis-password>
JWT_SECRET=<random-jwt-signing-secret>
TENANT_AES_KEY=<exactly-32-byte-key>
SYSTEM_AES_KEY=<exactly-32-byte-key>
```

Changing an AES key without migrating encrypted records makes existing secrets
unreadable. Back up keys independently from the database and restrict access to
both.

### Bootstrap the first System Admin

Bootstrap promotes an existing user; it does not create one.

1. Start WeKnora with registration enabled.
2. Register the intended administrator account.
3. Add its exact email to `.env`:

```dotenv
WEKNORA_BOOTSTRAP_SYSTEM_ADMIN_EMAIL=admin@example.com
```

4. Restart the application:

```bash
docker compose restart app
```

5. Sign in again and open:

```text
/platform/settings?section=system-global
```

Bootstrap promotion runs only while no System Admin exists. If the account does
not yet exist, startup continues with a warning and a later restart retries.
Once bootstrapped, use System Admin APIs to promote additional administrators.
The server prevents self-revocation and revocation of the last System Admin.

List, promote, or revoke System Admins through the CLI escape hatch described
in [CLI administration](#10-cli-administration):

```bash
weknora api '/api/v1/system/admin/list?offset=0&limit=50'

printf '%s' '{"email":"second-admin@example.com"}' |
  weknora api -X POST /api/v1/system/admin/promote --input - --dry-run

printf '%s' '{"user_id":"<user-id>"}' |
  weknora api -X POST /api/v1/system/admin/revoke --input - --dry-run
```

Promotion and revocation use `POST`, so remove `--dry-run` only after reviewing
the operation. The generic CLI does not add a confirmation prompt to POST.

### Close public registration

After provisioning the required accounts or configuring an external identity
provider, set:

```dotenv
DISABLE_REGISTRATION=true
```

Restart the app after changing environment-only startup configuration.
OIDC configuration is documented in [OIDC authentication](docs/OIDC认证调用流程.md).

## 3. User and workspace membership management

Open **Settings > Workspace > Member Management**, or navigate to:

```text
/platform/settings?section=members
```

All tenant members can view the roster. Owner permission is required for
membership mutations.

### Member operations

An Owner can:

- Search members by username or email.
- Invite a registered email and assign a role.
- Generate a reusable invitation link for a selected role.
- View, copy, and revoke pending invitations.
- Change another member's role.
- Remove another member.

The available roles are `owner`, `admin`, `contributor`, and `viewer`. New
invitations default to `contributor`. The UI presents a seven-day invitation
lifetime; the server-provided `expires_at` value is authoritative.

Safety constraints include:

- The current user cannot edit or remove their own roster row.
- The last active Owner cannot be demoted, removed, or allowed to leave.
- Direct member creation through the API requires an existing account.
- Normal UI invitations support a pending acceptance flow.
- Tenant IDs in the URL must match the active tenant unless explicitly allowed
  cross-tenant administration is configured.

Relevant API routes are:

```text
GET    /api/v1/tenants/:tenant_id/members
POST   /api/v1/tenants/:tenant_id/members
PUT    /api/v1/tenants/:tenant_id/members/:user_id
DELETE /api/v1/tenants/:tenant_id/members/:user_id
POST   /api/v1/tenants/:tenant_id/leave
```

Example role change:

```bash
printf '%s' '{"role":"admin"}' |
  weknora api -X PUT \
    /api/v1/tenants/<tenant-id>/members/<user-id> \
    --input - --dry-run
```

### SDPivot operations users

The optional SDPivot development/test application exposes an operations login
at `/ops-login` and integrated operations page at `/ops`. There are no default
OPS credentials. The account must be active and marked as an OPS administrator.

The operations user page supports search, active/disabled filtering, user
creation, CSV/XLSX import, and enable/disable actions. Import constraints are:

- `.csv` or `.xlsx`, maximum 10 MiB and 1,000 data rows.
- `password` and either `phone` or `email` are required.
- Passwords must have at least eight characters.
- Each valid row commits independently; invalid rows do not roll back prior
  successful rows.

The hardened SDPivot OP build intentionally excludes the browser OPS surface.
Do not expose an ad hoc OPS build without a security review. The underlying
operations APIs remain privileged and must be protected accordingly.

## 4. Departments and roles

### Department management

Department management is currently API-only in the main WeKnora application;
there is no mounted department administration page.

Reads require Viewer or higher. Writes require Admin or higher:

```text
GET    /api/v1/tenants/:tenant_id/departments
GET    /api/v1/tenants/:tenant_id/departments/tree
GET    /api/v1/tenants/:tenant_id/departments/:department_id
POST   /api/v1/tenants/:tenant_id/departments
PUT    /api/v1/tenants/:tenant_id/departments/:department_id
DELETE /api/v1/tenants/:tenant_id/departments/:department_id
```

Create a root department:

```bash
printf '%s' \
  '{"name":"Research","description":"Research organization","sort_order":10}' |
  weknora api -X POST \
    /api/v1/tenants/<tenant-id>/departments \
    --input - --dry-run
```

Create a child by adding `parent_id`. Names must be unique among siblings.
Names are trimmed and limited to 255 characters; descriptions are limited to
2,000 characters. The server rejects missing parents, cross-tenant parents,
hierarchy cycles, and deletion of a department that still has children.

Delete preview and execution:

```bash
weknora api -X DELETE \
  /api/v1/tenants/<tenant-id>/departments/<department-id> \
  --dry-run

# Execute only after explicit approval.
weknora api -X DELETE \
  /api/v1/tenants/<tenant-id>/departments/<department-id> \
  -y
```

### SDPivot product roles

SDPivot defines a separate product-role matrix:

| Role | Permissions |
| --- | --- |
| `super_admin` | Assign roles, manage departments, write and read knowledge |
| `department_admin` | Manage departments, write and read knowledge |
| `knowledge_editor` | Write and read knowledge |
| `knowledge_viewer` | Read knowledge |

Assigning `department_admin` requires a `department_id` in the user's tenant.
Changing away from that role clears the department assignment. The API exists,
but the current integrated operations user table does not expose a full role or
department editor.

## 5. Model configuration

### Tenant models

Open **Settings > Models and Runtime > Model Management**:

```text
/platform/settings?section=models
```

Viewer and higher can inspect models. Admin and Owner can create, update, test,
credential, and delete tenant models.

Supported categories include chat/KnowledgeQA, embedding, rerank,
VLLM/multimodal, and ASR. Administrators can:

- Add local Ollama or remote models.
- Edit and delete non-built-in models.
- Test remote connections and run the saved-model debugger.
- Download Ollama models and monitor progress.
- Configure custom headers, model parameters, embedding dimensions, vision
  capability, and provider-specific options.
- Update API keys and app secrets through the dedicated credential endpoint.

Important constraints:

- Remote base URLs are required and validated against SSRF policy.
- The UI constrains embedding dimensions to 128-4096.
- Local Ollama is not supported for rerank models in the editor.
- Built-in models are read-only and their sensitive endpoint/configuration
  fields are removed from tenant responses.
- Secrets are not returned by normal model APIs.
- A model referenced by a knowledge base or agent cannot be deleted.
- Debug invokes the upstream provider and can incur cost.

Prefer the `/credentials` subresource for secrets. Do not put credentials in
logs, shell history, or committed JSON files.

### Platform model configurations

System Admins can open:

```text
/platform/settings?section=system-models
```

Platform model configurations are separate from tenant models. Each record has
a unique name, provider, optional endpoint/API key, and generation defaults:

| Field | Default | Constraint |
| --- | ---: | --- |
| Temperature | `0.7` | 0-2 |
| Max tokens | `2048` | Greater than 0 |
| Top-p | `1` | 0-1 |

Endpoints receive SSRF validation. API keys are encrypted and never returned;
the response only indicates whether a key is configured. Leaving the key blank
while editing retains the current key.

For declaratively provisioned shared models, see
[Built-in model management](docs/BUILTIN_MODELS.md).

## 6. System configuration

System Admins open **Settings > Platform > System**:

```text
/platform/settings?section=system-global
```

### Global processing defaults

| Setting | Default | Allowed range |
| --- | ---: | ---: |
| Chunk size | 512 | 1-65,536 |
| Retrieval threshold | 0.5 | 0-1 |
| Token limit | 4,096 | 0-1,000,000 |
| Concurrency | 32 | 1-1,024 |

Validate workload impact before increasing concurrency or token limits.

### Object storage

The platform storage card supports MinIO and S3 configuration, including
endpoint, region, bucket, path prefix, credentials, SSL, and force-path-style.
Endpoint and bucket are required, and the endpoint must be HTTP(S). Blank secret
fields preserve the currently configured value.

Changing the global platform card does not replace a separate disaster-recovery
backup of object storage. Verify read/write access after any endpoint, bucket,
credential, or path-style change.

### Login and messaging providers

The platform cards support:

- WeChat login: enabled state, App ID, App Secret, and HTTP(S) redirect URL.
- SMS: Aliyun, Tencent, Huawei, or custom provider; endpoint, region, signing,
  template, sender, app ID, custom headers, credentials, and timeout.

When WeChat login is enabled, App ID and a valid redirect URL are required.
SMS timeout must be between 1 and 120 seconds. Blank secret fields preserve
existing values.

### Global tags

System Admins can maintain the global tag dictionary. Each entry requires a
dimension and name and may include color and sort order. The backend accepts a
maximum of 500 entries.

### Generic runtime settings API

Additional registered settings are available through:

```text
GET    /api/v1/system/admin/settings
GET    /api/v1/system/admin/settings/:key
PUT    /api/v1/system/admin/settings/:key
DELETE /api/v1/system/admin/settings/:key
```

Resolution order is database override, environment variable, then built-in
default. Deleting an override resets it to the environment/default value.
Settings metadata identifies type, category, enum values, secret status, and
whether a restart is required.

The feature-complete generic settings component is not mounted in the current
settings navigation. Use the API for settings not exposed by the structured
cards.

Example update and reset:

```bash
printf '%s' '{"value":20}' |
  weknora api -X PUT \
    /api/v1/system/admin/settings/tenant.default_storage_quota_gb \
    --input - --dry-run

weknora api -X DELETE \
  /api/v1/system/admin/settings/tenant.default_storage_quota_gb \
  --dry-run
```

Some changes are distributed to replicas through Redis cache invalidation.
Restart every application replica when the setting metadata requires it.

## 7. Security configuration

Open the **Security Configuration** tab under the platform System page.

| Setting | Default | Notes |
| --- | ---: | --- |
| `security.ip_whitelist` | `[]` | IP/CIDR list; empty disables restriction |
| `auth.password.min_length` | 8 | 6-128 |
| `auth.password.complexity` | `true` | Upper, lower, number, and special character |
| `auth.password.rotation_days` | 90 | 0-3650; 0 disables expiry |
| `auth.login.max_failed_attempts` | 5 | 0-100; 0 disables lockout |
| `auth.login.lockout_minutes` | 30 | 1-10,080 |

Security-policy changes generally take effect immediately. Before enabling an
IP whitelist:

1. Include the operator's actual client IP or trusted proxy egress range.
2. Verify the deployment's trusted-proxy configuration.
3. Keep an out-of-band recovery method for correcting the setting.
4. Test from a second authenticated session before ending the first.

The authenticated application route stack installs the IP whitelist
middleware. Incorrect values can lock administrators out.

Additional API-managed controls include `ssrf.whitelist` and
`auth.registration_mode` (`self_serve` or `invite_only`). Treat every SSRF
allowlist entry as a bypass of network destination protections. Add only the
specific host or CIDR required, for example:

```dotenv
SSRF_WHITELIST_EXTRA=mcp.internal,*.corp.example,10.20.0.0/16
```

Do not broadly allow private networks unless the application must reach them.

## 8. Audit logs and usage statistics

### Tenant audit log

Tenant Admin and Owner can open the audit drawer from **Member Management**.
The feed is append-only and ordered newest first. It records membership and
invitation changes, RBAC denials, and selected infrastructure events.

API:

```text
GET /api/v1/tenants/:tenant_id/audit-log
```

Supported filters are exact `action`, exact `outcome`, and actor user ID.
Pagination uses an immutable numeric cursor: pass `next_cursor` from one response
as `after_id` in the next request. Default page size is 50 and maximum is 100.

```bash
weknora api \
  '/api/v1/tenants/<tenant-id>/audit-log?limit=50&outcome=denied'
```

Do not use `weknora api --paginate` for this endpoint; it is cursor-based.

### Platform audit log

System Admins open the **Audit Logs** tab in the platform System page, or call:

```text
GET /api/v1/system/admin/audit-log
```

It supports the same cursor and filters and includes events such as generic
system-setting changes and System Admin promotion/revocation. Not every
structured configuration handler currently emits the same setting-change event,
so retain deployment configuration change records outside the application as
part of the operational audit trail.

### SDPivot usage statistics

SDPivot user-level endpoints are:

```text
GET /api/v1/sdp/usage/summary
GET /api/v1/sdp/usage/history
GET /api/v1/sdp/usage/by-model
```

Operations-wide endpoints are:

```text
GET /api/v1/sdp/ops/usage-stats
GET /api/v1/sdp/ops/usage-stats/export
```

Operations filters include `start_date`, `end_date`, `user_id`,
`department_id`, `tenant_id`, `page`, and `page_size`. Dates use `YYYY-MM-DD`.
The export applies the same filters without pagination.

```bash
weknora api \
  '/api/v1/sdp/ops/usage-stats?start_date=2026-07-01&end_date=2026-07-31'

weknora api \
  '/api/v1/sdp/ops/usage-stats/export?start_date=2026-07-01' \
  --format text > usage_statistics.csv
```

Current UI caveats:

- The main SDPivot usage page does not map the current backend summary envelope
  and field names correctly, so its cards may display zero. Use the API as the
  authoritative source until that UI is aligned.
- Operations-wide usage statistics have backend endpoints but no active tab in
  the integrated operations UI.
- OPS dashboard token counters count usage records, not summed token values.

SDPivot operations audit routes are:

```text
GET /api/v1/sdp/ops/audit-logs
GET /api/v1/sdp/ops/audit-logs/export
```

The CSV audit export is capped at the newest 10,000 records and currently does
not apply the list-view filters. Record the exported time range and limitations
with the artifact.

## 9. Backup and restore

Backup management is currently API/CLI-only and requires System Admin. It backs
up the primary database, not all WeKnora data stores.

### Prerequisites

For PostgreSQL, `pg_dump` and `pg_restore` must be installed in the application
container and the configured database user must be able to read and restore all
objects. The production app image installs the PostgreSQL client.

Configure persistent backup storage:

```dotenv
WEKNORA_BACKUP_DIR=/data/backups
WEKNORA_BACKUP_TIMEOUT=2h
```

Docker Compose mounts a `backup-data` volume at `/data/backups`. The timeout
controls backup creation; restore has a separate two-hour handler timeout.

Supported formats are:

| Database | Artifact | Format |
| --- | --- | --- |
| PostgreSQL | `.dump` | `pg_dump --format=custom` |
| SQLite | `.db` | SQLite `VACUUM INTO` snapshot |

### Create and inspect a backup

```bash
weknora api -X POST /api/v1/system/admin/backups --dry-run
weknora api -X POST /api/v1/system/admin/backups
```

Creation returns HTTP 202 and runs asynchronously. Capture the returned `id`,
then poll:

```bash
weknora api /api/v1/system/admin/backups/<backup-id>
weknora api '/api/v1/system/admin/backups?limit=50&offset=0'
```

Wait for `status` to become `succeeded` or `failed`. For success, verify that
`size_bytes` is positive and `checksum_sha256` contains 64 lowercase hexadecimal
characters.

Statuses are `pending`, `running`, `succeeded`, and `failed`. Only one backup,
restore, or deletion can hold the process-local lock at a time; a competing
request returns HTTP 409. In a multi-replica deployment this lock is not shared,
so route maintenance operations to one designated replica.

### Download and verify

The CLI's JSON mode wraps raw API responses and is not suitable for artifact
download. Use an authenticated HTTP client and verify both size and SHA-256
against the backup record:

```bash
TOKEN=$(weknora auth token)
BASE_URL=http://127.0.0.1:8080

curl -fSL \
  -H "Authorization: Bearer $TOKEN" \
  -o weknora-backup.dump \
  "$BASE_URL/api/v1/system/admin/backups/<backup-id>/download"

sha256sum weknora-backup.dump
stat -c '%s' weknora-backup.dump
```

If the profile uses API-key mode, use `X-API-Key` instead of `Authorization`.
Store an externally verified copy outside the application host.

### Restore

> **Restore is destructive.** PostgreSQL restore uses `--clean --if-exists`.
> Schedule a maintenance window, stop or drain writes, create and externally
> verify a pre-restore backup, and use a matching WeKnora/database version.

The restore body must contain the exact confirmation string:

```json
{"confirmation":"RESTORE <backup-id>"}
```

Preview the raw API request without a real secret-bearing body:

```bash
printf '%s' '{"confirmation":"RESTORE <backup-id>"}' |
  weknora api -X POST \
    /api/v1/system/admin/backups/<backup-id>/restore \
    --input - --dry-run
```

After explicit approval, remove `--dry-run`. The server recomputes SHA-256 and
refuses a mismatched artifact. The endpoint does not create a pre-restore backup,
drain traffic, pause workers, or restart services. Restart the application and
verify health, migrations, integrations, and representative data after restore.

### Delete and schedule

```bash
weknora api -X DELETE \
  /api/v1/system/admin/backups/<backup-id> \
  --dry-run

# Execute only after explicit approval.
weknora api -X DELETE \
  /api/v1/system/admin/backups/<backup-id> \
  -y
```

Schedule API:

```text
GET /api/v1/system/admin/backups/schedule
PUT /api/v1/system/admin/backups/schedule
```

Default schedule configuration is disabled, with cron `0 0 2 * * *` and 30-day
retention. Cron has six fields including seconds, so this expression means
02:00:00 daily. `retention_days` accepts 0-3650; zero disables expiry. Expired
artifacts are purged only after a later backup succeeds.

### Disaster-recovery scope

Database backup does **not** include:

- Local files under `/data/files`.
- MinIO/S3 or other object storage.
- External vector databases or search indexes.
- Neo4j and other graph stores.
- Redis data, deployment secrets, TLS material, or encryption keys.

Back up those systems separately and test a complete restore procedure. There is
no API to upload/register an external dump; server restore operates only on an
existing backup record and artifact in the configured backup directory.

## 10. CLI administration

The `weknora` CLI has typed commands for profiles, authentication, knowledge
bases, documents, chunks, search, sessions, agents, and MCP serving. It does not
currently have typed `admin`, `user`, `department`, `model`, `audit`, `usage`,
`backup`, or `restore` command groups. Use `weknora api` for those endpoints.

### Install and authenticate

Pre-built binaries are published with releases. To build from source, use Go
1.26 or later:

```bash
cd cli
go build -o weknora .
sudo mv weknora /usr/local/bin/
```

Configure a profile and authenticate:

```bash
weknora profile add prod --host https://weknora.example.com --use
weknora auth login
```

For automation, pipe an API key or API token through stdin:

```bash
printf '%s' "$WEKNORA_API_KEY" | weknora auth login --with-token
```

Verify context before administration:

```bash
weknora auth status
weknora doctor
weknora profile list
```

Use a one-shot profile instead of changing persistent state:

```bash
weknora --profile staging auth status
WEKNORA_PROFILE=staging weknora kb list
```

Credentials are stored in the OS keyring when available, with a mode-0600 file
fallback under `$XDG_CONFIG_HOME/weknora/secrets/`.

### Output and automation contract

JSON is the default output and uses a success envelope with `ok`, `data`,
`meta`, and `profile`. Use `--format text` for human output, `--format ndjson`
for line-oriented lists/streams, and `--jq` against the complete JSON envelope:

```bash
weknora kb list --jq '.data[].id'
weknora auth status --jq '.data.tenant_id'
weknora version --format json
```

Important exit codes are:

| Exit | Meaning |
| ---: | --- |
| 0 | Success |
| 2 | Command/flag error |
| 3 | Authentication or authorization failure |
| 4 | Resource not found |
| 5 | Invalid input |
| 7 | Server or network failure |
| 10 | Destructive action requires explicit approval |
| 124 | Wait timeout |
| 130 | Signal cancellation |

Never automatically retry exit 10 with `-y`. Each destructive invocation
requires explicit human approval.

### Raw API safety

Syntax:

```bash
weknora api <path> [-X METHOD] [--input <file|->] [--paginate]
```

Rules:

- Paths must begin with `/`.
- `--input` without `-X` implies POST; otherwise the default is GET.
- DELETE is confirmation-gated and requires `-y` in non-interactive/JSON use.
- POST, PUT, PATCH, and custom methods are not confirmation-gated.
- Use `--dry-run` for writes before execution.
- Dry-run prints the full body. Do not use it with real passwords, API keys, or
  other secrets when stdout is logged.
- `--paginate` only handles `page`/`page_size` response shapes, not cursor feeds.

Examples:

```bash
weknora api /api/v1/system/info
weknora api /api/v1/models

printf '%s' '{"value":true}' |
  weknora api -X PUT /api/v1/system/admin/settings/example.key \
    --input - --dry-run
```

For the complete typed command reference, dry-run contract, streaming format,
and exit-code behavior, see [CLI documentation](cli/README.md).

## 11. MCP setup

There are three distinct MCP use cases:

1. Configure a remote MCP service for WeKnora agents to call.
2. Expose WeKnora through the curated Go CLI stdio MCP server.
3. Expose the broader Python MCP server through stdio, SSE, or Streamable HTTP.

### Configure MCP services inside WeKnora

Open **Settings > MCP Services**. Viewer can inspect services; Admin and Owner
can create, edit, enable/disable, test, delete, credential, and configure tool
approval policies.

Currently usable inbound transports are:

| Value | Support |
| --- | --- |
| `sse` | Supported |
| `http-streamable` | Supported |
| `stdio` | Rejected for security, despite legacy types/docs containing it |

Configure the service name, description, state, transport, URL, custom headers,
authentication, timeout, and retry metadata. The current client persists retry
count/delay but does not yet apply those fields as automatic retry loops.

The **From code** importer accepts a standard `mcpServers` object or one bare
server object. It recognizes `sse`, `http`, `http-streamable`, and common
Streamable HTTP aliases. If multiple servers are present, only the first is
imported. A config with `command` is rejected because application-side stdio is
disabled.

Example import:

```json
{
  "mcpServers": {
    "documentation": {
      "type": "http",
      "url": "https://mcp.example.com/mcp",
      "headers": {
        "X-Organization": "example"
      }
    }
  }
}
```

The importer fills the form but does not save automatically. Test the connection
after saving to discover tools and resources.

### MCP credentials and OAuth

Normal service responses never return secrets. Update static credentials through:

```text
PUT    /api/v1/mcp-services/:id/credentials
DELETE /api/v1/mcp-services/:id/credentials/:field
```

Supported fields are `api_key` and `token`. Omitted or empty fields preserve the
current value; DELETE clears a configured field. Credential changes recycle
cached connections.

Remote MCP OAuth supports authorization code with PKCE, metadata discovery,
dynamic client registration, and per-user token storage. The public callback is:

```text
/api/v1/mcp-oauth/callback
```

OAuth state expires after ten minutes. Use Redis in multi-replica deployments so
the callback can land on any replica; otherwise it must return to the same live
process.

If an internal service URL is rejected by SSRF validation, add only its exact
host or narrow CIDR to `SSRF_WHITELIST_EXTRA` and document the exception.

Built-in MCP services are tenant-visible but read-only. Their connection and
authentication details are stripped from responses. Provisioning a built-in
service currently requires a controlled database operation; back up first and
avoid plaintext secret insertion that bypasses application encryption.

For human-in-the-loop protection, mark sensitive discovered tools as requiring
approval. Approval timeout is configured with:

```dotenv
WEKNORA_AGENT_TOOL_APPROVAL_TIMEOUT=600
```

Without Redis, pending approvals are instance-local and require sticky routing
in a multi-replica deployment.

### Curated CLI MCP server

Use this option when the AI client can launch a local stdio process and a small,
safer tool surface is preferred.

After configuring the CLI profile and credentials:

```bash
weknora --profile prod mcp serve
```

Generic client configuration:

```json
{
  "mcpServers": {
    "weknora-prod": {
      "command": "/usr/local/bin/weknora",
      "args": ["--profile", "prod", "mcp", "serve"]
    }
  }
}
```

The server is stdio-only. Stdout is reserved for MCP JSON-RPC; logs go to
stderr. It exposes ten curated tools:

```text
kb_list, kb_view, doc_list, doc_view, doc_download,
chunk_list, search_chunks, agent_list, chat, session_ask
```

Destructive create/edit/delete operations are deliberately excluded. `chat` and
`session_ask` still create conversation/message records and may consume model
tokens.

### Python MCP server

Use the Python server when SSE/Streamable HTTP or its broader administrative
tool set is required.

```bash
cd mcp-server
python -m venv .venv
. .venv/bin/activate
pip install -r requirements.txt

export WEKNORA_BASE_URL=https://weknora.example.com/api/v1
export WEKNORA_API_KEY=<least-privilege-api-key>
python main.py --check-only
```

Run one transport:

```bash
python main.py --transport stdio
python main.py --transport sse --host 127.0.0.1 --port 8000
python main.py --transport http --host 127.0.0.1 --port 8000
```

Network endpoints are:

```text
SSE:             http://127.0.0.1:8000/sse
Streamable HTTP: http://127.0.0.1:8000/mcp
```

Docker Compose starts the Python MCP server only in the `full` profile:

```dotenv
WEKNORA_API_KEY=<least-privilege-api-key>
MCP_PORT=8082
```

```bash
docker compose --profile full up -d
```

The default host endpoint is `http://127.0.0.1:8082/mcp`.

> **Network security warning:** `WEKNORA_API_KEY` authenticates the Python MCP
> server to WeKnora; it does not authenticate clients connecting to the MCP
> network port. Anyone who can reach that port can use the configured key's
> capabilities. Keep it on loopback/private networking or place it behind an
> authenticated TLS reverse proxy and firewall.

The Python server exposes a much broader tool set, including tenant/model
creation, ingestion, and deletion. Use a least-privilege key and restrict client
access. File-ingestion tools read paths on the MCP server host, not the remote
client host.

## 12. Operational checklist

### Routine checks

- Verify `/health`, container status, migration state, and recent app errors.
- Review tenant and platform audit logs for denied or unexpected actions.
- Review pending invitations and remove stale accounts promptly.
- Test critical models, object storage, parser, vector store, and MCP services.
- Verify backup status, external artifact copies, size, and checksum.
- Monitor database, object storage, vector store, queue, and token usage capacity.
- Rotate credentials through dedicated credential APIs and retain encryption
  keys in a separate secrets backup.

### Before a high-risk change

- Confirm the active CLI profile and tenant with `weknora auth status`.
- Use `--dry-run` where supported, without placing real secrets in the body.
- Capture current settings and resource IDs.
- Create and externally verify a fresh backup.
- Schedule a maintenance window for restore or storage/database changes.
- Define rollback and post-change validation steps.

### After a restore or major configuration change

- Restart affected application replicas and workers.
- Confirm health and migration status.
- Verify administrator login and tenant switching.
- Test model, storage, parser, vector, MCP, and OAuth connections.
- Verify representative knowledge, sessions, and audit records.
- Reconcile external object/vector/graph data with the restored database.
- Record the change and evidence in the external operational audit trail.

## 13. Related documentation

- [Main installation guide](README.md)
- [RBAC reference](docs/RBAC说明.md)
- [API documentation index](docs/api/README.md)
- [Authentication API](docs/api/auth.md)
- [System API](docs/api/system.md)
- [MCP service API](docs/api/mcp-service.md)
- [MCP tool approval](docs/zh/mcp-approval.md)
- [Built-in MCP services](docs/BUILTIN_MCP_SERVICES.md)
- [Built-in models](docs/BUILTIN_MODELS.md)
- [CLI reference](cli/README.md)
- [Python MCP server](mcp-server/README.md)
- [Lite deployment](docs/LITE.md)
- [Troubleshooting and QA](docs/QA.md)

When handwritten API documentation differs from the running server, use the
current route implementation and Swagger generated by the deployed version as
the endpoint authority.
