#!/bin/bash
PATH=/usr/bin:/bin
LC_ALL=C
export PATH LC_ALL
unset CDPATH ENV BASH_ENV
set -Eeuo pipefail
umask 077

readonly SOURCE_ROOT="$(/usr/bin/realpath -- "${1:-$(pwd)}")"
readonly TMP_ROOT="$(/usr/bin/mktemp -d /tmp/op-deploy-test.XXXXXX)"
readonly ROOT="$TMP_ROOT/repo"
LAUNCHER_RESTORE=""
cleanup_test() {
    local status=$?
    trap - EXIT HUP INT TERM
    if [[ -n "$LAUNCHER_RESTORE" && -f "$LAUNCHER_RESTORE" && -d "$ROOT/deploy" ]]; then
        /usr/bin/cp -- "$LAUNCHER_RESTORE" "$ROOT/deploy/op-deploy.sh" || status=1
    fi
    /bin/rm -rf -- "$TMP_ROOT" || status=1
    exit "$status"
}
trap cleanup_test EXIT HUP INT TERM
/bin/mkdir -p -- "$ROOT/deploy/tests" "$ROOT/deploy/migration" "$ROOT/migrations/postgres-bootstrap" "$ROOT/docker" "$ROOT/frontend/sdpivot/dist" \
    "$ROOT/frontend/sdpivot/deps/cppjieba/dict" "$ROOT/cmd" "$ROOT/config" "$ROOT/dataset" \
    "$ROOT/deps" "$ROOT/docs" "$ROOT/internal" "$ROOT/migrations" "$ROOT/packages" \
    "$ROOT/scripts" "$ROOT/skills" "$ROOT/docreader"
for relative in \
    .dockerignore go.mod go.sum Makefile \
    deploy/docker-compose.op.yml deploy/migrate-op.sh deploy/migration/Dockerfile \
    migrations/postgres-bootstrap/000012_sdpivot_op_baseline.up.sql \
    deploy/validate-op-deployment.sh deploy/op-deploy.sh deploy/tests/test-op-deployment.sh \
    docker/Dockerfile.app docker/Dockerfile.docreader \
    frontend/sdpivot/Dockerfile.backend frontend/sdpivot/Dockerfile.frontend \
    frontend/sdpivot/nginx.conf; do
    /usr/bin/cp -- "$SOURCE_ROOT/$relative" "$ROOT/$relative"
done
printf '%s\n' '#!/bin/sh' 'exit 0' > "$ROOT/frontend/sdpivot/sdp-server"
/bin/chmod 700 "$ROOT/frontend/sdpivot/sdp-server"
printf '<!doctype html>\n' > "$ROOT/frontend/sdpivot/dist/index.html"
/bin/mkdir -p -- "$ROOT/docs/中文目录"
printf 'UTF-8 artifact fixture\n' > "$ROOT/docs/中文目录/数据源说明.md"

readonly ENV_FILE="$ROOT/deploy/test.env"
write_good_env() {
    /bin/cat > "$ENV_FILE" <<'EOF'
COMPOSE_PROJECT_NAME=sdpivot-op
OP_DB_USER=svcuser
OP_DB_PASSWORD=J7mQ2_vL9xR4-tN6.kP8
OP_DB_NAME=opdata
OP_REDIS_PASSWORD=V5@pH8!zC3#sK7%yD2&m
OP_JWT_SECRET=G7!vN2@qR9#xL4%tB6&kP3^sF8*mC5!z
OP_SDP_JWT_SECRET=Y4@cM8!wH2#rT7%pD5&nK9^xQ3*zV6@a
OP_TENANT_AES_KEY=A7!cD2@fG9#hJ4%kL6&mN3^pQ8*rC5xy
OP_SYSTEM_AES_KEY=Z5@xC8!vB2#nM7%qW4&eR9^tY3*uH6ab
OP_ALLOWED_ORIGINS=https://op.internal.test:8443
OP_HTTP_BIND=127.0.0.1
OP_HTTP_PORT=8080
OP_POSTGRES_IMAGE=paradedb/paradedb:v0.22.2-pg17
OP_REDIS_IMAGE=redis:7.0-alpine
OP_APP_IMAGE=registry.internal/weknora-app:v1.2.3
OP_DOCREADER_IMAGE=registry.internal/weknora-docreader:v1.2.3
OP_MIGRATION_IMAGE=sdpivot-op-migration:local
OP_SDP_BACKEND_IMAGE=sdpivot-op-backend:local
OP_SDP_FRONTEND_IMAGE=sdpivot-op-frontend:local
SDP_ENABLE_LEGACY_ALIAS=false
DISABLE_REGISTRATION=true
EOF
    /bin/chmod 600 "$ENV_FILE"
}
write_good_env

