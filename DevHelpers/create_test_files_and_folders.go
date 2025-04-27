package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func createDirectories(baseDir string) error {
	subDirs := []string{"text_files", "image_files", "audio_files", "greek_files"}
	for _, subDir := range subDirs {
		dirPath := filepath.Join(baseDir, subDir)
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
	}
	return nil
}

func createSampleFiles(baseDir, prefix string) error {
	// Create text files
	textDir := filepath.Join(baseDir, "text_files")
	if err := os.WriteFile(filepath.Join(textDir, prefix+"-has_duplicate.txt"), []byte("This is a sample text file."), 0644); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(textDir, prefix+"-has_duplicate.txt"), filepath.Join(textDir, prefix+"-has_duplicate1.txt")); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(textDir, prefix+"-has_no_duplicate.txt"), []byte("Another unique text file."), 0644); err != nil {
		return err
	}

	// Create image files
	imageDir := filepath.Join(baseDir, "image_files")
	if err := createRandomFile(filepath.Join(imageDir, prefix+"-has_many_duplicates.jpg"), 1024); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(imageDir, prefix+"-has_many_duplicates.jpg"), filepath.Join(imageDir, prefix+"-has_many_duplicates1.jpg")); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(imageDir, prefix+"-has_many_duplicates.jpg"), filepath.Join(imageDir, prefix+"-has_many_duplicates2.jpg")); err != nil {
		return err
	}
	if err := createRandomFile(filepath.Join(imageDir, prefix+"-has_no_duplicate.png"), 2048); err != nil {
		return err
	}

	// Create audio files
	audioDir := filepath.Join(baseDir, "audio_files")
	if err := createRandomFile(filepath.Join(audioDir, prefix+"-has_duplicate.mp3"), 512); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(audioDir, prefix+"-has_duplicate.mp3"), filepath.Join(audioDir, prefix+"-has_duplicate1.mp3")); err != nil {
		return err
	}
	if err := createRandomFile(filepath.Join(audioDir, prefix+"-has_no_duplicate.wav"), 1024); err != nil {
		return err
	}

	// Create Greek files
	greekDir := filepath.Join(baseDir, "greek_files")
	if err := createRandomFile(filepath.Join(greekDir, prefix+"-έχει-αντίγραφο.txt"), 512); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(greekDir, prefix+"-έχει-αντίγραφο.txt"), filepath.Join(greekDir, prefix+"-έχει-αντίγραφο1.txt")); err != nil {
		return err
	}
	if err := createRandomFile(filepath.Join(greekDir, prefix+"-δέν-έχει-αντίγραφο.txt"), 1024); err != nil {
		return err
	}

	// Create unique files
	if err := createRandomFile(filepath.Join(baseDir, prefix+"-unique1.pdf"), 512); err != nil {
		return err
	}
	if err := createRandomFile(filepath.Join(baseDir, prefix+"-unique2.png"), 1024); err != nil {
		return err
	}
	if err := createRandomFile(filepath.Join(baseDir, prefix+"-unique3.zip"), 2048); err != nil {
		return err
	}

	// Create files with no permissions
	noAccessFile := filepath.Join(baseDir, prefix+"-no_access_file.txt")
	if err := createRandomFile(noAccessFile, 512); err != nil {
		return err
	}
	if err := os.Chmod(noAccessFile, 0); err != nil { // Remove all permissions
		return fmt.Errorf("failed to set no permissions for file %s: %w", noAccessFile, err)
	}

	noAccessFile1 := filepath.Join(baseDir, prefix+"-no_access_file1.txt")
	if err := createRandomFile(noAccessFile1, 512); err != nil {
		return err
	}
	if err := os.Chmod(noAccessFile1, 0); err != nil {
		return fmt.Errorf("failed to set no permissions for file %s: %w", noAccessFile1, err)
	}

	noAccessFile2 := filepath.Join(baseDir, prefix+"-no_access_file2.txt")
	if err := createRandomFile(noAccessFile2, 512); err != nil {
		return err
	}
	if err := os.Chmod(noAccessFile2, 0); err != nil {
		return fmt.Errorf("failed to set no permissions for file %s: %w", noAccessFile2, err)
	}

	return nil
}

func createRandomFile(filePath string, size int) error {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return fmt.Errorf("failed to generate random data: %w", err)
	}
	return os.WriteFile(filePath, data, 0644)
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file from %s to %s: %w", src, dst, err)
	}

	return nil
}

func copyAndRenameFiles(sourceDir, targetDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(sourceDir, path)
			if err != nil {
				return err
			}
			newName := fmt.Sprintf("copied_%s", filepath.Base(relPath))
			targetPath := filepath.Join(targetDir, filepath.Dir(relPath), newName)
			if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
				return err
			}
			if err := copyFile(path, targetPath); err != nil {
				return err
			}
		}
		return nil
	})
}

func main() {
	baseDir := "../test_files/source"
	baseTargetDir := "../test_files/target"

	// Create directories
	if err := createDirectories(baseDir); err != nil {
		fmt.Println("Error creating directories:", err)
		return
	}
	if err := createDirectories(baseTargetDir); err != nil {
		fmt.Println("Error creating directories:", err)
		return
	}

	// Create sample files
	if err := createSampleFiles(baseDir, "source"); err != nil {
		fmt.Println("Error creating sample files:", err)
		return
	}
	if err := createSampleFiles(baseTargetDir, "target"); err != nil {
		fmt.Println("Error creating sample files:", err)
		return
	}

	// Copy and rename files
	if err := copyAndRenameFiles(baseDir, baseTargetDir); err != nil {
		fmt.Println("Error copying and renaming files:", err)
		return
	}

	fmt.Println("Sample files created and copied successfully.")
}
