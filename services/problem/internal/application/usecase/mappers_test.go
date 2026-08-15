package usecase

import (
	"encoding/json"
	"strings"
	"testing"

	"go-judge-system/services/problem/internal/domain/entity"
)

func TestTestCaseMetadataDoesNotSerializeStorageKey(t *testing.T) {
	t.Parallel()
	problem := &entity.Problem{ID: 42}
	testcase := &entity.TestCase{ID: 7, ProblemID: 42, ZipObjectKey: "problems/42/testcases/v1.zip", TestCount: 2, Version: 1}

	for name, value := range map[string]any{
		"testcase metadata": MapTestCaseToMetadataResponse(testcase),
		"problem detail":    MapProblemToAdminDetailResponse(problem, testcase),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			body, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(body), "zip_object_key") || strings.Contains(string(body), testcase.ZipObjectKey) {
				t.Fatalf("response leaked testcase storage key: %s", body)
			}
		})
	}
}