pass() { printf 'PASS %s\n' "$1"; }
expect_fail() {
    local name="$1"
    shift
    if "$@" > "$TMP_ROOT/out" 2> "$TMP_ROOT/err"; then
        printf 'FAIL expected rejection: %s\n' "$name" >&2
        exit 1
    fi
    pass "$name"
}

/bin/bash -n "$ROOT/deploy/validate-op-deployment.sh"
/usr/bin/python3 -m py_compile "$ROOT/deploy/op-deploy.sh"
/usr/bin/python3 - "$ROOT/deploy/op-deploy.sh" <<'PY'
import runpy
import signal
import sys

module = runpy.run_path(sys.argv[1])
assert module["FORWARDED_SIGNALS"] == (signal.SIGTERM, signal.SIGINT, signal.SIGHUP)
PY
pass syntax

/usr/bin/python3 - "$ROOT/deploy/docker-compose.op.yml" <<'PY_COMPOSE_HEALTH'
from pathlib import Path
import sys

text = Path(sys.argv[1]).read_text(encoding="utf-8")
expected = 'test "$$(cat /proc/1/comm)" = postgres && pg_isready -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'
assert text.count(expected) == 1
PY_COMPOSE_HEALTH
pass postgres_health_waits_for_final_pid1

/usr/bin/python3 - "$ROOT/deploy/migrate-op.sh" <<'PY_MIGRATION_EXTENSION_OBJECTS'
from pathlib import Path
import sys

text = Path(sys.argv[1]).read_text(encoding="utf-8")
required = (
    "FROM pg_catalog.pg_class application_object",
    "JOIN pg_catalog.pg_namespace application_schema",
    "application_object.relkind IN ('r', 'p')",
    "FROM pg_catalog.pg_depend extension_dependency",
    "JOIN pg_catalog.pg_extension owner_extension",
    "extension_dependency.classid = 'pg_catalog.pg_class'::regclass",
    "extension_dependency.objid = application_object.oid",
    "extension_dependency.deptype = 'e'",
)
for marker in required:
    assert text.count(marker) == 1, marker
assert "FROM pg_catalog.pg_tables" not in text
PY_MIGRATION_EXTENSION_OBJECTS
pass migration_greenfield_ignores_extension_owned_tables

/usr/bin/python3 - "$ROOT/deploy/migrate-op.sh" <<'PY_MIGRATION_BOOL_STATE'
from pathlib import Path
import sys

text = Path(sys.argv[1]).read_text(encoding="utf-8")
expected = "SELECT version::text || ':' || CASE WHEN dirty THEN 't' ELSE 'f' END FROM public.${table_name}"
assert text.count(expected) == 1
assert "dirty::text FROM public.${table_name}" not in text
PY_MIGRATION_BOOL_STATE
pass migration_state_normalizes_postgres_boolean

/usr/bin/python3 - "$ROOT/deploy/migrate-op.sh" "$ROOT/migrations/postgres-bootstrap/000012_sdpivot_op_baseline.up.sql" <<'PY_CORE_AUDIT_COMPAT'
from pathlib import Path
import sys

migration = Path(sys.argv[1]).read_text(encoding="utf-8")
baseline = Path(sys.argv[2]).read_text(encoding="utf-8")
for marker in (
    "core_audit_required(column_name, allowed_types, is_nullable)",
    "sdpivot_audit_required(column_name, allowed_types, is_nullable)",
    "object_name <> 'audit_logs'",
    "core_audit_shape",
    "'core_exact'",
    "'baseline_exact'",
):
    assert marker in migration, marker
for marker in (
    "requires Core table public.audit_logs",
    "requires public.audit_logs to be a table",
    "core_column_count <> 13",
    "sdpivot_column_count NOT IN (0, 6)",
    "total_column_count NOT IN (13, 19)",
    "requires the completed Core audit_logs schema",
    "ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS user_id VARCHAR(36)",
    "ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS ip VARCHAR(50)",
):
    assert marker in baseline, marker
assert "ALTER TABLE IF EXISTS audit_logs" not in baseline
assert baseline.index("requires Core table public.audit_logs") < baseline.index("ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS user_id")
assert baseline.index("ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS ip") < baseline.index("SELECT sdpivot_op_bootstrap_000012_assert_schema(FALSE)")
PY_CORE_AUDIT_COMPAT
pass migration_greenfield_preserves_core_audit_schema

payload="$TMP_ROOT/bash-env-payload"
printf 'touch %q\n' "$TMP_ROOT/bash-env-executed" > "$payload"
expect_fail bash_env_ignored /usr/bin/env BASH_ENV="$payload" \
    "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" rejected-action
[[ ! -e "$TMP_ROOT/bash-env-executed" ]] || { printf 'FAIL BASH_ENV executed before launcher isolation\n' >&2; exit 1; }

