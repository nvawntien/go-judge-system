package entity

import "time"

type TestCase struct {
	ID           int64
	ProblemID    int64
	ZipObjectKey string
	TestCount    int
	Version      string
	CreatedAt    time.Time
}

func NewTestCase(problemID int64, zipObjectKey string, testCount int, version string) *TestCase {
	return &TestCase{
		ProblemID:    problemID,
		ZipObjectKey: zipObjectKey,
		TestCount:    testCount,
		Version:      version,
		CreatedAt:    time.Now(),
	}
}

func (c *TestCase) UpdateMetadata(newZipKey string, newTestCount int, newVersion string) {
	c.ZipObjectKey = newZipKey
	c.TestCount = newTestCount
	c.Version = newVersion
}
