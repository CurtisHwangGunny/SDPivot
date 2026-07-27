package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func readDeployFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return strings.ToLower(string(data))
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write executable %s: %v", path, err)
	}
}

type migrationScenario struct {
	name            string
	coreState       string
	sdpivotState    string
	publicTables    string
	coreFingerprint string
	coreAuditShape  string
	sdpFingerprint  string
	sdpV13Schema    string
	sdpV13Account   string
	sdpLatest       string
	sdpMarkers      string
	failAt          string
	wantSuccess     bool
	wantCalls       []string
}

func runMigrationScenario(t *testing.T, scenario migrationScenario) (string, []string) {
	t.Helper()
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	stateDir := filepath.Join(tmp, "state")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "core"), []byte(scenario.coreState), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "sdpivot"), []byte(scenario.sdpivotState), 0o600); err != nil {
		t.Fatal(err)
	}

	writeExecutable(t, filepath.Join(binDir, "psql"), `#!/usr/bin/env bash
set -Eeuo pipefail
sql="${!#}"
state_dir="${FAKE_STATE_DIR:?}"
log="$state_dir/calls"
class=
output=
case "$sql" in
  *"to_regclass('public.schema_migrations')"*)
    class=core-exists
    [[ "$(<"$state_dir/core")" == absent ]] && output=f || output=t
    ;;
  *"FROM public.schema_migrations"*)
    class=core-state
    output="$(<"$state_dir/core")"
    ;;
  *"to_regclass('public.sdpivot_schema_migrations')"*)
    class=sdp-exists
    [[ "$(<"$state_dir/sdpivot")" == absent ]] && output=f || output=t
    ;;
  *"FROM public.sdpivot_schema_migrations"*)
    class=sdp-state
    output="$(<"$state_dir/sdpivot")"
    ;;
  *"FROM pg_catalog.pg_tables"*|*"application_object"*)
    class=public-count
    output="${FAKE_PUBLIC_TABLES:-0}"
    ;;
  *"op-probe:core-audit-m44"*)
    class=core-audit-m44
    if [[ -e "$state_dir/core-migrated" ]]; then
      output=migration44_exact
    elif [[ -e "$state_dir/bootstrap-migrated" ]]; then
      output=baseline_exact
    elif [[ -n "${FAKE_CORE_AUDIT_SHAPE+x}" ]]; then
      output="$FAKE_CORE_AUDIT_SHAPE"
    elif [[ "$(<"$state_dir/core")" == absent || "${FAKE_CORE_VERSION_NUM:-0}" -lt 44 ]]; then
      output=missing
    elif [[ "$(<"$state_dir/sdpivot")" != absent || "${FAKE_SDP_FINGERPRINT:-complete}" == complete ]]; then
      output=baseline_exact
    else
      output=migration44_exact
    fi
    ;;
  *"op-probe:core-v12-fp"*)
    class=core-fingerprint
    output="${FAKE_CORE_FINGERPRINT:-complete}"
    ;;
  *"op-probe:sdpivot-flags"*)
    class=sdp-flags
    output="${FAKE_SDP_MARKERS:-0}"
    ;;
  *"op-probe:sdpivot-v12"*)
    class=sdp-fingerprint
    if [[ "${FAKE_SDP_FINGERPRINT:-complete}" == empty && "$(<"$state_dir/sdpivot")" != absent ]]; then
      output=complete
    else
      output="${FAKE_SDP_FINGERPRINT:-complete}"
    fi
    ;;
  *"op-probe:sdpivot-v13-schema"*)
    class=sdp-v13-schema
    output="${FAKE_SDP_V13_SCHEMA:-complete}"
    ;;
  *"op-probe:sdpivot-v13-account"*)
    class=sdp-v13-account
    output="${FAKE_SDP_V13_ACCOUNT:-complete}"
    ;;
  *"op-probe:sdpivot-latest"*)
    class=sdp-latest
    output="${FAKE_SDP_LATEST_FINGERPRINT:-complete}"
    ;;
  *)
    class=unknown-psql
    ;;
esac
printf 'psql:%s\n' "$class" >>"$log"
if [[ "${FAKE_FAIL_AT:-}" == "psql:$class" ]]; then
  printf 'fake psql error containing %s\n' "${OP_DATABASE_URL:-missing}" >&2
  exit 9
fi
[[ "$class" != unknown-psql ]] || exit 8
printf '%s\n' "$output"
`)
	writeExecutable(t, filepath.Join(binDir, "migrate"), `#!/usr/bin/env bash
set -Eeuo pipefail
state_dir="${FAKE_STATE_DIR:?}"
log="$state_dir/calls"
path_value=
while (($#)); do
  if [[ "$1" == -path ]]; then
    path_value="$2"
    break
  fi
  shift
done
case "$path_value" in
  /migrations/versioned) class=core ;;
  /migrations/postgres-bootstrap) class=bootstrap ;;
  /migrations/postgres) class=sdpivot ;;
  *) class=unknown-migrate ;;
esac
printf 'migrate:%s\n' "$class" >>"$log"
if [[ "${FAKE_FAIL_AT:-}" == "migrate:$class" ]]; then
  printf '2026/07/27 12:00:00 Start buffering 15/u 000015_remove_billing_subscription\n' >&2
  printf 'error: migration failed: pq: relation "billing_subscriptions" does not exist; connection=%s; user=secret-user; password=secret-password\n' "${OP_DATABASE_URL:-missing}" >&2
  exit 9
fi
case "$class" in
  core) printf '63:f' >"$state_dir/core"; touch "$state_dir/core-migrated" ;;
  bootstrap) printf '12:f' >"$state_dir/sdpivot"; touch "$state_dir/bootstrap-migrated" ;;
  sdpivot) printf '17:f' >"$state_dir/sdpivot" ;;
  *) exit 8 ;;
esac
`)

	cmd := exec.Command("bash", "migrate-op.sh")
	markers := scenario.sdpMarkers
	if markers == "" {
		markers = "0"
	}
	coreVersionNum := strings.TrimSuffix(scenario.coreState, ":f")
	if coreVersionNum == "absent" || strings.Contains(coreVersionNum, ":") {
		coreVersionNum = "0"
	}
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+":"+os.Getenv("PATH"),
		"PSQL_BIN="+filepath.Join(binDir, "psql"),
		"MIGRATE_BIN="+filepath.Join(binDir, "migrate"),
		"OP_DATABASE_URL=postgres://secret-user:secret-password@db.internal/sdpivot",
		"FAKE_STATE_DIR="+stateDir,
		"FAKE_PUBLIC_TABLES="+scenario.publicTables,
		"FAKE_CORE_FINGERPRINT="+scenario.coreFingerprint,
		"FAKE_CORE_VERSION_NUM="+coreVersionNum,
		"FAKE_SDP_FINGERPRINT="+scenario.sdpFingerprint,
		"FAKE_SDP_V13_SCHEMA="+scenario.sdpV13Schema,
		"FAKE_SDP_V13_ACCOUNT="+scenario.sdpV13Account,
		"FAKE_SDP_MARKERS="+markers,
		"FAKE_SDP_LATEST_FINGERPRINT="+scenario.sdpLatest,
		"FAKE_FAIL_AT="+scenario.failAt,
	)
	if scenario.coreAuditShape != "" {
		cmd.Env = append(cmd.Env, "FAKE_CORE_AUDIT_SHAPE="+scenario.coreAuditShape)
	}
	output, err := cmd.CombinedOutput()
	if scenario.wantSuccess && err != nil {
		t.Fatalf("scenario %s failed: %v\n%s", scenario.name, err, output)
	}
	if !scenario.wantSuccess && err == nil {
		t.Fatalf("scenario %s unexpectedly succeeded\n%s", scenario.name, output)
	}
	if strings.Contains(string(output), "secret-password") {
		t.Fatalf("scenario %s leaked database credentials: %s", scenario.name, output)
	}
	callsData, readErr := os.ReadFile(filepath.Join(stateDir, "calls"))
	if readErr != nil {
		t.Fatalf("read calls for %s: %v", scenario.name, readErr)
	}
	calls := strings.Fields(strings.TrimSpace(string(callsData)))
	return string(output), calls
}