"$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate >/dev/null
pass compliant_validate
expect_fail internal_lock_argv_bypass "$ROOT/deploy/op-deploy.sh" --internal-locked --env-file "$ENV_FILE" validate
expect_fail up_build_equals "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" up --build=true
expect_fail up_no_build_equals "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" up --no-build=false
expect_fail up_unapproved_arg "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" up --wait

# Input identity, permission, and location.
/bin/chmod 644 "$ENV_FILE"
expect_fail env_mode_0644 "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
write_good_env
/bin/ln "$ENV_FILE" "$ROOT/deploy/test-hardlink.env"
expect_fail env_hardlink "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
/bin/rm -f "$ROOT/deploy/test-hardlink.env"
write_good_env
/bin/ln -s "$ENV_FILE" "$ROOT/deploy/test-symlink.env"
expect_fail env_symlink "$ROOT/deploy/op-deploy.sh" --env-file "$ROOT/deploy/test-symlink.env" validate
/bin/rm -f "$ROOT/deploy/test-symlink.env"
/usr/bin/cp "$ENV_FILE" "$TMP_ROOT/outside.env"
/bin/chmod 600 "$TMP_ROOT/outside.env"
expect_fail env_outside_repo "$ROOT/deploy/op-deploy.sh" --env-file "$TMP_ROOT/outside.env" validate
expect_fail env_option_like "$ROOT/deploy/op-deploy.sh" --env-file -bad validate
/bin/mkdir "$ROOT/deploy/test-dir.env"
expect_fail env_directory "$ROOT/deploy/op-deploy.sh" --env-file "$ROOT/deploy/test-dir.env" validate
/bin/rm -rf "$ROOT/deploy/test-dir.env"
/usr/bin/mkfifo "$ROOT/deploy/test-fifo.env"
expect_fail env_fifo "$ROOT/deploy/op-deploy.sh" --env-file "$ROOT/deploy/test-fifo.env" validate
/bin/rm -f "$ROOT/deploy/test-fifo.env"

# PATH hijack attempts must not execute.
/bin/mkdir "$TMP_ROOT/fakebin"
for tool in dirname readlink realpath stat docker env flock sha256sum; do
    printf '#!/bin/sh\ntouch %q\nexit 91\n' "$TMP_ROOT/hijacked-$tool" > "$TMP_ROOT/fakebin/$tool"
    /bin/chmod 700 "$TMP_ROOT/fakebin/$tool"
done
PATH="$TMP_ROOT/fakebin:/usr/bin:/bin" "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate >/dev/null
for tool in dirname readlink realpath stat docker env flock sha256sum; do
    [[ ! -e "$TMP_ROOT/hijacked-$tool" ]] || { printf 'FAIL PATH hijack %s\n' "$tool" >&2; exit 1; }
done
pass path_hijack_absent

# The launcher must retain its lock until a signalled child process group is fully reaped.
/usr/bin/cp -- "$ROOT/deploy/op-deploy.sh" "$TMP_ROOT/op-deploy.original"
LAUNCHER_RESTORE="$TMP_ROOT/op-deploy.original"
fake_docker="$TMP_ROOT/fake-docker"
/usr/bin/python3 - "$fake_docker" "$TMP_ROOT/fake-docker.pid" "$TMP_ROOT/fake-docker-signalled" <<'PY'
from pathlib import Path
import shlex
import sys

script, pid_file, signal_file = map(Path, sys.argv[1:])
script.write_text(
    "#!/bin/bash\n"
    f"printf '%s\\n' \"$$\" > {shlex.quote(str(pid_file))}\n"
    f"trap 'touch {shlex.quote(str(signal_file))}; /bin/sleep 2; exit 143' TERM INT HUP\n"
    "while :; do /bin/sleep 1; done\n"
)
script.chmod(0o700)
PY
/usr/bin/python3 - "$ROOT/deploy/op-deploy.sh" "$fake_docker" <<'PY'
from pathlib import Path
import sys

launcher = Path(sys.argv[1])
source = launcher.read_text()
old = '/usr/bin/docker'
assert source.count(old) == 1
launcher.write_text(source.replace(old, sys.argv[2]))
PY
"$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" config > "$TMP_ROOT/signal.out" 2> "$TMP_ROOT/signal.err" &
launcher_pid=$!
for _ in $(/usr/bin/seq 1 50); do
    [[ -s "$TMP_ROOT/fake-docker.pid" ]] && break
    /bin/sleep 0.1
done
[[ -s "$TMP_ROOT/fake-docker.pid" ]] || { printf 'FAIL signal child did not start\n' >&2; /bin/kill -KILL "$launcher_pid" 2>/dev/null || :; exit 1; }
child_pid="$(/bin/cat "$TMP_ROOT/fake-docker.pid")"
/bin/kill -TERM "$launcher_pid"
for _ in $(/usr/bin/seq 1 50); do
    [[ -e "$TMP_ROOT/fake-docker-signalled" ]] && break
    /bin/sleep 0.1
