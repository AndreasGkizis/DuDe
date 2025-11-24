package unit_tests

import (
	common "DuDe-wails/internal/common"
	handlers "DuDe-wails/internal/handlers"

	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFileArguments(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	testCases := []struct {
		name         string
		inputArgs    string
		expectedArgs map[string]string
		setupDirs    []string // Directories to create for test
		Err          bool
	}{
		{
			name: "Basic Test",
			inputArgs: common.ArgFilename_sourceDir + `=[ ` + filepath.Join(tempDir, "source") + `]
					` + common.ArgFilename_targetDir + ` = ` + filepath.Join(tempDir, "target") + `]
					` + common.ArgFilename_resDir + ` = ` + filepath.Join(tempDir, "results"),
			expectedArgs: map[string]string{
				common.ArgFilename_sourceDir: filepath.Join(tempDir, "source"),
				common.ArgFilename_targetDir: filepath.Join(tempDir, "target"),
				common.ArgFilename_resDir:    filepath.Join(tempDir, "results"),
				common.ArgFilename_cacheDir:  common.Def,
			},
			setupDirs: []string{"source", "target", "results"},
			Err:       false,
		},
		{
			name: "defaults untouched Test",
			inputArgs: common.ArgFilename_sourceDir + `=` + common.Path_prefix + common.ArgFilename_sourceDir_example + common.Path_suffix + "\n" +
				common.ArgFilename_targetDir + `=` + common.Path_prefix + common.ArgFilename_targetDir_example + common.Path_suffix + "\n" +
				common.ArgFilename_resDir + `=` + common.Path_prefix + common.ArgFilename_resDir_example + common.Path_suffix,
			expectedArgs: map[string]string{
				common.ArgFilename_sourceDir: common.Def,
				common.ArgFilename_targetDir: common.Def,
				common.ArgFilename_resDir:    common.Def,
				common.ArgFilename_cacheDir:  common.Def,
			},
			setupDirs: []string{"source", "target", "results"},
			Err:       false,
		},
		{
			name: "removed examples Test",
			inputArgs: common.ArgFilename_sourceDir + `=` + common.Path_prefix + common.ArgFilename_sourceDir_example + common.Path_suffix + "\n" +
				common.ArgFilename_targetDir + `=` + common.Path_prefix + common.Path_suffix + "\n" +
				common.ArgFilename_resDir + `=` + common.Path_prefix + common.Path_suffix,
			expectedArgs: map[string]string{
				common.ArgFilename_sourceDir: common.Def,
				common.ArgFilename_targetDir: common.Def,
				common.ArgFilename_resDir:    common.Def,
				common.ArgFilename_cacheDir:  common.Def,
			},
			setupDirs: []string{"source", "target", "results"},
			Err:       false,
		},
		{
			name: "removed examples Test no braces",
			inputArgs: common.ArgFilename_sourceDir + `=` + common.Path_prefix + common.ArgFilename_sourceDir_example + common.Path_suffix + "\n" +
				common.ArgFilename_targetDir + `=` + "\n" +
				common.ArgFilename_resDir + `=`,
			expectedArgs: map[string]string{
				common.ArgFilename_sourceDir: common.Def,
				common.ArgFilename_targetDir: common.Def,
				common.ArgFilename_resDir:    common.Def,
				common.ArgFilename_cacheDir:  common.Def,
			},
			setupDirs: []string{"source", "target", "results"},
			Err:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup directories
			for _, dir := range tc.setupDirs {
				os.Mkdir(filepath.Join(tempDir, dir), 0755)
			}

			// Write test args file
			argsFilePath := filepath.Join(tempDir, common.ArgFilename)
			err := os.WriteFile(argsFilePath, []byte(tc.inputArgs), 0644)
			if err != nil {
				t.Fatalf("Failed to create test arguments file: %v", err)
			}

			// Initialize args map
			args := make(map[string]string)
			args[common.ArgFilename_sourceDir] = common.Def
			args[common.ArgFilename_targetDir] = common.Def
			args[common.ArgFilename_resDir] = common.Def
			args[common.ArgFilename_cacheDir] = common.Def

			// Call the GetFileArguments function
			result, err := handlers.GetFileArguments(argsFilePath, args)

			// Assert that the result matches the expected arguments
			assert.Equal(t, tc.expectedArgs, result)
			if tc.Err {
				assert.Equal(t, tc.Err, err)
			}
		})
	}
}