func assertCalls(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected calls\nwant:\n%s\ngot:\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))
	}
}

func expectedCallsWithInitialSDPivotInspection(scenario migrationScenario, calls []string) []string {
	initial := []string{"psql:sdp-exists"}
	if scenario.sdpivotState != "absent" {
		initial = append(initial, "psql:sdp-state")
	}
	return append(initial, calls...)
}

func TestMigrationStateMachineSuccessPaths(t *testing.T) {
	tests := []migrationScenario{
		{
			name:      "greenfield empty",
			coreState: "absent", sdpivotState: "absent", publicTables: "0",
			coreFingerprint: "complete", sdpFingerprint: "empty", wantSuccess: true,
			wantCalls: []string{
				"psql:core-exists", "psql:core-audit-m44", "psql:public-count", "migrate:core",
				"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:sdp-exists", "psql:sdp-fingerprint",
				"migrate:bootstrap", "psql:sdp-exists", "psql:sdp-state",
				"migrate:sdpivot", "psql:sdp-exists", "psql:sdp-state",
				"psql:sdp-fingerprint", "psql:sdp-v13-schema", "psql:sdp-v13-account", "psql:sdp-latest",
			},
		},
		{
			name:            "complete historical v12 without dedicated ledger",
			coreState:       "63:f",
			sdpivotState:    "absent",
			publicTables:    "1",
			coreFingerprint: "complete",
			sdpFingerprint:  "complete",
			wantSuccess:     true,
			wantCalls: []string{
				"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:core-exists", "psql:core-state", "psql:core-audit-m44",
				"psql:sdp-exists", "psql:sdp-fingerprint", "migrate:bootstrap",
				"psql:sdp-exists", "psql:sdp-state", "migrate:sdpivot",
				"psql:sdp-exists", "psql:sdp-state", "psql:sdp-fingerprint", "psql:sdp-v13-schema", "psql:sdp-v13-account", "psql:sdp-latest",
			},
		},
		{
			name:      "dedicated sdpivot ledger at v12",
			coreState: "63:f", sdpivotState: "12:f", publicTables: "1",
			coreFingerprint: "complete", sdpFingerprint: "complete", wantSuccess: true,
			wantCalls: []string{
				"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:sdp-exists", "psql:sdp-state",
				"psql:sdp-fingerprint", "migrate:sdpivot", "psql:sdp-exists", "psql:sdp-state",
				"psql:sdp-fingerprint", "psql:sdp-v13-schema", "psql:sdp-v13-account", "psql:sdp-latest",
			},
		},
		{
			name:      "dedicated sdpivot ledger at v13",
			coreState: "63:f", sdpivotState: "13:f", publicTables: "1",
			coreFingerprint: "complete", sdpFingerprint: "complete", wantSuccess: true,
			wantCalls: []string{
				"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:core-exists", "psql:core-state", "psql:core-audit-m44",
				"psql:sdp-exists", "psql:sdp-state", "psql:sdp-fingerprint", "psql:sdp-v13-schema", "psql:sdp-v13-account",
				"migrate:sdpivot", "psql:sdp-exists", "psql:sdp-state",
				"psql:sdp-fingerprint", "psql:sdp-v13-schema", "psql:sdp-v13-account", "psql:sdp-latest",
			},
		},
		{
			name:      "proven core ledger at ambiguous numeric v12",
			coreState: "12:f", sdpivotState: "absent", publicTables: "1",
			coreFingerprint: "complete", sdpFingerprint: "empty", wantSuccess: true,
			wantCalls: []string{
				"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:core-fingerprint",
				"psql:sdp-exists", "psql:sdp-flags", "migrate:core",
				"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:sdp-exists", "psql:sdp-fingerprint",
				"migrate:bootstrap", "psql:sdp-exists", "psql:sdp-state",
				"migrate:sdpivot", "psql:sdp-exists", "psql:sdp-state",
				"psql:sdp-fingerprint", "psql:sdp-v13-schema", "psql:sdp-v13-account", "psql:sdp-latest",
			},
		},
		{
			name:      "already current ledgers have zero migration side effects",
			coreState: "63:f", sdpivotState: "17:f", publicTables: "1",
			coreFingerprint: "complete", sdpFingerprint: "complete", wantSuccess: true,
			wantCalls: []string{
				"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:core-exists", "psql:core-state", "psql:core-audit-m44",
				"psql:sdp-exists", "psql:sdp-state",
				"psql:sdp-fingerprint", "psql:sdp-v13-schema", "psql:sdp-v13-account", "psql:sdp-latest",
				"psql:sdp-exists", "psql:sdp-state",
				"psql:sdp-fingerprint", "psql:sdp-v13-schema", "psql:sdp-v13-account", "psql:sdp-latest",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, calls := runMigrationScenario(t, test)
			assertCalls(t, calls, expectedCallsWithInitialSDPivotInspection(test, test.wantCalls))
		})
	}
}

