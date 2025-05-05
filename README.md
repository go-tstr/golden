[![Go Reference](https://pkg.go.dev/badge/github.com/go-tstr/golden.svg)](https://pkg.go.dev/github.com/go-tstr/golden) [![codecov](https://codecov.io/github/go-tstr/golden/graph/badge.svg?token=H3u7Ui9PfC)](https://codecov.io/github/go-tstr/golden) ![main](https://github.com/go-tstr/golden/actions/workflows/go.yaml/badge.svg?branch=main)

# golden

Golden file testing is a technique where the expected output of a test is

- generated automatically from test
- stored in a separate file (the "golden file")
- verified to never change unless explicitly regerated

During testing, the actual output is compared against the contents of the golden file. If the outputs match, the test passes; if not, the test fails. This approach is especially useful for testing complex outputs such as JSON, HTML, or other structured data, as it makes it easy to review and update expected outputs.
