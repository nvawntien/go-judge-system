package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	pkgauth "go-judge-system/pkg/auth"
	"go-judge-system/pkg/rbac"
	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/outbound"
	"go-judge-system/services/auth/internal/domain"
	"go-judge-system/services/auth/internal/domain/entity"
)

const testUserID = "b249d9c8-3764-4ea0-a335-f1d99eb701a8"

func TestAdminUsersListFiltersAndPagination(t *testing.T) {
	active, suspended := true, false
	role := rbac.RoleModerator
	repo := &adminUsersRepository{listResult: outbound.ListUsersResult{
		Items: []*entity.User{{ID: testUserID, Username: "Ada", Email: "ada@example.test", Role: role, IsActive: true, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0)}},
		Total: 23,
	}}
	uc := NewAdminUsersUseCase(repo, &adminUsersIATStore{})
	page, limit := 2, 10
	result, err := uc.List(context.Background(), adminClaims(), dto.ListAdminUsersRequest{Page: &page, Limit: &limit, Search: " ada ", Role: "moderator", Status: "active"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if result.Pagination.Page != 2 || result.Pagination.Limit != 10 || result.Pagination.Total != 23 || result.Pagination.TotalPages != 3 {
		t.Fatalf("pagination = %+v", result.Pagination)
	}
	if len(result.Items) != 1 || result.Items[0].Email != "ada@example.test" || result.Items[0].IsSuspended {
		t.Fatalf("items = %+v", result.Items)
	}
	if repo.filter.Search != "ada" || repo.filter.Role == nil || *repo.filter.Role != role || repo.filter.IsActive == nil || *repo.filter.IsActive != active || repo.filter.IsSuspended == nil || *repo.filter.IsSuspended != suspended || repo.filter.Offset != 10 {
		t.Fatalf("filter = %+v", repo.filter)
	}
}

func TestAdminUsersListRejectsInvalidFiltersAndNonAdmin(t *testing.T) {
	uc := NewAdminUsersUseCase(&adminUsersRepository{}, &adminUsersIATStore{})
	if _, err := uc.List(context.Background(), pkgauth.Claims{Role: rbac.RoleModerator}, dto.ListAdminUsersRequest{}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("non-admin error = %v, want forbidden", err)
	}
	zero, overLimit := 0, 101
	for _, req := range []dto.ListAdminUsersRequest{{Page: &zero}, {Limit: &overLimit}, {Role: "owner"}, {Status: "disabled"}} {
		if _, err := uc.List(context.Background(), adminClaims(), req); err == nil {
			t.Fatalf("List(%+v) error = nil", req)
		}
	}
}

func TestAdminUsersListMapsRepositoryFailure(t *testing.T) {
	repo := &adminUsersRepository{listErr: errors.New("database unavailable")}
	_, err := NewAdminUsersUseCase(repo, &adminUsersIATStore{}).List(context.Background(), adminClaims(), dto.ListAdminUsersRequest{})
	if !errors.Is(err, domain.ErrInternalServer) {
		t.Fatalf("List() error = %v, want internal server", err)
	}
}

func TestAdminUsersGetRequiresAdminAndReturnsUser(t *testing.T) {
	user := &entity.User{ID: testUserID, Username: "ada", Email: "ada@example.test", Role: rbac.RoleUser}
	uc := NewAdminUsersUseCase(&adminUsersRepository{users: map[string]*entity.User{user.ID: user}}, &adminUsersIATStore{})

	result, err := uc.Get(context.Background(), adminClaims(), dto.UserIDRequest{UserID: testUserID})
	if err != nil || result.ID != testUserID || result.Email != user.Email {
		t.Fatalf("Get() result=%+v err=%v", result, err)
	}
	if _, err := uc.Get(context.Background(), pkgauth.Claims{Role: rbac.RoleUser}, dto.UserIDRequest{UserID: testUserID}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("non-admin error = %v, want forbidden", err)
	}
	if _, err := uc.Get(context.Background(), adminClaims(), dto.UserIDRequest{UserID: "not-a-uuid"}); err == nil {
		t.Fatal("invalid ID error = nil")
	}
}

func TestAdminUsersSuspensionRevokesBeforePersistenceAndUnsuspendKeepsCutoff(t *testing.T) {
	user := &entity.User{ID: testUserID, IsActive: true, UpdatedAt: time.Unix(1, 0)}
	repo := &adminUsersRepository{users: map[string]*entity.User{user.ID: user}}
	store := &adminUsersIATStore{}
	uc := NewAdminUsersUseCase(repo, store)
	suspended := true
	response, err := uc.SetSuspension(context.Background(), adminClaims(), dto.UserIDRequest{UserID: testUserID}, dto.SetUserSuspensionRequest{Suspended: &suspended})
	if err != nil {
		t.Fatalf("suspend error = %v", err)
	}
	if !response.IsSuspended || !user.IsSuspended || store.value == 0 || repo.updateCalls != 1 {
		t.Fatalf("suspend response=%+v user=%+v store=%d updates=%d", response, user, store.value, repo.updateCalls)
	}

	cutoff := store.value
	unsuspended := false
	response, err = uc.SetSuspension(context.Background(), adminClaims(), dto.UserIDRequest{UserID: testUserID}, dto.SetUserSuspensionRequest{Suspended: &unsuspended})
	if err != nil {
		t.Fatalf("unsuspend error = %v", err)
	}
	if response.IsSuspended || user.IsSuspended || store.value != cutoff || repo.updateCalls != 2 {
		t.Fatalf("unsuspend response=%+v user=%+v store=%d updates=%d", response, user, store.value, repo.updateCalls)
	}
}

func TestAdminUsersSuspensionStopsWhenInvalidationFails(t *testing.T) {
	user := &entity.User{ID: testUserID, IsActive: true}
	repo := &adminUsersRepository{users: map[string]*entity.User{user.ID: user}}
	store := &adminUsersIATStore{setErr: errors.New("redis unavailable")}
	suspended := true
	_, err := NewAdminUsersUseCase(repo, store).SetSuspension(context.Background(), adminClaims(), dto.UserIDRequest{UserID: testUserID}, dto.SetUserSuspensionRequest{Suspended: &suspended})
	if !errors.Is(err, domain.ErrInternalServer) || user.IsSuspended || repo.updateCalls != 0 {
		t.Fatalf("error=%v suspended=%t updates=%d", err, user.IsSuspended, repo.updateCalls)
	}
}

func TestAdminUsersSuspensionMapsPersistenceFailure(t *testing.T) {
	user := &entity.User{ID: testUserID, IsActive: true}
	repo := &adminUsersRepository{users: map[string]*entity.User{user.ID: user}, updateErr: errors.New("database unavailable")}
	suspended := true
	_, err := NewAdminUsersUseCase(repo, &adminUsersIATStore{}).SetSuspension(context.Background(), adminClaims(), dto.UserIDRequest{UserID: testUserID}, dto.SetUserSuspensionRequest{Suspended: &suspended})
	if !errors.Is(err, domain.ErrInternalServer) || !user.IsSuspended || repo.updateCalls != 1 {
		t.Fatalf("error=%v suspended=%t updates=%d", err, user.IsSuspended, repo.updateCalls)
	}
}

func adminClaims() pkgauth.Claims { return pkgauth.Claims{UserID: "admin", Role: rbac.RoleAdmin} }

type adminUsersIATStore struct {
	value  int64
	setErr error
}

func (s *adminUsersIATStore) SetLogoutAllIAT(_ context.Context, _ string, issuedAt int64) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.value = issuedAt
	return nil
}
func (s *adminUsersIATStore) GetLogoutAllIAT(context.Context, string) (int64, error) {
	return s.value, nil
}

