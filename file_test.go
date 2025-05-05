package golden_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-tstr/golden"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type test struct {
	name         string
	mt           mockT
	data         string
	expectedData string
	expectedPath string
}

func TestFile(t *testing.T) {
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll("./testdata/TestFunc"), "failed to remove testdata") })
	tests := []test{
		{
			name:         "sub test",
			mt:           mockT{name: "TestFunc/subtest"},
			data:         "other data",
			expectedData: "other data",
			expectedPath: "./testdata/TestFunc/subtest.golden",
		},
		{
			name:         "second sub test",
			mt:           mockT{name: "TestFunc/subtest_other"},
			data:         "yet another data",
			expectedData: "yet another data",
			expectedPath: "./testdata/TestFunc/subtest_other.golden",
		},
		{
			name:         "nested sub test",
			mt:           mockT{name: "TestFunc/subtest/nested"},
			data:         "nested data",
			expectedData: "nested data",
			expectedPath: "./testdata/TestFunc/subtest_nested.golden",
		},
		{
			name:         "parent of sub test",
			mt:           mockT{name: "TestFunc"},
			data:         "parent data",
			expectedData: "parent data",
			expectedPath: "./testdata/TestFunc/TestFunc.golden",
		},
	}

	fh := &golden.FileHandler{
		FileName:       golden.TestNameToFilePath,
		ShouldRecreate: golden.ParseRecreateFromEnv,
		Equal:          golden.EqualWithDiff,
		ProcessContent: nil,
	}

	t.Run("create", func(t *testing.T) {
		t.Setenv("GOLDEN_FILES_RECREATE", "true")
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mt := &tt.mt
				fh.Assert(mt, tt.data)
				assertResult(t, tt, mt)
			})
		}
	})

	t.Run("read only", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				mt := &tt.mt
				fh.Assert(mt, tt.data)
				assertResult(t, tt, mt)
			})
		}
	})

	const suffix = " overwrite"

	t.Run("overwrite", func(t *testing.T) {
		t.Setenv("GOLDEN_FILES_RECREATE", "true")
		for _, tt := range tests {
			tt.data += suffix
			tt.expectedData += suffix
			t.Run(tt.name, func(t *testing.T) {
				mt := &tt.mt
				fh.Assert(mt, tt.data)
				assertResult(t, tt, mt)
			})
		}
	})

	t.Run("read only after overwrite", func(t *testing.T) {
		for _, tt := range tests {
			tt.data += suffix
			tt.expectedData += suffix
			t.Run(tt.name, func(t *testing.T) {
				mt := &tt.mt
				fh.Assert(mt, tt.data)
				assertResult(t, tt, mt)
			})
		}
	})
}

func TestFolderDoesNotExist(t *testing.T) {
	mt := mockT{name: "TestDirFail"}
	golden.Assert(&mt, "data")
	assert.True(t, mt.failed)
	assert.Contains(t, mt.msg, "open testdata/TestDirFail/TestDirFail.golden: no such file or directory")
	assert.NoDirExists(t, "./testdata/TestDirFail")
}

func TestEqual(t *testing.T) {
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll("./testdata/TestSomeString"), "failed to remove testdata") })

	data := "some string"
	assert.NoError(t, os.MkdirAll("./testdata/TestSomeString", 0o755))
	assert.NoError(t, os.WriteFile("./testdata/TestSomeString/TestSomeString.golden", []byte(data), 0o600))

	mt := mockT{name: "TestSomeString"}
	got := golden.Assert(&mt, data)
	assert.True(t, got)
	assert.Empty(t, mt.msg)
	assert.False(t, mt.failed)
}

func TestEqual_No_Match(t *testing.T) {
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll("./testdata/TestSomeOtherString"), "failed to remove testdata") })

	data := []byte("some string")
	assert.NoError(t, os.MkdirAll("./testdata/TestSomeOtherString", 0o755))
	assert.NoError(t, os.WriteFile("./testdata/TestSomeOtherString/TestSomeOtherString.golden", data, 0o600))

	mt := mockT{name: "TestSomeOtherString"}
	got := golden.Assert(&mt, "other string")
	assert.False(t, got)
	assert.Contains(t, mt.msg, "Not equal:")
	assert.True(t, mt.failed)
}

func TestRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)
	golden.Request(t, http.DefaultClient, req, http.StatusBadRequest)
}

func TestProcessJSON(t *testing.T) {
	const (
		data1 = `{"name": "someone", "age": 41, "active": true}`
		data2 = `{"name": "someone", "age": 42, "active": true}`
	)

	fh := &golden.FileHandler{
		FileName:       golden.TestNameToFilePath,
		ShouldRecreate: func(t golden.T) bool { return true },
		Equal:          golden.EqualWithDiff,
		ProcessContent: golden.PrettyJSON,
	}

	t.Cleanup(func() { assert.NoError(t, os.RemoveAll("./testdata/TestSomeJSON"), "failed to remove testdata") })
	mt1 := mockT{name: "TestSomeJSON"}
	assert.True(t, fh.Assert(&mt1, data1))
	assert.Empty(t, mt1.msg)
	assert.False(t, mt1.failed)

	fh.ShouldRecreate = func(t golden.T) bool { return false }
	mt2 := mockT{name: "TestSomeJSON"}
	assert.True(t, fh.Assert(&mt2, data1))
	assert.Empty(t, mt2.msg)
	assert.False(t, mt2.failed)

	mt3 := mockT{name: "TestSomeJSON"}
	assert.False(t, fh.Assert(&mt3, data2))
	assert.Contains(t, mt3.msg, `-  "age": 41,`)
	assert.Contains(t, mt3.msg, `+  "age": 42,`)
	assert.True(t, mt3.failed)
}

func assertResult(t *testing.T, tt test, mt *mockT) {
	t.Helper()
	assert.Empty(t, mt.msg)
	assert.False(t, mt.failed)
	assert.FileExists(t, tt.expectedPath)
	b, err := os.ReadFile(tt.expectedPath)
	require.NoError(t, err)
	assert.Equal(t, tt.expectedData, string(b))
}

type mockT struct {
	name   string
	failed bool
	msg    string
}

func (m *mockT) Name() string                       { return m.name }
func (m *mockT) Logf(f string, args ...interface{}) { fmt.Printf(f, args...) }
func (m *mockT) FailNow()                           { m.failed = true }
func (m *mockT) Helper()                            {}
func (m *mockT) Errorf(f string, args ...interface{}) {
	m.failed = true
	m.msg += "\n" + fmt.Sprintf(f, args...)
}
