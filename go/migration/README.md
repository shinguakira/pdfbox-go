# migration/

Everything about *porting* PDFBox from Java to Go lives here. No Go source: the
directory carries no `.go` files on purpose, so `go build ./...` never sees it.

## What "port" means here

Every line of Go is newly written — none of it is generated, translated by a
tool, or bridged to a JVM at runtime. In that sense it is a full rewrite.

What is *not* rewritten is the design. The algorithms, the control flow, the
field-by-field state, and the format quirks are carried over from the Java, with
the original open alongside. A port is close to 100% new source and close to 0%
new design.

That is why the ported Java tests still apply: they exercise the algorithm, not
the syntax. A PDFBOX-numbered regression test written against Java in 2021 still
catches the same bug in Go, because the bug lived in the arithmetic and the
arithmetic came across.

Three things this is not:

- **A bytecode bridge** — the PDFBox .NET build recompiled Java bytecode to .NET
  IL via IKVM, wrote no new source, and exposed the Java API to callers. See
  [`conventions/prior-art.md`](conventions/prior-art.md).
- **A reimplementation from the PDF specification** — that is what pdfcpu and
  PDFsharp are. They owe PDFBox nothing and share none of its behaviour.
- **A redesign** — where the Java shape is awkward in Go it gets translated per
  [`conventions/java-to-go.md`](conventions/java-to-go.md), but the deviation is
  recorded at the point it happens. Improving on PDFBox is not the goal;
  matching it is.

| File | What it is |
| --- | --- |
| [`PLAN.md`](PLAN.md) | The porting plan — capability slices, what each needs, and in what order |
| [`BRANCHING.md`](BRANCHING.md) | Migration flow, branch roles, slice dependency graph. Carries the scope rule: no upstream sync, no PRs to Apache |
| [`mapping/modules.md`](mapping/modules.md) | What each of PDFBox's twelve Maven modules does, and how they depend on each other |
| [`STATUS.md`](STATUS.md) | Per-package progress, ported tests, and the deviations from Java recorded so far |
| [`conventions/tdd.md`](conventions/tdd.md) | **Test-driven porting — the Java test is ported before the Go implementation exists.** The rule the port runs on |
| [`conventions/java-to-go.md`](conventions/java-to-go.md) | How Java constructs are translated. Read this before porting anything |
| [`JAVA-BUGS.md`](JAVA-BUGS.md) | Java bugs found while porting. Recorded, never fixed — the port reproduces every one |
| [`conventions/prior-art.md`](conventions/prior-art.md) | How PDFBox was ported before (PdfPig in C#, .NET via IKVM), what carries over to Go and what does not |
| [`mapping/packages.tsv`](mapping/packages.tsv) | Java package to Go package. Hand maintained |
| [`mapping/inventory.tsv`](mapping/inventory.tsv) | Generated: files and lines per Java package, with the Go package each maps to |
| [`scripts/inventory.ps1`](scripts/inventory.ps1) | Regenerates `inventory.tsv` from the Java tree |

## Porting a package

1. Read [`conventions/tdd.md`](conventions/tdd.md) and
   [`conventions/java-to-go.md`](conventions/java-to-go.md). The first governs
   the order the work is done in; the second keeps 81 packages reading as one
   library.
2. Check the package has a row in [`mapping/packages.tsv`](mapping/packages.tsv);
   add one if not.
3. **Port that package's Java tests first, before the implementation exists.**
   Copy assertion values verbatim from the Java — never recompute them from your
   own code. Then port the implementation until they pass, then refactor to Go
   idiom with the tests green.
4. Comment every deliberate deviation from the Java behaviour at the point of
   difference, saying what Java does and why the port differs.
5. Update the package's row in [`STATUS.md`](STATUS.md) in the same commit. A
   package with implementation and no ported tests is recorded as *unverified*,
   not as partially done.

A package is done when it builds, `go vet` is clean, and its ported tests pass:

```bash
cd go && gofmt -l . && go vet ./... && go test ./...
```

## Checking a port against the running Java

When a ported package disagrees with a real PDF and the Java source does not
explain why, run the Java and compare. This is the only porting methodology
PdfPig — the C# port of PDFBox — has ever written down, and they published it as
a debugging practice rather than a document: clone PDFBox, put a `main` into
`CustomGraphicsStreamEngine`, point it at the problem file, and step through it
in a debugger. PDFBox is the oracle; the specification is the tiebreaker only
when PDFBox itself is wrong.

We are better placed than they were, because the Java is already here:

- [`examples/src/main/java/org/apache/pdfbox/examples/rendering/CustomGraphicsStreamEngine.java`](../../examples/src/main/java/org/apache/pdfbox/examples/rendering/CustomGraphicsStreamEngine.java)
  already has a `main` and takes a file argument — content stream operators.
- [`pdfbox/src/main/java/org/apache/pdfbox/contentstream/PDFStreamEngine.java`](../../pdfbox/src/main/java/org/apache/pdfbox/contentstream/PDFStreamEngine.java)
  is the class to breakpoint for operator dispatch.

```bash
mvn -pl pdfbox test          # run one module's Java tests
mvn -pl examples exec:java -Dexec.mainClass=org.apache.pdfbox.examples.rendering.CustomGraphicsStreamEngine -Dexec.args=some.pdf
```

When a comparison settles a question, put the answer in a Go test rather than in
a commit message. The point of the exercise is to convert "PDFBox does something
we did not expect" into a pinned assertion.

## Refreshing the inventory

The Java tree is frozen, so this only needs re-running when the mapping in
`packages.tsv` changes or a Go package is added:

```bash
pwsh go/migration/scripts/inventory.ps1
```

It rewrites `mapping/inventory.tsv` and lists any Java package missing from
`packages.tsv`, so new upstream packages surface instead of being silently
skipped.

## Relationship to the Java tree

The Java source stays where it is, untouched, in the module directories at the
repository root. It is the reference the port is checked against, and upstream
changes keep arriving — so this is a port living beside its original, not a
replacement of it. Nothing under `go/` should require editing a `.java` file.
