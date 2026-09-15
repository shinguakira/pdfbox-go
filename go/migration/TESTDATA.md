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
| 2 | Targeted third-party suites | 5,032 PDFs | `go/testdata/corpus/` | no — `.gitignore` |
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
| [`pdfjs`](https://github.com/mozilla/pdf.js/tree/master/test/pdfs) | 1,443, `-Suite pdfjs` | Apache-2.0 | Reduced reproducers from another reader's tracker. Not ground truth — pdf.js's expectations are pdf.js's — but files that broke a real implementation, which is what makes them worth opening. 982 are committed in `test/pdfs`. 459 are `.link` stubs holding a URL, which pdf.js downloads when its tests run; the script downloads them too and keeps each only with the md5 `test/test_manifest.json` records, so a file that cannot be had fails the run, and `_links.tsv` says where each came from. 2 are committed outside `test/pdfs` and land under `_repo/`. The passwords its tests open 12 of them with are written to `_passwords.tsv`. **Not on disk:** `test/pdfs/sig_corpus`, eight signed PDFs pdf.js neither commits nor downloads, which its `generate.py` builds from a mozilla-central checkout. Crash input, not assertions |

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

### One panic that looked faithful and is not

`qpdf/deep-pages.pdf` panics with `pdmodel: possible recursion found when
searching for page 1`. Read against the Java that looks like a faithful carry:
`PDPageTree.get` throws `IllegalStateException` with that message, declares no
checked exception, and so has no error channel the port could have used — the
line above it carries `IndexOutOfBoundsException` the same way.

**Running the Java says otherwise. PDFBox opens the file, reports one page, and
extracts from it without the guard firing.** So the port's recursion detection
fired where the Java's does not, on identical input, and that was a port defect
rather than a carry — fixed below, along with four other files that had looked
like unrelated single-character disagreements and were the same defect.

This section is worth its length for the reason it was wrong: reading two
implementations side by side is how the wrong conclusion got written down here in
the first place, and running them side by side is what caught it.

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

## Measured against the Java, 2026-09-12

The section above scores the port on its own: what it opens, what it reads. That
answers "where does it fall over", not "does it agree with PDFBox", and those are
different questions. This is the second one, and it is the first time it has been
asked at this scale.

`migration/scripts/run-oracle.ps1` compiles `io`, `fontbox` and `pdfbox` out of
the tree with `javac` — no Maven, and it fetches the four compile-scope jars the
poms name — and runs PDFBox over the same file list, writing the same table.
`corpus -oracle` joins the two.

```bash
pwsh go/migration/scripts/run-oracle.ps1
cd go && go run ./cmd/corpus -oracle testdata/oracle/java-corpus.tsv \
    ./testdata/corpus ../pdfbox/target/pdfs ../examples/target/pdfs
```

Over the same 3,646 files, the first run found twelve disagreements. All twelve
were the port's, eleven of them are fixed, and this is where it stands now:

```
3646 files, 27 of them encrypted and skipped
  open    both 3600, neither 46, behind 0, ahead 0
  pages   0 disagree
  text    both 3597, neither 3, behind 0, ahead 0
  chars   3596 the same length, 1 not

  1 of 3646 files disagree (0.03%)
```

**There is no document in the corpus that PDFBox reads and the port does not.**
Not one, at either stage. No page count disagrees anywhere, and of the 3,597
documents both extract, 3,596 come out the same length. The twelfth disagreement
is one character in 126,330, and it is described below.

For contrast, the first run of this comparison — before any of the fixes — read:

```
  open    both 3596, neither 46, behind 4, ahead 0
  text    both 3590, neither 3, behind 3, ahead 0
  chars   3585 the same length, 5 not
```

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

### The twelve, and what each one turned out to be

Every one of them was the port's defect, not a difference of opinion with
PDFBox. Eleven are fixed; the twelfth is a character and is described below.

| File | Was | Cause |
| --- | --- | --- |
| `qpdf/deep-pages.pdf` | text panicked on the page-tree recursion guard | **page tree** |
| `qpdf/fuzz-16214.pdf` | 1 char against 0 | page tree |
| `qpdf/many-nulls.pdf` | 0 chars against 1 | page tree |
| `qpdf/no-pages-types.pdf` | 7 chars against 0 | page tree |
| `qpdf/pages-loop.pdf` | 7 chars against 14 | page tree |
| `qpdf/issue-202.pdf` | `Page tree root must be a dictionary`; PDFBox reads 10 pages | **xref repair** |
| `qpdf/shared-images-errors.pdf` | text failed on a damaged Flate stream | **filter close** |
| `qpdf/shared-images-errors-2-out.pdf` | the same | filter close |
| `verapdf` PDF_A-1b 6.1.12 `t03-fail-c` | timeout; 65,540 chars on one page | **quadratic lookup** |
| `verapdf` TWG `A005-pdfa1-fail-c` | timeout; the same shape | quadratic lookup |
| `verapdf` Isartor PDFA-1b 6.1.12 `t01-fail-a` | timeout; 10,000 pages | **quadratic xref copy** |
| `pdfbox/target/pdfs/PDFBOX-3951-FIHUZ…` | 126,331 chars against 126,330 | font metrics — **open**; closed 2026-09-15 as a Type 0 displacement defect, below |

#### Five were one walk

`PDPageTree` guards against a cyclic page tree in two places and the two behave
differently **on purpose**. The indexed accessor, `get(int, COSDictionary, int)`,
throws `IllegalStateException`. The iterator's `enqueueKids` logs
`This page tree node has already been visited` and skips the kid, and the comment
on it cites PDFBOX-5009 and PDFBOX-3953. The port carries both, faithfully.

What it had wrong was which one text extraction reaches. Java's `processPages` is
`for (PDPage page : pages)`; the port's walked `pages.Get(i)` over
`pages.Count()`. On a well-formed file those are the same walk. On five corpus
files they are not, and `deep-pages.pdf` — where PDFBox prints that error and
carries on to extract — is the one that made it obvious. Iterating fixed all
five, including four that had looked like unrelated single-character
disagreements.

#### One was a repair that was computed and then dropped

`issue-202.pdf` has two cross-reference entries pointing at each other's objects.
Both implementations notice, both log it, and both compute the swap;
`XrefParser.validateXrefOffsets` in Java then applies it, because
`XrefTrailerResolver.getXrefTable()` hands out the resolver's own map and the
method edits it in place. The port's `XrefTable` answers a copy — the right shape
in Go — so the repair landed in a value that went out of scope at the end of
`checkXrefOffsets`. The failure path already wrote its result back; the success
path did not.

#### Two were a `Close` that undid its own `Read`

`shared-images-errors` carries a damaged Flate stream. The port's read side was
already right — Java's `FlateFilterDecoderStream.fetch` catches the
`DataFormatException`, keeps what inflated and reports end of data, and the port
does the same. But `compress/flate` remembers the error and hands it back from
`Close`, while Java's close is `inflater.end()` and cannot. So the stream said
"no more data", the caller believed it, and closing raised the damage the read
had already absorbed.

#### Three were speed, and speed is behaviour when there is a timeout

Neither of these changes an answer. Both change a complexity.

`PDFTextStripper`'s duplicate suppression asks Java's
`TreeMap<Float, TreeSet<Float>>` for `subMap(x - tolerance, x + tolerance)` and
then `subSet(y - tolerance, y + tolerance)`. Both are ordered, so both ranges are
found by search. The port used Go maps and scanned every key — the same answer,
quadratic in the glyphs on the page. On the two clause 6.1.12 files, which put
65,539 characters on a single page on purpose, that was 22 seconds against
PDFBox's fraction of one.

`COSDocument.getXrefTable` returns the live map in Java and a copy in the port,
because the Go table is keyed by internal hash rather than by the key object.
That is fine everywhere except `COSParser.getObjectKey`, which the port carries
line for line and which runs once per object read: copying the whole table to ask
how big it is is quadratic in the objects. A profile of the ten-thousand-page
Isartor file put 71% of 238 seconds inside it. With `XRefTableSize` and
`EachXRefKey` answering without copying, **238 seconds became 0.51**.

#### One is still open, and it is one character

`PDFBOX-3951-FIHUZ…` is 142 pages and differs from PDFBox by a single character
in 126,330: the copyright line reads `©  ECRI` here and `© ECRI` there. The extra
space is a word separator the port inserts because it thinks the gap is wider
than PDFBox does, and the gap is wider because the two disagree about the glyph
before it.

Measured on both sides, the `©` is drawn from `GHLILD+SymbolMT`, non-embedded,
and everything about the position agrees except the width — PDFBox makes it
8.6742 and the port 6.5891, a ratio of exactly 790 to 600.09766. Those two
numbers are the font's standard-14 width and the width its substitute reports.
What is *not* yet explained is that the two implementations also read a different
character code for the same glyph: PDFBox's `TextPosition` carries code 148 and
the port's `ShowGlyph` is handed 120. That is an encoding question in a symbolic
TrueType font, it is a layer below the text stripper, and it wants its own piece
of work rather than a guess.

Left open deliberately. It is recorded here with the measurements so the next
person starts where this stopped, and it is one character out of 126,330 in one
document out of 3,646.

**Closed on 2026-09-15**, by the defect the pdf.js corpus found — see
[pdf.js against the Java](#pdfjs-against-the-java-2026-09-15). Two things in
the paragraphs above were wrong. The 790 is not a standard-14 width:
`GHLILD+SymbolMT` is a Type 0 font over a non-embedded CIDFontType2, 790 is
the glyph's entry in its `/W`, and PDFBox advances by it; 600.09766 is the
substitute font's own width, which the port advanced by instead. And the code is
148 on both sides — measured again with every `TextPosition` dumped, the port's
carries 148 as well. With the displacement fixed the document's text is
identical to PDFBox's, all 126,330 characters.


### What this does and does not establish

It compares **length**, not content: two documents of the same length are not the
same document. What length catches is a stage that silently produced nothing, a
page that was not walked, a glyph that came out as two characters. What it cannot
catch is a wrong character in the right place. The ported Java tests are what
covers that, because their assertions are the Java's own values — this is the
layer underneath them, and its job is to say that nothing is wrong at a scale
those tests cannot reach.

## pdf.js against the Java, 2026-09-15

Every PDF pdf.js tests with, fetched and put through both sides:

```bash
pwsh go/migration/scripts/fetch-corpus.ps1 -Suite pdfjs
pwsh go/migration/scripts/run-oracle.ps1 -List <the 1,443 paths> -Out go/testdata/oracle/java-pdfjs.tsv `
    -Passwords go/testdata/corpus/pdfjs/_passwords.tsv
cd go && go run ./cmd/corpus -passwords testdata/corpus/pdfjs/_passwords.tsv \
    -oracle testdata/oracle/java-pdfjs.tsv ./testdata/corpus/pdfjs
```

**What is there.** 1,443 PDFs: the 982 committed in `test/pdfs`, the 459 its
`.link` stubs name — 176 of the stubs an archive.org capture and the rest the
file's own address, every file downloaded and matched against the md5 in
`test/test_manifest.json` — and the two committed elsewhere in the repository,
`web/compressed.tracemonkey-pldi-09.pdf` and `examples/learning/helloworld.pdf`.
**Not there:** `test/pdfs/sig_corpus`. Its eight PDFs are ignored by pdf.js's own
`.gitignore` and appear in no manifest; they exist only once its `generate.py`
has been run against a built mozilla-central checkout, for testing Firefox's
signature panel by hand. Fetching them means generating them, which is open in
the task file.

**Passwords.** Twelve of the files are encrypted. pdf.js's manifest gives seven
of their passwords and its tests the other five — `api_spec.js` for `pr6531_1`,
`pr6531_2`, `auth-event-ef-open` and `encrypted-attachment`, `viewer_spec.mjs`
for `print_protection` — and `fetch-corpus.ps1` writes all twelve to
`_passwords.tsv`, which both drivers are handed. Ten do not open without one;
the other two, `issue15893_reduced` and `pr6531_2`, open without one on both
sides and are opened with theirs because that is what pdf.js does. With the
passwords, all twelve open on both sides and agree.

### Fourteen disagreements, and a defect behind twelve of them

The first run, before any fix:

```
  open    both 1436, neither 7, behind 0, ahead 0
  pages   0 disagree
  text    both 1434, neither 2, behind 0, ahead 0
  chars   1420 the same length, 14 not
```

Nothing PDFBox opens failed to open here, and no page count disagreed; all
fourteen were the length of the text. Dumping every `TextPosition` from both
sides put twelve of them on one field: from the first glyph, the **width**
differed and nothing else did — same x, same y, same height, same space width.

| File | Go | PDFBox | Cause |
| --- | ---: | ---: | --- |
| `JBIG2Globals.pdf` | 8,662 | 8,475 | Type 0 displacement |
| `P020121130574743273239.pdf` | 14,723 | 14,667 | Type 0 displacement |
| `bug1749563.pdf` | 1,974 | 2,082 | Type 0 displacement |
| `bug951051.pdf` | 72,487 | 72,488 | Type 0 displacement |
| `issue15139.pdf` | 89 | 88 | Type 0 displacement |
| `issue15292.pdf` | 1,114 | 1,227 | Type 0 displacement |
| `issue1687.pdf` | 52 | 51 | Type 0 displacement |
| `issue1721.pdf` | 2,034,906 | 2,036,568 | Type 0 displacement |
| `issue18801.pdf` | 14,354 | 14,355 | Type 0 displacement |
| `issue7074_reduced.pdf` | 17 | 19 | Type 0 displacement |
| `issue9367.pdf` | 2,125 | 2,134 | Type 0 displacement |
| `mupdf-707147.pdf` | 81 | 78 | Type 0 displacement |
| `bug1175962.pdf` | 117 | 126 | JAVA-BUGS 15, fixed in the Go on purpose |
| `poppler-90-0-fuzzed.pdf` | 1,197 | 1,402 | JAVA-BUGS 30, fixed in the Go on purpose |

#### Type 0 glyphs advanced by the font program instead of `/W`

`PDFont.getDisplacement` is `new Vector(getWidth(code) / 1000, 0)`, and in Java
that `getWidth` is a virtual call: for a Type 0 font it reaches
`PDType0Font.getWidth`, which is the descendant's `/W`, then `/DW`, then 1000.
The port's `pdFont.Displacement` called `pdFont.Width` directly — Go embedding
does not dispatch — and that is the simple-font lookup, which finds no `/Widths`
in a Type 0 dictionary and falls through to the font program's advance. So every
glyph of every Type 0 font was advanced by its font program, embedded or
substituted, rather than by the widths the PDF gives.

`issue1687.pdf` shows the effect in one word. Its Arial Black is a CIDFontType2
with no `/W`, so PDFBox makes every glyph the default 1000 wide and the port made
it the font's 777.8. The stripper drops a glyph that repeats the one before it
within a third of its width. The two `l`s of "Ellis" are placed 4.264 apart; a
third of PDFBox's width is 4.267 and a third of the port's was 3.319, so PDFBox
extracts "Elis" and the port kept both. The other eleven are the same widths
reaching word breaks and duplicates: `issue15292.pdf` came out as
"A newstraightforwardandtransparentfeestructure", `bug1749563.pdf` lost
letters inside words.

It is not only text. The renderer advances by the same displacement, so Type 0
text was drawn at the font program's spacing too.
`TestLigaturesAndKerningRenderIdent` and `TestBidiRenderIdent` count the pixels
that differ between the port's page and the AWT reference, and both counts moved
— 2151 to 2160 and 2433 to 2440 — only inside the lines their `knownDeviations`
already list, where the two files draw different glyphs and so hold different
`/W` entries. Everywhere else both pages are unchanged, and the supplementary
plane page is still exact.

The fix is `f.self.Width(code)` in `pdfont.go`, the dispatch every other shared
method there already used. `TestType0DisplacementIsTheDescendantWidth` in
`pdfbox/pdmodel/font/corpusdefects_test.go` pins four dictionaries — no `/W` or
`/DW`, each alone, both — with PDFBox's own answers. **The same fix closed
`PDFBOX-3951`**, the last disagreement left in the 3,646 files above.

Two further things came out of reading around it. `LegacyPDFStreamEngine`'s
vertical branch scales a glyph's width by 1000 / `unitsPerEm` for a TrueType
program; the port had the `PDTrueTypeFont` half and had left the `PDType0Font`
half for a later slice that never brought it. It is ported, and
`TestVerticalGlyphWidthIsScaledByTheCIDFontsEm` pins it with PDFBox's values;
no file in either corpus reaches it. And the shape of the defect — a shared
method calling, on its own receiver, a method a type embedding it overrides —
was searched for across the whole Go tree, with a scratch program over the
syntax trees. Twenty-one calls have that shape, in six places:

| Where | Calls | What it comes to |
| --- | ---: | --- |
| `cos.Dictionary`, through `UpdateState` | 7 | nothing: a `Stream`'s constructor creates the state with the stream as its owner, so the dictionary's methods reach the stream's state |
| `ttf.Parser.parseAndClose`, through `ParseStream` | 1 | nothing: `OTFParser`'s overrides of `newFont` and `readTable` travel as fields its constructor sets |
| `function.pdFunctionBase`, through `rangeValues` | 3 | nothing for two — the identity function overrides `NumberOfOutputParameters`, and only types 2 and 3 clip to range. **The third was a difference, fixed 2026-09-15:** `RangeForOutput` on an identity function returns a range over no array in PDFBox, which fails only when it is read; the port panicked on the call. `TestIdentityRangeForOutputHasNoArray` |
| `ttf.TrueTypeFont.GetPath`, through `Glyph` | 1 | **a difference, fixed 2026-09-15.** An OpenType font with CFF2 outlines and no CFF, which neither side supports: PDFBox reaches `OpenTypeFont.getGlyph` and throws "OTF fonts do not have a glyf table", and the port dereferenced a nil table. It now panics with PDFBox's message. `TestOpenTypeWithoutCFFRefusesAGlyphTable` |
| `form.PDButton`, through `ExportValues`, `OnValues` and `Value` | 7 | **a difference, fixed 2026-09-15.** `SetValue`, `SetValueIndex` and `SetDefaultValue` on a `PDPushButton` checked against `PDButton`'s own values, where Java reaches the push button's empty ones and refuses every value but `Off`. `PDButton` now asks through the field's `self`. `TestPushButtonSettersCheckAgainstThePushButton` |
| `text.PDFTextStripper`, through `ProcessPage` and `WritePage` | 2 | **a difference, fixed 2026-09-15.** A `PDFTextStripperByArea` driven through `WriteText` wrote the page to the writer, where Java's `writeText` reaches the by-area `writePage` and fills the regions. `ExtractRegions`, the way the class is meant to be used, was right. `TestStripperByAreaWriteTextReachesItsWritePage` |

All four differences are fixed, each test-first with PDFBox's own answers; the
task file has the details.

#### The two that are left are fixes, not defects

`bug1175962.pdf` and `poppler-90-0-fuzzed.pdf` differ because
`track/java-bug-fixes` made the Go differ from the Java on purpose.
[`JAVA-BUGS.md`](JAVA-BUGS.md) 15 and 30 now say what each file shows: a
right-to-left run outside the basic plane that PDFBox splits into unpaired
halves, and an ASCIIHex content stream whose bad digits PDFBox turns into bytes
its parser reads past, where the Go's zeros end the page.

After both fixes:

```
  open    both 1436, neither 7, behind 0, ahead 0
  pages   0 disagree
  text    both 1434, neither 2, behind 0, ahead 0
  chars   1432 the same length, 2 not

  2 of 1443 files disagree (0.14%)
```

The seven that neither side opens are four `Missing root object specification in
trailer` and three `Page tree root must be a dictionary`, with the same message
from both; the two that neither extracts text from are `issue12823.pdf`, a
Type 0 font with no descendant, and `issue15604.pdf`, a number written `-.`.

And everything together — the 3,646 files above, pdf.js's 1,443, and
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

The 27 encrypted files are the earlier corpus's — 24 of qpdf's, one of
cabinet-of-horrors' and two PDFBox downloads — whose passwords are in qpdf's
test scripts and PDFBox's Java tests, and not yet in a table. Until they are,
both sides are only compared on refusing them.

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
