package services

import (
	"DuDe/internal/models"
	"bytes"
	"errors"
	"io"
	"testing"
)

type coverageTestFile struct {
	bytes.Buffer
	closeErr error
}

func (f *coverageTestFile) Close() error { return f.closeErr }

func TestSaveCoverageReportRemovesIncompleteFileAfterWriteFailure(t *testing.T) {
	expectedWrite := errors.New("write failed")
	expectedRemove := errors.New("remove failed")
	removed := false
	_, err := saveCoverageReport(
		models.MissingContentResult{}, models.ScanDiagnostics{}, t.TempDir(),
		func(string) (io.WriteCloser, error) { return &coverageFailingFile{err: expectedWrite}, nil },
		func(string) error { removed = true; return expectedRemove },
	)
	if !errors.Is(err, expectedWrite) || !errors.Is(err, expectedRemove) {
		t.Fatalf("expected write and cleanup errors, got %v", err)
	}
	if !removed {
		t.Fatal("expected incomplete report cleanup")
	}
}

func TestSaveCoverageReportReturnsCloseFailureAndRemovesFile(t *testing.T) {
	expectedClose := errors.New("close failed")
	removed := false
	_, err := saveCoverageReport(
		models.MissingContentResult{}, models.ScanDiagnostics{}, t.TempDir(),
		func(string) (io.WriteCloser, error) { return &coverageTestFile{closeErr: expectedClose}, nil },
		func(string) error { removed = true; return nil },
	)
	if !errors.Is(err, expectedClose) {
		t.Fatalf("expected close error, got %v", err)
	}
	if !removed {
		t.Fatal("expected incomplete report cleanup")
	}
}

type coverageFailingFile struct {
	err error
}

func (f *coverageFailingFile) Write([]byte) (int, error) { return 0, f.err }
func (f *coverageFailingFile) Close() error              { return nil }

func TestWriteCoverageCSVReportsFlushFailure(t *testing.T) {
	expected := errors.New("flush failed")
	writer := &coverageFlushFailingWriter{err: expected}
	if err := WriteCoverageCSV(writer, models.MissingContentResult{}, models.ScanDiagnostics{}); !errors.Is(err, expected) {
		t.Fatalf("expected flush error, got %v", err)
	}
}

type coverageFlushFailingWriter struct {
	bytes.Buffer
	err error
}

func (w *coverageFlushFailingWriter) Write(data []byte) (int, error) {
	if w.Len() > 0 {
		return 0, w.err
	}
	return w.Buffer.Write(data)
}
