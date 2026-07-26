package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestLoadAccountsRequiresCanonicalOPTenant(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.csv")
	err := os.WriteFile(path, []byte("email,username,password,tenant_id\nadmin@example.com,admin,ValidPass!123,2\n"), 0o600)
	require.NoError(t, err)

	_, err = loadAccounts(path)
	require.ErrorContains(t, err, "tenant_id must be 1 in OP mode")
}

func TestLoadAccountsDefaultsToCanonicalOPTenantAndAdminRole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.csv")
	csv := strings.Join([]string{
		"email,username,password,is_ops_admin",
		"admin@example.com,admin,ValidPass!123,true",
	}, "\n")
	require.NoError(t, os.WriteFile(path, []byte(csv), 0o600))

	accounts, err := loadAccounts(path)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, types.DefaultTenantID, accounts[0].TenantID)
	require.Equal(t, string(types.AccessRoleSuperAdmin), accountAccessRole(accounts[0]))
}

func TestPasswordHashPreservesMatchingBcryptHash(t *testing.T) {
	existing, err := bcrypt.GenerateFromPassword([]byte("ValidPass!123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	hash, changed, err := passwordHash(string(existing), "ValidPass!123")
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, existing, hash)

	hash, changed, err = passwordHash(string(existing), "NewPass!123")
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, bcrypt.CompareHashAndPassword(hash, []byte("NewPass!123")))
}
