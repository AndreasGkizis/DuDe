package unit_tests

import (
	"errors"
	"strings"
	"testing"

	"DuDe/internal/models"
	"DuDe/internal/processing"
)

func TestWriteResultsCSVReturnsFlushErrors(t *testing.T) {
	expectedErr := errors.New("disk write failed")
	output := &failAfterFirstWrite{err: expectedErr}

	err := processing.WriteResultsCSV(output, []models.ResultEntry{{
		Filename:          "original.txt",
		FullPath:          "/files/original.txt",
		DuplicateFilename: "duplicate.txt",
		DuplicateFullPath: "/files/duplicate.txt",
	}})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected flush error %v, got %v", expectedErr, err)
	}
	if !strings.Contains(err.Error(), "flush") {
		t.Fatalf("expected flush context in error, got %v", err)
	}
}

type failAfterFirstWrite struct {
	writes int
	err    error
}

func (writer *failAfterFirstWrite) Write(data []byte) (int, error) {
	writer.writes++
	if writer.writes > 1 {
		return 0, writer.err
	}
	return len(data), nil
}