type adminUsersRepository struct {
	users       map[string]*entity.User
	listResult  outbound.ListUsersResult
	listErr     error
	filter      outbound.ListUsersFilter
	updateCalls int
	updateErr   error
}

func (r *adminUsersRepository) CreateUser(context.Context, *entity.User) error { return nil }
func (r *adminUsersRepository) GetUserByEmail(context.Context, string) (*entity.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *adminUsersRepository) GetUserByUsername(context.Context, string) (*entity.User, error) {
	return nil, domain.ErrUserNotFound
}
func (r *adminUsersRepository) GetUserById(_ context.Context, id string) (*entity.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}
func (r *adminUsersRepository) ListUsers(_ context.Context, filter outbound.ListUsersFilter) (outbound.ListUsersResult, error) {
	r.filter = filter
	return r.listResult, r.listErr
}
func (r *adminUsersRepository) SearchPublicUsers(context.Context, outbound.SearchPublicUsersFilter) (outbound.SearchPublicUsersResult, error) {
	return outbound.SearchPublicUsersResult{}, nil
}
func (r *adminUsersRepository) UpdateUser(_ context.Context, user *entity.User) error {
	r.updateCalls++
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.users == nil {
		r.users = map[string]*entity.User{}
	}
	r.users[user.ID] = user
	return nil
}
func (r *adminUsersRepository) UpdatePassword(context.Context, string, string, time.Time) error {
	return nil
}
func (r *adminUsersRepository) UpdateProfile(context.Context, string, outbound.ProfileUpdates) error {
	return nil
}
func (r *adminUsersRepository) UpdateAvatar(context.Context, string, string, string, time.Time) error {
	return nil
}
func (r *adminUsersRepository) DeleteUser(context.Context, string) error { return nil }