done
[[ -e "$TMP_ROOT/fake-docker-signalled" ]] || { printf 'FAIL signal was not forwarded to child group\n' >&2; exit 1; }
expect_fail signal_lock_held_until_child_exit "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
/bin/kill -0 "$launcher_pid" 2>/dev/null || { printf 'FAIL launcher exited before child completed\n' >&2; exit 1; }
if wait "$launcher_pid"; then
    printf 'FAIL signalled launcher unexpectedly succeeded\n' >&2
    exit 1
fi
/bin/kill -0 "$child_pid" 2>/dev/null && { printf 'FAIL signalled child remains alive\n' >&2; exit 1; }
/usr/bin/cp -- "$TMP_ROOT/op-deploy.original" "$ROOT/deploy/op-deploy.sh"
LAUNCHER_RESTORE=""
"$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate >/dev/null
pass signal_child_reaped_and_lock_released

# Ambient OP_* values must not override the validated snapshot. Use a fake Docker
# config renderer so this suite never reaches the real Docker CLI.
/usr/bin/cp -- "$ROOT/deploy/op-deploy.sh" "$TMP_ROOT/op-deploy.pre-config-fake"
LAUNCHER_RESTORE="$TMP_ROOT/op-deploy.pre-config-fake"
config_fake_docker="$TMP_ROOT/config-fake-docker"
build_log="$TMP_ROOT/build.log"
fail_service_file="$TMP_ROOT/fail-service"
/usr/bin/python3 - "$config_fake_docker" "$build_log" "$fail_service_file" <<'PY'
from pathlib import Path
import sys

script, log, fail_service = map(Path, sys.argv[1:])
script.write_text("""#!/usr/bin/python3
import json
from pathlib import Path
import sys

LOG = Path({log!r})
FAIL_SERVICE = Path({fail!r})
args = sys.argv[1:]
if "config" in args and "--format" in args and "json" in args:
    env_file = Path(args[args.index("--env-file") + 1])
    env = dict(line.split("=", 1) for line in env_file.read_text().splitlines() if line.strip())
    dburl = "postgresql://{{}}:{{}}@postgres:5432/{{}}?sslmode=disable".format(
        env["OP_DB_USER"], env["OP_DB_PASSWORD"], env["OP_DB_NAME"])
    print(json.dumps({{"name": "sdpivot-op", "services": {{
        "postgres": {{"image": env["OP_POSTGRES_IMAGE"], "environment": {{"POSTGRES_PASSWORD": env["OP_DB_PASSWORD"]}}}},
        "redis": {{"image": env["OP_REDIS_IMAGE"], "environment": {{"REDIS_PASSWORD": env["OP_REDIS_PASSWORD"]}}}},
        "migration": {{"image": env["OP_MIGRATION_IMAGE"], "environment": {{"OP_DATABASE_URL": dburl}}}},
        "docreader": {{"image": env["OP_DOCREADER_IMAGE"]}},
        "app": {{"image": env["OP_APP_IMAGE"], "environment": {{
            "DB_PASSWORD": env["OP_DB_PASSWORD"], "REDIS_PASSWORD": env["OP_REDIS_PASSWORD"],
            "JWT_SECRET": env["OP_JWT_SECRET"], "TENANT_AES_KEY": env["OP_TENANT_AES_KEY"],
            "SYSTEM_AES_KEY": env["OP_SYSTEM_AES_KEY"]}}}},
        "sdp-backend": {{"image": env["OP_SDP_BACKEND_IMAGE"], "environment": {{
            "SDP_DB_PASSWORD": env["OP_DB_PASSWORD"], "SDP_REDIS_PASSWORD": env["OP_REDIS_PASSWORD"],
            "SDP_JWT_SECRET": env["OP_SDP_JWT_SECRET"]}}}},
        "sdp-frontend": {{"image": env["OP_SDP_FRONTEND_IMAGE"]}}
    }}}}))
    raise SystemExit(0)
if "build" in args:
    service = args[-1]
    with LOG.open("a") as handle:
        handle.write("build " + service + "\\n")
    if FAIL_SERVICE.exists() and FAIL_SERVICE.read_text().strip() == service:
        raise SystemExit(42)
    raise SystemExit(0)
raise SystemExit(90)
""".format(log=str(log), fail=str(fail_service)))
script.chmod(0o700)
PY
/usr/bin/python3 - "$ROOT/deploy/op-deploy.sh" "$config_fake_docker" <<'PY'
from pathlib import Path
import sys
launcher = Path(sys.argv[1])
source = launcher.read_text()
assert source.count('/usr/bin/docker') == 1
launcher.write_text(source.replace('/usr/bin/docker', sys.argv[2]))
PY
OP_DB_PASSWORD=password OP_APP_IMAGE=registry.invalid/app:latest \
    "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" config > "$TMP_ROOT/config.json"
