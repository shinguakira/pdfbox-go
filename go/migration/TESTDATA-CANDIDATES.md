# Test data candidates

Projects worth mining for test input and for the knowledge around it, written
down before any of them is acted on.

This is a survey, not a decision. Nothing here has been fetched, measured or
licence-checked by the port; [`TESTDATA.md`](TESTDATA.md) is the file that
records what actually is in use and what it scored. When one of these moves from
candidate to corpus, it gets a row there and a suite in
[`scripts/fetch-corpus.ps1`](scripts/fetch-corpus.ps1), and its entry here should
say so.

**Two of the five are AGPL.** iText and MuPDF both are. Studying how a problem
was solved and copying the code that solves it are different acts, and only the
first is available here. Test *data* and test *approach* travel; source does not.

---

## 1. PDFBox itself

The most important source, and not mainly for its code.

The issues and the JIRA history, the tests, and the recorded reasons for past
fixes carry more than the source does. PDF is not a format where handling
well-formed files is enough: broken xref tables, strange encodings, embedded
fonts, missing `ToUnicode`, malformed object streams — there is a great deal of
it in the wild, and each fix in PDFBox's history is a record of one real file
that broke something.

PDFBox says as much itself: `examples` and the test code are named as additional
sources of information, and known text-extraction problems are recorded there.

**Status here:** already the backbone. The 179 committed test documents, the 78
the poms download, and the JIRA reproducers behind them are what
[`TESTDATA.md`](TESTDATA.md) tiers 0 and 1 are made of. What is *not* yet mined
is the issue history as a source in its own right — the reasons, rather than the
files.

## 2. iText, Java and .NET

Worth looking at, and for a reason that is not "it is another PDF library".

iText has maintained **the same library in Java and in C# for years**. The Java
side still carries `sharpen` configuration — the Java-to-C# conversion machinery
is in the source tree — and `itext-java` and `itext-dotnet` exist as separate
repositories with deliberately aligned APIs.

So a file-by-file comparison is available:

```
iText, Java              iText, .NET
  PdfReader.java    ⟷      PdfReader.cs
```

and what falls out of it is a catalogue of where porting a PDF library across
languages actually hurts:

```
InputStream       →  Stream
IOException       →  IOException
byte / unsigned   →  ?
Closeable         →  IDisposable
Java collections  →  .NET collections
charset handling  →  Encoding
crypto provider   →  .NET / BouncyCastle
```

**Licence:** AGPL / commercial. Research the design and the solutions; do not
copy the code.

**Note for this repository:** the comparison above is Java→C#. This port is
Java→Go, so the mapping differs in its particulars — Go has no `IDisposable`, no
checked exceptions, and `byte` is already unsigned — but the *list of places that
hurt* transfers almost unchanged, and it is the list that is valuable.
[`conventions/prior-art.md`](conventions/prior-art.md) covers PdfPig and the
IKVM .NET build for the same reason; iText belongs beside them.

## 3. QPDF

A strong source for getting the low layer right and keeping it hard to break.

What to read:

- lexer and parser
- indirect objects
- xref table and xref stream
- object streams
- incremental update
- encryption
- damaged PDF recovery

Its tests are unusually practical — some rasterise the PDF and compare the
image. It is in OSS-Fuzz, and malformed PDFs that fuzzing finds are folded back
into the regression tests.

That operating model is worth taking wholesale:

```
unit tests over well-formed PDFs
        +
PDFBox's existing regression PDFs
        +
QPDF-style malformed PDF tests
        +
fuzzing
```

**Status here:** the corpus already carries qpdf's 639 test files
([`TESTDATA.md`](TESTDATA.md) tier 2), and they found four of the twelve
disagreements with PDFBox. What is *not* taken yet is the fourth line — nothing
in this port fuzzes, and nothing folds a fuzz finding back into a test.

## 4. PoDoFo

Unglamorous, and a genuine treasure house of test data.

There is a dedicated `podofo-resources` repository, laid out by the thing being
broken:

```
Corrupted/
Fonts/
FontsTTC/
FontsType1/
ParserTests/
PDFUA-Reference/
XMP/
```

with real cases in it:

```
invalid xref            encrypted strings
malformed annotation    AES
invalid image length    CID fonts
signatures              font width
```

Useful for exactly one question: can this implementation eat the PDFs that exist
in the world.

**Status here:** not fetched. The most obvious next addition to
`fetch-corpus.ps1` of anything on this page — it is organised the way the other
tier-2 suites are, one directory per failure mode.

## 5. MuPDF and PDFium

Lower priority.

The parser knowledge is very deep, but both are built as renderers and viewers,
which puts them at a distance from a PDFBox port. MuPDF is worth opening when
stuck on the content stream interpreter, colour spaces, fonts or graphics state;
it remains a large, mostly portable C implementation.

**Licence:** MuPDF is AGPL. Same rule as iText — read, do not lift.

**Note for this repository:** both are already ruled out as *dependencies* by the
pure-Go rule (go-fitz wraps MuPDF through cgo, go-pdfium ships a wasm blob).
Nothing changes about that. This entry is about reading them, and about their
test corpora.

---

## Where each would land

| Candidate | As test data | As knowledge | Next step |
| --- | --- | --- | --- |
| PDFBox | tiers 0 and 1, in use | the issue history, unmined | mine the reasons, not just the files |
| iText | — | Java↔C# diff, unmined | read `PdfReader` on both sides, write down the mapping |
| QPDF | tier 2, in use | the four-layer test model | add fuzzing; fold findings back |
| PoDoFo | **not fetched** | — | add a suite to `fetch-corpus.ps1` |
| MuPDF / PDFium | not fetched | renderer internals | open when the raster disagrees |
