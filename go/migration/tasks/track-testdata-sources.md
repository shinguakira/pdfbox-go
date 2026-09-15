# track/testdata-sources

Collect the PDFs other PDF libraries keep as test data or attach to their fixes,
run the Go version against PDFBox over them, and find the defects they expose.

**Branch: `track/testdata-sources`** — from and back to `migration-base`.
Reopened on 2026-09-15 at `migration-base` `195936170`, carrying the two
`track/performance` commits the merge of that branch did not include: the page
tree asking the document for its resource cache, and the decompressor pool's
record and tests.

What the test data is and how to fetch it is in
[`../TESTDATA.md`](../TESTDATA.md); what else there is to fetch, counted, is in
[`../TESTDATA-CANDIDATES.md`](../TESTDATA-CANDIDATES.md). This file is where the
work stands and what is left.

## Rules

- **The Java tree is read-only**, and PDFBox is the reference: a disagreement is
  the Go version's to explain, never the Java's to change.
- **Nothing fetched is committed.** Other projects' documents stay under their
  own licences; `go/testdata/corpus/` and `go/testdata/oracle/` are ignored.
- **A disagreement is fixed test-first**, with PDFBox's own output as the
  expected value. A bug found in the Java is recorded in
  [`../JAVA-BUGS.md`](../JAVA-BUGS.md), not fixed in the Java.
- **The oracle never gets worse.** After any change to the Go version,
  `corpus -oracle` must not report more disagreements than before it.
- **Pure Go**, and no new dependency without a decision.

## Where the test data stands, 2026-09-15

### On disk

| Source | PDFs | Where | Compared with PDFBox |
| --- | ---: | --- | --- |
| PDFBox, committed | 179 — 166 in the Java test resources, 13 elsewhere | the repository | through the ported Java tests |
| PDFBox, downloaded by its build | 58 PDFs, among 74 of the 78 files the poms declare | `pdfbox/target`, `examples/target`, `fontbox/target` | yes |
| veraPDF corpus | 2,908 | `go/testdata/corpus/verapdf` | yes |
| qpdf, `qpdf/qtest/qpdf` | 639 | `go/testdata/corpus/qpdf` | yes |
| pdf.js, everything its tests use | 1,443 — 982 committed in `test/pdfs`, 459 named by `.link` stubs, 2 committed elsewhere | `go/testdata/corpus/pdfjs` | yes, with the passwords its tests use for its 12 encrypted files |
| cabinet-of-horrors | 24 | `go/testdata/corpus/cabinet-of-horrors` | yes |
| SafeDocs targeted | 11 | `go/testdata/corpus/safedocs-targeted` | yes |
| PDF 2.0 examples | 7 | `go/testdata/corpus/pdf20examples` | yes |
| text-rendering-tests | none; test fonts | `go/testdata/corpus/text-rendering-tests` | no — shaping ground truth, not PDFs |

The four downloads that no longer fetch are all in `benchmark` and read by no
test.

### Not fetched

| Source | PDFs | How it would be fetched | What it is for |
| --- | ---: | --- | --- |
| pdf.js `test/pdfs/sig_corpus` | 8 | generated, not downloaded — see U7 | signed PDFs for testing Firefox's signature panel by hand |
| PoDoFo `podofo-resources` | 102 | not in `fetch-corpus.ps1` yet | one file per failure mode: every RC4 and AES key length, xref recovery, broken tables |
| qpdf, the rest | 78 | a change to the qpdf suite's subtree | `qpdf/qtest/storage` 2, `examples/qtest` 49, `compare-for-test/qtest` 21, `libtests/qtest` 6 |
| PDFium `testing/resources` | 301 | not in the script | renderer input, with the expected PNGs beside it |
| iText, Java and .NET | 6,897 and 6,960 | not in the script | the largest set by far; AGPL, and wants a decision on how much to carry |
| MuPDF `tests.git` | not counted | a separate repository | renderer questions only |
| SafeDocs issue-tracker corpus, batch 1 | 1.8 GB | tier 3 | every PDF attached to a PDFBOX issue; may contain malicious files |

### Compared with PDFBox

Over all 5,090 files on disk outside the repository:

```
5090 files, 27 of them encrypted and skipped
  open    both 5037, neither 53, behind 0, ahead 0
  pages   0 disagree
  text    both 5032, neither 5, behind 0, ahead 0
  chars   5030 the same length, 2 not

  2 of 5090 files disagree (0.04%)
```

- **2 disagreements, both Java bugs the Go fixes on purpose.**
  `pdfjs/bug1175962.pdf` is JAVA-BUGS 15 and `pdfjs/poppler-90-0-fuzzed.pdf` is
  JAVA-BUGS 30; each entry now says what the file shows. See U6.
- **27 encrypted files are compared only on refusing them.** See U8.
- Everything else agrees, `PDFBOX-4131-0.pdf` included.