/usr/bin/python3 - "$TMP_ROOT/config.json" <<'PY'
import json,sys
c=json.load(open(sys.argv[1]))
assert c['services']['app']['environment']['DB_PASSWORD']=='J7mQ2_vL9xR4-tN6.kP8'
assert c['services']['app']['image']=='registry.internal/weknora-app:v1.2.3'
PY
/usr/bin/cp -- "$TMP_ROOT/op-deploy.pre-config-fake" "$ROOT/deploy/op-deploy.sh"
LAUNCHER_RESTORE=""
pass ambient_env_isolated

# Image grammar negative and digest positive coverage.
for bad in ':latest' 'Repo/name:v1' '/repo/name:v1' 'repo//name:v1' 'repo/name:bad:tag' 'repo/name@sha256:ABC' 'repo/name:v1@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef' 'repo/name#bad:v1'; do
    write_good_env
    /usr/bin/python3 - "$ENV_FILE" "$bad" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]); bad=sys.argv[2]
p.write_text(p.read_text().replace('OP_APP_IMAGE=registry.internal/weknora-app:v1.2.3', 'OP_APP_IMAGE='+bad))
PY
    /bin/chmod 600 "$ENV_FILE"
    expect_fail "image_$bad" "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
done
write_good_env
/usr/bin/python3 - "$ENV_FILE" <<'PY'
from pathlib import Path
p=Path(__import__('sys').argv[1]); p.write_text(p.read_text().replace('OP_APP_IMAGE=registry.internal/weknora-app:v1.2.3','OP_APP_IMAGE=registry.internal/weknora-app@sha256:'+'0123456789abcdef'*4))
PY
/bin/chmod 600 "$ENV_FILE"
"$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate >/dev/null
pass image_digest

# Secret, origin, and bounded-input coverage.
for replacement in 'P@ssw0rd-Long-Value-2026!' 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA' 'Abcdefghijklmnopqrstuvwxyz12!' 'qwerty-QWERTY-1234567890!' 'sdpivotsdpivot123!AaBbCc'; do
    write_good_env
    /usr/bin/python3 - "$ENV_FILE" "$replacement" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]); p.write_text(p.read_text().replace('J7mQ2_vL9xR4-tN6.kP8',sys.argv[2]))
PY
    /bin/chmod 600 "$ENV_FILE"
    expect_fail secret_pattern "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
done
write_good_env
/usr/bin/python3 - "$ENV_FILE" <<'PY'
from pathlib import Path
p=Path(__import__('sys').argv[1]); s=p.read_text(); s=s.replace('V5@pH8!zC3#sK7%yD2&m','J7mQ2_vL9xR4-tN6.kP8'); p.write_text(s)
PY
/bin/chmod 600 "$ENV_FILE"
expect_fail correlated_secrets "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
write_good_env
/usr/bin/python3 - "$ENV_FILE" <<'PY'
from pathlib import Path
p=Path(__import__('sys').argv[1]); p.write_text(p.read_text().replace('J7mQ2_vL9xR4-tN6.kP8','J7!mQ2#vL9@xR4%tN6&k'))
PY
/bin/chmod 600 "$ENV_FILE"
expect_fail db_password_url_unsafe "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
for origin in 'https://user@host.test' 'https://host.test/path' 'https://host.test?x=1' 'https://host.test#x' 'https://*.host.test' 'https://example.com'; do
    write_good_env
    /usr/bin/python3 - "$ENV_FILE" "$origin" <<'PY'
from pathlib import Path
import sys
p=Path(sys.argv[1]); p.write_text(p.read_text().replace('https://op.internal.test:8443',sys.argv[2]))
PY
    /bin/chmod 600 "$ENV_FILE"
    expect_fail origin_invalid "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
done
write_good_env
/usr/bin/python3 - "$ENV_FILE" <<'PY'
from pathlib import Path
p=Path(__import__('sys').argv[1]); p.write_bytes(p.read_bytes()+b'TZ=bad\x01value\n')
PY
/bin/chmod 600 "$ENV_FILE"
expect_fail control_character "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
write_good_env
/usr/bin/python3 - "$ENV_FILE" <<'PY'
from pathlib import Path
p=Path(__import__('sys').argv[1]); p.write_text(p.read_text()+'TZ='+'A'*5000+'\n')
PY
/bin/chmod 600 "$ENV_FILE"
expect_fail oversized_line "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
write_good_env
/usr/bin/python3 - "$ENV_FILE" <<'PY'
from pathlib import Path
p=Path(__import__('sys').argv[1]); p.write_bytes(b'\xef\xbb\xbf'+p.read_bytes())
PY
/bin/chmod 600 "$ENV_FILE"
expect_fail utf8_bom "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
write_good_env
printf '%s\n' 'UNKNOWN_KEY=value' >> "$ENV_FILE"
expect_fail unknown_key "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
write_good_env
printf '%s\n' 'OP_DB_USER=duplicate' >> "$ENV_FILE"
expect_fail duplicate_key "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
write_good_env
/usr/bin/python3 - "$ENV_FILE" <<'PY'
from pathlib import Path
p=Path(__import__('sys').argv[1]); p.write_bytes(p.read_bytes()+b'#'+b'A'*65536+b'\n')
PY
/bin/chmod 600 "$ENV_FILE"
expect_fail oversized_file "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate

# Artifact provenance and manifest replacement detection.
write_good_env
/bin/ln -sf /bin/true "$ROOT/frontend/sdpivot/sdp-server"
expect_fail backend_symlink "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
/bin/rm "$ROOT/frontend/sdpivot/sdp-server"
printf '%s\n' '#!/bin/sh' 'exit 0' > "$ROOT/frontend/sdpivot/sdp-server"
/bin/chmod 700 "$ROOT/frontend/sdpivot/sdp-server"
/bin/rm "$ROOT/frontend/sdpivot/dist/index.html"
/bin/ln -s /etc/hosts "$ROOT/frontend/sdpivot/dist/index.html"
expect_fail index_symlink "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
/bin/rm "$ROOT/frontend/sdpivot/dist/index.html"
printf '<!doctype html>\n' > "$ROOT/frontend/sdpivot/dist/index.html"
/bin/ln "$ROOT/frontend/sdpivot/dist/index.html" "$ROOT/frontend/sdpivot/dist/index-hardlink.html"
expect_fail index_hardlink "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
/bin/rm "$ROOT/frontend/sdpivot/dist/index-hardlink.html"
/bin/ln "$ROOT/frontend/sdpivot/sdp-server" "$ROOT/frontend/sdpivot/sdp-server-hardlink"
expect_fail backend_hardlink "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
/bin/rm "$ROOT/frontend/sdpivot/sdp-server-hardlink"
/bin/ln -s missing-target "$ROOT/frontend/sdpivot/dist/ops.html"
expect_fail dangling_ops "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate
/bin/rm "$ROOT/frontend/sdpivot/dist/ops.html"
manifest_dir="$TMP_ROOT/manifest-work"
/bin/mkdir -m 700 "$manifest_dir"
manifest="$manifest_dir/test.manifest"
: > "$manifest"
/bin/chmod 600 "$manifest"
"$ROOT/deploy/validate-op-deployment.sh" artifacts "$manifest"
/usr/bin/python3 - "$manifest" <<'PY'
from pathlib import Path
import sys

data = Path(sys.argv[1]).read_bytes()
assert b"docs/\xe4\xb8\xad\xe6\x96\x87\xe7\x9b\xae\xe5\xbd\x95/\xe6\x95\xb0\xe6\x8d\xae\xe6\xba\x90\xe8\xaf\xb4\xe6\x98\x8e.md\n" in data
assert data.decode("utf-8").encode("utf-8") == data
PY
"$ROOT/deploy/validate-op-deployment.sh" compare-artifacts "$manifest"
pass artifact_utf8_generate_and_compare
stage_root="$TMP_ROOT/stage"
/bin/mkdir -m 700 "$stage_root"
override="$TMP_ROOT/stage.override.yml"
: > "$override"
/bin/chmod 600 "$override"
"$ROOT/deploy/validate-op-deployment.sh" stage "$manifest" "$stage_root" "$override"
"$ROOT/deploy/validate-op-deployment.sh" compare-stage "$manifest" "$stage_root"
[[ -f "$stage_root/docs/中文目录/数据源说明.md" ]] || { printf 'FAIL UTF-8 artifact was not staged\n' >&2; exit 1; }
pass artifact_utf8_stage_and_compare
/bin/chmod 700 "$stage_root" "$stage_root/docs" "$stage_root/docs/中文目录"
/bin/chmod 600 "$stage_root/docs/中文目录/数据源说明.md"
printf 'changed\n' >> "$stage_root/docs/中文目录/数据源说明.md"
/bin/chmod 400 "$stage_root/docs/中文目录/数据源说明.md"
/bin/chmod 500 "$stage_root/docs/中文目录" "$stage_root/docs" "$stage_root"
expect_fail artifact_utf8_staged_content_changed "$ROOT/deploy/validate-op-deployment.sh" compare-stage "$manifest" "$stage_root"
/usr/bin/python3 - "$stage_root" <<'PY'
import os
import sys