func TestMigrationStateMachineFailsClosedForUnsafeStates(t *testing.T) {
	tests := []migrationScenario{
		{name: "core dirty", coreState: "12:t", sdpivotState: "absent", coreFingerprint: "complete", sdpFingerprint: "empty", wantCalls: []string{"psql:core-exists", "psql:core-state", "psql:core-audit-m44"}},
		{name: "sdpivot dirty", coreState: "63:f", sdpivotState: "12:t", coreFingerprint: "complete", sdpFingerprint: "complete", wantCalls: []string{}},
		{name: "default ledger v12 ownership ambiguous", coreState: "12:f", sdpivotState: "absent", coreFingerprint: "invalid", sdpFingerprint: "complete", wantCalls: []string{"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:core-fingerprint"}},
		{name: "core incomplete with sdpivot markers", coreState: "12:f", sdpivotState: "absent", coreFingerprint: "complete", sdpFingerprint: "partial", sdpMarkers: "1", wantCalls: []string{"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:core-fingerprint", "psql:sdp-exists", "psql:sdp-flags"}},
		{name: "partial table set", coreState: "63:f", sdpivotState: "absent", coreFingerprint: "complete", sdpFingerprint: "partial"},
		{name: "critical column missing", coreState: "63:f", sdpivotState: "12:f", coreFingerprint: "complete", sdpFingerprint: "partial"},
		{name: "function missing or mismatched", coreState: "63:f", sdpivotState: "12:f", coreFingerprint: "complete", sdpFingerprint: "partial"},
		{name: "target rls missing", coreState: "63:f", sdpivotState: "12:f", coreFingerprint: "complete", sdpFingerprint: "partial"},
		{name: "document chunks not force or lacks dual tenant policy", coreState: "63:f", sdpivotState: "12:f", coreFingerprint: "complete", sdpFingerprint: "partial"},
		{name: "future sdpivot version", coreState: "63:f", sdpivotState: "18:f", coreFingerprint: "complete", sdpFingerprint: "complete", wantCalls: []string{}},
		{name: "future sdpivot version before core migration", coreState: "absent", sdpivotState: "18:f", publicTables: "0", coreFingerprint: "complete", sdpFingerprint: "empty", wantCalls: []string{}},
		{name: "v13 empty state table", coreState: "63:f", sdpivotState: "13:f", coreFingerprint: "complete", sdpFingerprint: "complete", sdpV13Schema: "partial"},
		{name: "v13 wrong state table structure", coreState: "63:f", sdpivotState: "13:f", coreFingerprint: "complete", sdpFingerprint: "complete", sdpV13Schema: "partial"},
		{name: "v13 known legacy hash still active", coreState: "63:f", sdpivotState: "13:f", coreFingerprint: "complete", sdpFingerprint: "complete", sdpV13Account: "partial"},
		{name: "latest policy missing", coreState: "63:f", sdpivotState: "17:f", coreFingerprint: "complete", sdpFingerprint: "complete", sdpLatest: "partial"},
		{name: "latest policy missing with check", coreState: "63:f", sdpivotState: "17:f", coreFingerprint: "complete", sdpFingerprint: "complete", sdpLatest: "partial"},
		{name: "latest extra permissive policy", coreState: "63:f", sdpivotState: "17:f", coreFingerprint: "complete", sdpFingerprint: "complete", sdpLatest: "partial"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.wantSuccess = false
			_, calls := runMigrationScenario(t, test)
			if test.wantCalls != nil {
				assertCalls(t, calls, expectedCallsWithInitialSDPivotInspection(test, test.wantCalls))
			}
			for _, call := range calls {
				if call == "migrate:bootstrap" || call == "migrate:sdpivot" {
					t.Fatalf("unsafe state reached SDPivot migration: %v", calls)
				}
			}
		})
	}
}

