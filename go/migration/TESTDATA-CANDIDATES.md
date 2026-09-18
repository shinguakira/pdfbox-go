# Test data candidates

Where else there are PDFs worth reading, counted rather than guessed at.

## The split this file runs on

**PDFBox alone is the oracle.** The question "what is the right answer for this
document" has one source and it is the Java in this repository, run by
[`scripts/run-oracle.ps1`](scripts/run-oracle.ps1). No other implementation gets
a vote: iText, QPDF, PoDoFo, MuPDF and PDFium disagree with PDFBox in places, and
where they do, PDFBox is right *by definition* — this is a port of PDFBox and
matching it is the whole job.

**Everything is fair game as test data.** A PDF is an input. What another project
believed about it decides nothing, and a file written to break qpdf's lexer
breaks a lexer. The value in these repositories is that somebody already went and
found the documents that hurt, and named them after the thing they hurt.

So: harvest the files from all of them, take the answers from none of them. That
is what settles the licensing, two of these being AGPL:
[`TESTDATA.md`](TESTDATA.md)'s rule that nothing fetched is committed, modified
or redistributed applies unchanged here. Test *data* and test *approach* travel;
source does not.

## What is actually there

Counted 2026-09-13 against each repository's default branch; the on-disk state
checked 2026-09-16. PDFBox's own are tiers 0 and 1 of
[`TESTDATA.md`](TESTDATA.md), which carries their counts.