**What "agrees" covers.** The comparison is whether a document opens, how many
pages it has, whether its text extracts, and **how many characters** that text
has. It does not compare which characters they are, nor rendering, forms,
annotations, saving, or image extraction. A wrong character in the right place,
or a wrong pixel, passes.

### Found and fixed on this branch

Each one test-first, with the expected values printed by the running PDFBox for
the same input, and the whole corpus compared again afterwards: 2 of 5,090 still
disagree, the same two.

- **Type 0 glyphs were advanced by the font program instead of `/W`.**
  `pdFont.Displacement` called its own `Width` where Java's `getWidth` is a
  virtual call. It was behind 12 of pdf.js's 14 first disagreements and behind
  `PDFBOX-3951`, and it moved Type 0 glyphs in rendering too. Fixed in
  `go/pdfbox/pdmodel/font/pdfont.go`; pinned by
  `TestType0DisplacementIsTheDescendantWidth`. `TESTDATA.md`, "pdf.js against
  the Java", has the rest.
- **The Type 0 half of the vertical width scaling was never ported.**
  `LegacyPDFStreamEngine.ShowGlyph` scaled a vertical `PDTrueTypeFont`'s width
  by 1000 / `unitsPerEm` and not a `PDType0Font`'s over a `PDCIDFontType2`.
  Ported in `go/pdfbox/text/legacystreamengine.go`; pinned by
  `TestVerticalGlyphWidthIsScaledByTheCIDFontsEm`. No corpus file reaches it.
- **A repeated filter was decoded twice by the reader and the view (U2).**
  `Filter.decode` keeps the first of each filter and reads `/DecodeParms` at
  each one's place in the reduced list; `createInputStream`, `createView` and
  `PDStream.createInputStream(List<String>)` all go through it. The port reduced
  the list for the last only. The reduction is now in `Stream.decode`
  (`go/pdfbox/cos/stream.go`), which all three reach.
  `TestRepeatedFilterIsDecodedOnceOnEveryPath` and
  `TestRepeatedFilterIsWrittenEveryTime` in `repeatedfilter_test.go`.
  `TestStreamTwoFilterChain`, which expected `[/Fl /Fl]` to come back whole, is
  replaced by ports of the two Java tests it stood in for.
- **A failing source ended the Flate data quietly (U4).** `Flate.Decode` ended
  the output at every error, and it and `NewFlateDecoderReader` took a header
  they could not read as an empty stream. PDFBox lets the source's
  `IOException` out of both and swallows only corrupt data and a source that
  ends. `go/pdfbox/filter/flate.go`; `TestFlateLetsAFailingSourceOut`.
- **`PDButton`'s setters did not reach `PDPushButton` (U9).** `SetValue`,
  `SetValueIndex` and `SetDefaultValue` checked against `PDButton`'s own
  `OnValues` and `ExportValues`, so a push button with an On appearance or an
  `/Opt` accepted values PDFBox refuses. `PDButton` now asks through the field's
  `self` (`go/pdfbox/pdmodel/interactive/form/pdbutton.go`);
  `TestPushButtonSettersCheckAgainstThePushButton`.
- **`PDFTextStripperByArea` driven through `WriteText` wrote the page (U10).**
  The base `ProcessPage` called its own `WritePage`. It now writes through a
  hook the by-area stripper installs its own in, the by-area copy of
  `ProcessPage` is gone, and `charactersByArticle` points at a shared list
  object, so that clearing and extending a region's lists in place reaches the
  region as Java's shared `ArrayList` does. `go/pdfbox/text/pdftextstripper.go`,
  `pdftextstripperbyarea.go`; `TestStripperByAreaWriteTextReachesItsWritePage`.
- **An unsupported OpenType font dereferenced a nil glyph table.**
  `OpenTypeFont.GetPath` on a font with CFF2 outlines and no CFF reaches
  `TrueTypeFont.getPath`, whose `getGlyph` is `OpenTypeFont`'s and throws "OTF
  fonts do not have a glyf table". The port's reached its own and dereferenced
  nil. `go/fontbox/ttf/truetypefont.go`, `opentype.go`;
  `TestOpenTypeWithoutCFFRefusesAGlyphTable`.
- **`RangeForOutput` on the identity function panicked.** PDFBox returns a range
  over no array, which fails only when it is read; the port's base method read
  the identity function's missing dictionary. `go/pdfbox/pdmodel/common/function/pdfunction.go`;
  `TestIdentityRangeForOutputHasNoArray`.
- **`/Identity` was accepted as a filter name.** `FilterFactory` refuses it with
  "Invalid filter"; `ByName` answered it, with a note that the parser depended
  on it. Nothing did. `go/pdfbox/filter/filter.go`; `TestByNameCoversEveryFilter`.

## Open tasks

### Unresolved