for current, dirs, names in os.walk(sys.argv[1]):
    os.chmod(current, 0o700)
    for name in names:
        os.chmod(os.path.join(current, name), 0o600)
PY
/bin/rm -rf "$stage_root"
printf '<!-- changed -->\n' >> "$ROOT/frontend/sdpivot/dist/index.html"
expect_fail artifact_manifest_content_changed "$ROOT/deploy/validate-op-deployment.sh" compare-artifacts "$manifest"
printf '<!doctype html>\n' > "$ROOT/frontend/sdpivot/dist/index.html"
/bin/mv "$ROOT/docs/中文目录/数据源说明.md" "$ROOT/docs/中文目录/改名说明.md"
expect_fail artifact_manifest_path_changed "$ROOT/deploy/validate-op-deployment.sh" compare-artifacts "$manifest"
/bin/mv "$ROOT/docs/中文目录/改名说明.md" "$ROOT/docs/中文目录/数据源说明.md"
control_path="$ROOT/docs/control"$'\001'"name.md"
printf 'unsafe\n' > "$control_path"
expect_fail artifact_control_path "$ROOT/deploy/validate-op-deployment.sh" compare-artifacts "$manifest"
/bin/rm -- "$control_path"
newline_path="$ROOT/docs/newline"$'\n'"name.md"
printf 'unsafe\n' > "$newline_path"
expect_fail artifact_newline_path "$ROOT/deploy/validate-op-deployment.sh" compare-artifacts "$manifest"
/bin/rm -- "$newline_path"
/bin/rm "$manifest"
pass artifact_provenance

# Compose builds must be serialized behind a fail-closed resource gate.
/usr/bin/cp -- "$ROOT/deploy/op-deploy.sh" "$TMP_ROOT/op-deploy.pre-build-gates"
LAUNCHER_RESTORE="$TMP_ROOT/op-deploy.pre-build-gates"
/usr/bin/python3 - "$ROOT/deploy/op-deploy.sh" "$config_fake_docker" <<'PY'
from pathlib import Path
import sys
p = Path(sys.argv[1])
s = p.read_text()
assert s.count('/usr/bin/docker') == 1
p.write_text(s.replace('/usr/bin/docker', sys.argv[2]))
PY
probe="$TMP_ROOT/resource-probe"
probe_count="$TMP_ROOT/probe.count"
probe_values="$TMP_ROOT/probe.values"
probe_args="$TMP_ROOT/probe.args"
/usr/bin/python3 - "$probe" "$probe_count" "$probe_values" "$probe_args" <<'PY'
from pathlib import Path
import sys
script, count, values, args = map(Path, sys.argv[1:])
script.write_text("""#!/usr/bin/python3
from pathlib import Path
import sys
count = Path({count!r})
values = Path({values!r})
args = Path({args!r})
with args.open("a") as handle:
    handle.write((sys.argv[1] if len(sys.argv) > 1 else "") + "\\n")
n = int(count.read_text()) if count.exists() else 0
n += 1
count.write_text(str(n) + "\\n")
lines = values.read_text().splitlines()
print(lines[n - 1] if n <= len(lines) else "4096 0 40960")
""".format(count=str(count), values=str(values), args=str(args)))
script.chmod(0o700)
PY
run_build() {
    OP_DEPLOY_INTERNAL_TESTING=1 OP_DEPLOY_TEST_RESOURCE_PROBE="$probe" \
        "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" compose-build
}
reset_build_fixture() {
    /bin/rm -f -- "$ROOT/deploy/.op-build-receipt.json" "$build_log" "$fail_service_file" \
        "$probe_count" "$probe_args"
    printf '%s\n' '4096 0 40960' > "$probe_values"
}
reset_build_fixture
printf '%s\n' 'old receipt' > "$ROOT/deploy/.op-build-receipt.json"
/bin/chmod 600 "$ROOT/deploy/.op-build-receipt.json"
run_build >/dev/null
/usr/bin/python3 - "$build_log" "$probe_count" "$probe_args" "$ROOT/deploy/.op-build-receipt.json" <<'PY'
from pathlib import Path
import json
import stat
import sys
log, count, args, receipt = map(Path, sys.argv[1:])
assert log.read_text().splitlines() == [
    'build migration', 'build sdp-backend', 'build sdp-frontend', 'build app', 'build docreader']
assert count.read_text().strip() == '6'
assert args.read_text().splitlines() == ['/'] * 6
metadata = receipt.stat()
assert stat.S_ISREG(metadata.st_mode) and metadata.st_nlink == 1 and stat.S_IMODE(metadata.st_mode) == 0o600
assert json.loads(receipt.read_text())['version'] == 1
PY
pass compose_build_serial_order_root_probe_and_replaced_receipt

