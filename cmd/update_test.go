package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/jarcoal/httpmock"
	"github.com/spf13/afero"
)

// Helper function to set up HTTP mock responses for version checking and updates
func setupMockResponses(latestVersion, assetVersion string, withUpdate bool) {
	httpmock.RegisterResponder("GET", "https://github.com/marianozunino/sdm-ui/releases/latest",
		func(req *http.Request) (*http.Response, error) {
			resp := httpmock.NewStringResponse(302, "")
			resp.Header.Set("Location", fmt.Sprintf("https://github.com/marianozunino/sdm-ui/releases/tag/v%s", latestVersion))
			resp.Request = req
			return resp, nil
		})

	httpmock.RegisterResponder("GET", fmt.Sprintf("https://github.com/marianozunino/sdm-ui/releases/tag/v%s", latestVersion),
		httpmock.NewStringResponder(200, ""))

	if withUpdate {
		assetName := getAssetName()
		downloadURL := fmt.Sprintf("https://github.com/marianozunino/sdm-ui/releases/download/v%s/%s", assetVersion, assetName)
		mockTarGz := createMockTarGzBinary()
		httpmock.RegisterResponder("GET", downloadURL,
			httpmock.NewBytesResponder(200, mockTarGz))
	}
}

// createMockTarGzBinary creates a mock tar.gz file containing a single executable file.
func createMockTarGzBinary() []byte {
	var buf bytes.Buffer

	// Create a gzip writer
	gzipWriter := gzip.NewWriter(&buf)
	defer gzipWriter.Close()

	// Create a tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Define the file to add to the tar.gz
	execFileName := "sdm-ui"
	execFileContent := []byte("mock binary data")

	// Create a tar header for the file
	header := &tar.Header{
		Name: execFileName,
		Size: int64(len(execFileContent)),
		Mode: 0o755, // Executable permissions
	}

	// Write the header to the tar
	if err := tarWriter.WriteHeader(header); err != nil {
		panic(err) // Handle error more gracefully in production code
	}

	// Write the file content to the tar
	if _, err := tarWriter.Write(execFileContent); err != nil {
		panic(err) // Handle error more gracefully in production code
	}

	// Ensure that the gzip writer flushes any buffered data
	if err := gzipWriter.Close(); err != nil {
		panic(err) // Handle error more gracefully in production code
	}

	return buf.Bytes()
}

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		current     string
		latest      string
		needsUpdate bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "2.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.1.0", "1.0.0", false},
		{"2.0.0", "1.9.9", false},
		{"1.0.0-alpha", "1.0.0", true},
		{"1.0.0", "1.0.1-alpha", true},
		{"1.0.0-beta", "1.0.0-alpha", false},
		{"v1.0.0", "1.0.1", true},
		{"1.0.0", "v1.0.1", true},
		{"v1.0.0", "v1.0.0", false},
	}

	for _, test := range tests {
		current, err := parseVersion(test.current)
		if err != nil {
			t.Errorf("Error parsing current version %s: %v", test.current, err)
			continue
		}

		latest, err := parseVersion(test.latest)
		if err != nil {
			t.Errorf("Error parsing latest version %s: %v", test.latest, err)
			continue
		}

		if result := latest.GreaterThan(current); result != test.needsUpdate {
			t.Errorf("Version comparison failed for current: %s, latest: %s. Expected needsUpdate: %v, got: %v",
				test.current, test.latest, test.needsUpdate, result)
		}
	}
}

// Helper function to parse semantic versions
func parseVersion(versionStr string) (*semver.Version, error) {
	version, err := semver.NewVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("error parsing version: %v", err)
	}
	return version, nil
}

// Modified verifyHTTPCalls function to accept `t *testing.T`
func verifyHTTPCalls(t *testing.T, expectUpdate bool, latestVersion string) {
	info := httpmock.GetCallCountInfo()

	// Verify latest release check
	if info["GET https://github.com/marianozunino/sdm-ui/releases/latest"] != 1 {
		t.Error("Expected one call to check the latest release")
	}

	// If update expected, verify download URL call
	if expectUpdate {
		assetName := getAssetName()
		downloadURL := fmt.Sprintf("https://github.com/marianozunino/sdm-ui/releases/download/v%s/%s", latestVersion, assetName)
		if info["GET "+downloadURL] != 1 {
			t.Error("Expected one call to download the update")
		}
	}
}

func TestRunSelfUpdate(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	fs := afero.NewMemMapFs()
	executablePath := "/path/to/sdm-ui"

	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		expectUpdate   bool
		mockResponses  func()
		expectedError  string
	}{
		{
			name:           "No update available",
			currentVersion: "1.0.0",
			latestVersion:  "1.0.0",
			expectUpdate:   false,
			mockResponses:  func() { setupMockResponses("1.0.0", "1.0.0", false) },
		},
		{
			name:           "Update available",
			currentVersion: "1.0.0",
			latestVersion:  "1.1.0",
			expectUpdate:   true,
			mockResponses:  func() { setupMockResponses("1.1.0", "1.1.0", true) },
		},
		{
			name:           "Invalid current version",
			currentVersion: "invalid",
			latestVersion:  "1.0.0",
			expectUpdate:   false,
			mockResponses:  func() { setupMockResponses("1.0.0", "1.0.0", false) },
			expectedError:  "error parsing current version",
		},
		{
			name:           "Invalid latest version",
			currentVersion: "1.0.0",
			latestVersion:  "invalid",
			expectUpdate:   false,
			mockResponses: func() {
				httpmock.RegisterResponder("GET", "https://github.com/marianozunino/sdm-ui/releases/latest",
					httpmock.NewStringResponder(200, "https://github.com/marianozunino/sdm-ui/releases/tag/invalid"))
			},
			expectedError: "error parsing latest version",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Reset()
			tc.mockResponses()

			Version = tc.currentVersion
			err := runSelfUpdate(http.DefaultClient, fs, executablePath)

			if tc.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
					t.Errorf("Expected error containing '%s', got '%v'", tc.expectedError, err)
				}
			} else {
				if tc.expectUpdate && err != nil {
					t.Errorf("Expected successful update, got error: %v", err)
				} else if !tc.expectUpdate && err != nil {
					t.Errorf("Expected no update, got error: %v", err)
				}
			}

			// Pass t into verifyHTTPCalls
			verifyHTTPCalls(t, tc.expectUpdate, tc.latestVersion)
		})
	}
}

func TestExtractTarGz(t *testing.T) {
	extractPath := "/tmp/test-extract603698108"

	// Add this to check the contents
	files, err := os.ReadDir(extractPath)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}
	for _, file := range files {
		t.Logf("Found file: %s", file.Name())
	}

	expectedFile := "sdm-ui"
	if _, err := os.Stat(filepath.Join(extractPath, expectedFile)); os.IsNotExist(err) {
		t.Fatalf("Expected extracted file at %s, but it does not exist", filepath.Join(extractPath, expectedFile))
	}
}
