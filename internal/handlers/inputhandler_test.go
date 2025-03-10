package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestGetFileArguments(t *testing.T) {
// 	// Create a temporary directory for testing
// 	tempDir := t.TempDir()

// 	testCases := []struct {
// 		name         string
// 		inputArgs    string
// 		expectedArgs map[string]string
// 		setupDirs    []string // Directories to create for test
// 		Err          bool
// 	}{
// 		{
// 			name: "Basic Test",
// 			inputArgs: common.ArgFilename_sourceDir + `=[ ` + tempDir + `/source]
// 		` + common.ArgFilename_targetDir + ` = ` + tempDir + `/target]
// 		` + common.ArgFilename_resDir + ` = [` + tempDir + `/results`,
// 			expectedArgs: map[string]string{
// 				common.ArgFilename_sourceDir: tempDir + "/source",
// 				common.ArgFilename_targetDir: tempDir + "/target",
// 				common.ArgFilename_resDir:    tempDir + "/results",
// 				common.ArgFilename_cacheDir:  common.Def,
// 				common.ArgFilename_Dbg:       common.Def,
// 				common.ArgFilename_Mode:      common.Def,
// 			},
// 			setupDirs: []string{"source", "target", "results"},
// 			Err:       false,
// 		},
// 		{
// 			name: "Basic Test with NO brackets",
// 			inputArgs: common.ArgFilename_sourceDir + `= ` + tempDir + `/source
// 		` + common.ArgFilename_targetDir + ` = ` + tempDir + `/target
// 		` + common.ArgFilename_resDir + ` = ` + tempDir + `/results`,
// 			expectedArgs: map[string]string{
// 				common.ArgFilename_sourceDir: tempDir + "/source",
// 				common.ArgFilename_targetDir: tempDir + "/target",
// 				common.ArgFilename_resDir:    tempDir + "/results",
// 				common.ArgFilename_cacheDir:  common.Def,
// 				common.ArgFilename_Dbg:       common.Def,
// 				common.ArgFilename_Mode:      common.Def,
// 			},
// 			setupDirs: []string{"source", "target", "results"},
// 			Err:       false,
// 		},
// 		{
// 			name: "Basic Test with MIXED brackets",
// 			inputArgs: common.ArgFilename_sourceDir + `= ` + tempDir + `/source
// 		` + common.ArgFilename_targetDir + ` = [` + tempDir + `/target
// 		` + common.ArgFilename_resDir + ` = ` + tempDir + `/results]`,
// 			expectedArgs: map[string]string{
// 				common.ArgFilename_sourceDir: tempDir + "/source",
// 				common.ArgFilename_targetDir: tempDir + "/target",
// 				common.ArgFilename_resDir:    tempDir + "/results",
// 				common.ArgFilename_cacheDir:  common.Def,
// 				common.ArgFilename_Dbg:       common.Def,
// 				common.ArgFilename_Mode:      common.Def,
// 			},
// 			setupDirs: []string{"source", "target", "results"},
// 			Err:       false,
// 		},
// 		{
// 			name: "Basic Test with MIXED and whitespaces brackets",
// 			inputArgs: common.ArgFilename_sourceDir + `= ` + tempDir + `/source
// 		` + common.ArgFilename_targetDir + ` =[ ` + tempDir + `/target
// 		` + common.ArgFilename_resDir + ` = ` + tempDir + `/results        ]           `,
// 			expectedArgs: map[string]string{
// 				common.ArgFilename_sourceDir: tempDir + "/source",
// 				common.ArgFilename_targetDir: tempDir + "/target",
// 				common.ArgFilename_resDir:    tempDir + "/results",
// 				common.ArgFilename_cacheDir:  common.Def,
// 				common.ArgFilename_Dbg:       common.Def,
// 				common.ArgFilename_Mode:      common.Def,
// 			},
// 			setupDirs: []string{"source", "target", "results"},
// 			Err:       false,
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			// Setup directories
// 			for _, dir := range tc.setupDirs {
// 				os.Mkdir(filepath.Join(tempDir, dir), 0755)
// 			}

// 			// Write test args file
// 			argsFilePath := filepath.Join(tempDir, common.ArgFilename)
// 			err := os.WriteFile(argsFilePath, []byte(tc.inputArgs), 0644)
// 			if err != nil {
// 				t.Fatalf("Failed to create test arguments file: %v", err)
// 			}

// 			// Initialize args map
// 			args := make(map[string]string)
// 			args[common.ArgFilename_sourceDir] = common.Def
// 			args[common.ArgFilename_targetDir] = common.Def
// 			args[common.ArgFilename_resDir] = common.Def
// 			args[common.ArgFilename_cacheDir] = common.Def
// 			args[common.ArgFilename_Dbg] = common.Def
// 			args[common.ArgFilename_Mode] = common.Def

// 			// Call the getFileArguments function
// 			result, err := getFileArguments(argsFilePath, args)

//				// Assert that the result matches the expected arguments
//				assert.Equal(t, tc.expectedArgs, result)
//				if tc.Err {
//					assert.Equal(t, tc.Err, err)
//				}
//			})
//		}
//	}
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeInput(tc.input)

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}
