#!/bin/bash
set -Eeuo pipefail

readonly SOURCE_ROOT="$(realpath -- "${1:-$(pwd)}")"
readonly TMP_ROOT="$(mktemp -d /tmp/op-migrate-audit-test.XXXXXX)"
trap 'rm -rf -- "$TMP_ROOT"' EXIT HUP INT TERM
readonly FAKE_BIN="$TMP_ROOT/bin"
mkdir -p "$FAKE_BIN"

cat > "$FAKE_BIN/psql" <<'PY'
#!/usr/bin/python3
import os
from pathlib import Path
import sys

sql = sys.argv[-1]
state = Path(os.environ["TEST_STATE"])
core_done = (state / "core-done").exists()
baseline_done = (state / "baseline-done").exists()
sdp_done = (state / "sdp-done").exists()

if "/* op-probe:core-audit-m44 */" in sql:
    if os.environ.get("REQUIRE_STRICT_BIGSERIAL") == "1":
        required = ["a.default_expr IS DISTINCT FROM pg_catalog.format", "p.seqstart = 1", "p.seqincrement = 1", "p.seqmax = 9223372036854775807", "p.seqmin = 1", "p.seqcache = 1", "NOT p.seqcycle"]
        if any(fragment not in sql for fragment in required):
            print("invalid")
            sys.exit(0)
    if baseline_done:
        print("baseline_exact")
    elif core_done:
        print("migration44_exact")
    else:
        print(os.environ["AUDIT_STATE"])
elif "to_regclass('public.schema_migrations') IS NOT NULL" in sql:
    print("t")
elif "FROM public.schema_migrations" in sql:
    print(("63" if core_done else os.environ["CORE_VERSION"]) + ":f")
elif "to_regclass('public.sdpivot_schema_migrations') IS NOT NULL" in sql:
    print("t" if baseline_done else "f")
elif "FROM public.sdpivot_schema_migrations" in sql:
    print(("14" if sdp_done else "12") + ":f")
elif "/* op-probe:core-v12-fp */" in sql:
    print("complete")
elif "/* op-probe:sdpivot-flags */" in sql:
    print("0")
elif "/* op-probe:sdpivot-v12 */" in sql:
    print("complete" if baseline_done else "empty")
elif "/* op-probe:sdpivot-v13-schema */" in sql or "/* op-probe:sdpivot-v13-account */" in sql or "/* op-probe:sdpivot-latest */" in sql:
    print("complete")
else:
    print("0")
PY
chmod 700 "$FAKE_BIN/psql"

cat > "$FAKE_BIN/migrate" <<'SH'
#!/bin/bash
set -Eeuo pipefail
path=""
while (($#)); do
    if [[ "$1" == "-path" ]]; then path="$2"; shift 2; else shift; fi
done
printf '%s\n' "$path" >> "$TEST_STATE/migrate.log"
case "$path" in
    /migrations/versioned) touch "$TEST_STATE/core-done" ;;
    /migrations/postgres-bootstrap) touch "$TEST_STATE/baseline-done" ;;
    /migrations/postgres) touch "$TEST_STATE/sdp-done" ;;
esac
SH
chmod 700 "$FAKE_BIN/migrate"

run_case() {
    local name="$1" core_version="$2" audit_state="$3" expected="$4"
    local state="$TMP_ROOT/$name"
    mkdir -p "$state"
    if CORE_VERSION="$core_version" AUDIT_STATE="$audit_state" TEST_STATE="$state" \
        PSQL_BIN="$FAKE_BIN/psql" MIGRATE_BIN="$FAKE_BIN/migrate" \
        OP_DATABASE_URL="postgresql://test:test@localhost/test" \
        "$SOURCE_ROOT/deploy/migrate-op.sh" >"$state/out" 2>"$state/err"; then
        status=0
    else
        status=$?
    fi

    case "$expected" in
        reject)
            [[ "$status" -ne 0 ]] || { printf 'FAIL %s unexpectedly succeeded\n' "$name" >&2; exit 1; }
            [[ ! -e "$state/migrate.log" ]] || { printf 'FAIL %s invoked migrate before rejection\n' "$name" >&2; exit 1; }
            ;;
        allow)
            [[ "$status" -eq 0 ]] || { printf 'FAIL %s failed: %s\n' "$name" "$(<"$state/err")" >&2; exit 1; }
            [[ "$(head -n 1 "$state/migrate.log")" == "/migrations/versioned" ]] || {
                printf 'FAIL %s did not invoke Core migration first\n' "$name" >&2; exit 1;
            }
            ;;
    esac
    printf 'PASS %s\n' "$name"
}

