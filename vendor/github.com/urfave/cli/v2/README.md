cli
===

[![GoDoc](https://godoc.org/github.com/urfave/cli?status.svg)](https://pkg.go.dev/github.com/urfave/cli/v2)
[![codebeat](https://codebeat.co/badges/0a8f30aa-f975-404b-b878-5fab3ae1cc5f)](https://codebeat.co/projects/github-com-urfave-cli)
[![Go Report Card](https://goreportcard.com/badge/urfave/cli)](https://goreportcard.com/report/urfave/cli)
[![codecov](https://codecov.io/gh/urfave/cli/branch/main/graph/badge.svg)](https://codecov.io/gh/urfave/cli)

cli is a simple, fast, and fun package for building command line apps in Go. The
goal is to enable developers to write fast and distributable command line
applications in an expressive way.

## Usage Documentation

Usage documentation exists for each major version. Don't know what version you're on? You're probably using the version from the `main` branch, which is currently `v2`.

- `v2` - [./docs/v2/manual.md](./docs/v2/manual.md)
- `v1` - [./docs/v1/manual.md](./docs/v1/manual.md)

Guides for migrating to newer versions:

- `v1-to-v2` - [./docs/migrate-v1-to-v2.md](./docs/migrate-v1-to-v2.md)

## Installation

Using this package requires a working Go environment. [See the install instructions for Go](http://golang.org/doc/install.html).

Go Modules are required when using this package. [See the go blog guide on using Go Modules](https://blog.golang.org/using-go-modules).

### Using `v2` releases

```
$ go get github.com/urfave/cli/v2
```

```go
...
import (
  "github.com/urfave/cli/v2" // imports as package "cli"
)
...
```

### Using `v1` releases

```
$ go get github.com/urfave/cli
```

```go
...
import (
  "github.com/urfave/cli"
)
...
```

### Build tags

You can use the following build tags:

#### `urfave_cli_no_docs`

When set, this removes `ToMarkdown` and `ToMan` methods, so your application
won't be able to call those. This reduces the resulting binary size by about
300-400 KB (measured using Go 1.18.1 on Linux/amd64), due to less dependencies.

### GOPATH

Make sure your `PATH` includes the `$GOPATH/bin` directory so your commands can
be easily used:
```
export PATH=$PATH:$GOPATH/bin
```

### Supported platforms

cli is tested against multiple versions of Go on Linux, and against the latest
released version of Go on OS X and Windows. This project uses Github Actions for
builds. To see our currently supported go versions and platforms, look at the [./.github/workflows/cli.yml](https://github.com/urfave/cli/blob/main/.github/workflows/cli.yml).

## License

See [`LICENSE`](./LICENSE)