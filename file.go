// Package golden provides standard way to write tests with golden files.
package golden

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stretchr/testify/assert"
)

var DefaultHandler = &FileHandler{
	FileName:       TestNameToFilePath,
	ShouldRecreate: ParseRecreateFromEnv,
	Equal:          EqualWithDiff,
	ProcessContent: nil,
}

type FileHandler struct {
	FileName       func(T) string
	ShouldRecreate func(T) bool
	ProcessContent func(T, string) string
	Equal          func(t T, expected, actual string, msgAndArgs ...interface{}) (ok bool)
}

type T interface {
	Logf(format string, args ...any)
	Errorf(format string, args ...interface{})
	FailNow()
	Name() string
	Helper()
}

// Client is an interface that allows using http.Client or any other client that implements the Do method.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// Request sends the request and asserts that the response status code is equal to the expectedStatusCode.
// It also asserts that the response body is equal to the golden file content using EqualString.
// Example test function:
//
//	func TestAPI(t *testing.T) {
//		tests := []struct {
//			name         string
//			method       string
//			path         string
//			body         io.Reader
//			expectedCode int
//		}{
//			{
//				name:   "create user",
//				method: "POST",
//				path:   "/api/v1/user",
//				body: strings.NewReader(`{"name": "someone"}`),
//				expectedCode: 200,
//			},
//		}
//
//		for _, tt := range tests {
//			t.Run(tt.name, func(t *testing.T) {
//				req, err := http.NewRequest(tt.method, "http://127.0.0.1:8080"+tt.path, tt.body)
//				require.NoError(t, err)
//				golden.Request(t, http.DefaultClient, req, tt.expectedCode)
//			})
//		}
//	}
func Request(t T, client Client, req *http.Request, expectedStatusCode int) (*http.Response, bool) {
	return DefaultHandler.Request(t, client, req, expectedStatusCode)
}

// Assert checks the golden file content against the given data.
func Assert(t T, data string) bool {
	return DefaultHandler.Assert(t, data)
}

func (h *FileHandler) Request(t T, client Client, req *http.Request, expectedStatusCode int) (*http.Response, bool) {
	resp, err := client.Do(req)
	NoError(t, err, "client.Do failed")

	ok := true
	if resp.StatusCode != expectedStatusCode {
		ok = false
		t.Errorf("expected status code %d, got %d", expectedStatusCode, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	NoError(t, err, "reading response body failed")

	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, h.Assert(t, string(body)) && ok
}

func (h *FileHandler) Assert(t T, data string) bool {
	t.Helper()
	if h.ProcessContent != nil {
		data = h.ProcessContent(t, data)
	}
	return h.Equal(t, h.loadAndSaveFile(t, data), data)
}

func (h *FileHandler) loadAndSaveFile(t T, data string) string {
	fileName := h.FileName(t)
	if h.ShouldRecreate(t) {
		t.Logf("recreating golden file: %s", fileName)
		NoError(t, os.MkdirAll(filepath.Dir(fileName), 0o755), "failed to create testdata directory for golden file")
		NoError(t, os.WriteFile(fileName, []byte(data), 0o600), "failed to write golden file")
	}

	b, err := os.ReadFile(fileName)
	NoError(t, err, "failed to read golden file")
	return string(b)
}

// TestNameToFilePath creates file name and path for the golden file using t.Name() with following rules:
// Top level: ./testdata/{testFuncName}/{testFuncName}.golden
// Subtest:   ./testdata/{testFuncName}/{subTestName}.golden
func TestNameToFilePath(t T) string {
	split := strings.SplitN(t.Name(), "/", 2)
	mainTestName := t.Name()
	testName := t.Name()
	if len(split) == 2 {
		mainTestName = split[0]
		testName = strings.ReplaceAll(split[1], "/", "_")
	}

	return strings.ReplaceAll(filepath.Join("./testdata/", mainTestName, testName+".golden"), " ", "_")
}

// ParseRecreateFromEnv checks if the environment variable GOLDEN_FILES_RECREATE is set to true.
func ParseRecreateFromEnv(t T) bool {
	str := os.Getenv("GOLDEN_FILES_RECREATE")
	if str == "" {
		return false
	}

	overwrite, err := strconv.ParseBool(str)
	NoError(t, err, fmt.Sprintf("failed to parse GOLDEN_FILES_RECREATE env variable: '%s' to bool", str))
	return overwrite
}

func NoError(t T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Errorf("%s: %s", msg, err)
		t.FailNow()
	}
}

func EqualWithDiff(t T, expected, actual string, msgAndArgs ...interface{}) (ok bool) {
	return assert.Equal(t, expected, actual, msgAndArgs...)
}
