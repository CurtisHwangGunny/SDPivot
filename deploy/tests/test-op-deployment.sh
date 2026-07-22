#!/bin/bash
PATH=/usr/bin:/bin
LC_ALL=C
export PATH LC_ALL
unset CDPATH ENV BASH_ENV
set -Eeuo pipefail
umask 077

readonly SOURCE_ROOT="$(/usr/bin/realpath -- "${1:-$(pwd)}")"
readonly TMP_ROOT="$(/usr/bin/mktemp -d /tmp/op-deploy-test.XXXXXX)"
trap '/bin/rm -rf -- "$TMP_ROOT"' EXIT HUP INT TERM
readonly ROOT="$TMP_ROOT/repo"
/bin/mkdir -p -- "$ROOT/deploy/tests" "$ROOT/deploy/migration" "$ROOT/docker" "$ROOT/frontend/sdpivot/dist" \
    "$ROOT/frontend/sdpivot/deps/cppjieba/dict" "$ROOT/cmd" "$ROOT/config" "$ROOT/dataset" \
    "$ROOT/deps" "$ROOT/docs" "$ROOT/internal" "$ROOT/migrations" "$ROOT/packages" \
    "$ROOT/scripts" "$ROOT/skills" "$ROOT/docreader"
for relative in \
    .dockerignore go.mod go.sum Makefile \
    deploy/docker-compose.op.yml deploy/migrate-op.sh deploy/migration/Dockerfile \
    deploy/validate-op-deployment.sh deploy/op-deploy.sh deploy/tests/test-op-deployment.sh \
    docker/Dockerfile.app docker/Dockerfile.docreader \
    frontend/sdpivot/Dockerfile.backend frontend/sdpivot/Dockerfile.frontend \
    frontend/sdpivot/nginx.conf; do
    /usr/bin/cp -- "$SOURCE_ROOT/$relative" "$ROOT/$relative"
done
printf '%s\n' '#!/bin/sh' 'exit 0' > "$ROOT/frontend/sdpivot/sdp-server"
/bin/chmod 700 "$ROOT/frontend/sdpivot/sdp-server"
printf '<!doctype html>\n' > "$ROOT/frontend/sdpivot/dist/index.html"

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
"$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" validate >/dev/null
pass signal_child_reaped_and_lock_released

# Ambient OP_* values must not override the validated snapshot.
OP_DB_PASSWORD=password OP_APP_IMAGE=registry.invalid/app:latest \
    "$ROOT/deploy/op-deploy.sh" --env-file "$ENV_FILE" config > "$TMP_ROOT/config.json"
/usr/bin/python3 - "$TMP_ROOT/config.json" <<'PY'
import json,sys
c=json.load(open(sys.argv[1]))
assert c['services']['app']['environment']['DB_PASSWORD']=='J7mQ2_vL9xR4-tN6.kP8'
assert c['services']['app']['image']=='registry.internal/weknora-app:v1.2.3'
PY
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
manifest="$ROOT/deploy/test.manifest"
: > "$manifest"
/bin/chmod 600 "$manifest"
"$ROOT/deploy/validate-op-deployment.sh" artifacts "$manifest"
printf '<!-- changed -->\n' >> "$ROOT/frontend/sdpivot/dist/index.html"
expect_fail artifact_manifest_changed "$ROOT/deploy/validate-op-deployment.sh" compare-artifacts "$manifest"
/bin/rm "$manifest"
pass artifact_provenance

printf '%s\n' 'All OP deployment trust-boundary tests passed.'
