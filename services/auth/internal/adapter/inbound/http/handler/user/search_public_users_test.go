package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-judge-system/services/auth/internal/application/dto"
	"go-judge-system/services/auth/internal/application/port/inbound"

	"github.com/gin-gonic/gin"
)

func TestSearchPublicUsersHandlerBindsPublicQueryAndDoesNotLeakAccountFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uc := &fakeSearchPublicUsersUseCase{}
	router := gin.New()
	router.GET("/api/v1/users/search", NewSearchPublicUsersHandler(uc).Handle)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/users/search?q=ada&page=2&limit=3", nil))
	if recorder.Code != http.StatusOK || uc.req.Query != "ada" || uc.req.Page == nil || *uc.req.Page != 2 || uc.req.Limit == nil || *uc.req.Limit != 3 {
		t.Fatalf("status=%d request=%+v", recorder.Code, uc.req)
	}
	if body := recorder.Body.String(); containsSensitivePublicSearchField(body) {
		t.Fatalf("response leaked sensitive field: %s", body)
	}
}

type fakeSearchPublicUsersUseCase struct {
	req dto.SearchPublicUsersRequest
}

func (f *fakeSearchPublicUsersUseCase) Execute(_ context.Context, req dto.SearchPublicUsersRequest) (dto.SearchPublicUsersResponse, error) {
	f.req = req
	return dto.SearchPublicUsersResponse{
		Items:      []dto.PublicUserSearchItem{{Username: "ada", FullName: "Ada Lovelace", Rating: 1800}},
		Pagination: dto.PublicUserSearchPagination{Page: 2, Limit: 3, Total: 1, TotalPages: 1},
	}, nil
}

func containsSensitivePublicSearchField(body string) bool {
	for _, field := range []string{"email", "role", "is_active", "is_suspended", "password", "avatar_object_key"} {
		if strings.Contains(body, field) {
			return true
		}
	}
	return false
}

var _ inbound.SearchPublicUsersUseCase = (*fakeSearchPublicUsersUseCase)(nil)
