# pdfbox-go

A Go port of [Apache PDFBox](https://pdfbox.apache.org/), living beside the Java
source it is ported from. The Java modules stay at the repository root and are
the reference; this directory holds the Go module.

**Status: early.** The foundation layer (`pdfio`) is ported and tested; nothing
above it exists yet. See [`migration/STATUS.md`](migration/STATUS.md) for what
is done and [`migration/PLAN.md`](migration/PLAN.md) for the order the rest
follows.

## Layout

```
go/
├── go.mod              module github.com/shinguakira/pdfbox-go/go
├── pdfio/              <- org.apache.pdfbox.io      (the io module)
├── fontbox/            <- org.apache.fontbox
├── xmpbox/             <- org.apache.xmpbox
├── pdfbox/             <- org.apache.pdfbox         (cos, filter, pdmodel, ...)
├── cmd/                <- org.apache.pdfbox.tools   (command line entry points)
├── internal/           helpers with no Java counterpart
└── migration/          the porting plan, conventions, mapping and status
```

The tree mirrors the Java package structure so that any Go file can be traced
back to the Java file it came from. The full package table is
[`migration/mapping/packages.tsv`](migration/mapping/packages.tsv).

## Building

Requires Go 1.26 or later.

```bash
cd go && go build ./...
```

```bash
cd go && gofmt -l . && go vet ./... && go test ./...
```

The Go module is self-contained: building it needs no JDK and no Maven, and
`mvn` at the repository root does not see it.

## Contributing to the port

Read [`migration/conventions/java-to-go.md`](migration/conventions/java-to-go.md)
first — it is the difference between a port and 77 unrelated translations. The
short version:

- Idiomatic Go at the API boundary, faithful algorithm inside.
- `error` returns rather than exceptions, with sentinel values in each package's
  `errors.go`.
- `io.EOF` for end of input, never a `-1` sentinel.
- Every package carries its Java tests, ported alongside the code.
- Every deliberate deviation from Java behaviour is commented where it happens.

## Licence

Apache License 2.0, the same as the upstream project. See
[`../LICENSE.txt`](../LICENSE.txt).
