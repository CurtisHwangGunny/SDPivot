package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"golang.org/x/crypto/bcrypt"
)

type provisioningUserRepo struct {
	interfaces.UserRepository
	created       *types.User
	updatedTenant uint64
}

func (r *provisioningUserRepo) GetUserByEmail(context.Context, string) (*types.User, error) {
	return nil, nil
}

func (r *provisioningUserRepo) GetUserByUsername(context.Context, string) (*types.User, error) {
	return nil, nil
}

func (r *provisioningUserRepo) CreateUser(_ context.Context, user *types.User) error {
	copy := *user
	r.created = &copy
	return nil
}

func (r *provisioningUserRepo) UpdateUser(_ context.Context, user *types.User) error {
	r.updatedTenant = user.TenantID
	return nil
}

type provisioningTenantService struct {
	interfaces.TenantService
	createCalls int
}

func (s *provisioningTenantService) CreateTenant(context.Context, *types.Tenant) (*types.Tenant, error) {
	s.createCalls++
	return &types.Tenant{ID: 99}, nil
}

func (s *provisioningTenantService) GetTenantByID(_ context.Context, id uint64) (*types.Tenant, error) {
	return &types.Tenant{ID: id}, nil
}

type provisioningMemberService struct {
	interfaces.TenantMemberService
	members []*types.TenantMember
}

type bootstrapUserRepo struct {
	interfaces.UserRepository
	user        *types.User
	createCalls int
	updateCalls int
}

func (r *bootstrapUserRepo) GetUserByEmail(context.Context, string) (*types.User, error) {
	return r.user, nil
}
func (r *bootstrapUserRepo) CreateUser(_ context.Context, user *types.User) error {
	r.createCalls++
	r.user = user
	return nil
}
func (r *bootstrapUserRepo) UpdateUser(_ context.Context, user *types.User) error {
	r.updateCalls++
	r.user = user
	return nil
}

type bootstrapMemberService struct {
	interfaces.TenantMemberService
	ensureCalls int
}

func (s *bootstrapMemberService) EnsureOwner(context.Context, string, uint64) (*types.TenantMember, error) {
	s.ensureCalls++
	return &types.TenantMember{Role: types.TenantRoleOwner}, nil
}

func (s *provisioningMemberService) ListByUser(context.Context, string) ([]*types.TenantMember, error) {
	return s.members, nil
}

func TestUserServiceRegisterTenantlessSkipsTenantCreation(t *testing.T) {
	repo := &provisioningUserRepo{}
	tenantSvc := &provisioningTenantService{}
	svc := &userService{userRepo: repo, tenantService: tenantSvc}

	user, err := svc.Register(context.Background(), &types.RegisterRequest{
		Username:           "alice",
		Email:              "alice@example.com",
		Password:           "supersecret",
		TenantProvisioning: types.TenantProvisioningTenantless,
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if tenantSvc.createCalls != 0 {
		t.Fatalf("tenant create calls = %d, want 0", tenantSvc.createCalls)
	}
	if user.TenantID != 0 || repo.created == nil || repo.created.TenantID != 0 {
		t.Fatalf("tenantless user persisted with tenant: user=%d created=%v", user.TenantID, repo.created)
	}
}

func TestBootstrapOPAdminIsIdempotentAndDoesNotResetExistingPassword(t *testing.T) {
	repo := &bootstrapUserRepo{}
	tenantSvc := &provisioningTenantService{}
	memberSvc := &bootstrapMemberService{}
	svc := &userService{userRepo: repo, tenantService: tenantSvc, memberService: memberSvc}

	if err := svc.BootstrapOPAdmin(context.Background(), "operator@example.com", "Bootstrap9"); err != nil {
		t.Fatalf("first BootstrapOPAdmin: %v", err)
	}
	if repo.createCalls != 1 || repo.user == nil || repo.user.MustChangePassword != true {
		t.Fatalf("first bootstrap state: creates=%d user=%+v", repo.createCalls, repo.user)
	}
	firstHash := repo.user.PasswordHash
	if err := svc.BootstrapOPAdmin(context.Background(), "operator@example.com", "Different9"); err != nil {
		t.Fatalf("second BootstrapOPAdmin: %v", err)
	}
	if repo.createCalls != 1 || repo.user.PasswordHash != firstHash {
		t.Fatalf("bootstrap reset or duplicated user: creates=%d hashChanged=%v", repo.createCalls, repo.user.PasswordHash != firstHash)
	}
	if repo.user.AccessRole != types.AccessRoleSuperAdmin || !repo.user.IsActive || tenantSvc.createCalls != 1 || memberSvc.ensureCalls != 2 {
		t.Fatalf("bootstrap invariants not maintained: user=%+v tenants=%d members=%d", repo.user, tenantSvc.createCalls, memberSvc.ensureCalls)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.user.PasswordHash), []byte("Bootstrap9")); err != nil {
		t.Fatalf("bootstrap password hash invalid: %v", err)
	}
}

func TestResolveLoginTenantIDRepairsTenantlessUserWithMembership(t *testing.T) {
	repo := &provisioningUserRepo{}
	tenantSvc := &provisioningTenantService{}
	memberSvc := &provisioningMemberService{members: []*types.TenantMember{
		{TenantID: 42, Status: types.TenantMemberStatusActive},
	}}
	svc := &userService{userRepo: repo, tenantService: tenantSvc, memberService: memberSvc}
	user := &types.User{ID: "alice", TenantID: 0}

	if got := svc.resolveLoginTenantID(context.Background(), user); got != 42 {
		t.Fatalf("resolved tenant = %d, want 42", got)
	}
	if repo.updatedTenant != 42 || user.TenantID != 42 {
		t.Fatalf("repair was not persisted: repo=%d user=%d", repo.updatedTenant, user.TenantID)
	}
}
