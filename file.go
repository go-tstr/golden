// Package golden provides standard way to write tests with golden files.
package golden

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var overwrite, _ = strconv.ParseBool(os.Getenv("OVERWRITE_GOLDEN_FILES"))

type T interface {
	Logf(format string, args ...any)
	Errorf(format string, args ...interface{})
	FailNow()
	Name() string
	Helper()
}

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
	t.Helper()
	resp, err := client.Do(req)
	require.NoError(t, err, "client.Do failed")

	ok := assert.Equal(t, expectedStatusCode, resp.StatusCode, "unexpected status code")
	body, err := io.ReadAll(resp.Body)
	ok = assert.NoError(t, err, "reading response body failed") && ok
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, Equal(t, string(body)) && ok
}

// Equal asserts that the golden file content is equal to the data in string format.
func Equal(t T, data string) bool {
	t.Helper()
	return assert.Equal(t, File(t, data), string(data))
}

// File returns the golden file content for the test.
// If OVERWRITE_GOLDEN_FILES env is set to true, the golden file will be created with the content of the data.
// OVERWRITE_GOLDEN_FILES is read only once at the start of the test and it's value is not updated.
// Depending of the test structure the golden file and it's directories arew created in
// ./testdata/{testFuncName}/{subTestName}.golden or ./testdata/{testFuncName}/{testFuncName}.golden.
func File(t T, data string) string {
	t.Helper()
	return file(t, data, overwrite)
}

func file(t T, data string, recreate bool) string {
	t.Helper()
	split := strings.SplitN(t.Name(), "/", 2)
	mainTestName := t.Name()
	testName := t.Name()
	if len(split) == 2 {
		mainTestName = split[0]
		testName = strings.ReplaceAll(split[1], "/", "_")
	}

	folderName := fmt.Sprintf("./testdata/%s", mainTestName)
	fileName := strings.ReplaceAll(fmt.Sprintf("%s/%s.golden", folderName, testName), " ", "_")
	if recreate {
		t.Logf("recreating golden file: %s", fileName)
		require.NoError(t, os.MkdirAll(folderName, 0o755), "failed to create testdata directory for golden file")
		require.NoError(t, os.WriteFile(fileName, []byte(data), 0o600), "failed to write golden file")
	}

	b, err := os.ReadFile(fileName)
	require.NoError(t, err, "failed to read golden file")
	return string(b)
}
