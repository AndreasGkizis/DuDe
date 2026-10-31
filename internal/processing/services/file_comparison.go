package services

import (
	"DuDe/internal/models"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

// FilesMatch applies metadata/hash matching and optionally confirms the match
// byte-for-byte. The primary file is supplied open so callers comparing many
// candidates can reuse the same descriptor.
func FilesMatch(ctx context.Context, primaryFile *os.File, primary, candidate models.FileHash, paranoid bool) (bool, error) {
	if primary.FileSize != candidate.FileSize {
		return false, nil
	}

	if !paranoid {
		// Normal mode requires both hashes. Matching size alone is not enough.
		if primary.Hash == "" || candidate.Hash == "" {
			return false, nil
		}
		return primary.Hash == candidate.Hash, nil
	}

	// In Paranoid Mode, a known hash mismatch is conclusive. Otherwise,
	// compare the actual bytes, including when one or both hashes are missing.
	if primary.Hash != "" && candidate.Hash != "" && primary.Hash != candidate.Hash {
		return false, nil
	}
	return FilesEqual(ctx, primaryFile, candidate.FilePath)
}

// FilesEqual compares two files byte for byte. It checks cancellation before
// opening and before each read, and leaves the first file rewound for reuse by
// callers comparing multiple candidates.
func FilesEqual(ctx context.Context, file1 *os.File, path2 string) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	file2, err := os.Open(path2)
	if err != nil {
		return false, fmt.Errorf("error opening duplicate file: %w", err)
	}
	defer file2.Close()

	const chunkSize = 4096
	buf1 := make([]byte, chunkSize)
	buf2 := make([]byte, chunkSize)

	for {
		select {
		case <-ctx.Done():
			return false, context.Canceled
		default:
		}

		n1, err1 := file1.Read(buf1)
		n2, err2 := file2.Read(buf2)
		if err1 != nil && !errors.Is(err1, io.EOF) || err2 != nil && !errors.Is(err2, io.EOF) {
			return false, fmt.Errorf("read error: %w", errors.Join(err1, err2))
		}
		if n1 != n2 || !bytes.Equal(buf1[:n1], buf2[:n2]) {
			return false, nil
		}
		if errors.Is(err1, io.EOF) && errors.Is(err2, io.EOF) {
			break
		}
	}

	if _, err := file1.Seek(0, io.SeekStart); err != nil {
		return false, fmt.Errorf("reset file %q: %w", file1.Name(), err)
	}
	return true, nil
}
