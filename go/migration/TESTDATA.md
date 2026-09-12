# Test data

What the port is checked against, where it comes from, and which of it is worth
the disk.

The Java tree is the specification and its tests are the assertions — that rule
does not change here. This file is about the *input*: the documents those tests
read, the ones they read but this repository does not carry, and the corpora
nobody has written an assertion for yet.

Four tiers, ordered by how targeted they are. A tier-1 file exists because one
numbered bug needed it. A tier-3 file exists because somebody crawled the web.
Both are useful; they answer different questions, and only the first two belong
in a test.

| Tier | What | Files | Where | Committed |
| --- | --- | ---: | --- | --- |
| 0 | In the repository already | 168 PDFs | `*/src/test/resources/` | yes |
| 1 | What the Java build downloads | 74 of 78 | `*/target/{pdfs,fonts,imgs}` | no — `.gitignore` |
| 2 | Targeted third-party suites | 3,589 PDFs | `go/testdata/corpus/` | no — `.gitignore` |
| 3 | Bulk, for scoring not asserting | millions | not fetched | no |

Nothing below tier 0 is ever committed. These are other projects' documents
under other projects' licences; they are fetched, read as test input, and not
redistributed from here.

## Tier 0 — already here

168 PDFs came with the Java snapshot, and they are the only ones a fresh clone
has. Most are small and each one was added alongside a fix.

| Where | Files | What it is for |
| --- | ---: | --- |
| `pdfbox/src/test/resources/input/` | 40 | The text-extraction corpus, with 81 expected-output `.txt` files beside it — `<name>.pdf.txt` unsorted and `<name>.pdf-sorted.txt` sorted by position. This is the only tier-0 set that ships its own ground truth |
| `pdfbox/src/test/resources/input/rendering/` | 6 | PDFs paired with reference PNGs, one per page. The raster ground truth |
| `pdfbox/src/test/resources/org/apache/pdfbox/multipdf/` | 28 | Merge, split and overlay inputs |
| `pdfbox/src/test/resources/input/merge/` | 15 | More of the same, for `PDFMergerUtility` |
| `.../pdmodel/interactive/form/` | 12 | AcroForm fields, appearances, flattening |
| `.../encryption/` | 10 | Every standard security handler revision, plus the keystores |
| `tools/src/test/resources/input/ImageIOUtil/` | 8 | Raster output |
| `pdfbox-layout-{awt,fop}/src/test/resources/pdf/` | 14 | The reference PDFs the two shaping backends render. `go/pdfbox/glyphlayout` is measured against these |
| `.../pdfparser/`, `.../input/compression/`, the rest | 35 | Missing catalogs, object streams, embedded files |

The Go tests reach these by relative path — `../../../pdfbox/src/test/resources/…`
— rather than copying them, so there is one copy and the Java and the Go read
the same bytes. Keep doing that. A fixture only belongs in a Go `testdata/`
directory when the Go generates it, as `multipdf/testdata/genjavabug82.go` does.

## Tier 1 — what the Java build downloads

This is the highest-value set in the whole document, and it was sitting unused.

PDFBOX-3974 added a `download-maven-plugin` block to `pdfbox/pom.xml`,
`fontbox/pom.xml`, `examples/pom.xml` and `benchmark/pom.xml`: **78 files, every
one pinned by a SHA-512 the pom carries, every one the reduced reproducer
attached to a numbered PDFBOX issue.** They are downloaded rather than committed
because they are large, third-party, or both.

Nothing here runs Maven, so `target/` has always been empty, and **29 Java test
classes read out of it** — 240 `@Test` methods sit in those classes, though
not every one of them wants a download. Twenty-seven comments across 22 Go files
name one of those directories, and eleven of them say a test is unported for that
reason — *"not ported: it reads `target/pdfs`, which this repository does not
carry"*. That sentence is now false.

```bash
pwsh go/migration/scripts/fetch-testdata.ps1          # all of it, ~106 MB
pwsh go/migration/scripts/fetch-testdata.ps1 -List    # what it would fetch
pwsh go/migration/scripts/fetch-testdata.ps1 -Module fontbox -Kind fonts
```

The script reads the manifest out of the poms rather than carrying a copy, so it
cannot drift from the frozen Java tree, and it skips a file already present whose
SHA-512 matches. No JDK needed.