func TestMigrationStateMachineStopsImmediatelyWhenCommandFails(t *testing.T) {
	tests := []migrationScenario{
		{name: "initial inspection fails", coreState: "absent", sdpivotState: "absent", sdpFingerprint: "empty", failAt: "psql:core-exists", wantCalls: []string{"psql:core-exists"}},
		{name: "core migration fails", coreState: "absent", sdpivotState: "absent", publicTables: "0", sdpFingerprint: "empty", failAt: "migrate:core", wantCalls: []string{"psql:core-exists", "psql:core-audit-m44", "psql:public-count", "migrate:core"}},
		{name: "fingerprint inspection fails", coreState: "63:f", sdpivotState: "absent", sdpFingerprint: "empty", failAt: "psql:sdp-fingerprint", wantCalls: []string{"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:sdp-exists", "psql:sdp-fingerprint"}},
		{name: "bootstrap migration fails", coreState: "63:f", sdpivotState: "absent", sdpFingerprint: "empty", failAt: "migrate:bootstrap", wantCalls: []string{"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:sdp-exists", "psql:sdp-fingerprint", "migrate:bootstrap"}},
		{name: "security migration fails", coreState: "63:f", sdpivotState: "12:f", sdpFingerprint: "complete", failAt: "migrate:sdpivot", wantCalls: []string{"psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:core-exists", "psql:core-state", "psql:core-audit-m44", "psql:sdp-exists", "psql:sdp-state", "psql:sdp-fingerprint", "migrate:sdpivot"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.wantSuccess = false
			_, calls := runMigrationScenario(t, test)
			assertCalls(t, calls, expectedCallsWithInitialSDPivotInspection(test, test.wantCalls))
		})
	}
}

