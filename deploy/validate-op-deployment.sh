#!/bin/bash
PATH=/usr/bin:/bin
LC_ALL=C
export PATH LC_ALL
unset CDPATH ENV BASH_ENV
set -Eeuo pipefail

fail() {
    printf 'OP deployment validation failed: %s\n' "$1" >&2
    exit 1
}

readonly REALPATH=/usr/bin/realpath
readonly PYTHON=/usr/bin/python3
[[ -x "$REALPATH" && -x "$PYTHON" ]] || fail "required system tools are unavailable"
readonly SCRIPT_PATH="$($REALPATH -- "${BASH_SOURCE[0]}")"
readonly DEPLOY_DIR="${SCRIPT_PATH%/*}"
readonly REPO_ROOT="${DEPLOY_DIR%/*}"

if (( $# < 1 )); then
    fail "internal usage: validate-op-deployment.sh MODE ..."
fi

mode="$1"
shift
case "$mode" in
    env) (( $# == 2 )) || fail "env mode requires ENV_FILE SNAPSHOT" ;;
    artifacts) (( $# == 1 )) || fail "artifacts mode requires MANIFEST" ;;
    compare-artifacts) (( $# == 1 )) || fail "compare-artifacts mode requires MANIFEST" ;;
    stage) (( $# == 3 )) || fail "stage mode requires MANIFEST STAGE_ROOT OVERRIDE" ;;
    compare-stage) (( $# == 2 )) || fail "compare-stage mode requires MANIFEST STAGE_ROOT" ;;
    receipt-write) (( $# == 4 )) || fail "receipt-write mode requires SNAPSHOT MANIFEST CONFIG RECEIPT" ;;
    receipt-verify) (( $# == 4 )) || fail "receipt-verify mode requires SNAPSHOT MANIFEST CONFIG RECEIPT" ;;
    all) (( $# == 3 )) || fail "all mode requires ENV_FILE SNAPSHOT MANIFEST" ;;
    config) (( $# == 2 )) || fail "config mode requires SNAPSHOT CONFIG_JSON" ;;
    *) fail "unsupported internal mode" ;;
esac

exec "$PYTHON" - "$mode" "$DEPLOY_DIR" "$REPO_ROOT" "$@" <<'PY'
import hashlib
import hmac
import ipaddress
import json
import math
import os
import re
import stat
import sys
from pathlib import Path
from urllib.parse import urlsplit

MODE, DEPLOY_DIR, REPO_ROOT, *ARGS = sys.argv[1:]
DEPLOY_DIR = os.path.realpath(DEPLOY_DIR)
REPO_ROOT = os.path.realpath(REPO_ROOT)
UID = os.getuid()
MAX_ENV_SIZE = 65536
MAX_LINE_SIZE = 4096
MAX_VALUE_SIZE = 2048

ALLOWED_KEYS = (
    "COMPOSE_PROJECT_NAME",
    "OP_DB_USER", "OP_DB_PASSWORD", "OP_DB_NAME",
    "OP_REDIS_PASSWORD", "OP_REDIS_DB", "OP_REDIS_PREFIX",
    "OP_JWT_SECRET", "OP_SDP_JWT_SECRET", "OP_TENANT_AES_KEY", "OP_SYSTEM_AES_KEY",
    "OP_ALLOWED_ORIGINS", "OP_HTTP_BIND", "OP_HTTP_PORT",
    "OP_POSTGRES_IMAGE", "OP_REDIS_IMAGE", "OP_APP_IMAGE", "OP_DOCREADER_IMAGE",
    "OP_MIGRATION_IMAGE", "OP_SDP_BACKEND_IMAGE", "OP_SDP_FRONTEND_IMAGE",
    "OP_BOOTSTRAP_ADMIN_EMAIL", "OP_BOOTSTRAP_ADMIN_PASSWORD",
    "OP_BOOTSTRAP_SYSADMIN_EMAIL", "OP_BOOTSTRAP_SYSADMIN_PASSWORD",
    "SDP_ENABLE_LEGACY_ALIAS", "DISABLE_REGISTRATION", "MAX_FILE_SIZE_MB",
    "WEKNORA_ASYNQ_CONCURRENCY", "CONCURRENCY_POOL_SIZE",
    "SDP_DB_MAX_OPEN_CONNS", "SDP_DB_MAX_IDLE_CONNS", "SDP_DB_CONN_MAX_LIFETIME",
    "SDP_READY_TIMEOUT", "DOCREADER_PDF_FORCE_SCANNED", "DOCREADER_ODL_MAX_WORKERS",
    "TZ", "APT_MIRROR", "APK_MIRROR_ARG",
)
ALLOWED = set(ALLOWED_KEYS)
IMAGE_KEYS = (
    "OP_POSTGRES_IMAGE", "OP_REDIS_IMAGE", "OP_APP_IMAGE", "OP_DOCREADER_IMAGE",
    "OP_MIGRATION_IMAGE", "OP_SDP_BACKEND_IMAGE", "OP_SDP_FRONTEND_IMAGE",
)
SECRET_KEYS = (
    "OP_DB_PASSWORD", "OP_REDIS_PASSWORD", "OP_JWT_SECRET",
    "OP_SDP_JWT_SECRET", "OP_TENANT_AES_KEY", "OP_SYSTEM_AES_KEY",
)
REQUIRED_KEYS = (
    "OP_DB_USER", "OP_DB_PASSWORD", "OP_DB_NAME", "OP_REDIS_PASSWORD",
    "OP_JWT_SECRET", "OP_SDP_JWT_SECRET", "OP_TENANT_AES_KEY", "OP_SYSTEM_AES_KEY",
    "OP_ALLOWED_ORIGINS", "OP_HTTP_BIND", "OP_HTTP_PORT", *IMAGE_KEYS,
)
PLACEHOLDERS = ("change_me", "changeme", "placeholder", "replace_me", "dummy", "sample", "todo", "your_", "insert_")
COMMON_WORDS = (
    "password", "admin", "postgres", "redis", "secret", "changeme", "default",
    "sdpivot", "weknora", "qwerty", "letmein", "welcome", "root", "testtest",
)
KEYBOARD = ("qwertyuiop", "asdfghjkl", "zxcvbnm", "1234567890", "0987654321")
COMPONENT_RE = re.compile(r"[a-z0-9]+(?:[._-][a-z0-9]+)*\Z")
HOST_LABEL_RE = re.compile(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\Z")
TAG_RE = re.compile(r"[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}\Z")


def fail(message):
    print(f"OP deployment validation failed: {message}", file=sys.stderr)
    raise SystemExit(1)


def inside(path, base):
    try:
        return os.path.commonpath((os.path.realpath(path), base)) == base
    except ValueError:
        return False


def validate_secure_regular(path, *, base=None, mode=None, executable=False, label="file"):
    try:
        before = os.lstat(path)
    except OSError:
        fail(f"{label} is missing or inaccessible")
    if stat.S_ISLNK(before.st_mode) or not stat.S_ISREG(before.st_mode):
        fail(f"{label} must be a regular non-symlink file")
    if before.st_nlink != 1:
        fail(f"{label} must have exactly one hard link")
    if before.st_uid != UID:
        fail(f"{label} must be owned by the current user")
    if mode is not None and stat.S_IMODE(before.st_mode) != mode:
        fail(f"{label} mode must be {mode:o}")
    if executable and not (before.st_mode & stat.S_IXUSR):
        fail(f"{label} must be executable by its owner")
    real = os.path.realpath(path)
    if base is not None and not inside(real, base):
        fail(f"{label} must resolve inside the allowed repository path")
    return before, real


def read_env_secure(path, *, base=DEPLOY_DIR):
    if not path or path.startswith("-"):
        fail("environment file path must not be empty or option-like")
    before, real = validate_secure_regular(path, base=base, mode=0o600, label="environment file")
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
    try:
        fd = os.open(path, flags)
    except OSError:
        fail("environment file could not be opened safely")
    try:
        opened = os.fstat(fd)
        identity = (before.st_dev, before.st_ino, before.st_uid, before.st_mode, before.st_nlink)
        opened_identity = (opened.st_dev, opened.st_ino, opened.st_uid, opened.st_mode, opened.st_nlink)
        if identity != opened_identity:
            fail("environment file changed while it was opened")
        chunks = []
        total = 0
        while total <= MAX_ENV_SIZE:
            chunk = os.read(fd, min(8192, MAX_ENV_SIZE + 1 - total))
            if not chunk:
                break
            chunks.append(chunk)
            total += len(chunk)
        if total > MAX_ENV_SIZE:
            fail("environment file exceeds 65536 bytes")
        data = b"".join(chunks)
        after = os.fstat(fd)
        if (opened.st_dev, opened.st_ino, opened.st_size, opened.st_mtime_ns, opened.st_ctime_ns) != (
            after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns
        ):
            fail("environment file changed while it was read")
    finally:
        os.close(fd)
    return data, real


def parse_env(data):
    for byte in data:
        if byte == 0x7F or byte > 0x7E or (byte < 0x20 and byte not in (0x0A, 0x0D)):
            fail("environment file contains unsupported control or non-ASCII bytes")
    for index, byte in enumerate(data):
        if byte == 0x0D and (index + 1 >= len(data) or data[index + 1] != 0x0A):
            fail("environment file contains a bare carriage return")
    text = data.replace(b"\r\n", b"\n").decode("ascii")
    values = {}
    for number, line in enumerate(text.splitlines(), 1):
        if len(line.encode("ascii")) > MAX_LINE_SIZE:
            fail(f"line {number} exceeds 4096 bytes")
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            fail(f"line {number} must use literal KEY=VALUE syntax")
        key, value = line.split("=", 1)
        if not re.fullmatch(r"[A-Z][A-Z0-9_]*", key):
            fail(f"line {number} has an invalid key")
        if key not in ALLOWED:
            fail(f"line {number} uses unsupported key {key}")
        if key in values:
            fail(f"duplicate key {key}")
        if len(value.encode("ascii")) > MAX_VALUE_SIZE:
            fail(f"{key} exceeds 2048 bytes")
        if any(ch.isspace() for ch in value):
            fail(f"{key} must be a whitespace-free literal")
        if any(token in value for token in ("$", "`", ";", "&&", "||", "<", ">", "'", '"', "\\")):
            fail(f"{key} contains unsupported shell syntax")
        values[key] = value
    for key in REQUIRED_KEYS:
        if not values.get(key):
            fail(f"required key {key} is missing or empty")
    for key in REQUIRED_KEYS:
        lowered = values[key].lower()
        if any(marker in lowered for marker in PLACEHOLDERS):
            fail(f"{key} still contains a known placeholder")
    return values


def write_secure(path, data, label):
    try:
        before = os.lstat(path)
    except OSError:
        fail(f"{label} output is missing")
    if not stat.S_ISREG(before.st_mode) or stat.S_ISLNK(before.st_mode) or before.st_uid != UID or before.st_nlink != 1 or stat.S_IMODE(before.st_mode) != 0o600:
        fail(f"{label} output must be a current-user 0600 regular file with one link")
    flags = os.O_WRONLY | os.O_TRUNC | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
    try:
        fd = os.open(path, flags)
    except OSError:
        fail(f"{label} output could not be opened safely")
    try:
        opened = os.fstat(fd)
        if (before.st_dev, before.st_ino) != (opened.st_dev, opened.st_ino):
            fail(f"{label} output changed while it was opened")
        view = memoryview(data)
        while view:
            view = view[os.write(fd, view):]
        os.fsync(fd)
    finally:
        os.close(fd)


def canonical_env(values):
    return "".join(f"{key}={values[key]}\n" for key in ALLOWED_KEYS if key in values).encode("ascii")


def normalized_secret(value):
    table = str.maketrans({"@": "a", "4": "a", "3": "e", "1": "i", "!": "i", "0": "o", "5": "s", "7": "t", "+": "t"})
    return "".join(ch for ch in value.lower().translate(table) if ch.isalnum())


def entropy(value):
    counts = {ch: value.count(ch) for ch in set(value)}
    length = len(value)
    return -sum((count / length) * math.log2(count / length) for count in counts.values())


def has_sequence(value):
    lowered = value.lower()
    sequences = ("abcdefghijklmnopqrstuvwxyz", "zyxwvutsrqponmlkjihgfedcba", "0123456789", "9876543210", *KEYBOARD, *(s[::-1] for s in KEYBOARD))
    return any(sequence[i:i + 4] in lowered for sequence in sequences for i in range(len(sequence) - 3))


def is_periodic(value):
    lowered = value.lower()
    for size in range(1, len(lowered) // 2 + 1):
        if len(lowered) % size == 0 and lowered == lowered[:size] * (len(lowered) // size):
            return True
    return False


def validate_secrets(values):
    minimums = {"OP_DB_PASSWORD": 20, "OP_REDIS_PASSWORD": 20, "OP_JWT_SECRET": 32, "OP_SDP_JWT_SECRET": 32, "OP_TENANT_AES_KEY": 32, "OP_SYSTEM_AES_KEY": 32}
    names = {normalized_secret(values["OP_DB_USER"]), normalized_secret(values["OP_DB_NAME"]), "sdpivot", "weknora"}
    for key in SECRET_KEYS:
        value = values[key]
        if key in ("OP_TENANT_AES_KEY", "OP_SYSTEM_AES_KEY"):
            if len(value) != 32:
                fail(f"{key} must be exactly 32 characters")
        elif len(value) < minimums[key]:
            fail(f"{key} is shorter than the required minimum")
        norm = normalized_secret(value)
        if any(word in norm for word in COMMON_WORDS) or any(name and len(name) >= 3 and name in norm for name in names):
            fail(f"{key} is derived from a common word, product, user, or database name")
        if len(set(value)) < 10 or entropy(value) < 3.0:
            fail(f"{key} lacks sufficient character diversity")
        if has_sequence(value) or is_periodic(value):
            fail(f"{key} contains a repeated, sequential, or keyboard pattern")
        classes = sum((any(c.islower() for c in value), any(c.isupper() for c in value), any(c.isdigit() for c in value), any(not c.isalnum() for c in value)))
        if key not in ("OP_TENANT_AES_KEY", "OP_SYSTEM_AES_KEY") and classes < 3:
            fail(f"{key} must use at least three character classes")
    normalized = {key: normalized_secret(values[key]) for key in SECRET_KEYS}
    for i, left_key in enumerate(SECRET_KEYS):
        for right_key in SECRET_KEYS[i + 1:]:
            left, right = normalized[left_key], normalized[right_key]
            if left == right or left in right or right in left:
                fail(f"{left_key} and {right_key} must be independent")


def validate_repository(repository):
    if not repository or repository.startswith("/") or repository.endswith("/") or "//" in repository or repository.lower() != repository:
        return False
    segments = repository.split("/")
    first_is_registry = len(segments) > 1 and ("." in segments[0] or ":" in segments[0] or segments[0] == "localhost")
    start = 0
    if first_is_registry:
        registry = segments[0]
        start = 1
        if registry.count(":") > 1:
            return False
        if ":" in registry:
            host, port = registry.rsplit(":", 1)
            if not port.isdigit() or not 1 <= int(port) <= 65535:
                return False
        else:
            host = registry
        if not host or len(host) > 253:
            return False
        try:
            ipaddress.IPv4Address(host)
        except ValueError:
            if host == "localhost":
                pass
            elif any(not HOST_LABEL_RE.fullmatch(label) for label in host.split(".")):
                return False
    return start < len(segments) and all(COMPONENT_RE.fullmatch(segment) for segment in segments[start:])


def validate_image(key, image):
    if any(ch in image for ch in "#[]") or image.count("@") > 1:
        fail(f"{key} has an invalid image reference")
    if "@" in image:
        repository, digest = image.split("@", 1)
        if ":" in repository.rsplit("/", 1)[-1]:
            fail(f"{key} must not combine a tag and digest")
        if not validate_repository(repository) or not re.fullmatch(r"sha256:[0-9a-f]{64}", digest):
            fail(f"{key} must use repository@sha256 with 64 lowercase hex characters")
        return
    last_slash = image.rfind("/")
    tag_separator = image.rfind(":")
    if tag_separator <= last_slash:
        fail(f"{key} must use an explicit non-latest tag or sha256 digest")
    repository, tag = image[:tag_separator], image[tag_separator + 1:]
    if not validate_repository(repository) or not TAG_RE.fullmatch(tag) or tag.lower() == "latest":
        fail(f"{key} has an invalid or latest image tag")


def validate_origin(value):
    if "," in value or "*" in value:
        fail("OP_ALLOWED_ORIGINS must contain exactly one non-wildcard origin")
    try:
        parsed = urlsplit(value)
        port = parsed.port
    except ValueError:
        fail("OP_ALLOWED_ORIGINS has an invalid host or port")
    if parsed.scheme not in ("http", "https") or not parsed.netloc or parsed.username is not None or parsed.password is not None:
        fail("OP_ALLOWED_ORIGINS must be a plain HTTP(S) origin without userinfo")
    if parsed.path not in ("", "/") or parsed.query or parsed.fragment:
        fail("OP_ALLOWED_ORIGINS must not contain path, query, or fragment data")
    host = parsed.hostname
    if not host or port is not None and not 1 <= port <= 65535:
        fail("OP_ALLOWED_ORIGINS has an invalid host or port")
    if host.lower().endswith("example.com") or host.lower().endswith("example.invalid"):
        fail("OP_ALLOWED_ORIGINS still uses an example domain")
    if ":" in host:
        if not parsed.netloc.startswith("["):
            fail("IPv6 origins must use bracket notation")
        try:
            ipaddress.IPv6Address(host)
        except ValueError:
            fail("OP_ALLOWED_ORIGINS has an invalid IPv6 host")
    else:
        try:
            ipaddress.IPv4Address(host)
        except ValueError:
            if len(host) > 253 or host.lower() != host or any(not re.fullmatch(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", label) for label in host.split(".")):
                fail("OP_ALLOWED_ORIGINS has an invalid hostname")


def validate_values(values):
    if not re.fullmatch(r"[A-Za-z0-9._~-]+", values["OP_DB_USER"]) or not re.fullmatch(r"[A-Za-z0-9._~-]+", values["OP_DB_NAME"]):
        fail("database user and name must use URL-safe literal characters")
    if not re.fullmatch(r"[A-Za-z0-9._~-]+", values["OP_DB_PASSWORD"]):
        fail("OP_DB_PASSWORD must use URL-unreserved characters for the migration connection URL")
    validate_secrets(values)
    validate_origin(values["OP_ALLOWED_ORIGINS"])
    if values["OP_HTTP_BIND"] != "127.0.0.1":
        fail("OP_HTTP_BIND must be exactly 127.0.0.1")
    http_port = values["OP_HTTP_PORT"]
    if not re.fullmatch(r"[1-9][0-9]{0,4}", http_port) or int(http_port) > 65535:
        fail("OP_HTTP_PORT must be a canonical decimal integer from 1 through 65535")
    for key in IMAGE_KEYS:
        validate_image(key, values[key])


def canonical_manifest_path(path):
    relative = os.path.relpath(path, REPO_ROOT)
    if os.sep != "/":
        relative = relative.replace(os.sep, "/")
    return relative


def validate_manifest_path(relative):
    if not relative or relative in (".", "..") or relative.startswith("/") or "\\" in relative:
        fail("artifact manifest contains an unsafe path")
    try:
        relative.encode("utf-8")
    except UnicodeEncodeError:
        fail("artifact manifest path is not valid UTF-8")
    if any(ord(ch) < 0x20 or 0x7F <= ord(ch) <= 0x9F for ch in relative):
        fail("artifact manifest path contains a control character")
    parts = relative.split("/")
    if any(part in ("", ".", "..") for part in parts):
        fail("artifact manifest contains an unsafe path")
    return relative


def manifest_sort_key(path):
    return validate_manifest_path(canonical_manifest_path(path)).encode("utf-8")


def secure_file_row(path, label):
    before, real = validate_secure_regular(path, base=REPO_ROOT, label=label)
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
    try:
        fd = os.open(path, flags)
    except OSError:
        fail(f"{label} could not be opened safely")
    try:
        opened = os.fstat(fd)
        if (before.st_dev, before.st_ino, before.st_uid, before.st_mode, before.st_nlink) != (
            opened.st_dev, opened.st_ino, opened.st_uid, opened.st_mode, opened.st_nlink
        ):
            fail(f"{label} changed while it was opened")
        digest = hashlib.sha256()
        total = 0
        while True:
            chunk = os.read(fd, 1024 * 1024)
            if not chunk:
                break
            digest.update(chunk)
            total += len(chunk)
        after = os.fstat(fd)
        if (opened.st_dev, opened.st_ino, opened.st_size, opened.st_mtime_ns, opened.st_ctime_ns) != (
            after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns
        ):
            fail(f"{label} changed while it was hashed")
    finally:
        os.close(fd)
    relative = validate_manifest_path(canonical_manifest_path(real))
    return f"{digest.hexdigest()}\t{opened.st_dev}\t{opened.st_ino}\t{stat.S_IMODE(opened.st_mode):o}\t{total}\t{opened.st_mtime_ns}\t{relative}\n"


def add_tree(files, relative, *, optional=False):
    root = os.path.join(REPO_ROOT, relative)
    if optional and not os.path.lexists(root):
        return
    try:
        root_stat = os.lstat(root)
    except OSError:
        fail(f"{relative} is missing")
    if stat.S_ISLNK(root_stat.st_mode) or not stat.S_ISDIR(root_stat.st_mode) or not inside(root, REPO_ROOT):
        fail(f"{relative} must be a real directory inside the repository")
    for current, dirs, names in os.walk(root, topdown=True, followlinks=False):
        dirs.sort()
        names.sort()
        for name in dirs:
            path = os.path.join(current, name)
            item = os.lstat(path)
            if stat.S_ISLNK(item.st_mode) or not stat.S_ISDIR(item.st_mode) or not inside(path, REPO_ROOT):
                fail(f"{relative} contains an unsafe directory entry")
        for name in names:
            files.add(os.path.join(current, name))


def artifact_manifest():
    files = set()
    individual = (
        ".dockerignore", "go.mod", "go.sum", "Makefile",
        "deploy/docker-compose.op.yml", "deploy/migration/Dockerfile", "deploy/migrate-op.sh",
        "docker/Dockerfile.app", "docker/Dockerfile.docreader",
        "frontend/sdpivot/Dockerfile.backend", "frontend/Dockerfile", "frontend/nginx.conf",
        "frontend/docker-entrypoint.sh", "frontend/sdpivot/sdp-server",
    )
    for relative in individual:
        path = os.path.join(REPO_ROOT, relative)
        validate_secure_regular(path, base=REPO_ROOT, executable=(relative == "frontend/sdpivot/sdp-server"), label=relative)
        files.add(path)
    for relative in (
        "cmd", "config", "dataset", "deps", "docs", "internal", "migrations", "packages",
        "scripts", "skills", "docreader", "frontend/dist",
    ):
        add_tree(files, relative)
    dist = os.path.join(REPO_ROOT, "frontend/dist")
    if os.path.lexists(os.path.join(dist, "ops.html")):
        fail("frontend/dist/ops.html must not exist, including as a dangling symlink")
    assets = os.path.join(dist, "assets")
    if os.path.isdir(assets) and any(
        "OpsPage" in name or "OpsLoginPage" in name for name in os.listdir(assets)
    ):
        fail("OP build contains ops page chunks")
    index = os.path.join(dist, "index.html")
    validate_secure_regular(index, base=REPO_ROOT, label="frontend/dist/index.html")
    rows = []
    for path in sorted(files, key=manifest_sort_key):
        rows.append(secure_file_row(path, canonical_manifest_path(path)))
    return "".join(rows).encode("utf-8")


def load_values_from_snapshot(path):
    data, _ = read_env_secure(path, base=None)
    values = parse_env(data)
    validate_values(values)
    return values


def secure_manifest_bytes(path):
    before, real = validate_secure_regular(path, mode=0o600, label="artifact manifest")
    parent = os.path.dirname(real)
    try:
        parent_stat = os.lstat(parent)
    except OSError:
        fail("artifact manifest workdir is inaccessible")
    if stat.S_ISLNK(parent_stat.st_mode) or not stat.S_ISDIR(parent_stat.st_mode) or parent_stat.st_uid != UID or stat.S_IMODE(parent_stat.st_mode) != 0o700:
        fail("artifact manifest must be inside a current-user 0700 private workdir")
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
    try:
        fd = os.open(path, flags)
    except OSError:
        fail("artifact manifest could not be opened safely")
    try:
        opened = os.fstat(fd)
        if (before.st_dev, before.st_ino, before.st_uid, before.st_mode, before.st_nlink) != (
            opened.st_dev, opened.st_ino, opened.st_uid, opened.st_mode, opened.st_nlink
        ):
            fail("artifact manifest changed while it was opened")
        chunks = []
        total = 0
        while total <= 1024 * 1024:
            chunk = os.read(fd, min(65536, 1024 * 1024 + 1 - total))
            if not chunk:
                break
            chunks.append(chunk)
            total += len(chunk)
        if total > 1024 * 1024:
            fail("artifact manifest is unexpectedly large")
        data = b"".join(chunks)
        after = os.fstat(fd)
        if (opened.st_dev, opened.st_ino, opened.st_size, opened.st_mtime_ns, opened.st_ctime_ns) != (
            after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns
        ):
            fail("artifact manifest changed while it was read")
    finally:
        os.close(fd)
    return data


def parse_manifest(path):
    rows = {}
    data = secure_manifest_bytes(path)
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError:
        fail("artifact manifest is not valid UTF-8")
    for line in text.splitlines():
        parts = line.split("\t")
        if len(parts) != 7 or not re.fullmatch(r"[0-9a-f]{64}", parts[0]) or not parts[4].isdigit():
            fail("artifact manifest contains an invalid row")
        relative = validate_manifest_path(parts[6])
        if relative in rows:
            fail("artifact manifest contains an unsafe path")
        rows[relative] = (parts[0], int(parts[4]), int(parts[3], 8))
    if not rows:
        fail("artifact manifest is empty")
    return rows, data


def open_regular_at(root_fd, relative, label):
    parts = Path(relative).parts
    current_fd = os.dup(root_fd)
    try:
        for part in parts[:-1]:
            flags = os.O_RDONLY | getattr(os, "O_DIRECTORY", 0) | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
            next_fd = os.open(part, flags, dir_fd=current_fd)
            os.close(current_fd)
            current_fd = next_fd
            directory = os.fstat(current_fd)
            if not stat.S_ISDIR(directory.st_mode) or directory.st_uid != UID:
                fail(f"{label} has an unsafe parent directory")
        flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
        fd = os.open(parts[-1], flags, dir_fd=current_fd)
    except OSError:
        fail(f"{label} could not be opened safely")
    finally:
        os.close(current_fd)
    opened = os.fstat(fd)
    if not stat.S_ISREG(opened.st_mode) or opened.st_uid != UID or opened.st_nlink != 1:
        os.close(fd)
        fail(f"{label} must be a current-user single-link regular file")
    return fd, opened


def stage_context(manifest_path, stage_root, override_path):
    rows, _ = parse_manifest(manifest_path)
    try:
        stage_stat = os.lstat(stage_root)
    except OSError:
        fail("stage root is unavailable")
    if stat.S_ISLNK(stage_stat.st_mode) or not stat.S_ISDIR(stage_stat.st_mode) or stage_stat.st_uid != UID or stat.S_IMODE(stage_stat.st_mode) != 0o700:
        fail("stage root must be a current-user 0700 directory")
    repo_fd = os.open(REPO_ROOT, os.O_RDONLY | getattr(os, "O_DIRECTORY", 0) | getattr(os, "O_CLOEXEC", 0))
    try:
        for relative, (expected_hash, expected_size, source_mode) in sorted(rows.items()):
            source_fd, opened = open_regular_at(repo_fd, relative, relative)
            target = os.path.join(stage_root, relative)
            parent = os.path.dirname(target)
            os.makedirs(parent, mode=0o700, exist_ok=True)
            digest = hashlib.sha256()
            total = 0
            flags = os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
            target_fd = os.open(target, flags, 0o400)
            try:
                while True:
                    chunk = os.read(source_fd, 1024 * 1024)
                    if not chunk:
                        break
                    digest.update(chunk)
                    total += len(chunk)
                    view = memoryview(chunk)
                    while view:
                        view = view[os.write(target_fd, view):]
                source_after = os.fstat(source_fd)
                if (opened.st_dev, opened.st_ino, opened.st_size, opened.st_mtime_ns, opened.st_ctime_ns) != (
                    source_after.st_dev, source_after.st_ino, source_after.st_size, source_after.st_mtime_ns, source_after.st_ctime_ns
                ):
                    fail(f"{relative} changed while the staged snapshot was copied")
                if total != expected_size or not hmac.compare_digest(digest.hexdigest(), expected_hash):
                    fail(f"{relative} does not match the trusted artifact manifest")
                os.fchmod(target_fd, 0o500 if source_mode & 0o100 else 0o400)
                os.fsync(target_fd)
            finally:
                os.close(source_fd)
                os.close(target_fd)
        for current, dirs, _ in os.walk(stage_root):
            for directory in dirs:
                os.chmod(os.path.join(current, directory), 0o500)
        os.chmod(stage_root, 0o500)
    finally:
        os.close(repo_fd)
    override = {
        "services": {
            "migration": {"build": {"context": stage_root, "dockerfile": "deploy/migration/Dockerfile"}},
            "docreader": {"build": {"context": stage_root, "dockerfile": "docker/Dockerfile.docreader"}},
            "app": {"build": {"context": stage_root, "dockerfile": "docker/Dockerfile.app"}},
            "sdp-backend": {"build": {"context": os.path.join(stage_root, "frontend/sdpivot"), "dockerfile": "Dockerfile.backend"}},
            "sdp-frontend": {"build": {"context": os.path.join(stage_root, "frontend/sdpivot"), "dockerfile": "Dockerfile.frontend"}},
        }
    }
    write_secure(override_path, (json.dumps(override, sort_keys=True) + "\n").encode("ascii"), "Compose stage override")


def compare_stage(manifest_path, stage_root):
    rows, _ = parse_manifest(manifest_path)
    stage_real = os.path.realpath(stage_root)
    for relative, (expected_hash, expected_size, _) in sorted(rows.items()):
        path = os.path.join(stage_real, relative)
        try:
            before = os.lstat(path)
        except OSError:
            fail(f"staged {relative} is missing")
        if not inside(path, stage_real) or stat.S_ISLNK(before.st_mode) or not stat.S_ISREG(before.st_mode) or before.st_uid != UID or before.st_nlink != 1:
            fail(f"staged {relative} must be a current-user single-link regular file")
        if stat.S_IMODE(before.st_mode) not in (0o400, 0o500):
            fail(f"staged {relative} must be read-only")
        flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
        try:
            fd = os.open(path, flags)
        except OSError:
            fail(f"staged {relative} could not be opened safely")
        digest = hashlib.sha256()
        try:
            opened = os.fstat(fd)
            if (before.st_dev, before.st_ino, before.st_uid, before.st_mode, before.st_nlink) != (
                opened.st_dev, opened.st_ino, opened.st_uid, opened.st_mode, opened.st_nlink
            ):
                fail(f"staged {relative} changed while it was opened")
            total = 0
            while True:
                chunk = os.read(fd, 1024 * 1024)
                if not chunk:
                    break
                digest.update(chunk)
                total += len(chunk)
            after = os.fstat(fd)
            if (opened.st_dev, opened.st_ino, opened.st_size, opened.st_mtime_ns, opened.st_ctime_ns) != (
                after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns
            ):
                fail(f"staged {relative} changed while it was hashed")
        finally:
            os.close(fd)
        if total != expected_size or not hmac.compare_digest(digest.hexdigest(), expected_hash):
            fail(f"staged {relative} differs from the trusted artifact manifest")


def secure_bytes(path, label, *, mode=0o600):
    before, _ = validate_secure_regular(path, mode=mode, label=label)
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
    fd = os.open(path, flags)
    try:
        opened = os.fstat(fd)
        if (before.st_dev, before.st_ino, before.st_uid, before.st_mode, before.st_nlink) != (opened.st_dev, opened.st_ino, opened.st_uid, opened.st_mode, opened.st_nlink):
            fail(f"{label} changed while it was opened")
        chunks = []
        while True:
            chunk = os.read(fd, 65536)
            if not chunk:
                break
            chunks.append(chunk)
        after = os.fstat(fd)
        if (opened.st_dev, opened.st_ino, opened.st_size, opened.st_mtime_ns, opened.st_ctime_ns) != (after.st_dev, after.st_ino, after.st_size, after.st_mtime_ns, after.st_ctime_ns):
            fail(f"{label} changed while it was read")
        return b"".join(chunks)
    finally:
        os.close(fd)


def receipt_payload(snapshot, manifest, config):
    values = load_values_from_snapshot(snapshot)
    return {
        "version": 1,
        "env_sha256": hashlib.sha256(secure_bytes(snapshot, "environment snapshot")).hexdigest(),
        "manifest_sha256": hashlib.sha256(secure_manifest_bytes(manifest)).hexdigest(),
        "compose_config_sha256": hashlib.sha256(secure_bytes(config, "Compose config output")).hexdigest(),
        "images": {key: values[key] for key in IMAGE_KEYS},
    }


def write_receipt(snapshot, manifest, config, receipt):
    receipt = os.path.abspath(receipt)
    if os.path.dirname(receipt) != DEPLOY_DIR:
        fail("build receipt must be stored in the deployment directory")
    payload = (json.dumps(receipt_payload(snapshot, manifest, config), sort_keys=True) + "\n").encode("ascii")
    directory_fd = os.open(DEPLOY_DIR, os.O_RDONLY | getattr(os, "O_DIRECTORY", 0) | getattr(os, "O_CLOEXEC", 0))
    temporary = f".op-build-receipt.{os.getpid()}.tmp"
    flags = os.O_WRONLY | os.O_CREAT | os.O_EXCL | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
    try:
        fd = os.open(temporary, flags, 0o600, dir_fd=directory_fd)
        try:
            view = memoryview(payload)
            while view:
                view = view[os.write(fd, view):]
            os.fsync(fd)
        finally:
            os.close(fd)
        os.replace(temporary, os.path.basename(receipt), src_dir_fd=directory_fd, dst_dir_fd=directory_fd)
        os.fsync(directory_fd)
    except OSError:
        try:
            os.unlink(temporary, dir_fd=directory_fd)
        except OSError:
            pass
        fail("build receipt could not be stored safely")
    finally:
        os.close(directory_fd)


def verify_receipt(snapshot, manifest, config, receipt):
    try:
        recorded = json.loads(secure_bytes(receipt, "build receipt").decode("ascii"))
    except Exception:
        fail("build receipt is invalid")
    expected = receipt_payload(snapshot, manifest, config)
    left = json.dumps(recorded, sort_keys=True, separators=(",", ":")).encode("ascii")
    right = json.dumps(expected, sort_keys=True, separators=(",", ":")).encode("ascii")
    if len(left) != len(right) or not hmac.compare_digest(hashlib.sha256(left).digest(), hashlib.sha256(right).digest()):
        fail("build receipt does not match the current validated deployment state")


def verify_config(snapshot, config_path):
    values = load_values_from_snapshot(snapshot)
    try:
        config = json.loads(Path(config_path).read_text(encoding="utf-8"))
    except Exception:
        fail("Compose config output is not valid JSON")
    services = config.get("services", {})
    expected_services = {"postgres", "redis", "migration", "docreader", "app", "sdp-backend", "sdp-frontend"}
    if not isinstance(services, dict) or set(services) != expected_services:
        fail("Compose config has an unexpected service set")
    project = config.get("name")
    networks = config.get("networks")
    expected_networks = {"op-internal", "op-ingress"}
    if not isinstance(project, str) or not project or not isinstance(networks, dict) or set(networks) != expected_networks:
        fail("Compose config has an unexpected network set")
    for network_name in expected_networks:
        network = networks.get(network_name)
        if not isinstance(network, dict) or network.get("name") != f"{project}_{network_name}" or network.get("external") is True:
            fail(f"Compose network {network_name} is not isolated by the Compose project")
    if networks["op-internal"].get("internal") is not True:
        fail("Compose network op-internal must be internal")
    if networks["op-ingress"].get("internal") not in (None, False):
        fail("Compose network op-ingress must not be internal")

    def service_networks(service):
        attached = services[service].get("networks")
        if not isinstance(attached, dict):
            fail(f"Compose service {service} must declare its networks explicitly")
        return set(attached)

    if service_networks("sdp-frontend") != expected_networks:
        fail("Compose service sdp-frontend must connect only to op-internal and op-ingress")
    for service in expected_services - {"sdp-frontend"}:
        if service_networks(service) != {"op-internal"}:
            fail(f"Compose service {service} must connect only to op-internal")

    for service in expected_services - {"sdp-frontend"}:
        if services[service].get("ports"):
            fail(f"Compose service {service} must not publish ports")
    frontend_ports = services["sdp-frontend"].get("ports")
    if not isinstance(frontend_ports, list) or len(frontend_ports) != 1 or not isinstance(frontend_ports[0], dict):
        fail("Compose service sdp-frontend must publish exactly one port")
    frontend_port = frontend_ports[0]
    required_port_fields = {"target", "published", "host_ip", "protocol"}
    if not required_port_fields.issubset(frontend_port):
        fail("Compose service sdp-frontend published port is missing required fields")
    if (type(frontend_port["target"]) is not int or frontend_port["target"] != 80
            or type(frontend_port["published"]) is not str or frontend_port["published"] != values["OP_HTTP_PORT"]
            or type(frontend_port["host_ip"]) is not str or frontend_port["host_ip"] != "127.0.0.1"
            or frontend_port["host_ip"] != values["OP_HTTP_BIND"]
            or type(frontend_port["protocol"]) is not str or frontend_port["protocol"] != "tcp"):
        fail("Compose service sdp-frontend published port does not match the validated snapshot")

    expected_images = {
        "postgres": values["OP_POSTGRES_IMAGE"], "redis": values["OP_REDIS_IMAGE"],
        "app": values["OP_APP_IMAGE"], "docreader": values["OP_DOCREADER_IMAGE"],
        "migration": values["OP_MIGRATION_IMAGE"], "sdp-backend": values["OP_SDP_BACKEND_IMAGE"],
        "sdp-frontend": values["OP_SDP_FRONTEND_IMAGE"],
    }
    for service, expected in expected_images.items():
        actual = services.get(service, {}).get("image", "")
        if len(actual) != len(expected) or not hmac.compare_digest(hashlib.sha256(actual.encode()).digest(), hashlib.sha256(expected.encode()).digest()):
            fail(f"Compose config does not match the validated snapshot for {service} image")
    pairs = (
        ("postgres", "POSTGRES_PASSWORD", "OP_DB_PASSWORD"),
        ("redis", "REDIS_PASSWORD", "OP_REDIS_PASSWORD"),
        ("app", "DB_PASSWORD", "OP_DB_PASSWORD"),
        ("app", "REDIS_PASSWORD", "OP_REDIS_PASSWORD"),
        ("app", "JWT_SECRET", "OP_JWT_SECRET"),
        ("app", "TENANT_AES_KEY", "OP_TENANT_AES_KEY"),
        ("app", "SYSTEM_AES_KEY", "OP_SYSTEM_AES_KEY"),
        ("sdp-backend", "SDP_DB_PASSWORD", "OP_DB_PASSWORD"),
        ("sdp-backend", "SDP_REDIS_PASSWORD", "OP_REDIS_PASSWORD"),
        ("sdp-backend", "SDP_JWT_SECRET", "OP_SDP_JWT_SECRET"),
    )
    for service, env_key, source_key in pairs:
        actual = str(services.get(service, {}).get("environment", {}).get(env_key, ""))
        expected = values[source_key]
        if len(actual) != len(expected) or not hmac.compare_digest(hashlib.sha256(actual.encode()).digest(), hashlib.sha256(expected.encode()).digest()):
            fail(f"Compose config does not match the validated snapshot for {service}.{env_key}")
    expected_url = f"postgresql://{values['OP_DB_USER']}:{values['OP_DB_PASSWORD']}@postgres:5432/{values['OP_DB_NAME']}?sslmode=disable"
    actual_url = str(services.get("migration", {}).get("environment", {}).get("OP_DATABASE_URL", ""))
    if len(actual_url) != len(expected_url) or not hmac.compare_digest(hashlib.sha256(actual_url.encode()).digest(), hashlib.sha256(expected_url.encode()).digest()):
        fail("Compose migration URL does not match the validated snapshot")


if MODE in ("env", "all"):
    env_path, snapshot = ARGS[0], ARGS[1]
    data, _ = read_env_secure(env_path)
    values = parse_env(data)
    validate_values(values)
    write_secure(snapshot, canonical_env(values), "environment snapshot")
if MODE in ("artifacts", "all"):
    manifest = ARGS[-1]
    write_secure(manifest, artifact_manifest(), "artifact manifest")
elif MODE == "compare-artifacts":
    expected = secure_manifest_bytes(ARGS[0])
    actual = artifact_manifest()
    if len(actual) != len(expected) or not hmac.compare_digest(hashlib.sha256(actual).digest(), hashlib.sha256(expected).digest()):
        fail("artifact state changed after its trusted manifest was recorded")
elif MODE == "stage":
    stage_context(ARGS[0], ARGS[1], ARGS[2])
elif MODE == "compare-stage":
    compare_stage(ARGS[0], ARGS[1])
elif MODE == "receipt-write":
    write_receipt(*ARGS)
elif MODE == "receipt-verify":
    verify_receipt(*ARGS)
elif MODE == "config":
    verify_config(ARGS[0], ARGS[1])
PY
