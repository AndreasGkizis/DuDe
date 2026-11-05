package services

import (
	"DuDe/internal/models"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestReportsRetryOnlyFileCollisions(t *testing.T) {
	diagnostics := models.ScanDiagnostics{Diagnostics: []models.Diagnostic{{Path: "/private", Reason: "permission denied"}}}
	for _, report := range []struct {
		name string
		save func(string, func(string) (io.WriteCloser, error)) (string, error)
	}{
		{"duplicate", func(directory string, create func(string) (io.WriteCloser, error)) (string, error) {
			return saveDuplicateReport(&sync.Map{}, diagnostics, directory, create, os.Remove)
		}},
		{"coverage", func(directory string, create func(string) (io.WriteCloser, error)) (string, error) {
			return saveCoverageReport(models.MissingContentResult{}, diagnostics, directory, create, os.Remove)
		}},
	} {
		t.Run(report.name, func(t *testing.T) {
			t.Run("collision preserves existing report", func(t *testing.T) {
				var existingPath string
				calls := 0
				path, err := report.save(t.TempDir(), func(path string) (io.WriteCloser, error) {
					calls++
					if calls == 1 {
						// Simulate another writer creating the chosen path just
						// before we attempt exclusive creation.
						existingPath = path
						if err := os.WriteFile(path, []byte("existing report"), 0o600); err != nil {
							return nil, err
						}
					}
					return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
				})
				if err != nil {
					t.Fatalf("save report after collision: %v", err)
				}
				wantPath := strings.TrimSuffix(existingPath, filepath.Ext(existingPath)) + "_1" + filepath.Ext(existingPath)
				if calls != 2 || path != wantPath {
					t.Fatalf("creation calls = %d, path = %q; want 2 calls and %q", calls, path, wantPath)
				}
				content, err := os.ReadFile(existingPath)
				if err != nil || string(content) != "existing report" {
					t.Fatalf("existing report changed: content = %q, error = %v", content, err)
				}
			})

			for _, expected := range []error{os.ErrPermission, os.ErrNotExist} {
				t.Run(expected.Error(), func(t *testing.T) {
					calls := 0
					path, err := report.save(t.TempDir(), func(path string) (io.WriteCloser, error) {
						calls++
						return nil, &os.PathError{Op: "open", Path: path, Err: expected}
					})
					if !errors.Is(err, expected) || calls != 1 || path != "" {
						t.Fatalf("path = %q, error = %v, calls = %d; want empty path, %v, and no retries", path, err, calls, expected)
					}
				})
			}
		})
	}
}

func TestConcurrentReportsHaveDistinctPathsAndContents(t *testing.T) {
	for _, report := range []struct {
		name string
		save func(models.ScanDiagnostics, string) (string, error)
	}{
		{"duplicate", func(diagnostics models.ScanDiagnostics, directory string) (string, error) {
			return SaveDuplicateReport(&sync.Map{}, diagnostics, directory)
		}},
		{"coverage", func(diagnostics models.ScanDiagnostics, directory string) (string, error) {
			return SaveCoverageReport(models.MissingContentResult{}, diagnostics, directory)
		}},
	} {
		t.Run(report.name, func(t *testing.T) {
			const writers = 16
			directory := t.TempDir()
			start := make(chan struct{})
			type outcome struct {
				path, marker string
				err          error
			}
			results := make(chan outcome, writers)
			for i := 0; i < writers; i++ {
				go func(i int) {
					marker := fmt.Sprintf("writer-%03d", i)
					<-start
					path, err := report.save(models.ScanDiagnostics{Diagnostics: []models.Diagnostic{{Path: marker, Reason: "skipped"}}}, directory)
					results <- outcome{path: path, marker: marker, err: err}
				}(i)
			}
			close(start)

			paths := make(map[string]bool, writers)
			for i := 0; i < writers; i++ {
				result := <-results
				if result.err != nil {
					t.Errorf("save report: %v", result.err)
					continue
				}
				if paths[result.path] {
					t.Errorf("report path reused: %q", result.path)
				}
				paths[result.path] = true
				content, err := os.ReadFile(result.path)
				if err != nil || !strings.Contains(string(content), result.marker) {
					t.Errorf("report %q lost its content: %q, error: %v", result.path, content, err)
				}
			}
		})
	}
}