func TestMigrationFailureReportsSafeDiagnosticsWithoutCredentials(t *testing.T) {
	output, _ := runMigrationScenario(t, migrationScenario{
		name:            "sdpivot v14 migration failure diagnostics",
		coreState:       "63:f",
		sdpivotState:    "14:f",
		coreFingerprint: "complete",
		sdpFingerprint:  "complete",
		sdpV13Schema:    "complete",
		sdpV13Account:   "complete",
		failAt:          "migrate:sdpivot",
		wantSuccess:     false,
	})

	for _, safeDetail := range []string{
		"000015_remove_billing_subscription",
		`pq: relation "billing_subscriptions" does not exist`,
	} {
		if !strings.Contains(output, safeDetail) {
			t.Errorf("migration failure output must contain %q:\n%s", safeDetail, output)
		}
	}
	for _, secret := range []string{
		"secret-user",
		"secret-password",
		"postgres://secret-user:secret-password@db.internal/sdpivot",
	} {
		if strings.Contains(output, secret) {
			t.Errorf("migration failure output leaked %q:\n%s", secret, output)
		}
	}
}

func TestCoreAuditMigration44GateRunsBeforeCoreMutation(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		shape     string
		wantAllow bool
	}{
		{name: "43 missing", version: "43:f", shape: "missing", wantAllow: true},
		{name: "44 exact", version: "44:f", shape: "migration44_exact", wantAllow: true},
		{name: "62 exact", version: "62:f", shape: "migration44_exact", wantAllow: true},
		{name: "43 migration44 exact", version: "43:f", shape: "migration44_exact"},
		{name: "43 baseline exact", version: "43:f", shape: "baseline_exact"},
		{name: "43 invalid", version: "43:f", shape: "invalid"},
		{name: "44 missing", version: "44:f", shape: "missing"},
		{name: "44 baseline exact", version: "44:f", shape: "baseline_exact"},
		{name: "44 invalid", version: "44:f", shape: "invalid"},
		{name: "62 missing", version: "62:f", shape: "missing"},
		{name: "62 baseline exact", version: "62:f", shape: "baseline_exact"},
		{name: "62 invalid", version: "62:f", shape: "invalid"},
		{name: "63 missing", version: "63:f", shape: "missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario := migrationScenario{name: tt.name, coreState: tt.version, coreAuditShape: tt.shape, sdpivotState: "absent", sdpFingerprint: "empty", wantSuccess: tt.wantAllow}
			_, calls := runMigrationScenario(t, scenario)
			probe, coreMigrate := -1, -1
			for i, call := range calls {
				if call == "psql:core-audit-m44" && probe < 0 {
					probe = i
				}
				if call == "migrate:core" {
					coreMigrate = i
				}
			}
			if probe < 0 {
				t.Fatal("Core audit fingerprint probe was not recorded")
			}
			if tt.wantAllow {
				if coreMigrate < 0 || probe > coreMigrate {
					t.Fatalf("Core migration was not gated by the audit probe: %v", calls)
				}
			} else {
				if coreMigrate >= 0 {
					t.Fatalf("unsafe audit shape entered Core migration: %v", calls)
				}
				for _, call := range calls {
					if call == "migrate:core" {
						t.Fatalf("unsafe audit shape entered Core migration: %v", calls)
					}
				}
			}
		})
	}
}

