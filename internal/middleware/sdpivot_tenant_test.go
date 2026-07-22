package middleware

import (
	"os"
	"strings"
	"testing"
)

func TestSDPivotTenantFallbackContextIsTransactionLocalAndNeverElevated(t *testing.T) {
	query, args := sdPivotTenantFallbackContext(uint64(42))

	want := "SELECT set_config('app.current_tenant_id', ?, true), set_config('app.is_ops_admin', 'false', true)"
	if query != want {
		t.Fatalf("fallback query = %q, want %q", query, want)
	}
	if len(args) != 1 {
		t.Fatalf("fallback args = %#v, want tenant ID only", args)
	}
	if got, ok := args[0].(string); !ok || got != "42" {
		t.Fatalf("fallback tenant arg = %#v, want string 42", args[0])
	}
	if strings.Contains(query, "app.is_ops_admin', ?") {
		t.Fatal("fallback must not bind the request role to the operations context")
	}
}

func TestSDPivotTenantFallbackFailureRollsBackAndHidesDatabaseDetails(t *testing.T) {
	content, err := os.ReadFile("sdpivot_tenant.go")
	if err != nil {
		t.Fatalf("read tenant middleware: %v", err)
	}
	source := strings.Join(strings.Fields(string(content)), " ")

	failureFlow := `if fallbackErr := tx.Exec(fallbackSQL, fallbackArgs...).Error; fallbackErr != nil {
		_ = tx.Rollback().Error
		log.Printf("failed to set tenant database context: %v", fallbackErr)
		c.JSON(500, gin.H{"error": "failed to set tenant database context"})
		c.Abort()
		return
	}`
	if !strings.Contains(source, strings.Join(strings.Fields(failureFlow), " ")) {
		t.Fatal("fallback failure must roll back before returning the generic client error")
	}
	if strings.Contains(source, `"detail": fallbackErr.Error()`) {
		t.Fatal("fallback failure must not expose database details to clients")
	}
}
