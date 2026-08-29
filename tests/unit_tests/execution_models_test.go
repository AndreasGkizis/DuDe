package unit_tests

import (
	"DuDe/internal/models"
	"encoding/json"
	"testing"
)

func TestExecutionParamsPreservesExplicitMode(t *testing.T) {
	tests := []struct {
		name string
		mode models.ExecutionMode
	}{
		{
			name: "explicit duplicate mode",
			mode: models.ExecutionModeDuplicates,
		},
		{
			name: "find missing content mode",
			mode: models.ExecutionModeFindMissingContent,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := models.ExecutionParams{Mode: test.mode}

			payload, err := json.Marshal(params)
			if err != nil {
				t.Fatalf("marshal execution params: %v", err)
			}

			var decoded models.ExecutionParams
			if err := json.Unmarshal(payload, &decoded); err != nil {
				t.Fatalf("unmarshal execution params: %v", err)
			}
			if decoded.Mode != test.mode {
				t.Fatalf("decoded Mode = %d, want explicit mode %d", decoded.Mode, test.mode)
			}
		})
	}
}

func TestExecutionParamsFieldsUseMatchingJSONNames(t *testing.T) {
	params := models.ExecutionParams{
		Mode:                 models.ExecutionModeFindMissingContent,
		FindMissingSourceDir: "/source",
		FindMissingTargetDir: "/target",
		WorkerCount:          3,
		BufferSizeBytes:      4096,
	}

	payload, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal execution params: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal execution params: %v", err)
	}

	assertJSONValue(t, decoded, "mode", float64(models.ExecutionModeFindMissingContent))
	assertJSONValue(t, decoded, "findMissingSourceDir", "/source")
	assertJSONValue(t, decoded, "findMissingTargetDir", "/target")
	assertJSONValue(t, decoded, "workerCount", float64(3))
	assertJSONValue(t, decoded, "bufferSizeBytes", float64(4096))

	for _, legacyName := range []string{"findmissingsourceDir", "findmissingtargetDir", "cpus", "bufSize"} {
		if _, ok := decoded[legacyName]; ok {
			t.Fatalf("legacy JSON field %q should not be present", legacyName)
		}
	}
}

func TestMissingContentResultUsesDedicatedMissingSourceFiles(t *testing.T) {
	result := models.MissingContentResult{
		TotalSourceFiles: 2,
		MissingFiles: []models.MissingSourceFile{
			{
				Filename: "missing.txt",
				FullPath: "/source/missing.txt",
			},
		},
	}

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal missing-content result: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal missing-content result: %v", err)
	}

	assertJSONValue(t, decoded, "totalSourceFiles", float64(2))
	if _, ok := decoded["missingFiles"]; !ok {
		t.Fatal("JSON field \"missingFiles\" is missing")
	}
	if _, ok := decoded["reportPath"]; ok {
		t.Fatal("empty reportPath should be omitted")
	}
}

func TestDuplicateResultUsesExistingFileHashGroups(t *testing.T) {
	result := models.DuplicateResult{
		Groups: []models.FileHash{
			{
				FileName: "original.txt",
				FilePath: "/files/original.txt",
				DuplicatesFound: []models.FileHash{
					{
						FileName: "copy.txt",
						FilePath: "/files/copy.txt",
					},
				},
			},
		},
	}

	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal duplicate result: %v", err)
	}

	var decoded models.DuplicateResult
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal duplicate result: %v", err)
	}
	if len(decoded.Groups) != 1 {
		t.Fatalf("decoded group count = %d, want 1", len(decoded.Groups))
	}
	if len(decoded.Groups[0].DuplicatesFound) != 1 {
		t.Fatalf("decoded duplicate count = %d, want 1", len(decoded.Groups[0].DuplicatesFound))
	}
}

func assertJSONValue(t *testing.T, values map[string]any, key string, want any) {
	t.Helper()
	got, ok := values[key]
	if !ok {
		t.Fatalf("JSON field %q is missing", key)
	}
	if got != want {
		t.Fatalf("JSON field %q = %v, want %v", key, got, want)
	}
}
