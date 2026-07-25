package handler

import (
	"strings"
	"testing"
)

func TestImportRowsFromRecordsAcceptsEnglishAndChineseHeaders(t *testing.T) {
	records := [][]string{
		{"手机号", "邮箱", "初始密码", "昵称"},
		{"13800138000", "ALICE@example.com", "password1", "Alice"},
	}
	rows, err := importRowsFromRecords(records)
	if err != nil {
		t.Fatalf("parse records: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one row, got %d", len(rows))
	}
	if rows[0].row != 2 || rows[0].phone != "13800138000" || rows[0].email != "alice@example.com" || rows[0].nickname != "Alice" {
		t.Fatalf("unexpected parsed row: %+v", rows[0])
	}
}

func TestImportRowsFromRecordsRequiresPasswordAndIdentityHeaders(t *testing.T) {
	for _, records := range [][][]string{
		{{"email"}, {"alice@example.com"}},
		{{"password"}, {"password1"}},
	} {
		if _, err := importRowsFromRecords(records); err == nil {
			t.Fatal("expected missing header error")
		}
	}
}

func TestImportRowsFromRecordsHandlesMissingOptionalColumns(t *testing.T) {
	rows, err := importRowsFromRecords([][]string{
		{"email", "password"},
		{"alice@example.com", "password1"},
	})
	if err != nil {
		t.Fatalf("parse records: %v", err)
	}
	if rows[0].phone != "" || rows[0].nickname != "" {
		t.Fatalf("missing optional columns used unrelated values: %+v", rows[0])
	}
}

func TestValidateUserImportRowDoesNotExposePassword(t *testing.T) {
	errs := validateUserImportRow(userImportRow{row: 3, email: "invalid", password: "short"})
	if len(errs) != 2 {
		t.Fatalf("expected email and password errors, got %+v", errs)
	}
	for _, item := range errs {
		if strings.Contains(item.Value, "short") {
			t.Fatalf("password leaked in validation error: %+v", item)
		}
	}
}
