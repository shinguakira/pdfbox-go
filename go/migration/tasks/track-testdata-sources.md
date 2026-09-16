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

## Where the test data stands

What is on disk, tier by tier, and what each suite reaches:
[`../TESTDATA.md`](../TESTDATA.md). What is not fetched yet, counted per
project: [`../TESTDATA-CANDIDATES.md`](../TESTDATA-CANDIDATES.md). Where the
comparison against PDFBox stands, over every file on disk outside the
repository: `TESTDATA.md`, "Where it stands".

Two files disagree, both Java bugs the Go fixes on purpose — JAVA-BUGS 15 and
30, and whether the second fix stays is U6. The 27 encrypted files that are
compared only on refusing them are U8. Since 2026-09-16 the comparison carries a
digest of the text as well as its length, and iText's 13,857 files are in it;
[`track-testdata-itext.md`](track-testdata-itext.md) is that branch's record and
it merged into this one.

### Found and fixed on this branch

Each one test-first, with the expected values printed by the running PDFBox for
the same input, and the whole corpus compared again afterwards: 2 of 5,090 still
disagree, the same two.

- **Type 0 glyphs were advanced by the font program instead of `/W`.**
  `pdFont.Displacement` called its own `Width` where Java's `getWidth` is a
  virtual call. It was behind 12 of pdf.js's 14 first disagreements and behind
  `PDFBOX-3951`, and it moved Type 0 glyphs in rendering too. Fixed in
  `go/pdfbox/pdmodel/font/pdfont.go`; pinned by
  `TestType0DisplacementIsTheDescendantWidth`.
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

### Found in review, 2026-09-16

Three findings of the branch's review, and two defects that checking the third
turned up. Each test-first, with PDFBox's output as the expected value. The
comparison over the 5,090 files, run again with all five in: the same 2
disagreements, and every file's row — open, pages, text, length — the same as
before them.

- **A codec that cannot be compared panicked the reduction of repeated
  filters.** The reduction above kept a map keyed on each filter's
  `StreamCodec`. That is an interface a `CodecProvider` answers, and nothing
  makes the value behind it comparable: one holding a slice panicked with "hash
  of unhashable type" on every stream with two filters or more, before
  decoding began. A codec that can be compared is still its own key — the
  provider in `pdfbox/filter` answers values that are equal exactly where
  Java's `FilterFactory` hands out the same instance — and one that cannot is
  keyed by the name the filter array gives it. `go/pdfbox/cos/stream.go`;
  `TestRepeatedFilterReductionTakesACodecThatCannotBeCompared`, which reads
  through `CreateReader`, `CreateView` and `CreateReaderStopping`, panicked
  before the change.
- **A pdf.js fetched by the earlier script could not be brought up to date.**
  When the linked files came in, the script started copying
  `test/test_manifest.json` into `_repo/`, and the links and the passwords are
  read from it. A `pdfjs`
  fetched before that has no manifest, and a rerun without `-Force` stopped at
  reading it, before any linked file. A present suite now has its missing
  extras fetched first, from the archive, and only those; its own files and the
  linked files already downloaded stay as they are.
  `go/migration/scripts/fetch-corpus.ps1`. Run with `_repo` moved aside: the
  rerun fetched the three files, which came back identical to the ones moved
  aside, and went on to find all 459 linked files present with the manifest's
  md5.
- **Nothing took a processed page's resources out of the cache.** Since
  `track/performance` every page reads through the document's cache, and
  `DefaultResourceCache` keeps its fonts, colour spaces, graphics states,
  patterns, property lists, shadings and XObjects. Java's
  `PDFTextStripper.processPage` ends with `page.removePageResourceFromCache()`;
  the port had left the call out, with a note that its pages had no cache, and
  had no such method on `PDPage`. Java's cache also holds its entries through a
  `SoftReference` the collector may clear, and the port's holds them outright,
  so everything any page read stayed for as long as the document was open.
  `PDPage.RemovePageResourceFromCache` is now ported — the page's own
  resources, not inherited ones; a Type 0 font's descendant font and its
  descriptor; and the resources of each form XObject removed, recursively — and
  `ProcessPage` calls it. `go/pdfbox/pdmodel/pdpage.go`,
  `go/pdfbox/text/pdftextstripper.go`;
  `TestRemovePageResourceFromCacheLeavesTheInheritedResources`, fifteen values
  PDFBox printed for the same objects, and
  `TestWriteTextTakesEachPagesResourcesOutOfTheCache`.

**The purge, checked over the corpus.** The tests pin the method; they do not
say that text extraction leaves the same cache behind. So both caches were
wrapped to count what text extraction put in and what was still in at the end,
per kind, over every one of the 5,037 files that open on both sides. The Go
side is `go/testdata/oracle/purgeprobe`, which git ignores; the PDFBox side was
a throwaway program wrapping `DefaultResourceCache` the same way. The count
found two defects, both fixed:

1. **A transparency group's resources stayed.** The recursion took a
   `*form.PDFormXObject` only. Java's `instanceof PDFormXObject` also takes the
   `PDTransparencyGroup` that extends it; the Go type embeds it instead, and
   has to be named. `pdfjs/geothermal.pdf` kept 7 XObjects PDFBox takes out.
   Added to the first test above, which failed on it.
