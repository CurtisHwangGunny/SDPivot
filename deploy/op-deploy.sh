#!/usr/bin/python3 -I
import fcntl
import os
import signal
import stat
import subprocess
import sys
import time

BODY = 'PATH=/usr/bin:/bin\nLC_ALL=C\nexport PATH LC_ALL\nunset CDPATH ENV BASH_ENV\nset -Eeuo pipefail\numask 077\n\nfail() {\n    printf \'OP deployment failed: %s\\n\' "$1" >&2\n    exit 1\n}\n\nreadonly REALPATH=/usr/bin/realpath\nreadonly MKTEMP=/usr/bin/mktemp\nreadonly ENV_BIN=/usr/bin/env\nreadonly DOCKER=/usr/bin/docker\nreadonly NPM=/usr/bin/npm\nreadonly PYTHON=/usr/bin/python3\nreadonly RM=/bin/rm\nreadonly SLEEP=/bin/sleep\nfor tool in "$REALPATH" "$MKTEMP" "$ENV_BIN" "$DOCKER" "$NPM" "$PYTHON" "$RM" "$SLEEP"; do\n    [[ -x "$tool" ]] || fail "required system tool is unavailable"\ndone\n\nreadonly SCRIPT_PATH="$($REALPATH -- "$1")"\nshift\nreadonly RESOURCE_PROBE="$1"\nreadonly MIN_MEM_MIB="$2"\nreadonly MAX_LOAD_MILLI="$3"\nreadonly MIN_DISK_MIB="$4"\nreadonly LOAD_WAIT_ATTEMPTS="$5"\nreadonly LOAD_WAIT_SECONDS="$6"\nreadonly LOAD_STABLE_SAMPLES="$7"\nshift 7\n[[ "$1" == "--" ]] || fail "invalid internal launcher arguments"\nshift\nreadonly DEPLOY_DIR="${SCRIPT_PATH%/*}"\nreadonly REPO_ROOT="${DEPLOY_DIR%/*}"\nreadonly VALIDATOR="$DEPLOY_DIR/validate-op-deployment.sh"\nreadonly COMPOSE_FILE="$DEPLOY_DIR/docker-compose.op.yml"\n[[ -x "$VALIDATOR" ]] || fail "validator is unavailable"\n[[ -f "$COMPOSE_FILE" ]] || fail "Compose file is unavailable"\n\nusage() {\n    printf \'%s\\n\' "usage: $SCRIPT_PATH [--env-file PATH] {validate|config|build-assets|compose-build|up} [up: -d|--detach]" >&2\n    exit 2\n}\n\nenv_file="$DEPLOY_DIR/.env.op"\nif [[ "${1:-}" == "--env-file" ]]; then\n    (( $# >= 3 )) || usage\n    env_file="$2"\n    shift 2\nfi\n(( $# >= 1 )) || usage\naction="$1"\nshift\ncase "$action" in\n    validate|config|build-assets)\n        (( $# == 0 )) || usage\n        ;;\n    compose-build)\n        (( $# == 0 )) || fail "compose-build accepts no additional arguments"\n        ;;\n    up)\n        for arg in "$@"; do\n            case "$arg" in\n                -d|--detach) ;;\n                *) fail "up accepts only -d or --detach; build and no-build overrides are forbidden" ;;\n            esac\n        done\n        ;;\n    *) usage ;;\nesac\n\ntmp_dir="$($MKTEMP -d /tmp/weknora-op-deploy.XXXXXX)"\ncleanup() {\n    local status=$?\n    trap - EXIT HUP INT TERM\n    if ! "$PYTHON" -I - "$tmp_dir" <<\'PY\'\nimport os\nimport stat\nimport sys\n\nroot = os.path.abspath(sys.argv[1])\ntry:\n    root_stat = os.lstat(root)\nexcept FileNotFoundError:\n    raise SystemExit(0)\nif os.path.dirname(root) != "/tmp" or not os.path.basename(root).startswith("weknora-op-deploy."):\n    raise SystemExit(1)\nif not stat.S_ISDIR(root_stat.st_mode) or stat.S_ISLNK(root_stat.st_mode) or root_stat.st_uid != os.getuid():\n    raise SystemExit(1)\nfor current, directories, files in os.walk(root, topdown=True, followlinks=False):\n    current_stat = os.lstat(current)\n    if not stat.S_ISDIR(current_stat.st_mode) or stat.S_ISLNK(current_stat.st_mode) or current_stat.st_uid != os.getuid():\n        raise SystemExit(1)\n    os.chmod(current, 0o700)\n    for name in directories + files:\n        path = os.path.join(current, name)\n        entry = os.lstat(path)\n        if stat.S_ISLNK(entry.st_mode) or entry.st_uid != os.getuid():\n            raise SystemExit(1)\n        os.chmod(path, 0o700 if stat.S_ISDIR(entry.st_mode) else 0o600)\nPY\n    then\n        (( status != 0 )) || status=1\n    fi\n    if ! "$RM" -rf -- "$tmp_dir"; then\n        (( status != 0 )) || status=1\n    fi\n    exit "$status"\n}\ntrap cleanup EXIT HUP INT TERM\nsnapshot="$tmp_dir/env.snapshot"\nmanifest="$tmp_dir/artifacts.manifest"\nconfig_json="$tmp_dir/config.json"\nstage_override="$tmp_dir/stage-override.json"\nstage_root="$tmp_dir/stage"\nprivate_home="$tmp_dir/home"\ndocker_config="$private_home/.docker"\nreceipt="$DEPLOY_DIR/.op-build-receipt.json"\n: > "$snapshot"\n: > "$manifest"\n: > "$config_json"\n: > "$stage_override"\n/bin/mkdir -m 700 -p -- "$docker_config" "$stage_root"\n/bin/chmod 600 "$snapshot" "$manifest" "$config_json" "$stage_override"\n\ncompose_clean() {\n    "$ENV_BIN" -i PATH=/usr/bin:/bin HOME="$private_home" DOCKER_CONFIG="$docker_config" LC_ALL=C \\\n        "$DOCKER" compose --env-file "$snapshot" -f "$COMPOSE_FILE" "$@"\n}\n\ncompose_staged() {\n    "$ENV_BIN" -i PATH=/usr/bin:/bin HOME="$private_home" DOCKER_CONFIG="$docker_config" LC_ALL=C \\\n        "$DOCKER" compose --env-file "$snapshot" -f "$COMPOSE_FILE" -f "$stage_override" "$@"\n}\n\ninvalidate_build_receipt() {\n    "$PYTHON" -I - "$receipt" "$DEPLOY_DIR" <<\'PY\'\nimport os\nimport stat\nimport sys\n\nreceipt = os.path.abspath(sys.argv[1])\ndeploy_dir = os.path.realpath(sys.argv[2])\nif os.path.dirname(receipt) != deploy_dir or os.path.basename(receipt) != ".op-build-receipt.json":\n    raise SystemExit(1)\nflags = os.O_RDONLY | getattr(os, "O_DIRECTORY", 0) | getattr(os, "O_CLOEXEC", 0)\ndirectory_fd = os.open(deploy_dir, flags)\ntry:\n    name = os.path.basename(receipt)\n    try:\n        before = os.stat(name, dir_fd=directory_fd, follow_symlinks=False)\n    except FileNotFoundError:\n        raise SystemExit(0)\n    if (not stat.S_ISREG(before.st_mode) or stat.S_ISLNK(before.st_mode)\n            or before.st_uid != os.getuid() or before.st_nlink != 1\n            or stat.S_IMODE(before.st_mode) != 0o600):\n        raise SystemExit(1)\n    open_flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)\n    fd = os.open(name, open_flags, dir_fd=directory_fd)\n    try:\n        opened = os.fstat(fd)\n        current = os.stat(name, dir_fd=directory_fd, follow_symlinks=False)\n        identity = lambda value: (value.st_dev, value.st_ino)\n        if identity(before) != identity(opened) or identity(opened) != identity(current):\n            raise SystemExit(1)\n        if (not stat.S_ISREG(opened.st_mode) or opened.st_uid != os.getuid()\n                or opened.st_nlink != 1 or stat.S_IMODE(opened.st_mode) != 0o600):\n            raise SystemExit(1)\n        os.unlink(name, dir_fd=directory_fd)\n        os.fsync(directory_fd)\n    finally:\n        os.close(fd)\nfinally:\n    os.close(directory_fd)\nPY\n}\n\nresource_gate() {\n    local output\n    local -a metrics\n    local value\n    local attempt=0\n    local stable_samples=0\n    local observed_high_load=0\n    local space_re=" "\n    while (( attempt < LOAD_WAIT_ATTEMPTS )); do\n        (( attempt += 1 ))\n        if [[ -n "$RESOURCE_PROBE" ]]; then\n            output="$($RESOURCE_PROBE /)" || fail "resource probe failed"\n        else\n            output="$($PYTHON -I - "$REPO_ROOT" <<\'PY\'\nimport math\nimport os\nimport sys\n\navailable_kib = None\nwith open("/proc/meminfo", "r", encoding="ascii") as handle:\n    for line in handle:\n        if line.startswith("MemAvailable:"):\n            available_kib = int(line.split()[1])\n            break\nif available_kib is None:\n    raise SystemExit(1)\nwith open("/proc/loadavg", "r", encoding="ascii") as handle:\n    load_milli = math.ceil(float(handle.read().split()[0]) * 1000)\nstats = os.statvfs("/")\ndisk_mib = stats.f_bavail * stats.f_frsize // (1024 * 1024)\nprint(available_kib // 1024, load_milli, disk_mib)\nPY\n            )" || fail "resource probe failed"\n        fi\n        [[ "$output" =~ ^([0-9]+)${space_re}([0-9]+)${space_re}([0-9]+)$ ]] || fail "resource probe returned invalid metrics"\n        metrics=("${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}" "${BASH_REMATCH[3]}")\n        for value in "${metrics[@]}"; do\n            [[ "$value" == "0" || "$value" =~ ^[1-9][0-9]*$ ]] || fail "resource probe returned invalid metrics"\n        done\n        (( metrics[0] >= MIN_MEM_MIB )) || fail "insufficient MemAvailable (${metrics[0]} MiB; require ${MIN_MEM_MIB} MiB)"\n        (( metrics[2] >= MIN_DISK_MIB )) || fail "insufficient root filesystem space (${metrics[2]} MiB; require ${MIN_DISK_MIB} MiB)"\n        if (( metrics[1] <= MAX_LOAD_MILLI )); then\n            if (( observed_high_load == 0 )); then\n                return 0\n            fi\n            (( stable_samples += 1 ))\n            if (( stable_samples >= LOAD_STABLE_SAMPLES )); then\n                return 0\n            fi\n        else\n            observed_high_load=1\n            stable_samples=0\n        fi\n        (( attempt < LOAD_WAIT_ATTEMPTS )) || fail "Load1 did not stabilize (${metrics[1]} milli; maximum ${MAX_LOAD_MILLI} milli)"\n        "$SLEEP" "$LOAD_WAIT_SECONDS"\n    done\n    fail "Load1 did not stabilize within the bounded wait"\n}\n\n"$VALIDATOR" env "$env_file" "$snapshot"\n\ncase "$action" in\n    validate)\n        "$VALIDATOR" artifacts "$manifest"\n        printf \'%s\\n\' \'OP deployment prerequisites validated.\'\n        ;;\n    config)\n        compose_clean config --format json > "$config_json"\n        "$VALIDATOR" config "$snapshot" "$config_json"\n        /bin/cat -- "$config_json"\n        ;;\n    build-assets)\n        "$ENV_BIN" -i PATH=/usr/bin:/bin HOME="$private_home" LC_ALL=C \\\n            "$NPM" --prefix "$REPO_ROOT/frontend/sdpivot" run build:op\n        "$VALIDATOR" artifacts "$manifest"\n        printf \'%s\\n\' \'OP frontend assets built and recorded.\'\n        ;;\n    compose-build)\n        invalidate_build_receipt || fail "existing build receipt is unsafe"\n        "$VALIDATOR" artifacts "$manifest"\n        compose_clean config --format json > "$config_json"\n        "$VALIDATOR" config "$snapshot" "$config_json"\n        "$VALIDATOR" stage "$manifest" "$stage_root" "$stage_override"\n        "$VALIDATOR" compare-stage "$manifest" "$stage_root"\n        for service in migration sdp-backend sdp-frontend app docreader; do\n            resource_gate\n            compose_staged build "$service"\n        done\n        resource_gate\n        "$VALIDATOR" compare-stage "$manifest" "$stage_root"\n        "$VALIDATOR" compare-artifacts "$manifest"\n        "$VALIDATOR" receipt-write "$snapshot" "$manifest" "$config_json" "$receipt"\n        ;;\n    up)\n        [[ -e "$receipt" ]] || fail "no successful compose-build receipt exists for this deployment"\n        "$VALIDATOR" artifacts "$manifest"\n        compose_clean config --format json > "$config_json"\n        "$VALIDATOR" config "$snapshot" "$config_json"\n        "$VALIDATOR" receipt-verify "$snapshot" "$manifest" "$config_json" "$receipt"\n        compose_clean up "$@" --no-build\n        ;;\nesac\n'
FORWARDED_SIGNALS = (signal.SIGTERM, signal.SIGINT, signal.SIGHUP)
child = None
pending_signals = []
kill_deadline = None
finished = False