func TestGetCLIArguments(t *testing.T) {

	// Create a temporary directory for testing paths
	tempDir := t.TempDir()

	initialArgsDefault := map[string]string{
		common.ArgFilename_sourceDir:    common.Def,
		common.ArgFilename_targetDir:    common.Def,
		common.ArgFilename_resDir:       common.Def,
		common.ArgFilename_cacheDir:     common.Def,
		common.ArgFilename_paranoidMode: common.Def,
	}
	testCases := []struct {
		name        string
		args        []string
		initialArgs map[string]string
		expected    map[string]string
	}{
		{
			name:        "No CLI Arguments",
			args:        []string{"test_program"},
			initialArgs: initialArgsDefault,
			expected: map[string]string{
				common.ArgFilename_sourceDir:    common.Def,
				common.ArgFilename_targetDir:    common.Def,
				common.ArgFilename_resDir:       common.Def,
				common.ArgFilename_cacheDir:     common.Def,
				common.ArgFilename_paranoidMode: common.Def, // due to nothing set this remains untoucheds
			},
		},
		{
			name: "INVALID CLI Arguments",
			args: []string{"test_program",
				"-" + common.ParanoidFlag + "=  false",
			},
			initialArgs: initialArgsDefault,
			expected: map[string]string{
				common.ArgFilename_sourceDir:    common.Def,
				common.ArgFilename_targetDir:    common.Def,
				common.ArgFilename_resDir:       common.Def,
				common.ArgFilename_cacheDir:     common.Def,
				common.ArgFilename_paranoidMode: common.Def,
			},
		},
		{
			name: "Boolean Args set to false",
			args: []string{"test_program",
				"-" + common.ParanoidFlag + "=false",
			},
			initialArgs: initialArgsDefault,
			expected: map[string]string{
				common.ArgFilename_sourceDir:    common.Def,
				common.ArgFilename_targetDir:    common.Def,
				common.ArgFilename_resDir:       common.Def,
				common.ArgFilename_cacheDir:     common.Def,
				common.ArgFilename_paranoidMode: "false",
			},
		},
		{
			name: "Source and Target Set",
			args: []string{"test_program",
				"-" + common.SourceFlag, filepath.Join(tempDir, "source_cli"),
				"-" + common.TargetFlag, filepath.Join(tempDir, "target_cli"),
			},
			initialArgs: initialArgsDefault,
			expected: map[string]string{
				common.ArgFilename_sourceDir:    filepath.Join(tempDir, "source_cli"),
				common.ArgFilename_targetDir:    filepath.Join(tempDir, "target_cli"),
				common.ArgFilename_resDir:       common.Def,
				common.ArgFilename_cacheDir:     common.Def,
				common.ArgFilename_paranoidMode: "false", // no flag = false
			},
		},
		{
			name: "All Flags Set",
			args: []string{
				"test_program",
				"-" + common.SourceFlag, filepath.Join(tempDir, "source_cli"),
				"-" + common.ResultDirFlag, filepath.Join(tempDir, "results_cli"),
				"-" + common.TargetFlag, filepath.Join(tempDir, "target_cli"),
				"-" + common.MemDirFlag, filepath.Join(tempDir, "cache_cli"),
				"-" + common.ParanoidFlag, // even the presense of the flag makes it true
			},
			initialArgs: initialArgsDefault,
			expected: map[string]string{
				common.ArgFilename_sourceDir:    filepath.Join(tempDir, "source_cli"),
				common.ArgFilename_targetDir:    filepath.Join(tempDir, "target_cli"),
				common.ArgFilename_resDir:       filepath.Join(tempDir, "results_cli"),
				common.ArgFilename_cacheDir:     filepath.Join(tempDir, "cache_cli"),
				common.ArgFilename_paranoidMode: "true",
			},
		},
		{
			name: "Long Flags Set",
			args: []string{
				"test_program",
				"--" + common.SourceFlag_long, filepath.Join(tempDir, "long_source"),
				"--" + common.TargetFlag_long, filepath.Join(tempDir, "long_target"),
				"--" + common.ResultDirFlag_long, filepath.Join(tempDir, "long_res"),
				"--" + common.MemDirFlag_long, filepath.Join(tempDir, "long_cache"),
				"--" + common.ParanoidFlag_long, // even the presense of the flag makes it true
			},
			initialArgs: initialArgsDefault,
			expected: map[string]string{
				common.ArgFilename_sourceDir:    filepath.Join(tempDir, "long_source"),
				common.ArgFilename_targetDir:    filepath.Join(tempDir, "long_target"),
				common.ArgFilename_resDir:       filepath.Join(tempDir, "long_res"),
				common.ArgFilename_cacheDir:     filepath.Join(tempDir, "long_cache"),
				common.ArgFilename_paranoidMode: "true",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Reset the flag set at the beginning of each subtest
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			// Save the original os.Args
			originalArgs := os.Args
			defer func() {
				// Restore os.Args after the test
				os.Args = originalArgs
			}()

			// Set up the test os.Args
			os.Args = tc.args

			// Call the function
			result := handlers.GetCLIArgs(tc.initialArgs)

			// Assert the results
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic Test",
			input:    `C:\\bla\blu`,
			expected: `C:\\bla\blu`,
		},
		{
			name:     "brackets",
			input:    `[/home]`,
			expected: `/home`,
		},
		{
			name:     "one bracket",
			input:    `[/home`,
			expected: `/home`,
		},
		{
			name:     "whitespace",
			input:    ` /home`,
			expected: `/home`,
		},
		{
			name:     "whitespace & brackets",
			input:    `[ /home`,
			expected: `/home`,
		},
		{
			name:     "whitespace & brackets",
			input:    `[ /home `,
			expected: `/home`,
		},
		{
			name:     "whitespace & brackets",
			input:    `[ /home   ]`,
			expected: `/home`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := handlers.SanitizeInput(tc.input)

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}
