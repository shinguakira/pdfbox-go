# pdfbox-go

A Go port of [Apache PDFBox](https://pdfbox.apache.org/), living beside the Java
source it is ported from. The Java modules stay at the repository root and are
the reference; this directory holds the Go module.

**Status.** Reading, writing, merging, form handling, text extraction and
rendering are in, and the command-line tool is `go/cmd/pdfbox`. What is
deliberately absent, and why, is per package in
[`migration/STATUS.md`](migration/STATUS.md);
[`migration/PLAN.md`](migration/PLAN.md) is the order the work was taken in.

## Layout

```
go/
├── go.mod              module github.com/shinguakira/pdfbox-go/go
├── pdfio/              <- org.apache.pdfbox.io      (the io module)
├── fontbox/            <- org.apache.fontbox
├── xmpbox/             <- org.apache.xmpbox
├── pdfbox/             <- org.apache.pdfbox         (cos, filter, pdmodel, ...)
├── tools/              <- org.apache.pdfbox.tools   (the commands themselves)
├── awt/                <- java.awt                  (Color, geom, image)
├── javatext/           <- java.text                 (Bidi)
├── w3c/                <- org.w3c.dom               (a reading DOM)
├── cmd/                the binaries: pdfbox, and corpus and bench for migration
├── internal/           helpers with no Java counterpart
└── migration/          the porting plan, conventions, mapping and status
```

The PDFBox modules mirror the Java package structure, so that any Go file can be
traced back to the Java file it came from; the full package table is
[`migration/mapping/packages.tsv`](migration/mapping/packages.tsv). The three
JDK directories are there because the Java being ported uses those classes and
Go has no equivalent — each says so in its package doc comment.

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
first — it is the difference between a port and as many unrelated translations
as there are packages. It is where the naming, the error convention, the class
translations and the recording of a deliberate deviation are written down.
[`migration/conventions/tdd.md`](migration/conventions/tdd.md) is the other half:
the Java test is ported before the Go implementation exists, and its assertion
values are copied from the Java rather than read off the Go.

## Licence

Apache License 2.0, the same as the upstream project. See
[`../LICENSE.txt`](../LICENSE.txt).