- [x] **U1. `PDFBOX-3951` is one character out.** Closed 2026-09-15 by the Type 0
  displacement defect above; its text is now identical to PDFBox's.
- [x] **U2. A repeated filter is decoded twice.** Closed 2026-09-15; above.
- [x] **U3. The PDFBox table is short a file.** Closed 2026-09-15:
  `testdata/oracle/java-corpus.tsv` regenerated over every file on disk, 5,090
  rows.
- [x] **U4. `Flate.Decode` swallows a failing source.** Closed 2026-09-15, by
  matching Java; above.
- [x] **U5. `BENCHMARK.md` predates the performance work.** Closed 2026-09-15:
  every table on the page measured again on the current code, and the README's
  with it. The rest of the machine used between 0.3 and 1.3 cores of twelve
  during the runs, recorded for each; a Perl process looping since 2026-09-12,
  left behind by an earlier session's scratch script, was stopped first. The
  port's best pass is 8,234 ms against PDFBox's 5,625, 1.46× where it was 6.4×;
  with four workers it is the faster of the two, and it uses 2.9× less CPU.
- [ ] **U6. Decide whether JAVA-BUGS 30's fix stays.** On
  `pdfjs/poppler-90-0-fuzzed.pdf` the fix makes the Go extract 1,197 characters
  where PDFBox extracts 1,402: the bad ASCIIHex digits read as zero leave a
  control byte that ends page 10's content stream, where PDFBox's -1 arithmetic
  leaves bytes its parser reads past. The entry has the bytes.
- [ ] **U7. pdf.js's `sig_corpus`, eight PDFs.** pdf.js neither commits nor
  downloads them: `test/pdfs/sig_corpus/generate.py` builds them, and needs a
  built mozilla-central checkout for its `pycms.py` and vendored Python modules.
  They are not encrypted. What they hold is signatures — a detached PKCS#7 over
  `/ByteRange` in an AcroForm `/Sig` field, one with a byte changed after
  signing, one with an unsupported `/SubFilter`, and two signatures by
  incremental update — and the comparison as it stands, open, pages, text and
  its length, would see a one-page Helvetica document and the incremental update
  and nothing of the signatures. Having them means writing that generator in Go,
  a CMS signer over mozilla-central's published test keys, and comparing
  signature fields; or deciding they are out of scope. Needs a decision.
- [ ] **U8. The earlier corpus's 27 encrypted files have no passwords.** 24 of
  qpdf's, `cabinet-of-horrors/encryption_openpassword.pdf`, and
  `PDFBOX-4517-cryptfilter.pdf` and `PDFBOX-5639.pdf`. Opening them is half of
  what they test — key derivation and the password check for R2 to R6, RC4 and
  AES, long passwords, short O and U, a bad `/Length`, crypt filters — and the
  pages and text after opening are the other half, the decryption of strings and
  streams. Only the passwords are needed from their sources; qpdf's expected
  output is qpdf's, and PDFBox stays the reference. Where they are, as far as
  checked on 2026-09-15: the two PDFBox files in its Java tests,
  `userpassword1234` in `TestFilters` and `JUL2023rfi` in
  `TestSymmetricKeyEncryption`; most of qpdf's in `qpdf/qtest/*.test`
  (`encryption.test`, `check-encryption.test`, `encryption-parameters.test`),
  and some in the file name, `U=view,O=master`; the rest not yet found. **The
  format corpus's `readme.md` does not give `encryption_openpassword.pdf`'s
  password** — it says only that the file needs one — so that file may have none
  to be had. qpdf's tests open files with the user and the owner password both,
  and a table with both would compare the owner path too.
- [x] **U9. `PDButton.SetValue` and `SetValueIndex` on a `PDPushButton`.** Closed
  2026-09-15; above, with `SetDefaultValue`.
- [x] **U10. `PDFTextStripperByArea` driven through `WriteText`.** Closed
  2026-09-15; above.

### Not yet done

- [ ] **T1.** Fetch PoDoFo's 102 and score them against PDFBox.
- [ ] **T2.** Fetch qpdf's remaining 78.
- [x] **T3.** Fetch pdf.js's test PDFs and score them. Done 2026-09-15: 1,443 on
  disk, all compared; the eight of U7 are not on disk.
- [ ] **T4.** Fetch PDFium's 301.
- [ ] **T5.** Decide how much of iText's 6,897 to carry, then fetch it.
- [ ] **T6. Compare text content, not only its length.** A hash of each
  document's text, and of each page's to find where two differ, from both
  `JavaCorpus` and `cmd/corpus`. This is the change most likely to find defects
  the current comparison cannot see.
- [ ] **T7.** Compare rendering with PDFBox, page by page.
- [ ] **T8.** MuPDF's `tests.git`, when a renderer question needs it.
- [ ] **T9.** Fuzzing, with what it finds folded back into tests — the one line of
  qpdf's test model this repository does not have.