| Source | PDFs | Where | Licence | State |
| --- | ---: | --- | --- | --- |
| [itext-java](https://github.com/itext/itext-java) | **6,897** | `*/src/test/resources` | AGPL / commercial | **all fetched** 2026-09-16, with the passwords and keys its tests open 229 of them with |
| [itext-dotnet](https://github.com/itext/itext-dotnet) | **6,960** | the same, mirrored | AGPL / commercial | **all fetched** 2026-09-16, the same for 235 |
| iText's [other 45 repositories](https://github.com/itext) | **20,913** | 32 of the 45; see below | AGPL / commercial | **all fetched** 2026-09-16, with the passwords and keystores eight of them open 28 encrypted files with |
| [qpdf](https://github.com/qpdf/qpdf) | 717 | `qpdf/qtest/qpdf` 639, `qpdf/qtest/storage` 2, and 76 more | Apache-2.0 | **639 fetched**, 78 not |
| [pdfium](https://github.com/chromium/pdfium) | **301** | `testing/resources` | BSD-3-Clause | not fetched |
| [podofo-resources](https://github.com/podofo/podofo-resources) | **102** | a flat root, plus six directories | none declared | **all fetched** 2026-09-17, with the passwords PoDoFo's tests open 19 of them with |
| [mupdf](https://github.com/ArtifexSoftware/mupdf) | **0** | — see below | AGPL-3.0 | n/a |

Two of those numbers are worth stopping on.

**MuPDF carries no PDFs at all.** Its repository has zero `.pdf` files; the test
suite is Artifex's separate `tests.git`, which is served from
<https://cgit.ghostscript.com/cgi-bin/cgit.cgi/tests.git/> and reachable. So
"read MuPDF for its test data" means going somewhere else than the obvious
place, and reading MuPDF's *source* is a different activity with an AGPL
attached. Worth opening when the content stream interpreter, a colour space, a
font or the graphics state is the thing that disagrees — and read, not lifted.

**iText is an order of magnitude larger than everything else combined.** Nearly
seven thousand PDFs in each of the two repositories. They are the same library in
two languages, so the two sets largely mirror each other — the 63-file difference
suggests near-mirroring rather than two independent corpora, which means fetching
one is most of the value and fetching both is worth doing only to find where they
diverge.

MuPDF and pdfium are both ruled out as *dependencies* by the pure-Go rule —
go-fitz wraps MuPDF through cgo, go-pdfium ships a wasm blob. Nothing here
changes that; this is about reading their files.

## What each is good for

### PDFBox

The part **not** mined is the history rather than the files — the JIRA entries,
and the recorded reason each fix exists. PDF is not a format where handling
well-formed files is enough (broken xref tables, strange encodings, embedded
fonts, missing `ToUnicode`, malformed object streams), and PDFBox's own
documentation points at `examples` and the test code as additional sources,
including for known text-extraction problems.

### iText, both halves

The biggest haul on the page by a wide margin, and organised by what the file
exercises: `layout` 2,342, `kernel` 1,779, `svg` 1,446, `forms` 682, `sign` 386.
`kernel` and `forms` map almost directly onto this port's `cos`/`pdfparser` and
`pdmodel/interactive/form`.

There is a second thing here that is not test data and is worth writing down
while it is in view: iText has maintained the same library in Java and in C# for
years, the Java tree still carries `sharpen` configuration, and the two
repositories keep their APIs deliberately aligned. Diffing `PdfReader.java`
against `PdfReader.cs` is a ready-made catalogue of where porting a PDF library
across languages hurts — `InputStream`→`Stream`, `Closeable`→`IDisposable`,
collections, charsets, the crypto provider, and the unsigned-byte problem. That
catalogue was written for Java→C#; this port is Java→Go, so the particulars
differ and the *list of painful places* does not.
[`conventions/prior-art.md`](conventions/prior-art.md) is where that belongs if
it is ever pursued.

**The rest of iText's organisation — fetched, and no longer a candidate.**
`itext-java` and `itext-dotnet` are iText Core; the other 45 public repositories
of [github.com/itext](https://github.com/itext) are its add-ons, samples and
tools, counted on 2026-09-16 from each default branch's tree. Thirty-two of them
hold PDFs, 20,913 between them, and **all 20,913 are on disk and scored**: what
they are and what the two sides made of them is
[`TESTDATA.md`](TESTDATA.md), "iText's other repositories", and the work is I2
of [`tasks/track-testdata-itext.md`](tasks/track-testdata-itext.md). The counts
below are what was counted before the fetch, and the fetch matched every one of
them:

| Repository | PDFs | What they are |
| --- | ---: | --- |
| `itext-pdfhtml-java`, `itext-pdfhtml-dotnet` | 7,630, 7,649 | pdfHTML's tests: HTML and CSS in, PDF out, nearly all `cmp_` files iText wrote |
| `itext-publications-samples-dotnet`, `-examples-java` | 1,013, 762 | the published examples' expected output |
| `itextpdf`, `itextsharp` | 854, 849 | iText 5, the previous generation, with its own test resources |
| `i5js-sandbox` (archived) | 515 | iText 5 examples |
| `itext-publications-book-java`, `-highlevel-java`, `-jumpstart-java`, `-signatures-java`, `-signing-examples-java` | 331, 112, 48, 173, 7 | the books' examples |
| `itext-pdfsweep-java`, `itext-pdfsweep-dotnet` | 252, 254 | redaction inputs and results |
| `itext-pdfocr-java`, `itext-pdfocr-dotnet` | 169, 196 | OCR output |
| 16 more with any PDFs — `i5ns-book` 18, `i5js-book` 12, `itext-2022-customer-event` 11, `rups` 10, `itext-android-ui` 8, `pdfcop` 7, `itext-python-example` 7, `GIDS2026` 5, `ndi-demo` 4, `i5js-tutorial` 4, `i5ns-tutorial` 4, `i7js-zugferd` 3, `pdfdeserializer` 3, `AndroidPdfViewer` 1, `pdfchain` 1, `wtpdf-demo` 1 | 99 | |
| the other 13 | 0 | |

### QPDF

Already tier 2, and it earned its place: its files are behind three of the six
port defects [`TESTDATA.md`](TESTDATA.md) records the oracle finding, and eight
of the twelve disagreements. What it is strong at is exactly the low layer — lexer and parser, indirect objects, xref
tables and streams, object streams, incremental update, encryption, damaged-PDF
recovery.

The 78 not fetched are `qpdf/qtest/storage` 2, `examples/qtest` 49,
`compare-for-test/qtest` 21 and `libtests/qtest` 6; the suite takes
`qpdf/qtest/qpdf` alone, so reaching them is a path change.

One line of qpdf's test model this repository does not have: qpdf is in
OSS-Fuzz, and the malformed PDFs fuzzing finds are folded back into its
regression tests. Unit tests over well-formed PDFs, PDFBox's own regression
PDFs and qpdf-style malformed PDF tests are all here; fuzzing is not.

### PoDoFo

The smallest set and possibly the best-aimed. `podofo-resources` is a separate
repository, and its 102 PDFs are named after the thing they break:

```
RC4V2-40 / 56 / 80 / 96 / 128        every RC4 key length
AESV2-128, AESV3R6-256               and each with a *_KeyLengthNNNViolation twin
TestXRefRecovery1 / 2                xref recovery
TestFixInvalidCrossReferenceTable    a broken table that is meant to be repaired
TestImageInvalidLength               an image whose declared length is a lie
TestMalformedAnnotationAction        a malformed annotation
TestEncryptedStringsEscaped 1 / 2    encrypted strings needing escape
TextExtraction1 / 2 / 5, AllRotations, PredefinedCmap
YCCK-jpeg, YCbCr-jpeg, inline-image  colour and inline images
blank-rotated-90 / 270, blank-with-offset-start
```

plus `Corrupted/`, `ParserTests/` (7), `TechDocs/` (28, mostly XMP),
`PDFUA-Reference/`, `PQC/` (2, post-quantum signatures), and `Fonts/`,
`FontsTTC/`, `FontsType1/`, `Std14Fonts/`, `Charmaps/` for fonts.

A file per failure mode, which is the shape the tier-2 suites already have. **The
cheapest addition to `fetch-corpus.ps1` of anything here.** Taken on 2026-09-17:
every file opens on both sides, and the one text that differs is JAVA-BUGS 23;
see [`TESTDATA.md`](TESTDATA.md), "PoDoFo against the Java".

### PDFium

301 files in `testing/resources`, BSD-licensed, and the pixel tests come with
expected `.png` output beside them. Lower priority for a reason that is about
shape rather than quality: pdfium is a renderer, so its corpus leans where a
renderer leans.

## Order to take them in

PoDoFo was first, the cheapest addition of anything here, and was taken on
2026-09-17. Then qpdf's remaining 78, which is a path change to a suite already
there; then PDFium, BSD and bringing expected images with it; and MuPDF's
`tests.git` only when a renderer question needs it. Each is an open task in
[`tasks/track-testdata-sources.md`](tasks/track-testdata-sources.md). iText was
fourth on that list and is no longer on it: it was taken whole, both languages,
on 2026-09-16.

Each one that lands gets a row in [`TESTDATA.md`](TESTDATA.md) with what it
scored, and its line here should then say so.
