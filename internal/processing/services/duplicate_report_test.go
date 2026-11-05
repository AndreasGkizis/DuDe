package services

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"

	"DuDe/internal/models"
)

type duplicateTestFile struct {
	bytes.Buffer
	closeErr error
}

func (f *duplicateTestFile) Close() error { return f.closeErr }

func TestSaveDuplicateReportReturnsCreateFailure(t *testing.T) {
	expected := errors.New("create failed")
	_, err := saveDuplicateReport(
		&sync.Map{}, models.ScanDiagnostics{Diagnostics: []models.Diagnostic{{Path: "/private"}}}, t.TempDir(),
		func(string) (io.WriteCloser, error) { return nil, expected },
		func(string) error { return nil },
	)
	if !errors.Is(err, expected) {
		t.Fatalf("expected create error, got %v", err)
	}
}

func TestSaveDuplicateReportRemovesIncompleteFileAfterWriteFailure(t *testing.T) {
	expectedWrite := errors.New("write failed")
	expectedRemove := errors.New("remove failed")
	removed := false
	_, err := saveDuplicateReport(
		&sync.Map{}, models.ScanDiagnostics{Diagnostics: []models.Diagnostic{{Path: "/private"}}}, t.TempDir(),
		func(string) (io.WriteCloser, error) { return &duplicateFailingFile{err: expectedWrite}, nil },
		func(string) error { removed = true; return expectedRemove },
	)
	if !errors.Is(err, expectedWrite) || !errors.Is(err, expectedRemove) {
		t.Fatalf("expected write and cleanup errors, got %v", err)
	}
	if !removed {
		t.Fatal("expected incomplete report cleanup")
	}
}

func TestSaveDuplicateReportReturnsCloseFailureAndRemovesFile(t *testing.T) {
	expectedClose := errors.New("close failed")
	removed := false
	_, err := saveDuplicateReport(
		&sync.Map{}, models.ScanDiagnostics{Diagnostics: []models.Diagnostic{{Path: "/private"}}}, t.TempDir(),
		func(string) (io.WriteCloser, error) { return &duplicateTestFile{closeErr: expectedClose}, nil },
		func(string) error { removed = true; return nil },
	)
	if !errors.Is(err, expectedClose) {
		t.Fatalf("expected close error, got %v", err)
	}
	if !removed {
		t.Fatal("expected incomplete report cleanup")
	}
}

func TestWriteDuplicateCSVReportsFlushFailure(t *testing.T) {
	expected := errors.New("flush failed")
	writer := &duplicateFlushFailingWriter{err: expected}
	if err := writeDuplicateCSV(writer, nil, models.ScanDiagnostics{Diagnostics: []models.Diagnostic{{Path: "/private"}}}); !errors.Is(err, expected) {
		t.Fatalf("expected flush error, got %v", err)
	}
}

type duplicateFailingFile struct {
	err error
}

func (f *duplicateFailingFile) Write([]byte) (int, error) { return 0, f.err }
func (f *duplicateFailingFile) Close() error              { return nil }

type duplicateFlushFailingWriter struct {
	bytes.Buffer
	err error
}

func (w *duplicateFlushFailingWriter) Write(data []byte) (int, error) {
	if w.Len() > 0 {
		return 0, w.err
	}
	return w.Buffer.Write(data)
}
