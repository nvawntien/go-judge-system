package dto

import "mime/multipart"

type UploadTestCaseRequest struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

type TestCaseMetadataResponse struct {
	ID           int64  `json:"id"`
	ProblemID    int64  `json:"problem_id"`
	ZipObjectKey string `json:"zip_object_key"`
	TestCount    int    `json:"test_count"`
	Version      int    `json:"version"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type InternalTestCaseResponse struct {
	TestCount      int    `json:"test_count"`
	Version        int    `json:"version"`
	ZipDownloadURL string `json:"zip_download_url"`
}