run_case core43_preexisting_audit_rejected_before_migrate 43 migration44_exact reject
run_case core44_exact_allowed 44 migration44_exact allow
run_case core62_exact_allowed 62 migration44_exact allow
run_case core44_mixed_rejected_before_migrate 44 invalid reject
run_case core62_mixed_rejected_before_migrate 62 invalid reject
run_case core63_missing_rejected_before_mutation 63 missing reject


if [[ -n "${OP_TEST_PG_CONTAINER:-}" ]]; then
    export SOURCE_ROOT OP_TEST_PG_CONTAINER
    python3 - <<'PYPG'
import os
from pathlib import Path
import subprocess
import time

source_root = Path(os.environ["SOURCE_ROOT"])
container = os.environ["OP_TEST_PG_CONTAINER"]
script = (source_root / "deploy/migrate-op.sh").read_text()
marker = "        /* op-probe:core-audit-m44 */"
start = script.index(marker)
end = script.index("\n    \"\n}", start)
query = "\n".join(line[8:] if line.startswith("        ") else line for line in script[start:end].splitlines()) + ";\n"
migration = (source_root / "migrations/versioned/000044_audit_log.up.sql").read_text()
base = ["docker", "exec", "-i", container, "psql", "-U", "test"]

def psql(database, sql, capture=False):
    args = base + ["-d", database, "-v", "ON_ERROR_STOP=1"] + (["-Atq"] if capture else [])
    result = subprocess.run(args, input=sql, text=True, capture_output=True)
    if result.returncode:
        raise RuntimeError(result.stderr)
    return result.stdout.strip()

cases = [
    ("exact", "", "migration44_exact"),
    ("default_plus_one", "ALTER TABLE audit_logs ALTER COLUMN id SET DEFAULT (nextval(pg_get_serial_sequence('public.audit_logs','id')) + 1);", "invalid"),
    ("sequence_cycle", "ALTER SEQUENCE audit_logs_id_seq CYCLE;", "invalid"),
    ("sequence_start", "ALTER SEQUENCE audit_logs_id_seq START WITH 2;", "invalid"),
    ("sequence_min", "ALTER SEQUENCE audit_logs_id_seq MINVALUE 0;", "invalid"),
    ("sequence_max", "ALTER SEQUENCE audit_logs_id_seq MAXVALUE 9223372036854775806;", "invalid"),
    ("sequence_increment", "ALTER SEQUENCE audit_logs_id_seq INCREMENT BY 2;", "invalid"),
    ("sequence_cache", "ALTER SEQUENCE audit_logs_id_seq CACHE 2;", "invalid"),
    ("sequence_rename", "ALTER SEQUENCE audit_logs_id_seq RENAME TO audit_logs_serial_custom;", "migration44_exact"),
]

for number, (name, mutation, expected) in enumerate(cases, 1):
    database = f"op_m44_catalog_{os.getpid()}_{int(time.time())}_{number}"
    try:
        psql("postgres", f"CREATE DATABASE {database};")
        psql(database, migration)
        if mutation:
            psql(database, mutation)
        actual = psql(database, query, capture=True)
        if actual != expected:
            raise RuntimeError(f"{name}: expected {expected}, got {actual}")
        print(f"PASS pg17_{name}={actual}")
    finally:
        subprocess.run(base + ["-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", f"DROP DATABASE IF EXISTS {database} WITH (FORCE);"], text=True, capture_output=True)
PYPG
fi
