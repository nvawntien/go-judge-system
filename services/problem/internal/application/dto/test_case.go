package dto

import "mime/multipart"

type UploadTestCaseRequest struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

type UploadTestCasesResponse struct {
	TestCount int    `json:"test_count"`
	Version   string `json:"version"`
}

type TestCaseMetadataResponse struct {
	ProblemID   int64  `json:"problem_id"`
	TestCount   int    `json:"test_count"`
	Version     string `json:"version"`
	DownloadURL string `json:"download_url,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type InternalTestCaseResponse struct {
	ProblemID      int64  `json:"problem_id"`
	TestCount      int    `json:"test_count"`
	Version        string `json:"version"`
	ZipDownloadURL string `json:"zip_download_url"`
}
