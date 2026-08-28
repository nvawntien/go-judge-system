package cache

import (
	"testing"

	"go-judge-system/services/problem/internal/application/port/outbound"
)

func TestDecodeSubmissionProblemMetadata(t *testing.T) {
	valid := []byte(`{"id":42,"title":"Two Sum","slug":"two-sum","time_limit":1000,"memory_limit":256,"author_id":"author","is_hidden":true}`)

	got, err := decodeSubmissionProblemMetadata(42, valid)
	if err != nil {
		t.Fatalf("decode error = %v", err)
	}
	want := outbound.SubmissionProblemMetadata{
		ID: 42, Title: "Two Sum", Slug: "two-sum", TimeLimit: 1000, MemoryLimit: 256, AuthorID: "author", IsHidden: true,
	}
	if got != want {
		t.Fatalf("metadata = %+v, want %+v", got, want)
	}
}

func TestDecodeSubmissionProblemMetadataRejectsCorruptOrPartialEntries(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "malformed JSON", data: []byte(`{`)},
		{name: "wrong key ID", data: []byte(`{"id":43,"title":"Two Sum","slug":"two-sum","time_limit":1000,"memory_limit":256,"author_id":"author","is_hidden":false}`)},
		{name: "missing hidden state", data: []byte(`{"id":42,"title":"Two Sum","slug":"two-sum","time_limit":1000,"memory_limit":256,"author_id":"author"}`)},
		{name: "missing author", data: []byte(`{"id":42,"title":"Two Sum","slug":"two-sum","time_limit":1000,"memory_limit":256,"is_hidden":false}`)},
		{name: "empty title", data: []byte(`{"id":42,"title":"","slug":"two-sum","time_limit":1000,"memory_limit":256,"author_id":"author","is_hidden":false}`)},
		{name: "zero memory limit", data: []byte(`{"id":42,"title":"Two Sum","slug":"two-sum","time_limit":1000,"memory_limit":0,"author_id":"author","is_hidden":false}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := decodeSubmissionProblemMetadata(42, tt.data); err == nil {
				t.Fatal("decode error = nil, want corrupt cache entry rejection")
			}
		})
	}
}