2. **An AES-128 stream of the initialization vector alone could not be read.**
   This one is not in the purge; the count found it. `aesCBC` answered "no data to
   decrypt" where Java's `Cipher.doFinal` decrypts nothing to nothing, and on
   the AES-128 path that error made the whole object unreadable.
   `pdfjs/ichiji.pdf` has two such form XObjects, 16 bytes each; PDFBox reads
   them as empty forms, and the port read them as null — the text did not
   change, because they are empty. `go/pdfbox/pdmodel/encryption/securityhandler.go`;
   `TestAESPathsAnswerShortInputAsPDFBoxDoes`, twelve rows PDFBox's two AES
   paths answered when called through reflection, of which this was the one
   that failed.

With both fixed, 5,021 of the 5,037 files leave PDFBox's cache exactly: the same
entries put in, kind by kind, and the same left at the end. 15 fail on both
sides — 10 of pdf.js's encrypted files, which the count opened without their
passwords, and the 5 whose text extraction fails on both sides in the table
above. The one that differs is `pdfjs/poppler-90-0-fuzzed.pdf`, JAVA-BUGS 30
again: the port's page 10 content stream ends sooner, so it reads one font
fewer, 9 put and 2 left where PDFBox has 10 and 3.

**What the purge lets go**, measured on three long documents with the purge and
with a cache that ignores removals: how much more heap is live once text
extraction has walked every page than before it began, the document still
open, and how many entries the cache still holds.

| document | pages | without | with | entries left, without → with |
| --- | ---: | ---: | ---: | --- |
| `pdfjs/geothermal.pdf` | 372 | 27.5 MB | **10.9 MB** | 836 → 102 |
| `pdfjs/issue2386.pdf` | 141 | 14.6 MB | 14.1 MB | 1,088 → 100 |
| `pdfbox/target/pdfs/PDFBOX-3949-…` | 234 | 16.6 MB | 16.4 MB | 40 → 21 |

**What it costs.** `BENCHMARK.md`'s 3,597 documents, one pass each, alternating
three times between a build with the purge and one without it — the same code
with the call left out through `go build -overlay`, so the tree was not changed
for it. Both extract 1,834,959 characters and fail none. With the purge a pass
allocates 6,208 MB where it allocated 5,944, **4.4% more**, in every round: a
resource pages share is read again after each purge, until the stable-cache
bookkeeping keeps it, as in PDFBox. Time did not come apart from the noise: CPU
time 25.7–27.3 s without and 24.3–27.8 s with, and the peak live heap 242–259 MB
without and 240–271 MB with, while other programs, a game among them, used 4.8
to 6.2 cores of the twelve. `BENCHMARK.md` was measured before the
purge and has not been measured again; that is T10.

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
- [ ] **U11. Outside text extraction, the cache still keeps everything.** Found
  2026-09-16, with the purge above. `removePageResourceFromCache` has one caller
  in PDFBox, `PDFTextStripper.processPage`. Rendering a page, or any other
  stream engine walking pages, leaves what it read in the document's cache,
  and Java lets that go when memory runs short, through the cache's
  `SoftReference`s; the port has no such reference, so a document rendered page
  by page holds every page's fonts, colour spaces and XObjects until it is
  closed. The purge is public, so a caller can call it after each page; whether
  the port's renderer should, where PDFBox's does not, needs a decision.

### Not yet done

- [ ] **T1.** Fetch PoDoFo's 102 and score them against PDFBox.
- [ ] **T2.** Fetch qpdf's remaining 78.
- [x] **T3.** Fetch pdf.js's test PDFs and score them. Done 2026-09-15: 1,443 on
  disk, all compared; the eight of U7 are not on disk.
- [ ] **T4.** Fetch PDFium's 301.
- [x] **T5.** Decide how much of iText's 6,897 to carry, then fetch it. Done
  2026-09-16 on `track/testdata-itext`, which the user had made from this branch
  for it: all of it, the Java repository's 6,897 and the .NET one's 6,960. See
  [`track-testdata-itext.md`](track-testdata-itext.md).
- [ ] **T6. Compare text content, not only its length.** A hash of each
  document's text, and of each page's to find where two differ, from both
  `JavaCorpus` and `cmd/corpus`. This is the change most likely to find defects
  the current comparison cannot see. **Half done 2026-09-16 on
  `track/testdata-itext`:** both tables carry a digest of each document's text,
  and `-oracle` compares it. Per page is not done.
- [ ] **T7.** Compare rendering with PDFBox, page by page.
- [ ] **T8.** MuPDF's `tests.git`, when a renderer question needs it.
- [ ] **T9.** Fuzzing, with what it finds folded back into tests — the one line of
  qpdf's test model this repository does not have.
- [ ] **T10. Measure `BENCHMARK.md` again with the page purge in.** Its tables
  are the code before 2026-09-16. Not done that day because the machine was not
  quiet: the page's runs had the rest of it at 0.3 to 1.3 cores, and it was at
  4.8 to 6.2. What is known is above: 4.4% more allocated, the same output.
