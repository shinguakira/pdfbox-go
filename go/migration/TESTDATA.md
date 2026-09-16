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
| 0 | In the repository already | 166 PDFs, and 13 more elsewhere in the tree | `*/src/test/resources/` | yes |
| 1 | What the Java build downloads | 74 of 78 | `*/target/{pdfs,fonts,imgs}` | no — `.gitignore` |
| 2 | Targeted third-party suites | 5,032 PDFs | `go/testdata/corpus/` | no — `.gitignore` |
| 3 | Bulk, for scoring not asserting | millions | not fetched | no |

Nothing below tier 0 is ever committed. These are other projects' documents
under other projects' licences; they are fetched, read as test input, and not
redistributed from here.

## Tier 0 — already here

166 PDFs came with the Java snapshot under `src/test/resources`, and 13 more sit
elsewhere in the tree; they are the only ones a fresh clone has. Most are small,
and each one was added alongside a fix.

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
| `.../pdfparser/`, `.../input/compression/`, the rest | 33 | Missing catalogs, object streams, embedded files |

The Go tests reach these by relative path — `../../../pdfbox/src/test/resources/…`
— rather than copying them, so there is one copy and the Java and the Go read
the same bytes. Keep doing that. A fixture only belongs in a Go `testdata/`
directory when the Go generates it, as `multipdf/testdata/genjavabug82.go` does.

## Tier 1 — what the Java build downloads

PDFBOX-3974 added a `download-maven-plugin` block to `pdfbox/pom.xml`,
`fontbox/pom.xml`, `examples/pom.xml` and `benchmark/pom.xml`: **78 files, every
one pinned by a SHA-512 the pom carries, every one the reduced reproducer
attached to a numbered PDFBOX issue.** They are downloaded rather than committed
because they are large, third-party, or both. Nothing here runs Maven, so
`target/` stayed empty until `fetch-testdata.ps1` filled it on 2026-09-11; which
Java tests that unblocked, class by class, is in [`STATUS.md`](STATUS.md), "What
the fetch unblocked".

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
| `pdfbox/target/pdfs` | 58, 55 of them PDFs | `TestPDFParser`'s xref-recovery suite, `PDFMergerUtilityTest`'s 30 cases, `PDButtonTest`, `TestRadioButtons`, `TestSymmetricKeyEncryption`, `TestQuality`, `ContentStreamWriterTest` |
| `pdfbox/target/fonts` | 12 | The IPA fonts, `PDFBOX-5484.ttf`, `n019003l.pfb` |
| `pdfbox/target/imgs` | 3 | `JPEGFactoryTest`, `LosslessFactoryTest` — a 16-bit PNG and two JPEGs |
| `fontbox/target/fonts` | 13 | `CFFParserTest`, `TestCMap`, `PfbParserTest`, `TTFSubsetterTest` — all four used to skip on every machine |
| `examples/target/pdfs` | 3 | `TestCreateSignature` |

