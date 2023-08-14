## confi

[![Go Reference](https://pkg.go.dev/badge/github.com/jfk9w-go/confi.svg)](https://pkg.go.dev/github.com/jfk9w-go/confi)
[![Go Report](https://goreportcard.com/badge/github.com/jfk9w-go/confi)](https://goreportcard.com/report/github.com/jfk9w-go/confi)
[![Go Coverage](https://github.com/jfk9w-go/confi/wiki/coverage.svg)](https://raw.githack.com/wiki/jfk9w-go/confi/coverage.html)
[![CodeQL](https://github.com/jfk9w-go/confi/workflows/CodeQL/badge.svg)](https://github.com/jfk9w-go/confi/actions?query=workflow%3ACodeQL)

Configuration parser for Go.

### Features

* Read and merge configuration values from environment variables, stdin and files.
* Generate JSON schema for configuration struct based on types and tags.
* Apply default values for configuration values.
* Support for JSON, YAML and Gob.

### Usage

```bash
go get github.com/jfk9w-go/confi@latest
```

**Command-line options**

| Option | Description                                                                                                                                                                      |
|---|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `--config.stdin=<codec>` | Read configuration from stdin.<br>Supported codecs: `yaml` or `yml`, `json`, `gob`.                                                                                              |
| `--config.file=<path>` | Read configuration from file.<br>Option may be used several times in order to pass multiple files.<br>Codec is resolved based on filename extension. See supported codecs above. |

**Environment variables**

Environment variables are filtered based on prefix passed to `confi.Get()` call.

A single configuration file may be specified via `<prefix>_CONFIG_FILE` environment variable.

**Priority**

When properties are specified in multiple ways (e.g. environment variable and CLI option), they have the following priority:

1. CLI options.
2. Configuration files.
3. Environment variables.

Arrays (slices) and maps are overridden as a whole.

### Example

**TODO**