| Lands in | Files | What it unlocks |
| --- | ---: | --- |
| `pdfbox/target/pdfs` | 56 | `TestPDFParser`'s xref-recovery suite, `PDFMergerUtilityTest`'s 30 cases, `PDButtonTest`, `TestRadioButtons`, `TestSymmetricKeyEncryption`, `TestQuality`, `ContentStreamWriterTest` |
| `pdfbox/target/fonts` | 12 | The IPA fonts, `PDFBOX-5484.ttf`, `n019003l.pfb` |
| `pdfbox/target/imgs` | 3 | `JPEGFactoryTest`, `LosslessFactoryTest` — a 16-bit PNG and two JPEGs |
| `fontbox/target/fonts` | 13 | `CFFParserTest`, `TestCMap`, `PfbParserTest`, `TTFSubsetterTest` — all four used to skip on every machine |
| `examples/target/pdfs` | 3 | `TestCreateSignature` |

Four entries no longer fetch, all four in `benchmark`, none read by a test: two
are hosted at `crossasia-books.ub.uni-heidelberg.de`, which no longer resolves,
one is the ECI Altona suite, whose host now serves something other than the
SHA-512 the pom pinned, and one is Adobe's own copy of ISO 32000-1. The script
reports them and exits non-zero; that is the expected result of a clean run.

**What landing this changed on the first try:** every skip in `fontbox` that read
`the font the Java build downloads is not present` now runs and passes —
`cff`, `cmap`, `type1` and `ttf`. The remaining skips there are the OS-specific
ones, which want SimHei or a Mac.

## Tier 2 — targeted third-party suites

Tier 1 is targeted at the bugs PDFBox has had. That is not the same as targeted
at the format: it says nothing about the parts of ISO 32000 PDFBox has never been
handed a bad file for. These suites are organised the other way round — one file
per clause, per feature, per construct — so a gap shows up as a named file rather
than as a document that happens not to open.

```bash
pwsh go/migration/scripts/fetch-corpus.ps1            # every suite marked small, ~130 MB
pwsh go/migration/scripts/fetch-corpus.ps1 -List
pwsh go/migration/scripts/fetch-corpus.ps1 -Suite pdfjs
```