Four entries no longer fetch, all four in `benchmark`, none read by a test: two
are hosted at `crossasia-books.ub.uni-heidelberg.de`, which no longer resolves,
one is the ECI Altona suite, whose host now serves something other than the
SHA-512 the pom pinned, and one is Adobe's own copy of ISO 32000-1. The script
reports them and exits non-zero; that is the expected result of a clean run.

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
| [`safedocs-targeted`](https://github.com/pdf-association/safedocs) | 11 | Apache-2.0 | Hand-coded by the DARPA SafeDocs programme to break parsers on purpose: dual `startxref`, Type 3 inside Type 3, a page with no `/Contents`, a font inside a shading pattern, UTF-16LE strings. Its `ContentStreamCycleType3insideType3.pdf` is [`JAVA-BUGS.md`](JAVA-BUGS.md) 87, the unbounded Type 3 recursion neither side guards; it is why `cmd/corpus` isolates each file |
| [`pdf20examples`](https://github.com/pdf-association/pdf20examples) | 7 | see archive | PDF 2.0 features one per file: UTF-8 strings, page-level output intents, black point compensation, 2.0 reached by incremental save, a non-zero start offset |
| [`text-rendering-tests`](https://github.com/unicode-org/text-rendering-tests) | 0 | OFL fonts | No PDFs — test fonts with expected glyph ids and positions per string. This is the only **shaping ground truth independent of PDFBox**, and `go/pdfbox/glyphlayout` is the one part of the port written from a specification rather than ported, so it is the one part with no Java to check against |
| [`pdfjs`](https://github.com/mozilla/pdf.js/tree/master/test/pdfs) | 1,443, `-Suite pdfjs` | Apache-2.0 | Reduced reproducers from another reader's tracker. Not ground truth — pdf.js's expectations are pdf.js's — but files that broke a real implementation, which is what makes them worth opening. 982 are committed in `test/pdfs`, 459 are `.link` stubs the script downloads and matches against the md5 in `test/test_manifest.json`, and two are committed elsewhere in the repository; `_links.tsv` says where each came from, and the manifest and those two are copied under `_repo/`. Twelve are encrypted and open with the passwords its own tests use, which the script writes to `_passwords.tsv`. `test/pdfs/sig_corpus` is not fetched: pdf.js generates its eight signed PDFs rather than committing or downloading them, and [`tasks/track-testdata-sources.md`](tasks/track-testdata-sources.md) U7 has what having them would take. Crash input, not assertions |

## Tier 3 — bulk

Not fetched, and not for assertions. These answer "does it crash", "does it hang",
and "has the open rate moved", over inputs nobody curated.

| Corpus | Size | Why it is on this list |
| --- | --- | --- |
| [SafeDocs issue-tracker corpus](https://labs.pdfa.org/stressful-corpus/), `pdfs_202011/batch1.tgz` | 1.8 GB | **Batch 1 is PDFBOX.** Every PDF attached to a public PDFBOX issue, crawled by NASA JPL — the superset of tier 1, without the curation or the checksums. The most on-point bulk corpus for this repository by a distance. Batches 2–6 are Ghostscript, Tika, Mozilla, LibreOffice, pdf.js, poppler, qpdf and 28 others. Note the README's own warning: the collection may contain malicious files |
| [GovDocs1](https://digitalcorpora.org/corpora/files) | ~231k PDFs | Public-domain real-world government documents. Unbiased in a way none of the above are: nothing in it was chosen because it broke something |
| [CC-MAIN-2021-31-PDF-UNTRUNCATED](https://digitalcorpora.org/cc-main-2021-31-pdf-untruncated/) | ~8M PDFs | The web as it is. Only worth it for a producer-distribution question |
| [pdf-association/pdf-corpora](https://github.com/pdf-association/pdf-corpora) | — | The index the three above came from, and the place to look before going hunting. Around fifty corpora with sizes and licences |

## Measured against the Java

Scoring the port on its own answers "where does it fall over", not "does it agree
with PDFBox", and those are different questions. This is the second one.

`migration/scripts/run-oracle.ps1` compiles `io`, `fontbox` and `pdfbox` out of
the tree with `javac` — no Maven, and it fetches the four compile-scope jars the
poms name — and runs PDFBox over the same file list, writing the same table.
`corpus -oracle` joins the two.

```bash
pwsh go/migration/scripts/run-oracle.ps1
cd go && go run ./cmd/corpus -oracle testdata/oracle/java-corpus.tsv \
    ./testdata/corpus ../pdfbox/target/pdfs ../examples/target/pdfs
```

pdf.js is fetched and compared with the passwords its own tests use, on both
sides:

```bash
pwsh go/migration/scripts/fetch-corpus.ps1 -Suite pdfjs
pwsh go/migration/scripts/run-oracle.ps1 -List <the 1,443 paths> -Out go/testdata/oracle/java-pdfjs.tsv `
    -Passwords go/testdata/corpus/pdfjs/_passwords.tsv
cd go && go run ./cmd/corpus -passwords testdata/corpus/pdfjs/_passwords.tsv \
    -oracle testdata/oracle/java-pdfjs.tsv ./testdata/corpus/pdfjs
```

What this compares is whether a document opens, how many pages it has, whether
its text extracts, and **how many characters** that text has. It is length, not
content: two documents of the same length are not the same document. What length
catches is a stage that silently produced nothing, a page that was not walked, a
glyph that came out as two characters. A wrong character in the right place
passes, and the ported Java tests are what covers that — this is the layer
underneath them, and its job is to say that nothing is wrong at a scale those
tests cannot reach.

### The separator, which has to be dealt with before any of this means anything

The first run of this comparison said 3,470 of 3,590 documents differed in text
length by more than 2%, which read as a broken extractor. It is not. Java's
`PDFTextStripper` initialises both `lineSeparator` and `pageEnd` from
`System.lineSeparator()`, which on Windows is CRLF; the port hardcodes LF and
says so at `text/pdftextstripper.go:22`. Every line of every document therefore
costs one character more on the Java side. The delta distribution says it
outright — 2,925 files differed by exactly **-1**, and the eight that matched
were the eight with no text at all.

So the oracle forces both to LF, and `run-oracle.ps1 -Crlf` reproduces the
confusion on demand. The deviation is real and deliberate; it is a convention,
not a defect, and comparing content requires taking it out first.

### What the runs found

Three runs, each scored again after its fixes. Every disagreement any of them
found was the port's, and none was a difference of opinion with PDFBox.

**The port on its own over tier 1 and tier 2, 2026-09-12.** 3,646 files in under
ten minutes, 3,596 of them opened. One port defect: `text.handleDirection`
panicked on a word made only of paragraph separators, where
`golang.org/x/text/unicode/bidi` resolves it to zero runs and `java.text.Bidi`
has no such edge. Fixed and pinned by `TestHandleDirectionTakesAWordWithNoRuns`
in `go/pdfbox/text/corpusdefects_test.go`; [`STATUS.md`](STATUS.md) carries it.
Three veraPDF files in clause 6.1.12, *implementation limits*, timed out, which
is close to what those files test for.

**The same 3,646 against PDFBox, 2026-09-12.** Twelve disagreements, and six
defects behind them, all closed:

| Defect | Files | What pins it |
| --- | ---: | --- |
| `processPages` walked `pages.Get(i)` where Java's is `for (PDPage page : pages)`. `PDPageTree` guards a cyclic page tree in two places on purpose — the indexed accessor throws, the iterator logs and skips the kid — and text extraction reaches the iterator's | 5 | `TestProcessPagesWalksTheTreeTheWayJavaDoes`, `go/pdfbox/text/corpusdefects_test.go` |
| A computed cross-reference repair was dropped. `XrefTrailerResolver.getXrefTable` hands out the live map and `validateXrefOffsets` edits it in place; the port's `XrefTable` answers a copy, so the swap went out of scope | 1 | `TestSwappedXrefEntriesAreRepaired`, `go/pdfbox/pdfparser/corpusdefects_test.go` |
| `Close` on a damaged Flate stream raised the damage `Read` had already absorbed. `compress/flate` remembers the error and hands it back from `Close`; Java's close is `inflater.end()` and cannot | 2 | `TestFlateDecoderReaderCloseSwallowsDamage`, `go/pdfbox/filter/flate_test.go` |
| The duplicate suppression scanned every key where Java asks a `TreeMap` for `subMap` and then `subSet` — the same answer, quadratic in the glyphs on a page, 22 seconds on two clause 6.1.12 files | 2 | commented at `go/pdfbox/text/pdftextstripper.go` |
| `COSParser.getObjectKey` copied the whole cross-reference table to ask how big it is, once per object read. On the ten-thousand-page Isartor file that was 71% of 238 seconds; answering without copying made it 0.51 | 1 | `XRefTableSize` and `EachXRefKey`, `go/pdfbox/cos/document.go` |
| Type 0 glyphs advanced by the font program instead of `/W`, which cost `PDFBOX-3951` one character in 126,330 | 1 | below |

Running the Java also settled the question qpdf's damaged files raise, which is
whether PDFBox's cross-reference reconstruction recovers them where the port does
not. It does not: seventeen of the eighteen fail in PDFBox with the identical
message, and the eighteenth was `issue-202.pdf`, the repair above.

**pdf.js against the Java, 2026-09-15.** Its 1,443 files gave fourteen
disagreements, all of them the length of the text; nothing PDFBox opened failed
to open, and no page count disagreed. Twelve were one defect, and fixing it
closed `PDFBOX-3951` above with them: `pdFont.Displacement` called its own
`Width`, where Java's `getWidth` is a virtual call reaching
`PDType0Font.getWidth` and the descendant's `/W`, so every glyph of every Type 0
font was advanced by its font program, embedded or substituted, in extraction
and in rendering alike.
`TestType0DisplacementIsTheDescendantWidth` pins it, and
[`tasks/track-testdata-sources.md`](tasks/track-testdata-sources.md) has it and
the four further differences that reading around it turned up. The other two are
[`JAVA-BUGS.md`](JAVA-BUGS.md) 15 and 30, where `track/java-bug-fixes` made the
Go differ from the Java on purpose.

### Where it stands

Everything on disk outside the repository — the 3,646 above, pdf.js's 1,443, and
`PDFBOX-4131-0.pdf`, which the 2026-09-12 PDFBox table was missing — with
`run-oracle.ps1`'s default list and the passwords table on both sides:

```
5090 files, 27 of them encrypted and skipped
  open    both 5037, neither 53, behind 0, ahead 0
  pages   0 disagree
  text    both 5032, neither 5, behind 0, ahead 0
  chars   5030 the same length, 2 not

  2 of 5090 files disagree (0.04%)
```

**There is no document in the corpus that PDFBox reads and the port does not.**
Not one, at either stage. No page count disagrees anywhere, and the two lengths
that differ are the two deliberate Java-bug fixes above.

The 27 encrypted files are 24 of qpdf's, one of cabinet-of-horrors' and two
PDFBox downloads, whose passwords are in qpdf's test scripts and PDFBox's Java
tests and not yet in a table. Until they are, both sides are compared only on
refusing them; how far finding them has got is
[`tasks/track-testdata-sources.md`](tasks/track-testdata-sources.md) U8.

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

`-passwords` takes a table of encrypted files and their passwords, one
`<path ending>`, tab, `<password>` per line, in UTF-8. `fetch-corpus.ps1` writes
one as `_passwords.tsv` for a suite whose project publishes them — so far pdf.js
— and `run-oracle.ps1 -Passwords` gives PDFBox the same table. A file with no
password that will not open is reported `encrypted` and left out of the rates; a
file the table has a password for keeps its error if it still will not open.

Note what this does **not** do: it does not compare against the Java. Running
PDFBox over three thousand files is not cheap either, and the comparison that
matters is per-case rather than per-corpus. For that, keep to what
[`README.md`](README.md) already says — run the Java on the one file, and convert
the answer into a pinned Go assertion rather than a commit message.

## Where a new document should go

1. **Does a tier-0 file already show it?** Use that one. There are more of them
   than the Go tests currently read.
2. **Does the Java download one for exactly this?** Then the pom has it, the
   SHA-512 is already written down, and `fetch-testdata.ps1` already fetches it.
3. **Is the smallest file that shows the behaviour one you can write?** Write it,
   as a Go generator under the package's `testdata/`, the way
   `multipdf/testdata/genjavabug82.go` does. A generated fixture can be read in a
   text editor and re-made when the question changes; a downloaded one cannot.
4. **Only then reach for a corpus.** And when a corpus file settles a question,
   the answer belongs in a Go test with the file's name in the comment — not in
   a directory nobody re-runs.
