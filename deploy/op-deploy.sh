#!/usr/bin/python3 -I
import fcntl
import os
import signal
import stat
import subprocess
import sys
import time

BODY = 'PATH=/usr/bin:/bin\nLC_ALL=C\nexport PATH LC_ALL\nunset CDPATH ENV BASH_ENV\nset -Eeuo pipefail\numask 077\n\nfail() {\n    printf \'OP deployment failed: %s\\n\' "$1" >&2\n    exit 1\n}\n\nreadonly REALPATH=/usr/bin/realpath\nreadonly MKTEMP=/usr/bin/mktemp\nreadonly ENV_BIN=/usr/bin/env\nreadonly DOCKER=/usr/bin/docker\nreadonly NPM=/usr/bin/npm\nreadonly RM=/bin/rm\nfor tool in "$REALPATH" "$MKTEMP" "$ENV_BIN" "$DOCKER" "$NPM" "$RM"; do\n    [[ -x "$tool" ]] || fail "required system tool is unavailable"\ndone\n\nreadonly SCRIPT_PATH="$($REALPATH -- "$1")"\nshift\nreadonly DEPLOY_DIR="${SCRIPT_PATH%/*}"\nreadonly REPO_ROOT="${DEPLOY_DIR%/*}"\nreadonly VALIDATOR="$DEPLOY_DIR/validate-op-deployment.sh"\nreadonly COMPOSE_FILE="$DEPLOY_DIR/docker-compose.op.yml"\n[[ -x "$VALIDATOR" ]] || fail "validator is unavailable"\n[[ -f "$COMPOSE_FILE" ]] || fail "Compose file is unavailable"\n\nusage() {\n    printf \'%s\\n\' "usage: $SCRIPT_PATH [--env-file PATH] {validate|config|build-assets|compose-build|up} [up: -d|--detach]" >&2\n    exit 2\n}\n\nenv_file="$DEPLOY_DIR/.env.op"\nif [[ "${1:-}" == "--env-file" ]]; then\n    (( $# >= 3 )) || usage\n    env_file="$2"\n    shift 2\nfi\n(( $# >= 1 )) || usage\naction="$1"\nshift\ncase "$action" in\n    validate|config|build-assets)\n        (( $# == 0 )) || usage\n        ;;\n    compose-build)\n        (( $# == 0 )) || fail "compose-build accepts no additional arguments"\n        ;;\n    up)\n        for arg in "$@"; do\n            case "$arg" in\n                -d|--detach) ;;\n                *) fail "up accepts only -d or --detach; build and no-build overrides are forbidden" ;;\n            esac\n        done\n        ;;\n    *) usage ;;\nesac\n\ntmp_dir="$($MKTEMP -d /tmp/weknora-op-deploy.XXXXXX)"\ncleanup() {\n    "$RM" -rf -- "$tmp_dir"\n}\ntrap cleanup EXIT HUP INT TERM\nsnapshot="$tmp_dir/env.snapshot"\nmanifest="$tmp_dir/artifacts.manifest"\nconfig_json="$tmp_dir/config.json"\nstage_override="$tmp_dir/stage-override.json"\nstage_root="$tmp_dir/stage"\nprivate_home="$tmp_dir/home"\ndocker_config="$private_home/.docker"\nreceipt="$DEPLOY_DIR/.op-build-receipt.json"\n: > "$snapshot"\n: > "$manifest"\n: > "$config_json"\n: > "$stage_override"\n/bin/mkdir -m 700 -p -- "$docker_config" "$stage_root"\n/bin/chmod 600 "$snapshot" "$manifest" "$config_json" "$stage_override"\n\ncompose_clean() {\n    "$ENV_BIN" -i PATH=/usr/bin:/bin HOME="$private_home" DOCKER_CONFIG="$docker_config" LC_ALL=C \\\n        "$DOCKER" compose --env-file "$snapshot" -f "$COMPOSE_FILE" "$@"\n}\n\ncompose_staged() {\n    "$ENV_BIN" -i PATH=/usr/bin:/bin HOME="$private_home" DOCKER_CONFIG="$docker_config" LC_ALL=C \\\n        "$DOCKER" compose --env-file "$snapshot" -f "$COMPOSE_FILE" -f "$stage_override" "$@"\n}\n\n"$VALIDATOR" env "$env_file" "$snapshot"\n\ncase "$action" in\n    validate)\n        "$VALIDATOR" artifacts "$manifest"\n        printf \'%s\\n\' \'OP deployment prerequisites validated.\'\n        ;;\n    config)\n        compose_clean config --format json > "$config_json"\n        "$VALIDATOR" config "$snapshot" "$config_json"\n        /bin/cat -- "$config_json"\n        ;;\n    build-assets)\n        "$ENV_BIN" -i PATH=/usr/bin:/bin HOME="$private_home" LC_ALL=C \\\n            "$NPM" --prefix "$REPO_ROOT/frontend/sdpivot" run build:op\n        "$VALIDATOR" artifacts "$manifest"\n        printf \'%s\\n\' \'OP frontend assets built and recorded.\'\n        ;;\n    compose-build)\n        "$VALIDATOR" artifacts "$manifest"\n        compose_clean config --format json > "$config_json"\n        "$VALIDATOR" config "$snapshot" "$config_json"\n        "$VALIDATOR" stage "$manifest" "$stage_root" "$stage_override"\n        "$VALIDATOR" compare-stage "$manifest" "$stage_root"\n        compose_staged build\n        "$VALIDATOR" compare-stage "$manifest" "$stage_root"\n        "$VALIDATOR" receipt-write "$snapshot" "$manifest" "$config_json" "$receipt"\n        ;;\n    up)\n        [[ -e "$receipt" ]] || fail "no successful compose-build receipt exists for this deployment"\n        "$VALIDATOR" artifacts "$manifest"\n        compose_clean config --format json > "$config_json"\n        "$VALIDATOR" config "$snapshot" "$config_json"\n        "$VALIDATOR" receipt-verify "$snapshot" "$manifest" "$config_json" "$receipt"\n        compose_clean up "$@" --no-build\n        ;;\nesac\n'
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
            ["/bin/bash", "-s", "--", script, *sys.argv[1:]],
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