def forward_signal(signum, _frame):
    global kill_deadline
    if kill_deadline is None:
        kill_deadline = time.monotonic() + 5.0
    if child is None:
        pending_signals.append(signum)
        return
    if finished:
        return
    try:
        os.killpg(child.pid, signum)
    except ProcessLookupError:
        pass


def fail(message):
    raise SystemExit(f"OP deployment failed: {message}")


def main():
    global child, finished
    script = os.path.realpath(__file__)
    resource_probe = ""
    min_mem_mib = 800
    max_load_milli = 1500
    min_disk_mib = 20 * 1024
    load_wait_attempts = 25
    load_wait_seconds = 5
    load_stable_samples = 2
    test_root = os.path.realpath(os.path.join(os.path.dirname(script), ".."))
    if os.environ.get("OP_DEPLOY_INTERNAL_TESTING") == "1" and test_root.startswith("/tmp/op-deploy-test."):
        candidate = os.environ.get("OP_DEPLOY_TEST_RESOURCE_PROBE", "")
        if candidate:
            candidate = os.path.realpath(candidate)
            probe = os.stat(candidate, follow_symlinks=False)
            if not stat.S_ISREG(probe.st_mode) or probe.st_uid != os.getuid() or probe.st_nlink != 1 or not (probe.st_mode & stat.S_IXUSR):
                fail("test resource probe is unsafe")
            resource_probe = candidate
        try:
            min_mem_mib = int(os.environ.get("OP_DEPLOY_TEST_MIN_MEM_MIB", min_mem_mib))
            max_load_milli = int(os.environ.get("OP_DEPLOY_TEST_MAX_LOAD_MILLI", max_load_milli))
            min_disk_mib = int(os.environ.get("OP_DEPLOY_TEST_MIN_DISK_MIB", min_disk_mib))
            load_wait_attempts = int(os.environ.get("OP_DEPLOY_TEST_LOAD_WAIT_ATTEMPTS", load_wait_attempts))
            load_wait_seconds = int(os.environ.get("OP_DEPLOY_TEST_LOAD_WAIT_SECONDS", load_wait_seconds))
            load_stable_samples = int(os.environ.get("OP_DEPLOY_TEST_LOAD_STABLE_SAMPLES", load_stable_samples))
        except ValueError:
            fail("test resource thresholds are invalid")
        if min(min_mem_mib, max_load_milli, min_disk_mib, load_wait_seconds) < 0:
            fail("test resource thresholds are invalid")
        if load_wait_attempts < 2 or load_wait_attempts > 25:
            fail("test resource wait policy is invalid")
        if load_wait_seconds < 0 or load_wait_seconds > 5 or load_stable_samples != 2:
            fail("test resource wait policy is invalid")
    try:
        tmp = os.lstat("/tmp")
    except OSError:
        fail("/tmp is unavailable")
    if stat.S_ISLNK(tmp.st_mode) or not stat.S_ISDIR(tmp.st_mode) or tmp.st_uid != 0 or not (tmp.st_mode & stat.S_ISVTX):
        fail("/tmp must be a root-owned sticky directory")

    lock_path = f"/tmp/weknora-op-deploy-{os.getuid()}.lock"
    flags = os.O_RDWR | os.O_CREAT | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_CLOEXEC", 0)
    try:
        lock_fd = os.open(lock_path, flags, 0o600)
    except OSError:
        fail("repository lock could not be opened safely")

    try:
        opened = os.fstat(lock_fd)
        named = os.lstat(lock_path)
        if not stat.S_ISREG(opened.st_mode) or opened.st_uid != os.getuid() or opened.st_nlink != 1:
            fail("repository lock must be a current-user single-link regular file")
        if (opened.st_dev, opened.st_ino) != (named.st_dev, named.st_ino):
            fail("repository lock changed while it was opened")
        os.fchmod(lock_fd, 0o600)
        try:
            fcntl.flock(lock_fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            fail("another OP deployment action holds the repository lock")

        for signum in FORWARDED_SIGNALS:
            signal.signal(signum, forward_signal)

        child_env = {
            "PATH": "/usr/bin:/bin",
            "LANG": "C",
            "LC_ALL": "C",
        }
        child = subprocess.Popen(
            [
                "/bin/bash", "-s", "--", script, resource_probe,
                str(min_mem_mib), str(max_load_milli), str(min_disk_mib),
                str(load_wait_attempts), str(load_wait_seconds), str(load_stable_samples),
                "--", *sys.argv[1:]
            ],
            stdin=subprocess.PIPE,
            env=child_env,
            close_fds=True,
            start_new_session=True,
        )
        for signum in pending_signals:
            try:
                os.killpg(child.pid, signum)
            except ProcessLookupError:
                break
        try:
            child.stdin.write(BODY.encode("utf-8"))
            child.stdin.close()
        except BrokenPipeError:
            pass

        while True:
            try:
                returncode = child.wait(timeout=0.1)
                finished = True
                break
            except subprocess.TimeoutExpired:
                if kill_deadline is not None and time.monotonic() >= kill_deadline:
                    try:
                        os.killpg(child.pid, signal.SIGKILL)
                    except ProcessLookupError:
                        pass
                    returncode = child.wait()
                    finished = True
                    break
        return returncode if returncode >= 0 else 128 - returncode
    finally:
        os.close(lock_fd)


if __name__ == "__main__":
    raise SystemExit(main())