func TestMigrationFingerprintCoversVersion13AndLatestContracts(t *testing.T) {
	script := readDeployFile(t, "migrate-op.sh")
	for _, fragment := range []string{
		"op-probe:sdpivot-v13-schema", "op-probe:sdpivot-v13-account",
		"sdpivot_disable_legacy_ops_admin_000013_state", "key_constraint.contype = 'p'",
		"admin@smartknora.com", `\$2a\$10\$l4fdcgy48s7whdmzzqaaf.sql5nna1.0uewfbbvzvasmsv0qyugs2`,
		"remaining_legacy_credential", "target.password_hash = '\\$2a\\$10\\$l4fdcgy48s7whdmzzqaaf.sql5nna1.0uewfbbvzvasmsv0qyugs2'",
		"!sdpivot-disabled-legacy-ops-admin:", "target.is_active is distinct from false",
		"target.must_change_password is distinct from true", "expected_policies(table_name, policy_name, is_chunk_policy)",
		"sdpivot_secure_000014_document_chunks", "policy.polcmd", "policy.polpermissive", "policy.polroles",
		"pg_get_expr(policy.polqual", "pg_get_expr(policy.polwithcheck", "'::text', '', 'g'", "'public.', ''", "extra_permissive_policies",
		"policy.using_expression <> 'existsselect1fromdocumentsdwhered.id=document_chunks.document_idandd.tenant_id=document_chunks.tenant_idandd.tenant_id=get_current_tenant_id'",
		"p.proconfig = array['search_path=pg_catalog, public']", "regexp_replace(lower(p.prosrc)",
		"sdpivot_schema_migrations version ${sdpivot_version} is newer than supported",
	} {
		if !strings.Contains(script, fragment) {
			t.Errorf("migration fingerprint must contain %q", fragment)
		}
	}
}

func TestMigrationFingerprintCoversVersion12Contract(t *testing.T) {
	script := readDeployFile(t, "migrate-op.sh")
	for _, fragment := range []string{
		"core_v12_fingerprint", "('kb_shares', 'knowledge_base_id'", "('organization_join_requests', 'request_type'",
		"('agent_shares', 'source_tenant_id'", "('tenant_disabled_shared_agents', 'source_tenant_id'",
		"sdpivot_v12_fingerprint", "('documents', 'tenant_id'", "('document_chunks', 'document_id'",
		"('document_chunks', 'tenant_id'", "('document_chunks', 'chunk_index'", "('document_chunks', 'content'",
		"set_tenant_context(bigint,boolean)", "get_current_tenant_id()", "is_ops_admin_context()",
		"prorettype = 'void'::regtype", "prorettype = 'bigint'::regtype", "prorettype = 'boolean'::regtype",
		"relrowsecurity", "relforcerowsecurity", "polcmd = '*'", "d.id=document_chunks.document_id",
		"d.tenant_id=document_chunks.tenant_id", "d.tenant_id=get_current_tenant_id",
		"write_category_config", "knowledge_spaces", "documents", "qa_sessions", "qa_messages",
		"writing_drafts", "announcements", "token_usage", "document_chunks",
		"pg_get_expr(policy.polqual", "pg_get_expr(policy.polwithcheck",
	} {
		if !strings.Contains(script, fragment) {
			t.Errorf("migration fingerprint must contain %q", fragment)
		}
	}
}

func TestMigrationImagePinsToolingAndScriptForbidsForce(t *testing.T) {
	dockerfile := readDeployFile(t, "migration/Dockerfile")
	if !strings.Contains(dockerfile, "golang_migrate_version=v4.19.1") {
		t.Fatal("migration image must pin golang-migrate v4.19.1")
	}
	script := readDeployFile(t, "migrate-op.sh")
	for _, forbidden := range []string{"migrate force", "set -x", "echo \"$op_database_url\""} {
		if strings.Contains(script, forbidden) {
			t.Errorf("migration script contains forbidden fragment %q", forbidden)
		}
	}
}