reset_build_fixture
printf '%s\n' 'old receipt' > "$ROOT/deploy/.op-build-receipt.json"
/bin/chmod 600 "$ROOT/deploy/.op-build-receipt.json"
printf '%s\n' 'sdp-backend' > "$fail_service_file"
expect_fail compose_build_second_service_failure run_build
[[ "$(/bin/cat "$build_log")" == $'build migration\nbuild sdp-backend' ]] || { printf 'FAIL build continued after second service failure\n' >&2; exit 1; }
[[ ! -e "$ROOT/deploy/.op-build-receipt.json" ]] || { printf 'FAIL old receipt survived service failure\n' >&2; exit 1; }
pass compose_build_service_failure_stops_and_invalidates_receipt

reset_build_fixture
receipt_target="$TMP_ROOT/receipt-target"
printf '%s\n' 'target unchanged' > "$receipt_target"
/bin/ln -s "$receipt_target" "$ROOT/deploy/.op-build-receipt.json"
expect_fail compose_build_receipt_symlink run_build
[[ -L "$ROOT/deploy/.op-build-receipt.json" && "$(/bin/cat "$receipt_target")" == 'target unchanged' && ! -e "$build_log" ]] || {
    printf 'FAIL unsafe receipt symlink was changed or build started\n' >&2; exit 1;
}
/bin/rm -f -- "$ROOT/deploy/.op-build-receipt.json" "$receipt_target"
pass compose_build_receipt_symlink_preserved

reset_build_fixture
receipt_peer="$TMP_ROOT/receipt-peer"
printf '%s\n' 'hardlink unchanged' > "$receipt_peer"
/bin/chmod 600 "$receipt_peer"
/bin/ln "$receipt_peer" "$ROOT/deploy/.op-build-receipt.json"
expect_fail compose_build_receipt_hardlink run_build
[[ -f "$ROOT/deploy/.op-build-receipt.json" && -f "$receipt_peer" && "$(/bin/cat "$receipt_peer")" == 'hardlink unchanged' && ! -e "$build_log" ]] || {
    printf 'FAIL unsafe receipt hardlink was changed or build started\n' >&2; exit 1;
}
/bin/rm -f -- "$ROOT/deploy/.op-build-receipt.json" "$receipt_peer"
pass compose_build_receipt_hardlink_preserved

reset_build_fixture
printf '%s\n' '4096 0 40960' '4096 0 40960' '700 0 40960' > "$probe_values"
expect_fail compose_build_mid_resource_drop run_build
[[ "$(/bin/cat "$build_log")" == $'build migration\nbuild sdp-backend' && ! -e "$ROOT/deploy/.op-build-receipt.json" ]] || {
    printf 'FAIL resource drop continued building or wrote receipt\n' >&2; exit 1;
}
pass compose_build_mid_resource_drop_no_receipt

reset_build_fixture
printf '%s\n' '4096 0 14336' > "$probe_values"
expect_fail compose_build_default_disk_gate run_build
[[ ! -e "$build_log" && ! -e "$ROOT/deploy/.op-build-receipt.json" ]] || { printf 'FAIL disk gate allowed build\n' >&2; exit 1; }
pass compose_build_default_disk_gate_no_receipt
expect_fail compose_build_extra_service "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" compose-build app
expect_fail compose_build_parallel "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" compose-build --parallel

# Inject only the fake probe into the copied launcher. Without the internal-testing
# switch, ambient threshold overrides must be ignored and the default 20 GiB wins.
/usr/bin/cp -- "$ROOT/deploy/op-deploy.sh" "$TMP_ROOT/op-deploy.fake-before-ambient-threshold"
/usr/bin/python3 - "$ROOT/deploy/op-deploy.sh" "$probe" <<'PY'
from pathlib import Path
import sys
p = Path(sys.argv[1])
s = p.read_text()
old = '    resource_probe = ""\n'
assert s.count(old) == 1
p.write_text(s.replace(old, '    resource_probe = ' + repr(sys.argv[2]) + '\n'))
PY
reset_build_fixture
printf '%s\n' '4096 0 14336' > "$probe_values"
OP_DEPLOY_TEST_MIN_DISK_MIB=1 expect_fail ambient_test_min_disk_override_ignored \
    "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" compose-build
[[ ! -e "$build_log" && ! -e "$ROOT/deploy/.op-build-receipt.json" ]] || {
    printf 'FAIL ambient threshold override bypassed default disk gate\n' >&2; exit 1;
}
/usr/bin/cp -- "$TMP_ROOT/op-deploy.fake-before-ambient-threshold" "$ROOT/deploy/op-deploy.sh"
/usr/bin/cp -- "$TMP_ROOT/op-deploy.pre-build-gates" "$ROOT/deploy/op-deploy.sh"
LAUNCHER_RESTORE=""
pass compose_build_ambient_threshold_override_ignored

printf '%s\n' 'All OP deployment trust-boundary tests passed.'
