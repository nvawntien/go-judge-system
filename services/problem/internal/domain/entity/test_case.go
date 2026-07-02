package entity

import "time"

type TestCase struct {
	ID           int64
	ProblemID    int64
	ZipObjectKey string
	TestCount    int
	Version      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewTestCase(problemID int64, zipObjectKey string, testCount int, version int) *TestCase {
	now := time.Now()

	return &TestCase{
		ProblemID:    problemID,
		ZipObjectKey: zipObjectKey,
		TestCount:    testCount,
		Version:      version,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (c *TestCase) UpdateMetadata(newZipKey string, newTestCount int, newVersion int) {
	c.ZipObjectKey = newZipKey
	c.TestCount = newTestCount
	c.Version = newVersion
	c.UpdatedAt = time.Now()
}