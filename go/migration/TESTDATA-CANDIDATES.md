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

So: harvest the files from all of them, take the answers from none of them.

That also settles the licensing, which is the same policy
[`TESTDATA.md`](TESTDATA.md) already runs on and needs saying again here because
two of these are AGPL: **nothing fetched is committed, modified or
redistributed.** It is downloaded into a gitignored directory, read as input, and
that is all. Test *data* and test *approach* travel; source does not.

## What is actually there

Counted 2026-09-13 against each repository's default branch.

| Source | PDFs | Where | Licence | State |
| --- | ---: | --- | --- | --- |
| [itext-java](https://github.com/itext/itext-java) | **6,897** | `*/src/test/resources` | AGPL / commercial | not fetched |
| [itext-dotnet](https://github.com/itext/itext-dotnet) | **6,960** | the same, mirrored | AGPL / commercial | not fetched |
| [qpdf](https://github.com/qpdf/qpdf) | 717 | `qpdf/qtest/qpdf` 639, `qpdf/qtest/storage` 2, and 76 more | Apache-2.0 | **639 fetched**, 78 not |
| [pdfium](https://github.com/chromium/pdfium) | **301** | `testing/resources` | BSD-3-Clause | not fetched |
| [podofo-resources](https://github.com/podofo/podofo-resources) | **102** | a flat root, plus six directories | none declared | not fetched |
| [mupdf](https://github.com/ArtifexSoftware/mupdf) | **0** | — see below | AGPL-3.0 | n/a |
| PDFBox | 179 committed + 78 declared | this repository | Apache-2.0 | in use, tiers 0 and 1 |

Two of those numbers are worth stopping on.

**MuPDF carries no PDFs at all.** Its repository has zero `.pdf` files; the test
suite is Artifex's separate `tests.git`, which is served from
<https://cgit.ghostscript.com/cgi-bin/cgit.cgi/tests.git/> and reachable. So
"read MuPDF for its test data" means going somewhere else than the obvious
place, and reading MuPDF's *source* is a different activity with an AGPL
attached.

**iText is an order of magnitude larger than everything else combined.** Nearly
seven thousand PDFs in each of the two repositories. They are the same library in
two languages, so the two sets largely mirror each other — the 63-file difference
suggests near-mirroring rather than two independent corpora, which means fetching
one is most of the value and fetching both is worth doing only to find where they
diverge.

## What each is good for

### PDFBox

Already the backbone: tiers 0 and 1 of [`TESTDATA.md`](TESTDATA.md) are its
committed test resources and the 78 files its poms download, and every one of the
latter is the reduced reproducer of a numbered issue.

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

### QPDF

Already tier 2, and it earned its place: qpdf's files found four of the twelve
disagreements the oracle run turned up. What it is strong at is exactly the low
layer — lexer and parser, indirect objects, xref tables and streams, object
streams, incremental update, encryption, damaged-PDF recovery.

Two things are still on the table:

- **78 PDFs not being fetched.** The suite takes `qpdf/qtest/qpdf` and leaves
  the two beside it in `qpdf/qtest/storage`, `examples/qtest` (49),
  `compare-for-test/qtest` (21) and `libtests/qtest` (6). Recounted 2026-09-15
  from the repository tree: 639 + 2 + 49 + 21 + 6 = 717. The first count missed
  the two in `qpdf/qtest/storage`, so its fetched and not fetched added up to
  715.
- **The test model, of which one line is missing here.** qpdf rasterises some
  PDFs and compares the image; it is in OSS-Fuzz, and malformed PDFs that fuzzing
  finds are folded back into the regression tests. This port has the first three
  lines and not the fourth:

  ```
  unit tests over well-formed PDFs        ✓
  PDFBox's existing regression PDFs       ✓
  qpdf-style malformed PDF tests          ✓
  fuzzing, folded back into tests         ✗
  ```

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
cheapest addition to `fetch-corpus.ps1` of anything here.**

### PDFium

301 files in `testing/resources`, BSD-licensed, and the pixel tests come with
expected `.png` output beside them. Lower priority for a reason that is about
shape rather than quality: pdfium is a renderer, so its corpus leans where a
renderer leans.

### MuPDF

Lowest priority, and the entry is mostly a correction: no PDFs in the repository,
AGPL on the source, and a test suite that lives at Artifex's `tests.git`. Worth
opening when the content stream interpreter, a colour space, a font or the
graphics state is the thing that disagrees — and read, not lifted.

Both MuPDF and pdfium are already ruled out as *dependencies* by the pure-Go rule
(go-fitz wraps MuPDF through cgo; go-pdfium ships a wasm blob). Nothing here
changes that; this is about reading their files.

## Order to take them in

1. **PoDoFo** — 102 files, one per failure mode, a few lines in
   `fetch-corpus.ps1`. Best ratio on the page.
2. **qpdf's remaining 78** — the suite is already there; it is a path change.
3. **pdfium** — 301, BSD, and brings expected images with it.
4. **iText** — 6,897, and the reason it is fourth rather than first is that it is
   larger than everything above put together and wants its own decision about how
   much of it to carry.
5. **MuPDF via `tests.git`** — only when a renderer question needs it.

Each one that lands gets a row in [`TESTDATA.md`](TESTDATA.md) with what it
scored, and its line here should then say so.