| Suite | PDFs | Licence | What it is, and what it reaches |
| --- | ---: | --- | --- |
| [`verapdf`](https://github.com/veraPDF/veraPDF-corpus) | 2,908 | CC BY 4.0 | **The best one-file-one-rule corpus that exists.** Each file asserts a single requirement of ISO 19005 (PDF/A 1–4), ISO 14289 (PDF/UA 1–2), ISO 32000-1 or ISO 32000-2; the directory and filename name the clause, and each file documents itself in its own outline. Reaches `cos`, `pdfparser`, `xmpbox`, `documentinterchange`, `graphics/color` |
| [`qpdf`](https://github.com/qpdf/qpdf) | 639 | Apache-2.0 | qpdf's own regression inputs: cross-reference tables and streams in every arrangement, object streams of every shape, linearised and not, damaged xrefs, every encryption revision qpdf can write. Reaches `pdfparser`, `pdfwriter`, `cos`, `pdmodel/encryption` |
| [`cabinet-of-horrors`](https://github.com/openpreserve/format-corpus/tree/master/pdfCabinetOfHorrors) | 24 | see repo | The files digital preservation keeps tripping over: broken embedded fonts, encrypted-without-password, malformed page trees |
| [`safedocs-targeted`](https://github.com/pdf-association/safedocs) | 11 | Apache-2.0 | Hand-coded by the DARPA SafeDocs programme to break parsers on purpose: dual `startxref`, Type 3 inside Type 3, a page with no `/Contents`, a font inside a shading pattern, UTF-16LE strings. Small, and it earns its place — see below |
| [`pdf20examples`](https://github.com/pdf-association/pdf20examples) | 7 | see archive | PDF 2.0 features one per file: UTF-8 strings, page-level output intents, black point compensation, 2.0 reached by incremental save, a non-zero start offset |
| [`text-rendering-tests`](https://github.com/unicode-org/text-rendering-tests) | 0 | OFL fonts | No PDFs — test fonts with expected glyph ids and positions per string. This is the only **shaping ground truth independent of PDFBox**, and `go/pdfbox/glyphlayout` is the one part of the port written from a specification rather than ported, so it is the one part with no Java to check against |
| [`pdfjs`](https://github.com/mozilla/pdf.js/tree/master/test/pdfs) | 982, `-Suite pdfjs` | Apache-2.0 | Reduced reproducers from another reader's tracker. Not ground truth — pdf.js's expectations are pdf.js's — but 982 committed files that broke a real implementation, which is what makes them worth opening. A further 459 are `.link` stubs pdf.js fetches on demand and this does not. Crash input, not assertions |

## What the first run found

`corpus` over all of tier 1 and tier 2 — 3,646 files in under ten minutes. After
the one fix below:

```
3646 files, 27 of them encrypted and skipped
  open   3596/3619 (99.4%)
  text   3590/3619 (99.2%)
```

Per suite, `pdf20examples` 7/7, `safedocs-targeted` 11/11, `verapdf` 2905/2908,
`cabinet-of-horrors` 23/24, `pdfbox/target` 52/54, `examples/target` 3/3, `qpdf`
595/639.

The 27 encrypted ones are documents whose passwords live in the Java test that
reads them and which this tool has no way to know. They are reported as
`encrypted` and left out of the rate rather than counted against it — anything
else makes the number quietly wrong.

The 23 that are left are four groups, and they are not equally interesting.

### One port defect, fixed

Three files reached `panic: runtime error: index out of range [0] with length 0`
in text extraction — two of them from tier 1, which means they are in front of
the Java's own tests:

```
pdfbox/target/pdfs/PDFBOX-4418-000314.pdf
pdfbox/target/pdfs/PDFBOX-4418-000671.pdf
verapdf PDF_A-4 6.2.10.9 "Use of .notdef glyph" t01-fail-b
```

All three in `text.handleDirection`, and the cause is the one substitution that
method makes: Java uses `java.text.Bidi`, the standard library has no
counterpart, and the port uses `golang.org/x/text/unicode/bidi`. That library
resolves a word made only of paragraph separators — the empty string, or one
character of Unicode bidi class B — to **zero runs**, and then its
`Ordering.Direction` indexes the first of them. `java.text.Bidi` has no such
edge; the paragraph is simply not mixed with a base level of
`DIRECTION_LEFT_TO_RIGHT`, and Java returns the word untouched.

A defect the port introduced, so it is the one kind that gets fixed rather than
recorded: `direction.go` now answers a word with no runs before anything asks the
paragraph its direction. `TestHandleDirectionTakesAWordWithNoRuns` in
`pdfbox/text/corpusdefects_test.go` pins all seven inputs and fails without it.
`PDFBOX-4418-000314.pdf` went from a panic to 36,716 characters, and the
re-run against the first run's table is what that looks like from here:

```
against corpus-before.tsv: 0 worse, 3 better, 0 not in the baseline
  BETTER text   pdfbox/target/pdfs/PDFBOX-4418-000314.pdf: panic -> ok
  BETTER text   pdfbox/target/pdfs/PDFBOX-4418-000671.pdf: panic -> ok
  BETTER text   verapdf …/6.2.10.9 Use of .notdef glyph/…t01-fail-b.pdf: panic -> ok
```

### Fifteen qpdf files, and the gap they point at

Fifteen `qpdf/issue-*.pdf` come back `Missing root object specification in
trailer`, three more `Page tree root must be a dictionary`. These are qpdf's own
reduced reproducers of damaged files, and the question they raise is whether
PDFBox's cross-reference reconstruction recovers them where the port does not.

**That is the area [`STATUS.md`](STATUS.md) already names as the port's largest
untested hole:** `TestCOSParser` and `TestPDFParser` are the recovery suite for
broken cross-reference tables and truncated objects, five test classes and 47
`@Test` methods, and nothing in the port has run them. Eighteen qpdf files
landing on it is corroboration, not a new finding. Settling it means running the
Java on one of them, per [`README.md`](README.md).

### Three verapdf timeouts, which are the point of those files

```
Isartor PDFA-1b 6.1.12 Implementation Limits t01-fail-a
veraPDF PDF_A-1b 6.1.12 Implementation limits t03-fail-c
TWG A005-pdfa1-fail-c
```

All three are in clause 6.1.12, *implementation limits* — files built to exceed
them on purpose. Not opening one inside twenty seconds is close to what they
test for. Worth revisiting only with a number attached: which limit, and what
PDFBox does with the same file.

### One faithful panic, left alone

`qpdf/deep-pages.pdf` panics with `pdmodel: possible recursion found when
searching for page 1`, which is the port's carry of Java's
`IllegalStateException("Possible recursion found when searching for page …")` in
`PDPageTree.get`. That method declares no checked exception, so the port has no
error channel to use and carries the unchecked throw as a panic — the same
line above it does it for `IndexOutOfBoundsException`. Nothing to do.

### And the one that ends the process

`safedocs-targeted/ContentStreamCycleType3insideType3.pdf` is a Type 3 font whose
glyph procedure draws itself. Rendering it recurses 3.9 million frames deep and
ends in Go's `fatal error: stack overflow`.

The port is not wrong to do that. Neither PDFBox nor the port bounds Type 3
recursion: the `level` guard both carry — `increaseLevel`, `getLevel() > 50` — is
wired into the three `DrawObject` operators and **not** into `showType3Glyph`, so
the Java reaches `StackOverflowError` on the same file. PDFBox's own
[`SECURITY.md`](../../SECURITY.md) calls that a known limitation of reading
malformed PDFs rather than a vulnerability.

What *is* different is the blast radius, and it is a Go fact rather than a port
defect: a `StackOverflowError` is an `Error` and a Java caller can catch it, while
Go's stack overflow is fatal and no deferred `recover` runs. Same input, same
recursion, unrecoverable on one side. That is why `cmd/corpus` scores each file
in a child process by default.

## Tier 3 — bulk

Not fetched, and not for assertions. These answer "does it crash", "does it hang",
and "has the open rate moved", over inputs nobody curated.

| Corpus | Size | Why it is on this list |
| --- | --- | --- |
| [SafeDocs issue-tracker corpus](https://labs.pdfa.org/stressful-corpus/), `pdfs_202011/batch1.tgz` | 1.8 GB | **Batch 1 is PDFBOX.** Every PDF attached to a public PDFBOX issue, crawled by NASA JPL — the superset of tier 1, without the curation or the checksums. The most on-point bulk corpus for this repository by a distance. Batches 2–6 are Ghostscript, Tika, Mozilla, LibreOffice, pdf.js, poppler, qpdf and 28 others. Note the README's own warning: the collection may contain malicious files |
| [GovDocs1](https://digitalcorpora.org/corpora/files) | ~231k PDFs | Public-domain real-world government documents. Unbiased in a way none of the above are: nothing in it was chosen because it broke something |
| [CC-MAIN-2021-31-PDF-UNTRUNCATED](https://digitalcorpora.org/cc-main-2021-31-pdf-untruncated/) | ~8M PDFs | The web as it is. Only worth it for a producer-distribution question |
| [pdf-association/pdf-corpora](https://github.com/pdf-association/pdf-corpora) | — | The index the three above came from, and the place to look before going hunting. Around fifty corpora with sizes and licences |

## Scoring a corpus

Three thousand files cannot become assertions by hand, and they should not: the
question they answer is not "is this byte right" but "which of these can the port
open, read and draw at all, and did that change".

`go/cmd/corpus` is that. It is migration tooling, not a port of anything — PDFBox
has no such command.

```bash
cd go
go run ./cmd/corpus -o corpus.tsv ./testdata/corpus            # one row per file
go run ./cmd/corpus -render ./testdata/corpus/safedocs-targeted
go run ./cmd/corpus -baseline corpus.tsv ./testdata/corpus     # only what changed
```

One row per file, one column per stage — open, pages, text, chars, render — each
cell `ok` or the first thing that went wrong. Saved and passed back through
`-baseline`, the table is a regression check: a file that used to open and no
longer does is a defect the unit tests did not catch, and it exits non-zero so CI
can hold the line. A file that used to fail and now opens is a slice landing.

`-isolate` is on by default and scores each file in a child process. That is not
belt-and-braces: `recover` cannot catch a Go stack overflow, and a goroutine that
will not return cannot be stopped from outside, so without it one file ends the
run. `-isolate=false` is faster on a corpus already known to be survivable.

Note what this does **not** do: it does not compare against the Java. Running
PDFBox over three thousand files is not cheap either, and the comparison that
matters is per-case rather than per-corpus. For that, keep to what
[`README.md`](README.md) already says — run the Java on the one file, and convert
the answer into a pinned Go assertion rather than a commit message.

## Where a new document should go

1. **Does a tier-0 file already show it?** Use that one. 168 is more than the Go
   tests currently read.
2. **Does the Java download one for exactly this?** Then the pom has it, the
   SHA-512 is already written down, and `fetch-testdata.ps1` already fetches it.
3. **Is the smallest file that shows the behaviour one you can write?** Write it,
   as a Go generator under the package's `testdata/`, the way
   `multipdf/testdata/genjavabug82.go` does. A generated fixture can be read in a
   text editor and re-made when the question changes; a downloaded one cannot.
4. **Only then reach for a corpus.** And when a corpus file settles a question,
   the answer belongs in a Go test with the file's name in the comment — not in
   a directory nobody re-runs.
