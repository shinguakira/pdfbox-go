# Porting status

Hand maintained. Update the row for a package in the same commit that ports it.

The machine-generated counterpart is
[`mapping/inventory.tsv`](mapping/inventory.tsv), which knows how much Java
source sits behind each row but only distinguishes "has Go files" from "has
none". This file is where partial work and the reasons for it get recorded.

Status values: `done` · `in progress` · `blocked` · `not started` · `out of scope`

Last updated: 2026-09-16

## Summary

| Phase | Area | Java files | Status |
| --- | --- | ---: | --- |
| 0 | `pdfio` | 18 | **done — all 18 files**, finished by `track/scratchfile` |
| 1 | `pdfbox/cos` | 24 | **done — 22 of 24 ported**. Slice 7 closed the incremental-save deferral: `COSIncrement`, `COSUpdateInfo` and `COSUpdateState` are in. The two left are deliberate — `COSInputStream`, which exists in Java only to carry a `DecodeResult`, and `COSOutputStream`, folded into `streamWriter` |
| 2 | `filter`, `pdfparser`, `pdfwriter` | 48 | **done — all 48**. `filter` 23 of 23, finished by slice 6, `DecodeOptions` included; `pdfparser` all 18 including `FDFParser`; `pdfwriter` all 7, `getDataToSign` included |
| 3 | `pdfbox/pdmodel` | 433 | **done bar names left unported on purpose** — every file of `interactive`, `documentinterchange`, `fdf`, `fixup`, `common`, `graphics/optionalcontent`, `graphics/pattern` and `graphics/form`, and the model half of `graphics/shading`; `pdmodel/font` **at 39 of 39**, finished by `track/font-embedding`, all 12 encodings, `pdmodel/encryption` at 17 of 19, the two left being JCE lookups `crypto/*` answers. The 19 `java.awt.Paint` and `PaintContext` classes of `graphics/shading` stay unported by name; `track/raster` ported their arithmetic into `rendering/raster` |
| 4 | `fontbox` | 143 | **done — all 143 files**, finished by slice 4 |
| 5 | `contentstream`, `text` | 85 | **done — all 85 files**, finished by slice 9: the graphics engine, all 23 graphics operators, all 13 colour operators and the three `DrawObject`s |
| — | `awt/geom` (the JDK, not PDFBox) | — | in progress — `Point2D`, `AffineTransform`, `Path2D`, `Rectangle2D`, `Ellipse2D`, `FlatteningPathIterator`, and `Area` minus curves |
| 6 | `rendering`, `printing`, `shading` | 60 | **done** — slice 9 ported everything that computes, behind `rendering.Backend`; `track/raster` wrote the backend, in `go/pdfbox/rendering/raster`. The 19 `java.awt` shading classes stay unported by name -- their arithmetic is the ported `ShadingContext`s. See the slice 9 section and `track/raster`'s |
| — | `pdfbox` root (`Loader`) | 1 | done — the reading entry points, FDF and XFDF included |
| — | `w3c/dom`, `awt` (the JDK, not PDFBox) | — | in progress — a reading DOM for XFDF, and `Color` |
| 7 | `tools` | 26 | **25 of 26**, finished by `track/tools` as far as it could go and then by `track/imageio`, which took five: the four `tools/imageio` classes and `ExtractImages`. Then by `track/multipdf`, which took `PDFMerger` and `OverlayPDF`, and by `track/raster`, which took `PDFToImage`. The one left is `PrintPDF`, which waits for a printing system rather than for a raster. The package is `go/tools` and the one binary `go/cmd/pdfbox`, settled in that branch A0: the row used to say `cmd/pdfbox`, which `PLAN.md` never said, and that is the binary rather than the package. The count was 18 until the three tracks were planned and the classes counted against `go/tools/notbuilt.go`: 17 and 9 is 26, and 18 was not |
| — | `xmpbox` | 74 | **done — all 74 files**, and all 27 test files |
| — | `pdfbox/glyphlayout` | 7 | **the backend is built** — `track/pdfbox-layout`. Not a port: PDFBox has no shaper of its own, so `go/pdfbox/glyphlayout` is one, over ported GSUB and GPOS written from the specification. The four `*Awt`/`*Fop` classes stay unported by name; see its section |

## Test data

The Java build downloads 78 test inputs no checkout here had ever read, and on
2026-09-11 `migration/scripts/fetch-testdata.ps1` filled
`pdfbox/target/{pdfs,fonts,imgs}` and `fontbox/target/fonts` for the first time.
The corpora, the fetch scripts, what each directory unlocks, the oracle and
every comparison run against PDFBox are in [`TESTDATA.md`](TESTDATA.md); the
branch that runs it keeps its own record in
[`tasks/track-testdata-sources.md`](tasks/track-testdata-sources.md). This
section is what the fetch changed about the Go tests.

**No Java test in this tree waits on input any more.** Sentences further down
that give the input as the reason a test is unported were true when they were
written; each now carries a note saying the reason expired. What is left is
unwritten test code, and the unit that can be trusted for how much of it there
is is the test class, which the per-slice tables record by hand with a reason
for each one absent. Two proxies for the size of the gap were tried and both
overstate it. Counting distinct `PDFBOX-NNNN` strings counts a citation rather
than a test: six of the nine that put `PDAcroFormTest` at the top of that count
are javadoc lines on methods
`pdmodel/interactive/form/pdacroform_external_test.go` already has, the Go
simply not repeating the number in the comment. Matching method names — the port
keeps Java's, so `testBadDA` becomes `TestBadDA` — counts every case the port
consolidates: Java's twenty `GsubWorkerForBengaliTest.testApplyTransforms_*`
methods are one table-driven `TestBengaliApplyTransforms`, and `BlendModeTest`'s
per-mode methods are `TestSeparableBlendModes` and `TestNonSeparableBlendModes`.

`migration/scripts/fetch-flatten.ps1` fills
`pdfbox/target/test-output/flatten/in` with the twelve documents
`PDAcroFormFlattenTest` downloads: the ten in its `String[]` plus the two
written inline in `flattenTestPDFBOX5254` and `flattenTestPDFBOX5225`. That
class is the one whose input is not in a pom — it carries its own list of JIRA
URLs in its source — so `fetch-testdata.ps1` never saw it. The script rejects a
response that does not start with `%PDF`, because a JIRA attachment that has
gone away answers with an HTML page and an HTML page fails much later as a parse
error nobody traces back to the download. All twelve fetch today.

### What the fetch unblocked

| Java | Cases | Where |
| --- | ---: | --- |
| `TestPDFParser` | 18 | `go/pdfbox/testpdfparser_test.go`, and the render half of `testPDFBox3950` in `rendering/raster/pdfbox3950_test.go`, which is where the backend is |
| `PDAcroFormFlattenTest` | 13 | `go/pdfbox/pdmodel/interactive/form/pdacroformflatten_external_test.go` |
| `TestFontEmbedding`, the six that waited on `target/fonts` | 6 | `go/pdfbox/pdmodel/font/fontembeddingfonts_test.go`; the class is 17 of 17 |
| `TestQuality` | 4 | `go/pdfbox/rendering/raster/testquality_test.go`, mutation-checked on three of the four |
| `TestCMapSubtable` | 2 | `go/fontbox/ttf/testcmapsubtable_test.go`, mutation-checked on both |
| `CFFParserTest` | — | already ported and skipping; it reads `SourceSansProBold.otf`, so every case ran for the first time |
| `TestSymmetricKeyEncryption.testPDFBox5955`, `testPDFBox5639` | 2 | `pdmodel/encryption/downloaded_test.go` |
| `TestFilters.testPDFBOX4517` | 1 | `filter/pdfbox4517_test.go`, its own file because it loads a document and `pdfbox` imports `filter` |
| `PDFieldTreeTest.test5044`, `TestCheckBox.testPDFBox6207` | 2 | `pdmodel/interactive/form/downloaded_test.go` |
| `TestFDF.testPDFBox5894` | 1 | `pdmodel/fdf5894_test.go` |
| `COSDocumentCompressionTest.testPDFBox5927` | 1 | appended to `pdfwriter/compression_test.go` |
| `MergeAcroFormsTest`, `MergeAnnotationsTest` | 3 | `multipdf/mergedownloaded_test.go`; both classes are now complete |
| `PDFMergerUtilityTest` | 12 | `multipdf/mergerdownloaded_test.go`; the class is now 30 of 30 |

Twelve of `PDAcroFormFlattenTest`'s thirteen cases assert one thing and it is a
strong thing: **flattening must not change what the page looks like.** Each
renders the form as it arrives, flattens it, renders it again, and requires the
two to match. The thirteenth, `flattenSingleField`, reads a checked-in fixture
and asserts the field count.

### Deviations from Java

- **The text separators.** Java initialises `PDFTextStripper`'s `lineSeparator`
  and `pageEnd` from `System.lineSeparator()`, CRLF on Windows; the port
  hardcodes LF, commented at `text/pdftextstripper.go:22`. Any comparison of
  extracted text with PDFBox has to force both to LF before it means anything.
- **Flattening is compared by pixel, not by PNG.** Java's
  `PDAcroFormFlattenTest` compares the two renders byte for byte through
  `TestPDFToImage.filesAreIdentical`; the port compares pixels, which is the
  same claim without depending on the encoder.
- **No Go test downloads anything.** Two Java classes fetch inside the test
  body, `PDFieldTreeTest` and `PDAcroFormFlattenTest`, and that half is not
  ported; their files are fetched once, ahead of time, and read by path like
  every other fixture.
- **`testSurrogatePairCharacter` asserts only the round trip through the
  extractor.** Java also renders and compares, and does not fail on a difference
  — its own comment says rendering differs between systems and the result has to
  be looked at — so there is nothing there to assert.
- **The CFF charstring lock is held across the whole of `getType2CharString`**
  rather than around the map alone, in `cfffont.go`. `getParser` and
  `getLocalSubrIndex` beside it are lazy too and Java leaves both
  unsynchronised, a race the JVM survives because the worst of it is two parsers
  built and one dropped; Go calls a racing write undefined.

### Defects found and fixed

- **`text.handleDirection` panicked on a word of paragraph separators.**
  `golang.org/x/text/unicode/bidi`, which stands in for `java.text.Bidi`,
  resolves such a word to zero runs and then indexes the first of them, where
  Java returns the word untouched. `direction.go` now answers before asking;
  `pdfbox/text/corpusdefects_test.go` pins the empty string and all six code
  points of Unicode bidi class B.
- **`aesCBC` refused a trailing partial block, so an encrypted page came back
  blank and silently.** Java's `CipherInputStream` does not throw by contract
  and reports the end of the stream having written out every complete block.
  `securityhandler.go` now decrypts the whole blocks and drops the remainder;
  `encryption/trailingblock_test.go` pins the 160 bytes the running Java answers
  and the page's own text, and read 0 before the fix.
- **The tolerance was first applied to the wrong half of AES.** Java's
  `encryptDataAESother`, AES-128 and AES-192, ends on `Cipher.doFinal` and
  rethrows the short-block failure as an IOException; only `encryptDataAES256`
  reads through `CipherInputStream`. `aesCBC` now takes a
  `tolerateShortFinalBlock`, true only from the AES-256 path, and
  `TestAES128RefusesATrailingPartialBlock` in
  `encryption/trailingblockunit_test.go` pins all three cases.
  `aesCBCNoPadding` in `standardsecurityhandler.go` carries the same message and
  is left alone: it is the AES-256 key derivation, its inputs are fixed-length,
  and Java's `doFinal` does throw there.
- **`CFFType1Font` and `CFFCIDFont` raced on their charstring cache**, fatally —
  `fatal error: concurrent map writes` in `getType2CharString`, `cfffont.go`.
  Java declares that cache `new ConcurrentHashMap<>()` in both; both now carry a
  mutex. `CFFParserTest.testMultiThreadParse`.
- **`checkWithNumberTree` compared pages by pointer.** `mcr.getPage()` builds a
  fresh `PDPage` over the same dictionary on every call and Java's
  `PDPage.equals` compares `getCOSObject()`, so the helper reported every page
  as a different one, including on unmerged sources. `samePage`, in the helper
  `pdfmergerutility_test.go` carries for the merger cases, now compares the
  dictionary. It was a defect in the test helper, not in the merger.
- **Two helpers loaded a `target/pdfs` file without an `os.Stat` first**, so a
  fresh clone failed the package instead of skipping it: `openTarget` in
  `pdfbox/testpdfparser_test.go` and `TestPDFBox3950Renders`. Confirmed by
  moving the two fixtures aside and watching the three cases skip.
- **`TestGPOSKerningAgreesWithTheKernTable` asserted nothing in two of its three
  subtests.** `Arimo-Regular.ttf` and `FiraCode-Regular.ttf` skipped with `no
  kern table`, which was not a missing font: both parse, and both carry GPOS and
  no `kern` at all. Of the 114 fonts in the tree exactly two carry both tables,
  so the case now names `DejaVuSans.ttf` and `LiberationSans-Regular.ttf` and
  `t.Fatal`s if either table is missing. It went from 89 checked pairs to 152,
  all agreeing.

### Still skipped

`go test ./... -v` on 2026-09-12: **PASS 2749 / SKIP 8 / FAIL 0.**

| Skipped | Message | Checked |
| --- | --- | --- |
| `TestOnWindows/c:/windows/fonts/mingliu.ttc` | `the system font collection is not present: ... The system cannot find the file specified` | `C:\Windows\Fonts` holds 163 `.ttf`/`.ttc` and `mingliu.ttc` is not among them (`mingliub.ttc`, a different collection, is). Java skips too: `checkTrueTypeCollection` opens with `assumeTrue(file.exists())` |
| `TestOnMac` | `the Java test is @EnabledOnOs(OS.MAC)` | `TrueTypeFontCollectionTest.java:71` carries that annotation |
| `TestPDFBox3319` | `SimHei font not available on this machine, test skipped` | `C:\Windows\Fonts\simhei.ttf` absent. The message is Java's own, verbatim from `TTFSubsetterTest.java:156`. The font is Microsoft's and not ours to fetch, and the Java test skips itself without it |
| `TestLatinViaCns1NonEmbedded` | `no CID-keyed substitute for Adobe-CNS1 installed, can't test` | verbatim from `PDCIDFontType0SubstituteTest.java:56`, the same `assumeTrue(mapping.isCIDFont(), ...)`. `msjh.ttc` and `mingliub.ttc` are installed, but they are TrueType-outline, not CID-keyed CFF, so Java skips on this machine too |
| `TestSaveResources/JBIG2Image.pdf` | `this port has no JBIG2 decoder` | true, and **Java does not skip it** — see the filters section |
| `TestSaveResources/JPXTest{CMYK,Grey,RGB}.pdf` | `this port has no JPEG 2000 decoder` | the same: three more subtests that run in Java |

Seven of the eight are Java's own assumptions firing on this machine; the four
`TestSaveResources` ones are the port being behind. Four of eight overlaps
because the JBIG2 row covers one file and the JPX row three.

### Still open

**Four of the twelve documents `PDAcroFormFlattenTest` renders differ after
flattening**, and Java passes all twelve, so this is the port's `Flatten` doing
something the Java's does not.

| File | Pixels differing after flattening |
| --- | ---: |
| `test-2586.pdf` | 322 |
| `PDFBOX-5225.pdf` | 51, over two pages |
| `PDFBOX-4955.pdf` | 4 |
| `Signed-Document-1.pdf` | 2 |

No content appears, disappears or moves: on all four the differing pixels are
one to five levels of grey along an edge inside a region a few hundred pixels
across, and the test asserts that too — no channel may be more than 8 levels
out, and none is. A save-and-reload without flattening changes zero pixels on
all four, so the writer is not what does it, and the two flattened content
streams agree operator for operator. Two explanations were tried and are wrong:
Java's `resolveTransformationMatrix` works in double and rounds once where the
port narrowed to float32 first, and making the port match changed nothing; and
both sides write five digits, `pdpagecontentstream.go` raising the default of
four to five exactly as `PDPageContentStream` does. The counts are recorded in
the test as `differingAfterFlatten` with the rule that they go down and never
up: raising one is a regression, and dropping one to zero is the fix and should
delete the row. The AES trailing-block defect above was found through this one
and is not its cause — the counts are the same before and after that fix,
because the missing page text was missing from the render on both sides of the
flatten and cancelled out.

- **`Type1CharString.renderOnce` still races**, and `go test -race` on the `cff`
  package is not clean. It is deliberately unguarded and the reason is at the
  site: Java guards it with `synchronized(LOG)`, a re-entrant monitor, and seac
  renders another charstring from inside `render`, so `sync.Once` and
  `sync.Mutex` both deadlock on the self-referential seac that the
  `c.path == accentPath` check exists to catch. Rendering eagerly would build
  the path of every glyph a caller never draws. Giving `render` a re-entrancy it
  does not have is its own piece of work. The mutex above fixes the one failure
  that is fatal in Go and not in Java; it does not make the font thread-safe.

## The 891-class survey, and why it was replaced

**Superseded.** This was the answer to "what is actually left", taken from a
survey that compared all 891 in-scope Java main classes and 237 Java test
classes against the Go tree, class by class. "The audit that found what the
survey missed", below, replaces it and carries the live answer, its method and
the commands to re-run it. The one finding of the survey that outlived it is the
test gap that became `track/test-backfill`, the sixteen Java test classes merged
slices had left behind; every branch it named has since merged, and what is left
now is in "What is left, and the four branches that claim it", below.

What it got wrong, recorded because the cause will recur:

- **A name in the Go tree is not evidence of a port**, and the matcher treated
  one as such. `multipdf` was 3 of 6 — `PDFMergerUtility`, `LayerUtility` and
  `Overlay` were unported — and the survey counted all three as done, because
  `pdfcloneutility.go`'s package comment reads "PDFMergerUtility, LayerUtility
  and Overlay are **not** here". It read a name in a sentence saying the class
  is absent as evidence it is present. `track/multipdf` ported all three and
  built both commands.
- **A row it declared checked was not.** `pdmodel/font at 34 of 39` was wrong
  when it was written: `ToUnicodeWriter` had been ported by slice 7, with its
  own test and JAVA-BUGS 33. The matcher looked for a plain
  `Port of org.apache.pdfbox...` and `tounicodewriter.go`'s header reads `Port
  of **the package-private** org.apache.pdfbox...`, so it missed the class and
  then wrote the miss down as verified.
- **Three of its rows repeated deferrals that later slices had already closed**
  and recorded only in their own sections, leaving the summary row and the
  deferring slice's table saying the work was outstanding. The Summary table at
  the head of this file carries the corrected counts.

## Phase 0 — `pdfio`

| Java source | Go source | Status |
| --- | --- | --- |
| `RandomAccessRead.java` | `randomaccess.go` | done |
| `RandomAccessWrite.java` | `randomaccess.go` | done |
| `RandomAccess.java` | `randomaccess.go` | done |
| `RandomAccessReadBuffer.java` | `readbuffer.go` | done |
| `RandomAccessReadWriteBuffer.java` | `readwritebuffer.go` | done |
| `RandomAccessReadView.java` | `readview.go` | done |
| `SequenceRandomAccessRead.java` | `sequenceread.go` | done |
| `RandomAccessReadBufferedFile.java` | `bufferedfile.go` | done |
| `RandomAccessInputStream.java` | `adapters.go` | done |
| `RandomAccessOutputStream.java` | `adapters.go` | done |
| `RandomAccessStreamCache.java` | `streamcache.go` | done |
| `RandomAccessStreamCacheImpl.java` | `streamcache.go` | done |
| `IOUtils.java` | `ioutils.go`, `tempfile.go` | done — most of it maps to the Go stdlib instead, and the temporary file half arrived with `track/scratchfile`; see the file header. `createProtectedTempDir` is not ported: only `PDFDebugger` calls it, and it is a JVM shutdown hook |
| `ScratchFile.java` | `scratchfile.go` | done in `track/scratchfile` |
| `ScratchFileBuffer.java` | `scratchfilebuffer.go` | done in `track/scratchfile` |
| `MemoryUsageSetting.java` | `memoryusagesetting.go` | done in `track/scratchfile` |
| `RandomAccessReadMemoryMappedFile.java` | `mappedfile.go` | done in `track/scratchfile` — mapped through `golang.org/x/exp/mmap`, which is the decision B0 settled; see the file header for what it buys and what it costs |
| `NonSeekableRandomAccessReadInputStream.java` | `nonseekablestream.go` | done in `track/scratchfile` |

## Slice 1 — `pdfbox/cos`

Branch `slice/1-open-document`. Ported test-first.

19 of 24 files ported — everything slice 1 needs. Of the remaining five, four
are the incremental-save machinery deferred to slice 7 and one is folded into
`stream.go`.

| Java source | Go source | Status |
| --- | --- | --- |
| `COSBase.java` | `base.go` | done |
| `ICOSVisitor.java` | `visitor.go` | done — all 11 methods |
| `ICOSParser.java` | `object.go` | done |
| `COSBoolean.java` | `boolean.go` | done |
| `COSNull.java` | `null.go` | done |
| `COSObjectKey.java` | `objectkey.go` | done |
| `COSName.java` | `name.go`, `names.go` | done — 587 constants generated |
| `COSNumber.java` | `number.go` | done |
| `COSInteger.java` | `integer.go` | done |
| `COSFloat.java` | `float.go` | done |
| `COSString.java` | `string.go` | done |
| `PDFDocEncoding.java` | `pdfdocencoding.go` | done |
| `COSObject.java` | `object.go` | done — minus the update state |
| `COSArray.java` | `array.go` | done — minus the update state and the `COSObjectable` overloads |
| `COSDictionary.java` | `dictionary.go` | done — minus dates, `getCOSStream`, the `COSObjectable` overloads and the update state |
| `UnmodifiableCOSDictionary.java` | `unmodifiable.go` | done — as a read-only interface, see below |
| `COSStream.java` | `stream.go` | done; **written before its test**, see the method note (update state added in slice 7) |
| `COSInputStream.java` | — | not ported — `Stream.CreateReader` returns a plain `io.Reader`; the class exists in Java only to carry a `DecodeResult` |
| `COSOutputStream.java` | — | not ported — folded into `streamWriter` in `stream.go` |
| `COSDocument.java` | `document.go` | done (document state added in slice 7) |
| `COSDocumentState.java` | `documentstate.go` | done in slice 7 |
| `COSUpdateInfo.java` | `updateinfo.go` | done in slice 7 |
| `COSUpdateState.java` | `updatestate.go` | done in slice 7 |
| `COSIncrement.java` | `increment.go` | done in slice 7 |

### Ported tests — `cos`

| Java test | Go test | Notes |
| --- | --- | --- |
| `TestCOSBase` | `base_test.go` | abstract in Java; becomes `assertBaseContract`, called per type |
| `TestCOSBoolean` | `boolean_test.go` | complete; the COSWriter byte assertions are in `accept_external_test.go` |
| `COSObjectKeyTest` | `objectkey_test.go` | `testPDFBox5742` not ported — needs parser, writer, multipdf and a renderer |
| `TestCOSName` | `name_test.go` | all three Java tests drive documents; their real assertions (`/m#E4nnlich`, `/m#00nnlich`, PDFBOX-4076) are asserted against `WritePDF` directly |
| `TestCOSNumber` | `number_test.go` | complete |
| `TestCOSInteger` | `integer_test.go` | complete |
| `TestCOSFloat` | `float_test.go` | complete |
| `TestCOSString` | `string_test.go` | complete; the COSWriter serialisation assertions are in `accept_external_test.go` |
| `PDFDocEncodingTest` | `pdfdocencoding_test.go` | complete, including PDFBOX-3864 |
| `TestCOSArray` | `array_test.go` | complete |
| `COSDictionaryTest` | `dictionary_test.go` | `testCOSDictionaryNotEqualsCOSStream` needs `COSStream`; the identity semantics it guards are covered |
| `UnmodifiableCOSDictionaryTest` | `unmodifiable_test.go` | every assertion is a compile error in Go, so nothing is left to assert at run time |
| — | `null_test.go`, `object_test.go` | Java has no test for these; written from the source per the tdd rule |

### Deviations — `cos`

**The `accept()` assertions live outside package `cos`.** Java's `accept()`
tests drive a `COSWriter` and assert the emitted bytes.
`cos/accept_external_test.go` makes those assertions — for booleans, integers,
floats and strings, and for the static `COSWriter.writeString` — in package
`cos_test`, because a test file in package `cos` cannot import `pdfwriter`
without a cycle. The four tests named in `boolean_test.go`, `integer_test.go`,
`float_test.go` and `string_test.go` assert the visitor double dispatch and
point at it. Open debt until slice 7, where it was closed.

**`TestCOSFloat`'s sweep is not restored.** Java's `BaseTester` walks
`i * new Random(seed).nextFloat()` for `i` in `[-100000, 300000)` step 20000,
once with a fixed seed and once with the clock. `java.util.Random`'s sequence
cannot be reproduced in Go without porting the generator, so the accept test
uses the sweep slice 1 chose for the rest of `float_test.go`, plus the
PDFBOX-1778 corner case that `testWritePDF` adds.

Deliberate differences, each commented at the point it occurs:

- `Name` is interned through `weak.Pointer` and `runtime.AddCleanup`, the
  equivalents of Java's `WeakReference` plus `Cleaner`. Interning makes `==`
  equivalent to content equality and lets a `*Name` be a Go map key, which is
  what `Dictionary` needs.
- `Dictionary` keeps key insertion order in a slice, because Java's
  `LinkedHashMap` has it and Go maps do not. A dictionary is written back in the
  order it was read.
- `Dictionary` keys everything on `*Name`. Java declares a `String` and a
  `COSName` overload of nearly every accessor; Go has no overloading, and
  interned names make the pointer form cheap.
- `AsReadOnly` returns an interface with no mutating methods, so a write is a
  compile error. Java returns a `COSDictionary` that throws at run time.
- `cosEqual` dispatches to the `Equals` method of the types that define one.
  Java relies on every class overriding `equals`; Go has no single such hook,
  and every container comparison goes through this one function.
- `Visitor` and the ported types are deliberately smaller than the Java
  originals where a dependency is not ported yet; each says so in its doc
  comment.

Behaviour that is **not** a deviation, listed because it looks like one:
`Integer.Equals` truncates to 32 bits, `ParseHexString` honours a `ForceParsing`
flag, `assertBytesEqual` does not check lengths, and `Name.Bytes` returns the
internal slice. All four match the Java, and the first three are recorded in
[`JAVA-BUGS.md`](JAVA-BUGS.md) as defects carried over deliberately.

### Port defects found in review, fixed

- **`Integer.IntValue` did not truncate to 32 bits.** Go's `int(int64)` is a
  no-op where Java's `(int)` narrows, so `Equals` was not reproducing
  [`JAVA-BUGS.md`](JAVA-BUGS.md) 1 despite a comment claiming it did.
- **`Float.IntValue` and `LongValue` did not saturate.** Java clamps an
  out-of-range float to `MAX_VALUE`/`MIN_VALUE`; Go leaves the conversion
  undefined.
- **`Flate.Decode` propagated a corrupt-input error**, so `Stream.CreateReader`
  returned no reader and the decoded prefix was unreachable. Java catches, logs
  and returns what inflated — the damage tolerance the filter exists for.
- **`Stream.Length` answered while a writer was open**, returning a stale value.
  Java throws there; the Go now returns `ErrStreamWriting`.
- **`XrefTrailerResolver` resolved to no type when startxref pointed nowhere.**
  The Java constructor defaults it to `TABLE`.

## Slice 1 — `pdfbox/filter`

Only the filters slice 1 needs. The rest arrive in slice 6.

| Java source | Go source | Status |
| --- | --- | --- |
| `Filter.java` | `filter.go` | done — minus the `DecodeOptions` overload, which carries image subsampling |
| `FilterFactory.java` | `filter.go`, `provider.go` | done — as `ByName` plus a `Provider` type rather than a singleton. Until 2026-09-15 `ByName` also answered `/Identity`, which `FilterFactory` refuses with "Invalid filter"; it refuses it now, and nothing depended on it |
| `Predictor.java` | `predictor.go` | done — matched to PDFBox's output on 2026-09-15: `/Colors` clamped to 32 as `wrapPredictor` does, Java's 32-bit arithmetic throughout, the short last row completed with zeros. JAVA-BUGS 88 is not carried |
| `FlateFilter.java`, `FlateFilterDecoderStream.java` | `flate.go` | done — a source that fails, in the header or after it, fails `Decode` and the reader with its own error since 2026-09-15; see the deviations below |
| `IdentityFilter.java` | `filter.go` | done |
| `DecodeResult.java` | `filter.go` | partial — the JPX colour space and soft mask fields arrive with that filter |
| `DecodeOptions.java` | `decodeoptions.go` | done in slice 6 |
| the other 15 filters | `dct.go`, `ccittfax.go`, `lzw.go`, `runlength.go`, `asciihex.go`, `ascii85.go`, `imagereader.go` and the rest | done in slice 6 |

| Java test | Go test | Notes |
| --- | --- | --- |
| `PredictorTest` | `predictor_test.go` | complete |
| — | `predictorpath_test.go`, `javabug88_test.go` | no Java test reaches the predictor the way a PDF does, through FlateDecode and LZWDecode; these take PDFBox's own output for each case as the expected value |
| `TestFilters` | `flate_test.go` | the round-trip generator is ported; `testPDFBOX4517` needs a loader, `testPDFBOX1977` needs LZW, `testRLE` needs RunLength |

### Deviations — `filter`

Each commented at the point it occurs. The first two were taken on
`track/performance`; see [`PERFORMANCE-PLAN.md`](PERFORMANCE-PLAN.md) for why.

- **Decompressors are pooled.** Java makes a new `Inflater` for every stream,
  and `FlateFilterDecoderStream.close` ends it. `flate.go` keeps
  `compress/flate` decompressors in a pool and resets one for each stream, in
  `Flate.Decode` and in the reader `NewFlateDecoderReader` returns. That reader
  hands its decompressor back when its data ends rather than on `Close`,
  because the stream engine, like Java's, never closes a content stream it has
  parsed; a `Close` after that answers nil, which is what it answered before.
  Allocation and lifetime only: nothing a caller reads changes. Pinned by
  `TestFlateDecodeReusesOneDecompressor` and
  `TestFlateDecoderReaderPoolsItsDecompressorWithoutSharingIt`.
- **Unpredicted data is copied through a pooled 32 KB buffer** in
  `decodePredictor`, where Java's `transferTo` allocates one for each call.
  Allocation only.
- **A failing source comes out of `Flate.Decode` with its own error**, since
  2026-09-15 on `track/testdata-sources`. `FlateFilterDecoderStream` catches
  `DataFormatException` alone, so an `IOException` from the source comes out of
  `FlateFilter.decode`, and so does one from the two header bytes its
  constructor reads. The port's `Decode` had logged every error and carried on,
  and both it and `NewFlateDecoderReader` took a header they could not read,
  for any reason, as a stream with nothing in it. `endAtDamage` now ends the
  data at damage only, and a source that merely ends still ends the data.
  `TestFlateLetsAFailingSourceOut` holds PDFBox's answers.

## Slice 1 — `pdfbox/pdfparser`

8 of 18 files ported. This is the package `AGENTS.md` flags as historically
bug-prone, so it is ported line for line and every recovery path is kept.

| Java source | Go source | Status |
| --- | --- | --- |
| `BaseParser.java` | `base.go` | done — the lexer |
| `XrefTrailerResolver.java` | `xreftrailer.go` | done |
| `xref/XReferenceType.java` | `xref/xref.go` | done |
| `xref/XReferenceEntry.java` | `xref/xref.go` | done |
| `xref/AbstractXReference.java` | `xref/xref.go` | done |
| `xref/FreeXReference.java` | `xref/xref.go` | done |
| `xref/NormalXReference.java` | `xref/xref.go` | done |
| `xref/ObjectStreamXReference.java` | `xref/xref.go` | done |
| `COSParser.java` | `fileparser.go`, `objectparser.go` | next when this table was written, 2,021 lines; ported by slice 3, "The loader" |
| `XrefParser.java` | `xrefparser.go` | 695 lines; ported by slice 3, "The loader" |
| `BruteForceParser.java` | `bruteforceparser.go` | 857 lines, the damaged-file recovery path; ported by slice 3 |
| `PDFStreamParser.java` | `streamtokenparser.go` | done in slice 2 — named `StreamTokenParser`, because `StreamParser` is already COSParser's stream half |
| `PDFXRefStream.java` | `pdfxrefstream.go` | the writing path; slice 7 |
| `PDFXrefStreamParser.java` | `xrefstreamparser.go` | ported by slice 3 |
| `PDFParser.java` | `pdfparser.go` | the entry point; ported by slice 3 |
| `PDFObjectStreamParser.java` | `objectstreamparser.go` | ported by slice 3 |
| `EndstreamFilterStream.java` | `streamparser.go` | ported later in this slice, with `parseCOSStream` and `readUntilEndStream` |
| `FDFParser.java` | `fdfparser.go` | FDF, not needed for slice 1; slice 8 |

None of the Java files in this package have tests; the parsers are exercised
only through whole documents. Every test here is therefore written from the
source per the tdd rule, and the recovery paths named in the Java comments —
PDFBOX-3506, PDFBOX-276, brother_scan_cover.pdf — are pinned individually.

### Method note — what was not ported test-first

Two lapses against [`conventions/tdd.md`](conventions/tdd.md), recorded rather
than quietly fixed.

- **`pdfio`** predates the rule: the implementation was written first and the
  Java tests ported afterwards. The tests are faithful — assertion values, the
  sample byte arrays and the PDFBOX-numbered regression cases were copied from
  the Java test files, not recomputed from the Go — but they did not drive the
  implementation, so they cannot rule out a mistranslation the Java suite does
  not cover.
- **`cos/stream.go`** was written before `stream_test.go`. The test was then
  ported from `TestCOSStream` rather than written against the Go, so its
  assertions are still Java-derived.

Everything from `slice/1` onward follows the rule, and so did the five files
`track/scratchfile` added here: phase A ported the three Java test files before
any implementation was written.

`track/scratchfile`'s D9 re-read all thirteen files of `slice/0`. Not one
mistranslates its Java — `ReadBuffer`'s chunk arithmetic and `BufferedFile`'s
page-boundary handling are both faithful, the `-1` accumulation and the
redundant clamp included. What the re-read found instead was five defects in
the Java, [`JAVA-BUGS.md`](JAVA-BUGS.md) 67 to 71, and that entry 3 is worse
than it had been written up as.

### Ported tests

| Java test | Go test | Notes |
| --- | --- | --- |
| `RandomAccessReadBufferTest` | `readbuffer_test.go` | `testPDFBOX5111` not ported — it downloads a PDF over the network |
| `RandomAccessReadViewTest` | `readview_test.go` | complete |
| `RandomAccessReadWriteBufferTest` | `readwritebuffer_test.go` | complete |
| `SequenceRandomAccessReadTest` | `sequenceread_test.go` | complete |
| `RandomAccessReadBufferedFileTest` | `bufferedfile_test.go` | fixtures written to `t.TempDir()` instead of read from the source tree |
| `RandomAccessInputStreamTest` | `adapters_test.go` | complete |
| `ScratchFileBufferTest` | `scratchfilebuffer_test.go` | complete, plus `TestClearLeaksTheLastPage`, which pins JAVA-BUGS 62 |
| `NonSeekableRandomAccessReadInputStreamTest` | `nonseekablestream_test.go` | complete |
| `RandomAccessReadMemoryMappedFileTest` | `mappedfile_test.go` | complete |
| `TestIOUtils` | `streamcache_test.go` | mostly covers methods that map to the Go stdlib; the two stream cache factory cases are ported |
| — | `memoryusagesetting_test.go` | written from source: Java has no test for `MemoryUsageSetting`. Every expected value was read out of the running Java rather than reasoned about |

### Deviations from Java recorded so far

Each of these carries a comment at the point of difference in the Go source.

- `CreateView` hands every caller its own cursor instead of caching one clone
  per thread id. Go has no stable goroutine identity, and the result is safe for
  concurrent use where the Java version is not.
- `BufferedFile.Length` reports `ErrClosed` on a closed source, where Java's
  `length()` calls no `checkClosed` and answers the file length. Its sibling
  `RandomAccessReadMemoryMappedFile.length()` does check, so the Java is
  inconsistent between the two; the port follows the checking one in both.
  Slice 0's to settle.
- `BufferedFile` shares one mutex-guarded page cache across all cursors, where
  Java reopens the file per thread.
- The evicted page buffer is not reused for the next page read, so a cursor
  still holding an evicted page keeps reading valid bytes.
- `BufferedFile.IsEOF` compares offset against length instead of `peek() == -1`.
- End of input is `io.EOF` throughout, not a `-1` return.
- `MappedFile.CreateView` answers `ErrClosed` on a closed source, where Java
  reaches through the released buffer and raises NullPointerException.
  JAVA-BUGS 65, pinned by `TestMappedFileViewOfAClosedSource`.
- `NewSequenceRead` answers "empty list" for a list holding only zero-length
  sources, where Java checks `isEmpty()` before it filters and so raises
  IndexOutOfBoundsException. JAVA-BUGS 70.
- `SequenceRead.Close` closes every source and reports the first failure, where
  Java lets the first failure out of the loop and leaks the rest. JAVA-BUGS 71.
- `NewReadBufferSize` reads a chunk size of zero or less as the default. Java
  keeps a zero — `seek` carries `chunkSize > 0 ?` guards for exactly that state
  — and throws IllegalArgumentException for a negative one. The port drops those
  guards and forbids the state instead, because its chunk arithmetic divides by
  the chunk size. Slice 0's to settle.

Two things that are **not** deviations, confirmed by the D9 re-read:

- `ReadBuffer.Read` adds the `-1` its chunk helper answers to the running count,
  exactly as the Java loop does. JAVA-BUGS 2.
- `BufferedFile.Read` stops at the page boundary and clamps to the file length.
  It reads differently from Java only because Java guards its second clamp with
  `fileLength - fileOffset < PAGE_SIZE`, and where that guard is false the clamp
  cannot bite, so the two are the same function.

## Slice 2 — walk content streams

Branch `slice/2-content-streams`. The slice dumps the operator sequence of a
page and interprets nothing, so everything that draws or measures is left for
the slices that bring fonts, XObjects and a rasteriser.

### `awt/geom` — the JDK, not PDFBox

Go has no standard-library geometry, and `Matrix`, the graphics state and the
text machinery are all written against `java.awt.geom`. Only what PDFBox calls
is here; the geometry is ported and rasterisation left behind an interface, as
[`PLAN.md`](PLAN.md)'s slice 9 settles.

| Java source | Go source | Status |
| --- | --- | --- |
| `java.awt.geom.Point2D` | `point.go` | done — the abstract base as an interface, plus the Float and Double forms |
| `java.awt.geom.AffineTransform` | `affinetransform.go` | done — minus the state/type cache, which only lets the JDK skip terms it knows are zero |
| `java.awt.geom.PathIterator` | `shape.go` | done |
| `java.awt.Shape` | `shape.go` | partial — `contains` and `intersects` are absent; nothing in PDFBox calls them on a path, and they need the curve-crossing machinery in `sun.awt.geom` |
| `java.awt.geom.Path2D` | `path.go` | done — one type holding `float64`, rounding on the way in when it is a Float path |
| `java.awt.geom.Rectangle2D` | `rectangle.go` | done — one type; PDFBox only ever uses the Double form |
| `java.awt.Rectangle` | `rectangle.go` | done — the integer bounds only |
| `java.awt.geom.Area` | `area.go` | done in slice 9 — constructive area geometry, minus curves: an added shape is flattened first. That flattening is the only deviation; `equals` compares the geometries the way the JDK does, by exclusive-or. Written from the JDK contract, not from what the renderer needs |

### `pdfbox/util`, `fontbox/util`, `internal/javafmt`

| Java source | Go source | Status |
| --- | --- | --- |
| `pdfbox/util/Matrix.java` | `pdfbox/util/matrix.go` | done — `MatrixTest` ported |
| `pdfbox/util/Vector.java` | `pdfbox/util/vector.go` | done |
| `fontbox/util/BoundingBox.java` | `fontbox/util/boundingbox.go` | done — pulled in by `PDRectangle` |
| — | `internal/javafmt` | new — Java float rendering, which `geom`, `util` and `common` all need |

### `pdfbox/pdmodel`

| Java source | Go source | Status |
| --- | --- | --- |
| `common/COSObjectable.java` | `common/pdrectangle.go` | done |
| `common/PDRectangle.java` | `common/pdrectangle.go` | done |
| `common/PDImmutableRectangle.java` | `common/pdrectangle.go` | done — as a flag, since Go has no subclassing; `PDImmutableRectangleTest` ported |
| `common/PDDictionaryWrapper.java` | `common/pddictionarywrapper.go` | done |
| `common/PDTypedDictionaryWrapper.java` | `common/pdtypeddictionarywrapper.go` | done |
| `common/PDStream.java` | `common/pdstream.go` | done — the reading path here, the rest in slice 8 |
| `common/COSArrayList.java` | — | not started here — slice 8, with its Java test |
| `PDResources.java` | `pdresources.go`, `pdresources_colorspace.go`, `pdresources_graphics.go` | done — the dictionary plumbing and `getFont` with its direct cache here; `getColorSpace` and `getExtGState` came with slices 3 and 6, `getProperties` with slice 8, and `getShading`, `getPattern`, `getXObject` and the add and put family with slice 9 |
| `ResourceCache.java` | `pdmodel/font/resourcecache.go`, aliased in `resourcecache.go` | done — the font and font descriptor members here, the rest arriving with their types up to slice 9. The interface is declared in `pdmodel/font` because it names `PDFont` and `pdmodel` imports that package, so the five kinds it cannot name are asked of the cache by shape from `pdmodel` instead — and so are their removals, by `PDPage.RemovePageResourceFromCache` |
| `DefaultResourceCache.java` | `resourcecache.go` | done in slice 9 — all eight kinds, each with the stable-cache bookkeeping, which the port writes once as a generic map rather than eight times. Java holds each entry through a `SoftReference`; Go has none, so the port holds them outright |
| `PDPage.java` | `pdpage.go` | done — boxes, rotation, resources, contents here. The `PDStream` methods came with slice 7, everything else but `removePageResourceFromCache` with slice 8, and that with `track/testdata-sources` |
| `PDPageTree.java` | `pdpagetree.go` | done — the reading constructor takes the `PDDocument`, as Java's does, and asks it for the `ResourceCache` each time a page is handed out, by index or by the walk |
| `MissingResourceException.java` | `errors.go` | done |
| `PDDocument.java`, `PDDocumentCatalog.java`, `PDDocumentInformation.java` | — | not started here — slice 3 for the document and its information, slice 8 for the catalogue |

`PDPage.getContentsForStreamParsing` took the general path for the length of
this slice: its fast path decodes a single flate stream as it is read, which
needs `FlateFilterDecoderStream` and `NonSeekableRandomAccessReadInputStream`,
neither of which was ported then. `track/scratchfile` ported both and wired the
fast path back in, so `ContentsForStreamParsing` now branches the way Java does
— including onto the predictor bug the fast path carries, JAVA-BUGS 63.

### `PDPageTree` and the resource cache — two port defects, fixed on `track/performance`

Java hands a page its resource cache at the moment the page is handed out:
`PDPageTree.get(int)` and the iterator's `next()` each ask
`document.getResourceCache()` there and then. The port got that wrong twice.
Neither defect changes the text extracted or a page rendered. What they change
is which cache a page reads its fonts, colour spaces and images through, and so
what is built again and what memory can be let go.

- **The walk handed out pages with no cache at all.** `All` built each page with
  `NewPDPageOf`, where `Get` passed the tree's cache, so text extraction, which
  walks the tree, built every font again on every lookup.
  `pdfbox/pdmodel/pdpagetree.go`;
  `TestPDPageTreeAllHandsOutTheResourceCache`. What it cost is in
  [`PERFORMANCE-PLAN.md`](PERFORMANCE-PLAN.md).
- **The tree kept the cache it was made with.** `NewPDPageTreeOfCache` took
  `document.ResourceCache()` once, when the tree was built, and `Get` and `All`
  handed that out for as long as the tree lived, so a cache set on the document
  afterwards reached only trees taken afterwards. The tree now keeps the
  document, as Java's does, through `NewPDPageTreeOfDocument`.
  `pdfbox/pdmodel/pdpagetree.go`;
  `TestPDPageTreeAsksTheDocumentForTheCacheAsItHandsOutPages`. Only a caller
  that replaces or switches off a cache with `SetResourceCache` while holding a
  tree, or pages taken from one, could see it — `PDDocument.Pages()` builds a
  new tree on every call — and nothing in this repository calls
  `SetResourceCache` outside the test for it.

**A processed page's resources now leave the cache**, since
`track/testdata-sources`. Java's `PDFTextStripper.processPage` ends with
`page.removePageResourceFromCache()`, and the port had neither the call nor the
method, so with the cache handed to every page the resources every page read
stayed in it for as long as the document was open. Java's cache holds each entry
through a `SoftReference` and would let them go under memory pressure; the
port's holds them outright. What the purge lets go, what it costs, and the
transparency groups the port's purge was missing are in
[`tasks/track-testdata-sources.md`](tasks/track-testdata-sources.md), "Found in
review"; [`BENCHMARK.md`](BENCHMARK.md)'s tables are the code before it.

### `pdfbox/pdmodel/graphics`

| Java source | Go source | Status |
| --- | --- | --- |
| `PDLineDashPattern.java` | `graphics/pdlinedashpattern.go` | done |
| `blend/BlendMode.java` | `graphics/blend/blendmode.go` | done — blend functions included |
| `blend/BlendComposite.java`, `SoftMask.java` | `rendering/raster/composite.go`, `softmask.go` | both are `java.awt` raster classes and slice 9 put the raster half behind `rendering.Backend`, naming the blend mode and the alpha constants on the graphics state and the mask in `rendering.SoftMaskedPaint`. **`track/raster` ported both**, and every one of `BlendComposite`'s 340 reference pixels matches |
| `color/PDColor.java` | `graphics/color/pdcolor.go` | done |
| `color/PDColorSpace.java` | `graphics/color/colorspace.go` | partial — as an interface; the static `create` methods and the two `BufferedImage` methods are absent |
| `color/PDDeviceColorSpace.java` | `graphics/color/colorspace.go` | done |
| `color/PDDeviceGray.java` | `graphics/color/devicegray.go` | done — minus `toRGBImage` |
| the other 20 colour spaces | `graphics/color/` | done — `PDJPXColorSpace` excepted, which only the JPX filter constructs |
| `state/RenderingIntent.java` | `graphics/state/renderingintent.go` | done — `RenderingIntentTest` ported |
| `state/RenderingMode.java` | `graphics/state/renderingmode.go` | done |
| `state/PDTextState.java` | `graphics/state/pdtextstate.go` | done |
| `state/PDGraphicsState.java` | `graphics/state/pdgraphicsstate.go` | done in slice 9 — the soft mask, `getCurrentClippingPath` and the Area form of `intersectClippingPath` all arrived with `Area`. The two Java composites are not here and will not be: they wrap the blend mode and an alpha constant in a `java.awt.Composite`, which is the rasteriser's half |
| `state/PDSoftMask.java` | `graphics/state/pdsoftmask.go` | done in slice 9 — the transparency group is reached through `NewTransparencyGroup`, which `pdmodel` sets, because `graphics/form` imports this package |
| `state/PDExtendedGraphicsState.java` | `graphics/state/pdextendedgraphicsstate.go` | done in slice 8, with the `/SMask` arm in slice 9 |

### `pdfbox/contentstream`

| Java source | Go source | Status |
| --- | --- | --- |
| `PDContentStream.java` | `contentstream.go` | done — the default method becomes an interface plus a package function |
| `PDFStreamEngine.java` | `streamengine.go` | partial — the walk, the dispatch, the graphics stack and the accessors; everything that draws or measures is absent |
| `PDFGraphicsStreamEngine.java` | — | not started — the path operators hang off it |
| `operator/Operator.java` | `operator/operator.go` | done |
| `operator/OperatorName.java` | `operator/names.go` | done — 72 constants generated |
| `operator/OperatorProcessor.java` | `streamengine.go` | done — **moved into `contentstream`**, see below |
| `operator/MissingOperandException.java` | `operator/errors.go` | done |
| `operator/state/EmptyGraphicsStackException.java` | `operator/errors.go` | done — **moved into `contentstream/operator`**, see below |
| `operator/state/*` | `operator/state/state.go` | done, minus `SetGraphicsStateParameters` (gs) |
| `operator/text/*` | `operator/text/text.go` | partial — minus `Tf`, `Tj`, `TJ`, `'` and `"`, all of which need `PDFont` |
| `operator/markedcontent/*` | `operator/markedcontent/markedcontent.go` | partial — minus `DrawObject`, and minus resolving a property list named in the resources |
| `operator/color/*`, `operator/graphics/*` | — | not started — colour spaces and the graphics engine |

Two classes move package, both for the same reason: Java allows a package cycle
and Go does not. `OperatorProcessor` holds the engine and the engine holds
processors, so the interface lives with the engine in `contentstream`.
`EmptyGraphicsStackException` is raised by an operator in
`contentstream.operator.state` and caught by the engine, so the sentinel lives
in `contentstream/operator`, which both sides already import.

### Deviations — slice 2

- **The overridable methods of `PDFStreamEngine` are an interface, not
  embedding.** Go's embedding gives no virtual dispatch, so a subclass calling
  `SetOverrides(itself)` is what stands in for Java's dynamic dispatch from the
  superclass into the subclass. This is the pattern
  [`conventions/java-to-go.md`](conventions/java-to-go.md) prescribes.
- **`shouldProcessColorOperators` is always true.** The two cases that clear it
  are an uncoloured tiling pattern and a Type 3 char proc beginning with `d1`,
  and neither type is ported.
- **`java.util.zip.DataFormatException` becomes a check for the Go flate and
  zlib errors** in `operatorException`. The Go flate filter logs and returns
  what inflated rather than failing, so the branch may be unreachable.
- **The operator processors are one file per package**, where Java gives each a
  file of its own. Each is a few lines, and the package is the unit that
  matters.

### Port defects found and fixed

`COSDictionary.containsValue` had been written as `getKeyForValue` in disguise,
found while porting the slice. The two look in opposite directions:
`containsValue` unwraps an indirect reference given as the *argument*,
`getKeyForValue` unwraps the references *stored* in the dictionary. Collapsing
them made a dictionary holding a reference to `x` report that it contained `x`,
which would have made the PDFBOX-4509 search in `PDResources.add` dead code
rather than the fix it is. `getKeyForValue` also gained the guard Java puts in
front of `getObject`.

The rest came out of the slice 2 review, and each carries a test that fails
without the fix.

- **`COSStream.createView` asked a view for a view.** Java builds a second
  `RandomAccessReadView` around the one it holds; the port called `CreateView`
  on it, which a view refuses — in Go as in Java. Every unfiltered stream read
  from a file failed, and `PDPage.ContentsForRandomAccess` reported it as a
  malformed content stream and substituted a newline, so page content was being
  silently dropped.
- **`COSArray.toFloatArray` and the two numeric list conversions read the raw
  entry.** Java reads through `getObject`, which resolves an indirect reference.
  An indirect number yielded zero, so an indirect `/MediaBox` gave a zero-sized
  page.
- **`setString("")` removed the entry.** Java removes only for a null argument,
  and an empty string is a valid COS string. Go has no null string, so the port
  had used `""` for it; a caller wanting Java's null now calls `RemoveItem`, or
  `Set(index, nil)` on an array. `setEmbeddedString` had the same conflation.
- **`PDGraphicsState.renderingIntent` read as `AbsoluteColorimetric` from
  birth.** The Java field is null until `ri` or an extended graphics state sets
  it. It is a `*RenderingIntent` now, so "not specified" and "specified as
  absolute" are distinguishable again.
- **`Reader.Available` did not clamp.** Java is
  `Math.min(length - position, Integer.MAX_VALUE)`. The package-level
  `Available` already clamped; this one did not.
- **`ReadView.Close` kept its source when it owned it.** Java drops the
  reference either way, outside the ownership check.
- **`PDColor.Components` and `PDLineDashPattern.DashArray` returned nil when
  empty.** Both are documented as never nil, and Java's `clone()` of an empty
  array is not null. `append([]float32(nil))` with nothing to append is.
- **The `StreamCache` doc promised more than any implementation gives.** The
  memory-backed cache closes nothing, and neither does Java's. Comment only.

### Java bug carried over, after being fixed by mistake

`COSString.parseHex`'s handling of leading and trailing whitespace: the port had
sliced `hex[start:end]`, correcting it. Reverted, recorded as
[`JAVA-BUGS.md`](JAVA-BUGS.md) entry 11, and pinned by `string_test.go`, which
now asserts the error and the force-parsing substitution.

## Slice 3 — text from simple fonts

Branch `slice/3-text-simple-fonts`. The slice reads the font programs and the
encodings a simple font needs, decodes the text operators, and writes out the
text of a page. It also ports the loader, slice 1's unfinished half: nothing in
the tree could open a `.pdf`, and slice 3 is the first slice whose acceptance
criterion cannot be met without a file.

### `fontbox` — the root and the metrics

| Java | Go | Status |
| --- | --- | --- |
| `FontBoxFont.java` | `fontbox/fontboxfont.go` | done |
| `EncodedFont.java` | `fontbox/fontboxfont.go` | done |
| `afm/*.java` (8) | `fontbox/afm/` | done — all 8, with all 8 Java tests |
| `encoding/*.java` (4) | `fontbox/encoding/` | done — `EncodingTest` ported |

### `fontbox/ttf` — the reading path

15 of 44 files: the table directory, and the ten tables text extraction reads.

| Java | Go | Status |
| --- | --- | --- |
| `TTFDataStream`, `RandomAccessReadDataStream` | `datastream.go` | done |
| `TTFParser`, `TTFTable`, `TrueTypeFont` | `ttfparser.go`, `table.go`, `truetypefont.go` | done for the tables below |
| `HeaderTable`, `MaximumProfileTable`, `HorizontalHeaderTable`, `HorizontalMetricsTable`, `IndexToLocationTable` | `tables.go` | done |
| `NamingTable`, `NameRecord` | `naming.go` | done — minus `readHeaders`, the `FontHeaders` fast path, which is slice 4 |
| `PostScriptTable`, `WGL4Names` | `postscript.go`, `wgl4names.go` | done |
| `OS2WindowsMetricsTable` | `os2windowsmetrics.go` | done |
| `CmapTable`, `CmapSubtable`, `CmapLookup` | `cmap.go` | done |
| `GlyphTable`, `GlyphData`, `GlyphDescription`, `GlyfDescript`, `GlyfSimpleDescript`, `GlyfCompositeDescript`, `GlyfCompositeComp` | `glyph.go` | done — `GlyphData.getPath` needed `GlyphRenderer`, which slice 4 ported |
| `gsub/`, `TTFSubsetter`, `TrueTypeCollection`, `OTFParser`, `OpenTypeFont`, `CFFTable`, the vertical and kerning tables | — | not started when slice 3 closed; ported in slice 4 |

Every table the parser does not read keeps its place in the directory as an
`UnknownTable`, so a later slice can add the read without the file being walked
again.

### `pdmodel/font/encoding` — all 12 files

| Java | Go | Status |
| --- | --- | --- |
| `Encoding`, `BuiltInEncoding`, `DictionaryEncoding`, `Type1Encoding` | `encoding.go`, `dictionaryencoding.go` | done |
| `StandardEncoding`, `WinAnsiEncoding`, `MacRomanEncoding`, `MacOSRomanEncoding`, `MacExpertEncoding`, `SymbolEncoding`, `ZapfDingbatsEncoding` | `encodings.go`, `tables.go` | done — the seven tables generated from the Java |
| `GlyphList` | `glyphlist.go` | done |

`glyphlist.txt`, `zapfdingbats.txt` and `additional.txt` are copied byte for
byte into `pdfbox/resources` and embedded, since Go has no classpath.

### `pdmodel/font` — the simple font path

| Java | Go | Status |
| --- | --- | --- |
| `PDFontLike`, `PDVectorFont` | `pdfont.go` | done |
| `PDFont`, `PDSimpleFont` | `pdfont.go`, `pdsimplefont.go` | done — minus the ToUnicode CMap, below |
| `PDFontDescriptor`, `PDPanose`, `PDPanoseClassification` | `pdfontdescriptor.go`, `pdpanose.go` | done |
| `Standard14Fonts`, `UniUtil` | `standard14fonts.go` | done — the 15 AFM files embedded |
| `PDType1Font` | `pdtype1font.go` | partial — the standard 14 path is whole; the embedded PFB needs `fontbox/type1` and the substitute needs the font mapper, both slice 4 |
| `PDTrueTypeFont` | `pdtruetypefont.go` | done — an embedded font is read whole; the substitute for one that is not embedded arrived with slice 4's font mapper, and the `load` factories that write one with `track/font-embedding`, in `pdtruetypefont_embed.go` |
| `PDType3Font`, `PDType3CharProc` | `pdtype3font.go`, `pdtype3charproc.go` | done |
| `PDFontFactory` | `pdfontfactory.go` | partial here — Type 1, TrueType and Type 3; Type 0, Type 1C, Multiple Master and the CID fonts reported that they were not ported, and slice 4 added them |
| `PDType0Font`, `PDCIDFont*`, `PDType1CFont`, `PDMMType1Font`, the `FontMapper` chain, every `*Embedder`, `Subsetter`, `ToUnicodeWriter`, `CMapManager`, `FontCache`, `FileSystemFontProvider` | — | done elsewhere — slice 4 took all but the embedders, slice 7 took `ToUnicodeWriter`, and `track/font-embedding` took `TrueTypeEmbedder`, `PDTrueTypeFontEmbedder`, `PDCIDFontType2Embedder` and `Subsetter` |

### `pdmodel` — the holes slice 2 left

`PDResources.getFont` with its direct font cache, `ResourceCache` and
`DefaultResourceCache` for the font and font descriptor members, the font field
of `PDTextState`, and the reading half of `common/PDStream`. The stable-cache
bookkeeping in `removeFont` is ported as it stands, since it decides whether a
re-read gives the same font object.

### `contentstream` — the text path

The five text-showing operators, and the engine methods behind them: `showText`,
`showTextString`, `showTextStrings`, `applyTextAdjustment`, `showGlyph`,
`showFontGlyph`, `showType3Glyph`, `processType3Stream` and `getDefaultFont`.
The four glyph hooks join `StreamEngineOverrides`.

### `pdfbox/text` — 6 files

| Java | Go | Status |
| --- | --- | --- |
| `TextPosition`, `TextPositionComparator` | `textposition.go` | done |
| `LegacyPDFStreamEngine` | `legacystreamengine.go` | done — minus the `DrawObject` processor, which walks into a form XObject |
| `PDFTextStripper` | `pdftextstripper.go` | partial — the page walk is whole; `getText(PDDocument)`, `writeText`, the bookmark range and the article beads need types this slice does not carry |
| `PDFTextStripperByArea` | `pdftextstripperbyarea.go` | done |
| `PDFMarkedContentExtractor` | `pdfmarkedcontentextractor.go` | done — minus the XObject walk |
| `pdmodel/documentinterchange/markedcontent/PDMarkedContent` | `.../pdmarkedcontent.go` | done — `PDArtifactMarkedContent` folded in, since only its tag matters here |

`golang.org/x/text` is the module's first dependency, for the NFKC
normalisation and the bidi reordering the stripper needs and the Go standard
library does not carry.

### The loader — slice 1's unfinished half, ported here

`PLAN.md` slice 1 is "open a document" and lists `pdfbox/pdfparser` at 18 files;
the branch was merged to `migration-base` at 12 of 18, with `COSParser` "next"
and `PDFParser` "not started — the entry point", and `go/cmd/` empty. Slices 2
and 3 did not notice, because both take a `PDPage` a caller hands them.

| Java | Go | Status |
| --- | --- | --- |
| `COSParser.java` (the file half) | `pdfparser/fileparser.go` | done — minus encryption |
| `XrefParser.java` | `pdfparser/xrefparser.go` | done |
| `PDFXrefStreamParser.java` | `pdfparser/xrefstreamparser.go` | done |
| `PDFObjectStreamParser.java` | `pdfparser/objectstreamparser.go` | done |
| `BruteForceParser.java` | `pdfparser/bruteforceparser.go` | done |
| `PDFParser.java` | `pdfparser/pdfparser.go` | done — returns a `cos.Document`, and `Loader` wraps it |
| `Loader.java` | `pdfbox/loader.go` | done — the reading entry points |
| `PDDocument.java` | `pdmodel/pddocument.go` | partial — the reading path; signatures, form fields, importing a page and saving each need a package this port has not reached |
| `PDDocumentCatalog.java` | `pdmodel/pddocument.go` | partial — the pages and the version; forms, outlines, names, threads, metadata and actions wait on their types |
| `PDDocumentInformation.java` | `pdmodel/pddocument.go` | partial here — minus the dates, `getTrapped`, `getMetadataKeys` and `getPropertyStringValue`, which slice 8 added with `DateConverter` |
| `PDFXRefStream.java`, `EndstreamFilterStream.java`, `FDFParser.java` | `pdfparser/pdfxrefstream.go`, `pdfparser/streamparser.go`, `pdfparser/fdfparser.go` | not started here -- `PDFXRefStream` is the writing path and slice 7 took it; `EndstreamFilterStream` had landed with slice 1's stream half already; `FDFParser` is FDF, and slice 8 took it |

### The corpus — 16 of 40

`TestTextStripper` walks the 40 PDFs of `pdfbox/src/test/resources/input`
([`TESTDATA.md`](TESTDATA.md)) and scores rather than asserts, because a
document needing something this slice does not carry cannot match and a failing
assertion would say nothing new.

**40 of 40 open. 16 of 40 match the expected text exactly, sorted or
unsorted.** The 24 that do not:

| Cause | Files | Where it lands |
| --- | ---: | --- |
| Type 0 font | 12 | slice 4 — `PDType0Font` and the CID fonts |
| Type 1C font | 3 | slice 4 — `fontbox/cff` |
| ToUnicode CMap | 4 | slice 4 — `fontbox/cmap` |
| Article beads | 2 | needs `PDThreadBead`; both are `PDFBOX-3110-poems-beads` |
| Yields nothing, cause not yet established | 2 | `PDFBOX-3498-…` and `Liste732004001452_…` |
| One line differs in spacing | 1 | `cweb.pdf` line 249 |

### Deviations from Java

- **`TrueTypeFont.readTable` is not locked**, matching Java, which locks only
  `getTableBytes`. A table read reaches back into the font for the tables it
  depends on; a lock there would deadlock where Java simply re-enters its own
  monitor. `GlyphTable` does lock its own stream, so the recursion a composite
  glyph makes goes through `getGlyphLocked`. Commented where it is.
- **`GlyfCompositeDescript` shadows the parent's `contourCount`** with a field
  of its own. Go embedding does not shadow, so the composite count has its own
  name.
- **Three `HashMap` walks are ordered.** `BuiltInEncoding`,
  `Type1Encoding.fromFontBox` and the reverse glyph list depend in Java on an
  iteration order it leaves unspecified; the port walks the keys in order, so
  the same input always gives the same encoding.
- **`ResourceCache` names `PDFont`**, and Java puts it in `pdmodel`, which this
  package's parent imports. The interface is declared in `pdmodel/font` and
  `pdmodel` aliases it back under its Java name.
- **`PDType3Font.getResources` and `PDType3CharProc.getResources` return a
  `PDResources`**, likewise in `pdmodel`. They hand out the resource dictionary
  instead, and `contentstream` wraps it where the engine needs one.
- **The `PDFont` constructor calls the abstract `getName`.** Java dispatches
  virtually from a constructor and Go does not, so the port splits it: the
  concrete font sets `self`, then calls `initFromDictionary`.
- **The resource cache holds its entries outright.** Java holds every entry
  through a `SoftReference`; Go has none.
- **The default line separator is a line feed, not `System.lineSeparator()`.**
  Java's choice makes the text a document yields depend on the machine that read
  it. A caller wanting the platform separator sets it.
- **The sort is always `IterativeMergeSort`.** Java tries the JDK sort first and
  falls back when it throws on the intransitive comparator; Go's sort does not
  check, so trying it first would give a different order on exactly the
  documents the fallback exists for.
- **A cross-reference table is keyed on the packed object number and
  generation**, which is what `COSObjectKey.equals` compares — the stream index
  is left out. `XrefEntries` carries that, because a Go map on the key struct
  would compare the stream index too and split entries Java merges.
- **`cos.Document` gained `XRefOffset`, `PutXRefOffset` and `ClearXRefTable`,
  and `XrefTrailerResolver` gained `ReplaceXrefTable`.** Java writes through the
  live map its getter returns; the port's getters return copies, so each write
  is a method.
- **`PDPageTree` and `PDPage` now carry the `ResourceCache`.** Java passes a
  `PDDocument` into the reading constructor, which is only there to reach that
  cache. This closes the slice 1 hole recorded above.
- **`PDFTextStripperByArea` does not reimplement `ProcessPage`**, since
  2026-09-15. Go embedding does not dispatch, so the base `ProcessPage` writes
  the page through a hook the by-area stripper installs its `WritePage` in, and
  `charactersByArticle` is a pointer to the lists rather than a slice, so the
  base clearing and extending a region's lists in place reach the region as
  Java's shared `ArrayList` does.

### Deferred, and where each closed

- **The ToUnicode CMap.** `fontbox/cmap` is slice 4, so `PDFont.toUnicode`
  always fell through to what each font works out from its encoding, the path a
  font carrying no `/ToUnicode` takes. **Closed by slice 4.**
- **No font program for a font that is not embedded.** The font mapper chain and
  `fontbox/util/autodetect` are slice 4; every path that would read a width or
  an outline out of a substitute reported that rather than guessing, and the
  widths of a standard 14 font come from its AFM and were unaffected. A symbolic
  TrueType font that is not embedded therefore aborted the page, its encoding
  being synthesised from a font program that was not there, and the error
  travelled out through `Tf`. **Closed by slice 4:** the mapper always returns a
  font, down to the last-resort LiberationSans.
- **No glyph outlines.** `GlyphRenderer` and the CFF charstrings were listed
  here as slice 9; both landed in slice 4. **Closed.**
- **No embedded Type 1 program.** `fontbox/type1` is slice 4; until then such a
  font reads as damaged, which is what Java does with one it cannot parse.
  **Closed by slice 4.**
- **`processChildStream`**, so a form XObject was not walked. **Slice 9 ported
  it**, into `contentstream/forms.go`.
- **Encryption.** `pdmodel/encryption` is a package this port had not reached,
  so an encrypted document is reported rather than decrypted. **Closed by slice
  5.**
- **`TestCMapSubtable`, `TestTTFParser.testParseVertical` and
  `testParseHeaders`** read fonts the Java build downloads into `target/fonts`,
  which this repository did not carry. **[input present since 2026-09-11, and
  all three are ported; see "What the fetch unblocked"]**

### Defects found and fixed

- **`PDFont.getSpaceWidth` let a panic escape where Java catches.** A Type 3
  font's `encode` throws `UnsupportedOperationException` outright, so Java's
  catch is the ordinary path for every Type 3 font rather than an edge case;
  the escaping panic took down the whole page walk. `stringWidthOfSpace` now
  recovers it.
- **Four length comparisons counted runes where Java counts UTF-16 units** —
  `TextPosition.mergeDiacritic`, `isDiacritic`, and the duplicate-suppression
  tolerance in both `PDFTextStripper` and `PDFMarkedContentExtractor`. The port
  merged diacritics Java leaves alone and used half the tolerance; `utf16Length`
  now counts the way Java does at each. `text/review_test.go`.
- **`hasFontOrSizeChanged` dropped Java's fallback to `PDFont.hashCode`** where
  both font names are null. A Type 3 font with no `/Name` returns the empty
  string, so two different unnamed Type 3 fonts compared equal and the running
  average character width was never reset between them. `text/review_test.go`.
- **`removeContainedSpaces` did not shrink the article.** Java removes through
  the list iterator; the port returned a new slice and assigned it only to the
  local, so anything reading `getCharactersByArticle` after a page still saw the
  space. `text/review_test.go`.
- **`multiplyFloat` widened before multiplying.** Java multiplies in `float` and
  rounds that; the port's `float64` rounds the other way either side of a half.
  It decides whether a line is indented enough to start a paragraph.
  `text/review_test.go`.
- **`isDigitAt` and `XrefStreamParser.readNextValue` swallowed every read
  failure.** Java's `RandomAccessRead.read` throws for a failure and returns -1
  only at the end of the data, and both callers distinguish the two; both now
  return the error unless it is `io.EOF`.
- **`PDFTextStripperByArea.ProcessPage` did not clear the duplicate map**, so a
  stripper used twice reported nothing the second time. The reimplementation it
  lived in is gone, above; `TestStripperByAreaWriteTextReachesItsWritePage`.
- **`GetTextOfPages` did not reset the engine.** Java's `writeText` calls
  `resetEngine` first; the port left the page number where the previous call had
  pushed it.
- **`parseIntRadix` parsed at 64 bits**, where `Integer.parseInt` rejects
  anything outside the 32-bit range. An AFM carrying `Characters 2147483648` was
  accepted here and then looped on.
- **`readInternationalDate` overflowed.** A `time.Duration` is int64 nanoseconds
  and reaches about 292 years, so a `LONGDATETIME` past 2196 wrapped and came
  back as a date in the past. Both copies of the read now build the instant from
  the seconds.
- **The corpus harness only ran unsorted**, where Java runs every file against
  both `<name>.pdf.txt` and `<name>.pdf-sorted.txt`.
- **A Java bug the port had corrected, reverted:**
  `PDFTextStripper.handleDirection`, [`JAVA-BUGS.md`](JAVA-BUGS.md) entry 15,
  pinned by `text/feedback_test.go`.

### Reviewed and declined

- **The `aux` clone in `IterativeMergeSort` is dead but stays.** `mergeRuns`
  overwrites `aux[from:to]` before copying it back, so the initial copy is never
  read; Java writes `T[] aux = arr.clone()` and the port writes the clone.
- **`minYTopForLine` is computed and never read**, in the port as in Java, whose
  own comment says the check it was meant for caused regression failures.

### Still open

- `shouldProcessColorOperators` is always true: the Type 3 char proc case that
  clears it is `d0`/`d1` handling, which belongs with the renderer.

## Slice 4 — text from CID and CFF fonts

Branch `slice/4-text-cid-cff`. The slice finishes `fontbox` — the CFF and Type 1
font programs, the CMaps, the rest of the TrueType tables and the GSUB shaping
machinery — and the CID half of the font model, including the font mapper that
finds a substitute on the machine for a font a PDF does not embed. With it,
`fontbox` is whole: all 143 Java classes have a Go counterpart.

### `fontbox/cmap` — all 5 files

| Java | Go | Status |
| --- | --- | --- |
| `CMap`, `CMapParser`, `CMapStrings`, `CIDRange`, `CodespaceRange` | `cmap.go`, `cmapparser.go`, `cmapstrings.go` | done — all 5 Java tests |

The predefined CMaps under `org/apache/fontbox/resources/cmap` are copied byte
for byte into `fontbox/resources` and embedded; Go has no classpath.

### `fontbox/type1` and `fontbox/pfb` — all 7 files

| Java | Go | Status |
| --- | --- | --- |
| `Type1Font`, `Type1Parser`, `Type1Lexer`, `Token`, `DamagedFontException` | `type1font.go`, `type1parser.go`, `lexer.go`, `token.go` | done — `Type1LexerTest` and the `Type1Font` half of `PfbParserTest` |
| `Type1CharStringReader` | `cff/type1charstring.go` | moved — see the cycle note below |
| `pfb/PfbParser` | `pfb/pfbparser.go` | done — 3 of the 5 `PfbParserTest` cases; the other 2 need a font this repository does not carry |

### `fontbox/cff` — all 26 files

| Java | Go | Status |
| --- | --- | --- |
| `CFFParser`, `CFFFont`, `CFFCIDFont`, `CFFType1Font`, the four charsets and the two encodings, `CFFStandardString`, `CFFOperator`, `CharStringCommand`, `Type1CharString`, `Type2CharString`, the two charstring parsers, `DataInput` and its two implementations, `CharStringHandler`, `IndexData`, `FDSelect` and its two formats | `cff/` (15 files) | done — 5 of the 6 Java tests |

`CFFParserTest` skips: it reads a font the Java build downloads into
`target/fonts`. **[input present since 2026-09-11; see "What the fetch unblocked"]**

### `fontbox/ttf` — the other 29 files, and the 39 of the GSUB tree

| Java | Go | Status |
| --- | --- | --- |
| `OTFParser`, `OpenTypeFont`, `CFFTable`, `OTLTable` | `opentype.go` | done |
| `TrueTypeCollection`, `TTCDataStream`, `RandomAccessReadUnbufferedDataStream` | `collection.go` | done — `TrueTypeFontCollectionTest`, 2 of its 3 collections present on this machine |
| `FontHeaders`, and the `readHeaders` of every table that has one | `fontheaders.go` | done |
| `GlyphRenderer` | `glyphrenderer.go` | done |
| `TTFSubsetter` | `ttfsubsetter.go` | done — 5 of the 8 `TTFSubsetterTest` cases |
| `VerticalHeaderTable`, `VerticalMetricsTable`, `VerticalOriginTable`, `DigitalSignatureTable` | `verticaltables.go` | done |
| `KerningTable`, `KerningSubtable` | `kerning.go` | done |
| `OpenTypeScript` | `opentypescript.go`, `opentypescripttable.go` | done — `Scripts.txt` embedded, 953 ranges |
| `GlyphSubstitutionTable`, `SubstitutingCmapLookup` | `glyphsubstitution.go`, `opentype.go` | done — both `GlyphSubstitutionTable*Test` |
| `gsub/*.java` (13) | `ttf/gsub/` | done — `CompoundCharacterTokenizerTest`, `GlyphArraySplitterRegexImplTest`, `DefaultGsubWorkerTest`, `GsubWorkerForLatinTest`, `GsubWorkerForBengaliTest`, `GsubWorkerForDfltTest` |
| `model/*.java` (5) | `ttf/model/model.go` | done |
| `table/common/*.java` (12) | `ttf/table/common/common.go` | done |
| `table/gsub/*.java` (9) | `ttf/table/gsub/gsub.go` | done |

Go's `regexp` is leftmost-first the way Java's is, so the longest-first
alternation the compound-character tokenizer builds carries over unchanged.

`TrueTypeFont.getPath` reads the glyph and returns its path now that
`GlyphRenderer` is here; it was the last "not ported yet" in `fontbox`.

### `fontbox/util/autodetect` — all 7 files

| Java | Go | Status |
| --- | --- | --- |
| `FontDirFinder`, `NativeFontDirFinder`, `UnixFontDirFinder`, `MacFontDirFinder`, `OS400FontDirFinder`, `WindowsFontDirFinder`, `FontFileFinder` | `util/autodetect/autodetect.go` | done |

### `pdmodel/font` — the CID half, and the font mapper chain

34 of 39 files when the slice closed. The five it left were the embedders:
slice 7 took `ToUnicodeWriter` and `track/font-embedding` took
`TrueTypeEmbedder`, `PDTrueTypeFontEmbedder`, `PDCIDFontType2Embedder` and
`Subsetter`, so the package is 39 of 39.

| Java | Go | Status |
| --- | --- | --- |
| `PDType0Font` | `pdtype0font.go` | done |
| `PDCIDFont`, `PDCIDSystemInfo` | `pdcidfont.go` | done |
| `PDCIDFontType0` | `pdcidfonttype0.go` | done |
| `PDCIDFontType2` | `pdcidfonttype2.go` | done |
| `PDType1CFont`, `PDMMType1Font` | `pdtype1cfont.go`, `pdmmtype1font.go` | done |
| `CMapManager` | `cmapmanager.go` | done |
| `PDFont.toUnicode` and the ToUnicode CMap | `pdfont.go` | done — the hole slice 3 left |
| `FontMapper`, `FontMappers`, `FontMapperImpl` | `fontmapper.go`, `fontmappers.go`, `fontmapperimpl.go` | done — `CIDCharSetMatchTest` |
| `FontMapping`, `CIDFontMapping` | `fontmapping.go` | done |
| `FontProvider`, `FileSystemFontProvider` | `fontprovider.go`, `filesystemfontprovider.go` | done |
| `FontCache`, `FontInfo`, `FontFormat`, `CIDSystemInfo` | `fontcache.go`, `fontinfo.go`, `fontformat.go`, `cidsysteminfo.go` | done |

With the mapper here, the five font classes that had no font program for an
unembedded font — `PDType1Font`, `PDType1CFont`, `PDTrueTypeFont`,
`PDCIDFontType0`, `PDCIDFontType2` — all take the Java path and get a
substitute. Every "the font mapper is not ported yet" error is gone.

`PDCIDFontType0SubstituteTest` is ported and skips on this machine, as its Java
`assumeTrue` does: it needs a CID-keyed Adobe-CNS1 substitute installed.

### The corpus — 34 of 40 unsorted, 33 sorted

Slice 3 left it at **16 of 40 either way**; the Type 0 fonts, the Type 1C fonts
and the ToUnicode CMaps that accounted for 19 of the 24 failures are all read
now. See [`TESTDATA.md`](TESTDATA.md) for the corpus itself.

**40 of 40 open. 34 of 40 match unsorted, 33 sorted.** The 6 that do not:

| Cause | Files | Where it lands |
| --- | ---: | --- |
| Text drawn through a form XObject (`Do`) | 3 | the graphics slice — `PDFBOX-3498-…`, `PDFBOX-4322-Empty-ToUnicode-reduced`, `Liste732004001452_…` |
| Article beads | 2 | needs `PDThreadBead` in `pdmodel/interactive/pagenavigation`; both are `PDFBOX-3110-poems-beads` |
| Arabic diacritics ordered differently | 1 | `FC60_Times.pdf`; the merge logic matches the Java line for line, so the difference is upstream in the widths or in `XDirAdj` |

Sorted mode fails one more, `PDFBOX-3127-…VFont.pdf`, on a single missing space.

### Font substitution is environment-dependent — measured

`FileSystemFontProvider` scans the machine's fonts, so the same corpus could
score differently elsewhere. It does not: **with every system font directory
hidden from the finder the corpus scores 34 of 40**, the same as with the 215
fonts this machine has installed. Every lookup falls to the embedded
LiberationSans instead of to Arial, and nothing changes, because text extraction
takes its widths from the PDF's `/Widths` and `/W` arrays rather than from the
substitute.

The floor is guaranteed rather than incidental: `getTrueTypeFont`,
`getFontBoxFont` and `getCIDFont` all end at `lastResortFont`, which is compiled
into the binary. None of the three can return nil, and no document can fail for
want of a font on the machine.

### Deviations from Java

- **`CMapStrings` builds its 65536-entry and 256-entry tables eagerly**, where
  Java fills them in a static block. `getMapping` and `getIndexValue` return the
  comma-ok pair Java gets from a `null` return.
- **`PfbParser` accumulates its record size in `int32`**, so a record whose
  length byte sets bit 31 goes negative and trips the "record size is negative"
  check the way Java's 32-bit `int` does.
- **`Type1Parser.decrypt` needed brackets**: Java's `&` binds looser than `+`,
  so the whole sum is masked, while Go's binds tighter.
- **`DataInput.readByte` is `ReadSignedByte`**, because `go vet` reserves
  `ReadByte() (byte, error)` for `io.ByteReader`.
- **`cff` ↔ `type1`.** `Type1CharStringReader` lives in `type1` in Java and is
  implemented by `CFFType1Font`. The port declares it in `cff` and aliases it
  back from `type1`.
- **`cff` ↔ `ttf`.** `CFFParser.parseFirstSubFontROS` writes into a
  `FontHeaders`, which lives in `ttf`, which reads `cff` for its `CFF ` table.
  The port declares a `FontHeadersSink` interface in `cff` that `FontHeaders`
  satisfies.
- **`gsub` ↔ `ttf`.** `GsubWorker` takes a `CmapLookup`, which lives in `ttf`,
  which reaches into `gsub` for its worker factory. The port declares the
  two-method `CmapLookup` in `gsub` as well.
- **`model.GlyphKey` stands for Java's `List<Integer>` map key**, which Go
  cannot use: it is the ids joined with commas.
- **`autodetect` is a substitution rather than a transliteration**, as the task
  file called for. Java reads the `os.name` and `env.windir` system properties
  and the port reads `runtime.GOOS` and the environment; `File.isHidden` becomes
  a leading dot; `find()` gives back paths rather than URIs, which is what every
  caller turned them into anyway.
- **`FontCache` holds its fonts outright.** Java holds each through a
  `SoftReference`, so the collector may drop one and the next lookup re-reads it
  from disk. Go has no soft reference and no hook that stands in for one.
  Nothing observes the difference beyond memory use.
- **`java.util.PriorityQueue` is ported, not replaced with a sort.**
  `getFontMatches` scores every candidate and polls the best; two candidates of
  equal score come back in the order the heap gives them, which a sort would
  order differently. `siftUp` and `siftDown` are written out, and so is
  `Double.compare`.
- **`FontMapperImpl.fontInfoByName` keeps its insertion order.** Java's is a
  `LinkedHashMap`, and the order decides which of two equally good candidates
  wins. The port carries the key order in a slice beside the map.
- **The two system properties are package variables.** `pdfbox.fontcache` is
  `font.FontCacheDir` and `pdfbox.fontcache.skipchecksums` is
  `font.SkipChecksums`, each with the Java default. The on-disk `.pdfbox.cache`
  is written in exactly the Java format, including the platform line separator,
  so the two implementations can read each other's.

### Defects found and fixed

- **`PDFont.getSpaceWidth` had lost its first branch**, a stand-in written while
  `fontbox/cmap` was unported. Java measures the space at the code the
  `/ToUnicode` CMap maps to U+0020 when the font carries one, and takes the
  encoding branch only otherwise; the port always took the encoding branch,
  which moves where words break. `font/review_test.go`.
- **`PDCIDFontType2.codeToGID` swallowed an error Java lets out.**
  `name.equals(ttf.getName())` throws `IOException` and `codeToGID` declares it;
  the port discarded it and compared against the empty string, so a font whose
  name could not be read silently took the ToUnicode fallback. Fixed behind the
  same short-circuit Java's `&&` gives it.
- **`FileSystemFontProvider.createFSIgnored` had been quietly corrected**, the
  port passing the parent provider where Java passes `null`. Reverted to the
  Java; [`JAVA-BUGS.md`](JAVA-BUGS.md) entry 21.
- **`PDType0Font` was missing `getGsubData` and `getCmapLookup`.** Both were
  filed under the embedding half, but the reading constructor sets them too, to
  `GsubData.NO_DATA_FOUND` and `null`; they are now there with those values.
- **`CMap.toInt` accumulated at 64 bits.** Java's accumulator is an `int`, so a
  four-byte code whose first byte is 0x80 or more comes out negative, and
  `toUnicode(int)` — which tests the code against 256, 0xFFFF and 0xFFFFFF to
  work out how many bytes it had — then takes the two-byte branch and misses the
  mapping stored under that negative key, where the port found it.
  `cmap/feedback_test.go` pins both the arithmetic and the lookup it decides.
- **`PDCIDFont.readVerticalDisplacements` swallowed a malformed `/W2`.** Java
  casts every entry with `(COSNumber)` and indexes past the end without
  checking, so a bad array throws out of the font's constructor; the port's
  warning and partial read left some vertical metrics filled in and the rest
  defaulted. The casts are back, and a failed type assertion or an out-of-range
  index panics where Java throws.
- **`KerningTable.read` did not narrow its version 1 subtable count.** Java
  casts the unsigned count to a signed 32-bit `int`, so a count with bit 31 set
  goes negative and the `> 0` check skips it; the Go kept it positive and would
  have sized an allocation on it. The branch is unreachable —
  [`JAVA-BUGS.md`](JAVA-BUGS.md) entry 22 — so no test reaches it, but the cast
  is written out.
- **`PDFontFactory.createFont` was missing the Type 0 `/Subtype` repair**, a
  deferral left over from before `PDType0Font` was ported. `fixType0Subtype`,
  `getFontTypeFromFont`, `getFontHeader`, `getFontDescriptor`,
  `getDescendantFont`, the six header sniffers and the `FontType` inner class
  are now ported, so a Type 0 font whose descendant's `/Subtype` disagrees with
  the embedded font program has both the subtype and the `/FontFile2` ↔
  `/FontFile3` entry corrected as Java does, with the two dropped log lines.

Every ignored error in the slice was swept: all but `PDCIDFontType2.codeToGID`
above are comma-ok type assertions, and `CMap.readCode` ignores its read count
because Java does.

### Five more Java tests ported

`GsubWorkerForDevanagariTest`, `GsubWorkerForGujaratiTest`,
`GsubWorkerForTamilTest`, `GsubWorkerForAaltTest` and `GsubWorkerForSmcpTest`
had been passed over without a recorded reason, and all five run here: three
read fonts this repository carries, one reads `otf/FoglihtenNo07.otf`, and the
last reads Calibri, which its own `assumeTrue` guards. They are the first
coverage the shared reph worker has beyond Bengali, and the first of the type 3
alternate and type 2 multiple substitution paths.

`GsubWorkerForAalt` and `GsubWorkerForSmcp` live in the Java *test* tree rather
than the library — each is the Latin worker with one feature of its own, with
`applyGsubFeature` copied rather than shared — so the Go test copies them the
same way. The `@Disabled` cases are left out, as the Bengali port leaves its two
out: Devanagari drops `rkrf`, `cjct`, `abvs` and `psts`, and Gujarati drops
`psts`.

### Reviewed and declined — slice 4

- **`CMapStrings.getMapping` reads a zero-length code as the two-byte code 0.**
  Java does exactly this, because its ternary has two arms for three cases.
  Carried as written, [`JAVA-BUGS.md`](JAVA-BUGS.md) entry 23, and commented at
  the site so it is not "fixed" later.
- **`createDescendantFont` names the `/Type` in its error, not the `/Subtype`
  that failed to match.** Java does that too. The port now concatenates the
  `COSName` rather than its name, as Java does, so the message reads
  `Invalid font type: COSName{Font}` on both sides.

## Slice 5 — encrypted documents

Branch `slice/5-encryption`. The slice reads a password-protected or
certificate-protected PDF: the standard security handler for revisions 2 to 6,
the public key handler, and the wiring that decrypts every string and stream as
the parser reads it.

### `pdmodel/encryption` — 17 of 19 files

| Java | Go | Status |
| --- | --- | --- |
| `AccessPermission` | `accesspermission.go` | done |
| `InvalidPasswordException` | `decryptionmaterial.go` | done — an error type, since Java's extends IOException |
| `DecryptionMaterial`, `StandardDecryptionMaterial` | `decryptionmaterial.go` | done |
| `PublicKeyDecryptionMaterial`, `PublicKeyRecipient` | `publickeymaterial.go` | done |
| `ProtectionPolicy`, `StandardProtectionPolicy`, `PublicKeyProtectionPolicy` | `protectionpolicy.go`, `publickeymaterial.go` | done |
| `PDEncryption` | `pdencryption.go` | done |
| `PDCryptFilterDictionary` | `pdcryptfilterdictionary.go` | done |
| `RC4Cipher` | `rc4cipher.go` | done |
| `SaslPrep` | `saslprep.go` | done |
| `SecurityHandler` | `securityhandler.go` | done |
| `SecurityHandlerFactory` | `securityhandlerfactory.go` | done — a registry of constructors, since Java builds by reflection |
| `StandardSecurityHandler` | `standardsecurityhandler.go` | done |
| `PublicKeySecurityHandler` | `publickeysecurityhandler.go` | the reading half; see below |
| `MessageDigests` | — | not ported — three JCE lookups for MD5, SHA-1 and SHA-256, which are `crypto/md5`, `crypto/sha1` and `crypto/sha256` here |
| `SecurityProvider` | — | not ported — it holds a JCE Provider, and Go has no provider to hold |

`SecurityHandler` is an interface plus an embedded struct, the way the port
takes every abstract class; Java's type parameter `<TPOLICY extends
ProtectionPolicy>` says which policy a handler takes, and the two concrete
handlers narrow it themselves.

### What the tests reach

`TestSymmetricKeyEncryption.testPermissions` is ported whole, and its three
files cover the reading path end to end: **R2/V1** (RC4-40), **R3/V2**
(RC4-128) and **R6/V5** (AES-256), so `RC4Cipher`, algorithm 2, algorithm 2.A,
algorithm 2.B and `SaslPrep` all run. The four read-only tests of
`TestPublicKeyEncryption` are ported and pass. Every PDF and keystore those
tests read was made by something other than this port, three of them by Adobe
Acrobat, so nothing in the slice is checked against itself.

`fromsource_test.go` covers what neither reaches, and names it at the top:
`AccessPermission`'s bit arithmetic and its read-only lock,
`getPermissionBytesForPublicKey`, the protection policies, the handler factory,
`PDEncryption`'s setters, and the two ciphers. Its values come from outside the
port — RFC 2268's test vectors for RC2, Go's own `crypto/rc4` for RC4, and
RFC 4013's worked examples for SASLprep.

### Infrastructure the port supplies, which is not a migration

Java hands three things to BouncyCastle and the JCE. Go's standard library has
none of them and PDFBox has no code of its own to port for them, so this branch
writes them, and each file says so at the top:

- **`cms.go`** reads a CMS enveloped-data blob: the key transport recipients,
  their identifiers, and the content once the RSA key has unwrapped it. Only
  reading; the encrypting half would need an encoder.
- **`pkcs12.go`** reads a PKCS#12 keystore — the RFC 7292 SHA-1 derivation, the
  MAC, 3DES for the shrouded key bags and 40-bit RC2 for the certificate bags,
  which is what the checked-in keystores use.
- **`rc2.go`** is RC2 from RFC 2268, which nothing in Go has and the
  certificate bags need.

### Deviations from Java

- **The security handler's `objects` set is a Go map keyed on the interface.**
  Java uses an IdentityHashMap-backed set, for the reason its comment gives —
  two equal COSStrings must not be conflated. A Go map keyed on an interface
  holding a pointer already compares by identity, which is what that buys.
- **`ProtectionPolicy.setEncryptionKeyLength` returns an error** where Java
  throws IllegalArgumentException. The caller is asking for a key length a
  document cannot carry, not making a mistake it cannot see.
- **The AES-256 path keeps the plaintext of every block before a bad final
  one.** Java reads through a CipherInputStream, which has already written
  those blocks by the time `doFinal` throws, and swallows the exception; the
  port returns the same bytes. The AES-128 path does *not* swallow it, in Java
  or here.
- **`SaslPrep`'s prohibited-character message names the code point**, where
  Java names the character with `Character.getName`; Go has no name table.
- **`logIfStrongEncryptionMissing` does nothing.** It warns when the JCE
  unlimited strength policy files are missing, and Go has no key length policy.
  The call sites keep it so that the two read the same.
- **The error strings are Java's, character for character**, so that a caller
  matching on them sees the same text.
- **`LoadPDFFromWithKeyStore` takes one password for two jobs.** Where a
  keystore is given, the same string opens it — Java calls
  `KeyStore.load(keyStore, password.toCharArray())` — and is then handed on to
  `PublicKeyDecryptionMaterial` as the private key password. The keystore and
  the password are not alternatives.

### Deferrals

- **`PublicKeySecurityHandler.prepareDocumentForEncryption`** reports an error.
  It builds one CMS enveloped-data blob per recipient, which needs an encoder
  Go does not have; and nothing can save a document until slice 7, so it cannot
  be exercised either way. It is the one method of the nineteen files that does
  not do what the Java does.
- **`StandardSecurityHandler.prepareDocumentForEncryption` is ported** and does
  what Java does; it has no test, because a test would have to save.
- **The tests that encrypt and save are slice 7's**: `testProtection`,
  `testProtectionInnerAttachment`, `testPDFBox4308` and `testPDFBox4453` of the
  symmetric test, and `testProtection`, `testProtectionError` and
  `testMultipleRecipients` of the public key one. The branch ports the reading
  half and the encrypting code that needs no writer.
- **`testPDFBox5955` and `testPDFBox5639`** read PDFs the Java build downloads
  into `target/pdfs`, which this repository did not carry **[input present since
  2026-09-11; see "What the fetch unblocked"]**.

### Defects found and fixed

- **RC2's key expansion divided by zero.** `tm := byte(255 % (1 << n))` gives
  the untyped `1` the byte type of the conversion around it, so `1 << 8` is 0;
  a defect the port introduced, since Java's `1` is an int. `rc2.go`;
  `TestRC2Vectors`.
- **`validatePerms` refused a short /Perms with an error.** Java turns only a
  *misaligned* one into an IOException, through the IllegalBlockSizeException
  the cipher raises; a missing or empty one reaches `perms[9]` and throws an
  unchecked exception. The length check now covers the aligned case and the
  indexing covers the rest. `standardsecurityhandler.go`.
- **The CMS reader could not read an RC2 envelope, which is the only kind
  PDFBox writes.** `decryptCMSContent` read the initialisation vector as a bare
  OCTET STRING, the shape AES-CBC and DES-EDE3-CBC use; RFC 3370 section 5.3
  wraps RC2-CBC's version and IV in a SEQUENCE, so the unmarshal failed with a
  tag mismatch before the switch could reach the RC2 branch, and every
  `/Recipients` entry Java's `createDERForRecipient` produces would have been
  refused. The four checked-in public key fixtures use AES and 3DES, so nothing
  caught it. `cms.go`; `TestCMSContentParameters`.
- **A wrong keystore alias does not fail.** `PublicKeyDecryptionMaterial`
  ignores the alias when the store holds one entry, and all four checked-in
  keystores hold exactly one. `TestAliasIgnoredForASingleEntryKeyStore`.
- **A password-protected document opened with a keystore fails as the
  keystore**, not as incompatible material, because the keystore is loaded with
  the same password before the handler is reached.
  `TestPasswordDocumentWithKeyStore`.
- **`SecurityHandler`'s interface comment said `PrepareDocumentForEncryption`
  "prepares everything to decrypt the document".** Both implementations and
  Java's javadoc say encrypt. `securityhandler.go`.

### Java bugs

- **JAVA-BUGS.md 26** — `SecurityHandlerFactory.registerHandler` checks the
  filter name and not the policy, against its own javadoc; found here, in
  `securityhandlerfactory.go`.

## Slice 6 — the rest of the filters, and images

Branch `slice/6-filters-images`. The slice reads the image formats a PDF can
carry: the filters slice 1 left, the colour spaces those images are in, and the
image XObjects and inline images themselves.

### `pdfbox/filter` — all 23 files

| Java | Go | Status |
| --- | --- | --- |
| `Filter`, `FilterFactory` | `filter.go` | done — the factory is `ByName` and `allFilters`, the static `decode` is `Decode` |
| `ASCIIHexFilter` | `asciihex.go` | done — carries JAVA-BUGS 30 |
| `ASCII85Filter`, `ASCII85InputStream`, `ASCII85OutputStream` | `ascii85.go` | done — carries JAVA-BUGS 27 |
| `RunLengthDecodeFilter` | `runlength.go` | done |
| `LZWFilter` | `lzw.go`, `bitstream.go` | done — carries JAVA-BUGS 28 |
| `CryptFilter` | `crypt.go` | done |
| `CCITTFaxFilter`, `CCITTFaxDecoderStream`, `CCITTFaxEncoderStream` | `ccittfax.go`, `ccittfaxdecoderstream.go`, `ccittfaxencoderstream.go` | done |
| `DCTFilter` | `dct.go` | done over `image/jpeg`; see below |
| `JBIG2Filter`, `JPXFilter`, `MissingImageReaderException` | `imagereader.go` | declared, reporting the missing reader |
| `DecodeOptions` | `decodeoptions.go` | done |
| `TIFFExtension` | `tiffextension.go` | the constants the CCITT code uses |
| `FlateFilter`, `FlateFilterDecoderStream`, `IdentityFilter`, `Predictor`, `DecodeResult` | slice 1 | already done |

**JBIG2 and JPX are declared and unsupported**, which is what Java is on a
build without jbig2-imageio and the JAI Image I/O Tools: `findImageReader`
throws `MissingImageReaderException` before either filter decodes anything.
Neither format has PDFBox code to port — both are handed to the plugin — and Go
has no decoder for either. A document using one still opens; only that image is
missing, as in Java.

**PDFBox's own test build is not such a build.** `pdfbox/pom.xml` pulls
`org.apache.pdfbox:jbig2-imageio`, `com.github.jai-imageio:jai-imageio-core`
and `...:jai-imageio-jpeg2000`, all at `<scope>test</scope>` — the second and
third carry the comment that their licence forbids distributing them. So the
Java tests decode both formats and the port's do not: `TestSaveResources` skips
`JBIG2Image.pdf`, `JPXTestCMYK.pdf`, `JPXTestGrey.pdf` and `JPXTestRGB.pdf`,
four subtests that run in Java. That is a coverage gap, not parity, and it is
the only place in the suite where the port skips what Java runs.

### `pdfbox/util/filetypedetector` — all 3 files

Done, with tests written from the source. Java searches the whole array it
allocated rather than the part it filled, so a file shorter than the longest
signature is searched with trailing zeroes after it — a three byte file
`00 00 01` is detected as an ICO. The port pads to the same length so that it
reads the same files the same way.

### `pdmodel/common/function` — all 6 files, and all 11 of `type4`

Ported because `PDSeparation` and `PDDeviceN` evaluate a tint transform and are
useless without it. `TestOperators`, `TestParser` and `TestPDFunctionType4` are
ported whole.

The type 4 operand stack holds `int32`, `float32`, `bool` and
`*InstructionSequence`, matching Java's four `instanceof` distinctions exactly:
a dozen operators behave differently for an integer and a real, and collapsing
them would change the arithmetic. `not` carries JAVA-BUGS 29.

### `pdmodel/graphics/color` — 18 of 20 files

Everything but **`PDPattern`**, which takes a `PDResources` and builds pattern
dictionaries that only rendering reads, and **`PDJPXColorSpace`**, which only
`JPXFilter` constructs. Both are slice 9's, and `create` reports them the way
Java reports a colour space it cannot build.

`awt/image` is new: a `Raster` standing for the part of `java.awt.image` PDFBox
uses. Java has an interleaved raster and a banded one and PDFBox builds both;
every banded one it builds has a single band, where the two layouts are the
same, so the port stores interleaved throughout.

`convXYZtoRGB` is written out rather than handed to the platform: the D50 to
D65 Bradford adaptation folded into the sRGB primaries, then the sRGB transfer
function, which is the standard definition of Java's `CS_CIEXYZ.toRGB`. It
agrees with LittleCMS to within the rounding of the two, not bit for bit.

### `pdmodel/graphics` and `pdmodel/graphics/image` — 2 of 4, and all 9

`PDXObject` and `PDPostScriptXObject`; `PDFontSetting` and `PDLineDashPattern`
are elsewhere. The form XObject branch of `createXObject` reports that slice 9
owns it.

The image package is complete: `PDImage`, `SampledImageReader`,
`PDImageXObject`, `PDInlineImage`, `JPEGFactory`, `LosslessFactory`,
`CCITTFactory`, `PNGConverter` and `CustomFactory`.

Two things slice 1 deferred are brought in because this slice needs them:
`COSDictionary.getCOSStream`, for the /Mask and /SMask of an image, and
`PDStream.createInputStream(List<String>)`, which hands a caller the
still-encoded JPEG or fax data.

### Deviations from Java

- **`PDDeviceCMYK` converts naively.** Java converts through an ICC profile it
  ships as a resource — CGATS001Compat-v2-micro, an open stand-in for the
  "U.S. Web Coated (SWOP) v2" profile Acrobat uses — handed to
  `java.awt.color.ICC_ColorSpace` and from there to LittleCMS. Go has no ICC
  engine and PDFBox has no ICC code of its own to port: the whole conversion is
  three lines that call out to the platform. So the port uses
  R = (1-C)(1-K). **This is the largest gap in the slice**: it changes the
  colours every CMYK image and every CMYK fill comes out as, not by rounding.
- **`PDICCBased` always takes the /Alternate colour space.** That is the path
  Java takes when the profile will not load, and the path its own
  `org.apache.pdfbox.rendering.UseAlternateInsteadOfICCColorSpace` property
  forces; the port takes it deliberately rather than on an error.
- **DCT cannot be byte-identical to Java's.** `image/jpeg` has already applied
  the Adobe inversion a CMYK JPEG stores its samples with, where Java writes
  the samples as stored and lets the image's /Decode array invert them; the
  port takes that inversion back out, one subtraction per sample, for both the
  plain CMYK and the YCCK arms, read out of `applyBlack`. Beyond that, two JPEG
  decoders do not agree to the last bit: the inverse DCT and the YCbCr
  conversion are approximations, and `image/jpeg`'s differ from the JRE's in
  the last place on some samples.
- **Scaling.** Java scales a mask with an `AffineTransformOp`, bicubic for a
  small image and bilinear for a large one. Go has no resampler in its standard
  library at all, so the port writes bilinear where Java interpolates and
  nearest neighbour where it does not. A scaled mask differs softly, in the
  gradient of its edge.
- **`getStencilImage` takes a colour, not a `Paint`.** A `Paint` may be a
  gradient or a pattern, which are rendering objects, and slice 9 owns those.
- **A truncated image keeps the rows that read.** Java reads bits through
  `MemoryCacheImageInputStream`, whose `readBits` throws at the end of the
  stream, and nothing catches it, so a truncated image throws out of
  `getRGBImage`. The port returns zeroes and keeps what it had, which is the
  tolerance the filters already have for a damaged stream.
- **A CMYK JPEG is written as RGB.** Go's `image/jpeg` has no four component
  encoder, so the port converts a CMYK image to RGB before encoding and
  declares what it wrote; that loses the DeviceCMYK colour space Java writes.

### The DCT bound

The nearest available check on the encoder is the one Java's own test makes —
the mean difference between an image encoded by the port and its original — and
the port measured 5.03 against Java's bound of 5. Encoding `jpeg.jpg` at
quality 60 through 90 gives 7.05, 5.47, 5.03, 4.72, 2.78 and 0.41, a smooth
rate-distortion curve, so Go's encoder is simply lossier at the same nominal
quality. The bound in the port is 6, with that curve written beside it.

### Gaps Go's standard library leaves

- **No TIFF decoder.** `PDImageXObject.createFromByteArray` reads a TIFF the
  CCITT reader refuses by falling through to `ImageIO`, which decodes an LZW
  TIFF. `lzw.tif` loads in Java and does not here;
  `TestCreateFromByteArrayLZWTiff` pins that so it stays visible.
- **No BMP decoder**, for the same reason, on the same path.
- **A four component JPEG with no Adobe APP14 marker** is refused by
  `image/jpeg`, where Java's `getAdobeTransformByBruteForce` falls back to
  reading it as CMYK.
- **`JPEGFactory.createFromImage` ignores the DPI.** Java writes it by editing
  the JFIF APP0 marker through the writer's metadata tree; Go's `image/jpeg`
  gives no way to. Nothing in a PDF reads it — the image is scaled by the
  content stream — and PDFBOX-6235 notes that a CMYK JPEG has no JFIF marker to
  carry it either.

### Defects found and fixed

- **`SampledImageReader.from8bit` writes a region to the wrong rows**, which
  the port had quietly fixed: three bounds guards written while porting made
  the Go silently write nothing where Java fails, and they are gone, so it now
  panics where Java throws. JAVA-BUGS.md 31. `sampledimagereader.go`.
- **Two more bounds guards removed**, for the same reason:
  `PDIndexed.readColorTable` divides by the base colour space's component count
  without checking it, and `from1Bit` indexes its output without checking it.
  Both are unreachable in practice, and both now index the way the Java does.
  `indexed.go`, `sampledimagereader.go`.
- **A truncated ASCII85 stream repeats its last complete group**, which the
  port already did; the damage tolerance test was wrong and not the port, and
  now asserts the repeat. JAVA-BUGS.md 32. `ascii85.go`;
  `TestASCII85DamageTolerance`.
- **`ASCIIHexFilter` adds -1 for a digit that is not hexadecimal**, so `4Z`
  decodes to 63 and `Z4` to 0xF4. JAVA-BUGS.md 30. `asciihex.go`.
- **`COSDictionary.toString` ran the stack out on a dictionary holding
  itself.** Java delegates to a `getDictionaryString` that carries the objects
  it has already been through, and the Go had no such guard; a slice 1 defect
  this slice found, because the colour space `create` path reports exactly that
  case, PDFBOX-5315. `cos.Dictionary.String`; `TestCreateRecursiveDictionary`.
- **No sampled or calculator function could be built through the factory.**
  Java's `base instanceof COSDictionary` is satisfied by a `COSStream`, but a Go
  `*cos.Stream` embeds `cos.Dictionary` without being one, so the type
  assertion rejected every type 0 and type 4 function — and both are always
  streams — and with them every `/Separation` and `/DeviceN` whose tint
  transform is one. `TestPDFunctionType4` missed it by calling the type 4
  constructor directly. `pdfunction.go`; `TestCreateType4FromStream`,
  `TestCreateType0FromStream`, `TestCreateFromIndirectStream`.
- **A losslessly imported CMYK image came back blank.** The branch meant to
  read the four channels tested for a `CMYK()` method, and `image/color.CMYK`
  carries its channels as fields; every sample stayed zero, which in CMYK is
  white. It reads through `color.CMYKModel.Convert` now. `losslessfactory.go`;
  `TestLosslessCMYKRoundTrip`.
- **A CMYK JPEG was written as three components and declared as four.**
  `colorSpaceOfImage` read the Go image type and said DeviceCMYK, with an
  eight entry inverted decode array beside it, so a reader would take three
  samples per pixel as four. `jpegfactory.go`;
  `TestJPEGFromCMYKImageDeclaresWhatItEncoded`.
- **A repeated filter was decoded twice.** `COSInputStream.create`, `createView`
  and the static `Filter.decode` all reduce a repeated filter to one before
  applying any — a `/Filter` array naming the same filter twice is a malformed
  stream PDFBox repairs — and read `/DecodeParms` at each filter's place in the
  reduced list. The reduction is in `Stream.decode`, which all three reach; the
  writer still encodes through every entry. `cos/stream.go`;
  `TestRepeatedFilterIsDecodedOnceOnEveryPath` and
  `TestRepeatedFilterIsWrittenEveryTime`, which replace `TestStreamTwoFilterChain`.
- **`Raster.SetPixel` half wrote a pixel.** Handed fewer values than the raster
  has bands it stopped at the shorter of the two, leaving the remaining bands
  holding whatever the pixel had before; Java reads `numBands` values and
  throws, and the port panics now. A *longer* array is still fine in both,
  which is what lets the CIE colour spaces pass a three element one to a single
  band raster. `awt/image/raster.go`; `TestSetPixelRefusesAShortSlice`,
  `TestSetPixelAcceptsALongerSlice`.

### Which Java tests are ported, and which are not

| Java test | Ported |
| --- | --- |
| `TestFilters` | whole, except `testPDFBOX4517` |
| `PredictorTest` | whole, by slice 1 |
| `TestOperators`, `TestParser`, `TestPDFunctionType4` | whole |
| `PDLabTest` | whole |
| `PDICCBasedTest` | whole |
| `PDIndexedTest` | the parameter checks, and the first half of the factory test |
| `PDInlineImageTest` | the half that builds and checks the images |
| `JPEGFactoryTest` | the `validate` half of five of the ten |
| `PDDeviceCMYKTest` | no — both its tests load the ICC profile |
| `CCITTFactoryTest`, `LosslessFactoryTest`, `PNGConverterTest`, `PDImageXObjectTest` | no |

`testPDFBOX4517` reads `target/pdfs/PDFBOX-4517-cryptfilter.pdf`, which the Java
build downloads and this repository did not carry **[input present since
2026-09-11; see "What the fetch unblocked"]** — the same reason two of slice 5's
tests are absent.

The four image tests that are not ported all **save the document** and several
render it back, which is slice 7 and slice 9. In their place the port tests the
property each factory rests on, which needs neither: that what goes in comes
back out. `LosslessFactory` round trips a synthetic gradient, a grey ramp and
three checked-in PNGs pixel for pixel; `CCITTFactory` round trips a bitmap
through the fax encoder and reads the checked-in Group 3 and Group 4 TIFFs;
`PNGConverter` converts the checked-in truecolor and indexed PNGs and compares
every pixel against the same file decoded by `image/png`, and declines exactly
what Java declines.

The corpus score is unchanged: it measures text extraction, which this slice
does not touch.

### Still open

- **A3** — four of the seven image tests are not ported, because every one of
  their tests saves the document and several render it back. Closing A3 needs
  slice 7.

---

## Slice 7 — write and manipulate

Branch `slice/7-write-merge`. The first slice that produces a PDF rather than
consuming one: the writer and its compression pool, `PDFXRefStream`, the COS
update state slice 1 deferred, the save path, and `multipdf`. `track/multipdf`
finished the three `multipdf` classes and the `Splitter` half that slice 7 left
on `pdmodel/interactive` and `documentinterchange/logicalstructure`.

### `pdfbox/pdfwriter` — all 3 files

| Java file | Go file | Notes |
| --- | --- | --- |
| `COSWriter.java` | `coswriter.go` | done, minus `getDataToSign` — see below |
| `COSStandardOutputStream.java` | `cosstandardoutputstream.go` | done; unexported, because nothing outside the package uses it |
| `ContentStreamWriter.java` | `contentstreamwriter.go` | done |

`getDataToSign` was **not ported here**: it builds the byte range to be signed
out of `COSFilterInputStream`, which lives in
`pdmodel/interactive/digitalsignature`. Everything around it was ported —
`doWriteSignature` computes and writes the `/ByteRange`, and
`WriteExternalSignature` writes a signature made elsewhere into the reserved
space — and signing through a `SignatureInterface` returned an error naming the
gap. **Slice 8 closed it**: `COSWriter.DataToSign` is in `pdfwriter/coswriter.go`.

### `pdfbox/pdfwriter/compress` — all 4 files

| Java file | Go file |
| --- | --- |
| `CompressParameters.java` | `compressparameters.go` |
| `COSObjectPool.java` | `cosobjectpool.go` |
| `COSWriterCompressionPool.java` | `coswritercompressionpool.go` |
| `COSWriterObjectStream.java` | `coswriterobjectstream.go` |

### `pdfparser/PDFXRefStream` — done

`pdfparser/pdfxrefstream.go`. Java's `Collection<COSObjectKey>` and `Set<Long>`
become maps keyed on the key's internal hash and on the number, sorted where
Java's `TreeSet` iteration order matters.

### `pdfbox/cos` — the update state slice 1 deferred

`COSUpdateInfo`, `COSUpdateState`, `COSDocumentState` and `COSIncrement` are
ported, and wired into `Dictionary`, `Array`, `Object`, `Stream` and `Document`
at every site Java calls `getUpdateState().update(...)`.

Three slice 1 gaps next to that machinery are closed with it, because the writer
depends on them:

- **`COSArray.maybeWrap` and the same wrapping in `COSDictionary.setItem`.** A
  dictionary or array that is not direct and already has a key is stored as a
  `COSObject` referring to it. Without this the writer emits such an object
  inline at every use instead of once.
- **`COSDictionary.removeItem` updates unconditionally.** Java calls `update()`
  whether or not the key was there; the port did not fire at all when the key
  was absent.
- **`COSDictionary.resetObjectKeys` and `COSArray.resetObjectKeys`**, which
  `PDDocument.importPage` and `Splitter.createNewDocument` need to avoid
  overlapping object numbers.

`COSObject.setToNull` also stopped setting `isDereferenced`, which Java does not
do; `isDereferenced()` is read by `COSIncrement.collect(COSObject)`.

### `pdfbox/pdmodel` — the save path

`pddocument_save.go` holds `save`, `saveIncremental`, `setVersion`,
`getDocumentId`/`setDocumentId` and `importPage`. `PDPage` gained
`getContentStreams`, `getContents` and the two `setContents`; `PDStream` gained
the constructors that write into a document.

`COSWriter` implements `ICOSVisitor` only — no `Closeable`, no `close()` — and
`PDDocument.save` does not close the stream it is given. PDFBOX-4321.

`subsetDesignatedFonts` was a no-op here: Java walks `fontsToSubset` and calls
`font.subset()`, font subsetting is font embedding, and the set is always empty,
so the call site was ported and the body was not. **`track/font-embedding` wrote
the body**, in `pdmodel/pddocument_save.go`.

### `pdfbox/multipdf` — **6 of 6 files**, finished by `track/multipdf`

| Java file | Go file | Notes |
| --- | --- | --- |
| `PDFCloneUtility.java` | `pdfcloneutility.go` | done |
| `PageExtractor.java` | `pageextractor.go` | done |
| `Splitter.java` | `splitter.go`, `splitter_structure.go`, `splitter_kcloner.go` | done — the last two are `track/multipdf`'s; see below |
| `PDFMergerUtility.java` | `pdfmergerutility.go`, `pdfmergerutility_structure.go` | done — `track/multipdf` |
| `LayerUtility.java` | `layerutility.go` | done — `track/multipdf` |
| `Overlay.java` | `overlay.go` | done — `track/multipdf` |

Slice 7 left out `PDFMergerUtility`, `LayerUtility` and `Overlay`, and left out
of `Splitter` the seven private methods `fixDestinations`, `cloneStructureTree`,
`cloneIDTree`, `cloneRoleMap`, `cloneTreeElement`, `processResources` and
`processAnnotations`, the `KCloner` inner class and the four catalogue copies of
`createNewDocument`. Until `track/multipdf` ported them, a split left the
structure tree, the outline destinations and the annotations behind, and a split
document lost its viewer preferences, its language, its mark info and its
metadata.

### Which Java tests are ported, and which are not

| Java test | Go test | Notes |
| --- | --- | --- |
| `OperatorNameTest` | `contentstream/operator/names_test.go` | all 8, complete. Moved to the package the names live in, which is where a Go reader looks |
| `COSWriterTest` | `pdfwriter/coswriter_test.go` | 2 of 4 — see below |
| `PageExtractorTest` | `multipdf/pageextractor_test.go` | complete |
| `TestToUnicodeWriter` | `pdmodel/font/tounicodewriter_test.go` | all 8, complete — the A3 deferral from slice 3 |
| `COSWriterCompressionPoolTest` | `pdfwriter/compression_test.go` | needed `PDDocumentOutline` and `PDOutlineItem` -- slice 8; ported by `track/stale-deferrals` |
| `COSDocumentCompressionTest` | — | all 5 need `PDAcroForm`, `PDComplexFileSpecification`, `PDPageContentStream`, `PDCheckBox` or `protect` |
| `ContentStreamWriterTest` | `pdfwriter/contentstreamwriter_render_test.go` | **`track/raster`** — the round trip renders identically. Java reads a downloaded `PDFBOX-4750.pdf`; the port uses the page it writes for its own raster comparisons |
| `PDFCloneUtilityTest` | `multipdf/pdfcloneutility_test.go` | all 3 — `track/multipdf` |
| `OverlayTest` | `multipdf/overlay_test.go` | all 3 — `track/multipdf`, comparing content streams where the Java compares pixels |
| `TestLayerUtility` | `multipdf/layerutility_test.go` | complete — `track/multipdf` |
| `PDFMergerUtilityTest` | `multipdf/pdfmergerutility_test.go`, `multipdf/splitwithstructure_test.go` | 22 of 30 — `track/multipdf`; the other 8 read `target/pdfs` |
| `MergeAcroFormsTest` | `multipdf/mergeacroforms_test.go` | 1 of 3 — the other 2 read `target/pdfs` |
| `MergeAnnotationsTest` | `multipdf/mergedownloaded_test.go` | its one case reads `target/pdfs`, present since the fetch of 2026-09-11 |
| `TestFontEmbedding` | `pdmodel/font/fontembedding_test.go` | needed `PDPageContentStream` and `TestPDFToImage`; the other half of slice 3's A3 deferral, ported by `track/font-embedding` and finished after the fetch |

The inputs the rows above put in `target/pdfs` have been present since
2026-09-11; see "What the fetch unblocked".

`COSWriterTest`'s two that are not ported: `testPDFBox5945` builds an AcroForm
out of `PDAcroForm`, `PDTextField` and `PDAnnotationWidget`, and `testPDFBox6036`
downloads two PDFs from `issues.apache.org` — the port's tests do not reach the
network.

### Tests that are not ported from Java

`pdfwriter/writeverify_test.go` holds the two checks phase D asks for, and says
in its header that it is not a port:

- `TestEmittedBytesMatchPDFBox` compares the byte sequences this writer emits
  against `PDFBoxLegacyMerge-SameMerged.pdf` in the Java test resources, which
  PDFBox wrote. Header, object header, `endobj`, `endstream`, the two xref entry
  forms, the xref header, the trailer and `%%EOF` all match byte for byte. Each
  assertion fails the test if the reference stops containing the shape, so it
  cannot rot into a tautology.
- `TestIncrementalSaveAppends` checks that an incremental save leaves every byte
  of the original where it was, appends the update, and that the update is
  visible after reloading.
- `TestSaveRoundTrip` is the cheap first signal only; on its own it proves
  nothing, because writing and reading with the same broken port passes.

There is no Maven in this environment, so PDFBox itself could not be built to
diff a whole file against a Java run.

### What a plain save does to object numbers

`COSWriter.write` sets `number = getHighestXRefObjectNumber()` before it starts,
so a full, uncompressed save of a **loaded** document renumbers every object
from there — a file with 227 objects comes back with objects 228 to 454 and
`/Size 455`, and the xref table carries 227 free entries for the gap. That is
Java, not a port defect: `fillGapsWithFreeEntries` exists for exactly this. The
compressed path does not renumber, because the compression pool offers each
object's existing key back to `COSObjectPool.put`.

### Deviations from Java

- **`COSWriter`'s public byte constants and its static `writeString`** are
  declared in `pdfwriter/compress` and re-exported here under the Java names.
  `COSWriterObjectStream` needs them and `pdfwriter` imports `compress`, so Go
  forbids the direction Java uses; putting the definitions at the bottom of the
  dependency keeps one implementation rather than two. Commented in
  `compress/tokens.go` and at the top of `coswriter.go`.
- **`SignatureInterface` is declared in `pdfwriter`** rather than in the package
  Java has it in: it is one method, and the writer is the only thing in this
  slice that names it.
- **`COSWriterCompressionPool` takes a `PDDocument` in Java.** The port declares
  `compress.DocumentLike` — `Document()` and `Encryption()` — so that the
  dependency runs one way, the same device slice 5 used for the security
  handlers. `pdfwriter.PDDocumentLike` embeds it and `encryption.PDDocumentLike`.
- **Java's three default methods of `COSUpdateInfo` cannot be embedded**: they
  need the owner, and an embedded struct in Go has no way back to the value
  embedding it. Each implementor writes them out, one line each. `Stream`
  overrides the four it would inherit from the `Dictionary` it embeds, so that
  the state's owner is the stream — otherwise an increment would write the
  dictionary inside a stream and drop the stream data.
- **`new PDDocument()` sets `/Version` `1.4` on the catalogue, which Java's
  constructor does, and takes a `filter.Provider`.** Java resolves a filter
  through a static registry; the port passes the provider in, which is what
  keeps `cos` from importing `filter`, and without it a document built in memory
  could not write a Flate stream.
- **`PDFCloneUtility.cloneForNewDocument` is generic in Java and casts its
  result.** The port returns `cos.Base` and adds `CloneDictionaryForNewDocument`
  for the one caller shape that needs the concrete type; both are the same
  unchecked cast, in a different place. Commented in
  `multipdf/pdfcloneutility.go`.
- **`protected` and package-private members are exported**: `PDFCloneUtility`'s
  constructor and `cloneMerge`, and `Splitter.splitAtPage`, `createNewDocument`
  and the two document accessors. Go has no such level and the types are public.
  Commented in `multipdf/splitter.go`.
- **`PDStream(PDDocument, InputStream, ...)` closes the `InputStream`.** A Go
  `io.Reader` has nothing to close, so `NewPDStreamOfInput` leaves that to the
  caller.
- **`ToUnicodeWriter` is package-private and final in Java**, so the port keeps
  it unexported.
- **The xref entry number formats.** `DecimalFormat("0000000000")` and
  `("00000")` against `%010d` and `%05d` differ for a negative value, where Java
  keeps ten digits after the sign and Go counts the sign into the width; neither
  column is ever negative — one is a byte offset, the other a free object
  number.

### Defects found and fixed

The adversarial review read `coswriter.go`, `contentstreamwriter.go`, the four
`compress` files, `pdfxrefstream.go`, the four `cos` update-state files and the
three `multipdf` files against their Java. What it and the branch feedback found:

- **`visitFromDictionary` left `byteRangeArray` nil** where the entry is not an
  array, turning Java's `ClassCastException` into a nil dereference several
  steps later in `doWriteSignature`; it now asserts the way Java casts.
  `pdfwriter/coswriter.go`.
- **`COSDictionary.addAll` was routed through `setItem`**, so this slice's two
  new `SetItem` jobs — wrapping a keyed, non-direct value into a `COSObject`,
  and `getUpdateState().update(value)` — leaked into every `addAll` and into
  `importPage`, whose `new COSDictionary(page.getCOSObject())` is a copy
  constructor calling `addAll`. `AddAll` now uses the raw insertion helper.
  `cos`; `TestDictionaryAddAllIsARawPut`.
- **The trailer `/ID` digest hashed UTF-8** where Java hashes ISO-8859-1, so a
  document with a non-ASCII `/Title` came out with an `/ID` the reference would
  not produce. `encodeISO88591` in `pdfwriter/coswriter.go` now does what the
  charset does, including the `?` an unmappable character becomes; `encryption`
  keeps its own copy with the same body, as Java has no shared helper either.
  `TestDocumentIDDigestUsesISO88591` computes the expected digest from the Java
  rule, not from the port.
- **`Document.SetTrailer`'s doc comment** had Java's editorial note transcribed
  with its leading `//` intact, producing a doubled comment marker; it now says
  that the method links the trailer to the document state, which is what makes a
  later change count as an update, and attributes the note. `cos`.

### Java bugs recorded

- `JAVA-BUGS.md` 33 — `ToUnicodeWriter.allowDestinationRange` checks only
  `prev`; found reading this slice.
- `JAVA-BUGS.md` 34 — `PageExtractor.extract` throws for a start page beyond the
  document where its javadoc promises a blank one; pinned by
  `TestExtractBeyondTheDocumentPanics`.

`PageExtractor` indexing `splitted[0]` was reported with 34 and is not a defect:
reaching the index means both setters accepted their arguments, so
`1 <= startPage <= endPage <= numberOfPages`, and `processPages` therefore makes
at least one destination document.

### Still open

- `ObjectStreamXReference`'s `object` field is still not carried by
  `xref.ObjectStreamReference`. Its only accessor, `getObject()`, has no caller
  in the Java main tree.

## Slice 8 — forms, annotations, interactive features

Branch `slice/8-forms-annotations`. The largest slice by file count: every
subtree under `pdmodel/interactive`, the whole of `pdmodel/documentinterchange`
and `pdmodel/fdf`, the document fixups, the optional content, and the half of
`pdmodel/common` slice 2 left. Every Java class in those packages has a Go
counterpart; what is missing is named method by method below, and each gap is a
type a later slice brings.

### `pdmodel/interactive` — the four classes of the package itself

| Java file | Go file |
| --- | --- |
| `PlainText.java` | `interactive/plaintext.go` |
| `PlainTextFormatter.java` | `interactive/plaintextformatter.go` |
| `AppearanceStyle.java` | `interactive/appearancestyle.go` |
| `TextAlign.java` | `interactive/textalign.go` |

### `interactive/form` — all 21 files

`pdacroform.go`, `pdfield.go`, `pdterminalfield.go`, `pdnonterminalfield.go`,
`pdvariabletext.go`, `pdtextfield.go`, `pdbutton.go`, `buttons.go` (check box,
radio button, push button), `pdchoice.go`, `choices.go` (list box, combo box),
`pdsignaturefield.go`, `pdfieldfactory.go`, `pdfieldtree.go`, `fieldutils.go`,
`pddefaultappearancestring.go`, `pdxfaresource.go` and
`appearancegeneratorhelper.go`.

Five files carry what Go could not put where Java has it:

- **`catalogacroform.go`** — `getAcroForm`, `getAcroForm(PDDocumentFixup)` and
  `setAcroForm` are functions over a `*pdmodel.PDDocumentCatalog`, because they
  name `PDAcroForm` and `pdmodel` cannot import this package. The catalogue
  keeps the two private fields they use, typed `any`, and this package narrows
  them.
- **`fdf.go`** — `PDField.importFDF`, `PDTerminalField.importFDF`,
  `PDNonTerminalField.importFDF`, `PDField.exportFDF` and the two on
  `PDAcroForm`, as functions, because putting a method naming `FDFField` on the
  `PDField` interface would make every implementation name it too.
  `PDXFAResource.getDocument` is here for the same reason.
- **`widgetparent.go`** — `PDAnnotationWidget.setParent(PDField)`, which the
  annotation package cannot declare.
- **`documentsignatures.go`** — `PDDocument.getSignatureFields` and
  `getSignatureDictionaries`.
- **`handlerhooks.go`** — the two hooks `annotation/handlers` calls back into
  the form through: `AcroFormDefaultAppearance` and
  `AcroFormDefaultResourcesFont`.

`addsignature.go` holds `PDDocument`'s four `addSignature` overloads, their five
private helpers and `saveIncrementalForExternalSigning`, for the same reason:
they name `PDAcroForm`, `PDSignatureField` and `PDAnnotationWidget`.

### `interactive/annotation` — all 52 files

`pdannotation.go` holds the abstract base and its factory; `annotations.go` and
`annotations2.go` the concrete annotations; `dictionaries.go` the border
styles, the appearance dictionary and stream, the action dictionary, the
additional actions, the markup information, the external data and the rest.

`annotation/handlers` holds every appearance handler plus `cloudyborder.go` and
`pathwriter.go`. `register.go` fills the `DefaultAppearanceHandlers` registry
from its `init`, which is how `PDAnnotation.constructAppearances` reaches a
handler without this package naming each one.

`PDSquigglyAppearanceHandler.generateNormalAppearance` was **not ported** in
this slice — it fills the squiggle with a tiling pattern, which needs
`PDTilingPattern`, `PDPatternContentStream` and the `PDPattern` colour space.
**`track/raster` ported it**, and the appearance it generates matches PDFBox's
token for token across all three streams it is made of. The pattern reaches the
handler through `annotation.NewSquigglyPatternColor`, because
`graphics/pattern` imports `pdmodel`, which imports the handlers.

### `interactive/action` — all 25 files

`pdaction.go` (the abstract action and `PDActionFactory`), `actions.go` (the
concrete actions, `PDURIDictionary` and `PDWindowsLaunchParams`) and
`additionalactions.go` (the five additional-action dictionaries).

### `interactive/documentnavigation` — all 12 files

`destination/pddestination.go` and `destination/pdpagefitdestination.go` hold
the abstract destination, the factory, the named destination and the five page
destinations. `outline/` holds `PDOutlineNode`, `PDDocumentOutline`,
`PDOutlineItem` and `PDOutlineItemIterator`.

### `interactive/pagenavigation`, `measurement`, `viewerpreferences` — all 12 files

`pdthread.go` (thread and bead), `pdtransition.go` (transition, style,
dimension, direction, motion), `measurement/measurement.go` (viewport, measure,
rectilinear measure, number format) and
`viewerpreferences/pdviewerpreferences.go` with its five enums.

### `interactive/digitalsignature` — all 18 files

| Java file | Go file |
| --- | --- |
| `PDSignature.java` | `pdsignature.go` |
| `COSFilterInputStream.java` | `cosfilterinputstream.go` |
| `PDPropBuild.java`, `PDPropBuildDataDict.java` | `pdpropbuild.go` |
| `PDSeedValue.java` | `pdseedvalue.go` |
| `PDSeedValueCertificate.java`, `PDSeedValueMDP.java`, `PDSeedValueTimeStamp.java` | `pdseedvaluecertificate.go` |
| `SignatureInterface.java`, `ExternalSigningSupport.java`, `SigningSupport.java`, `SignatureOptions.java` | `signing.go` |
| `visible/*.java` — all 6 | `visible/` — 6 files |

**This closes slice 7's `getDataToSign` gap.** `COSWriter.DataToSign` is ported,
and with it the real signing path: `PDDocument.saveIncremental` with a
`SignatureInterface` now signs rather than returning the error slice 7 left.
`COSFilterInputStream` is what it needed.

### `pdmodel/documentinterchange` — all 24 files

`logicalstructure/` — `PDStructureNode`, `PDStructureElement`,
`PDStructureTreeRoot`, `PDAttributeObject`, `PDUserAttributeObject`,
`PDMarkInfo`, `PDMarkedContentReference`, `PDObjectReference`, `Revisions`,
`PDParentTreeValue`. `markedcontent/` — `PDMarkedContent`, `PDPropertyList`.
`taggedpdf/` — the standard attribute objects and `StandardStructureTypes`.
`prepress/` — `PDBoxStyle`.

`PDStructureElementNameTreeNode` lives in top-level `pdmodel`, where Java has
it, because it is the `/IDTree` of the structure tree root.

### `pdmodel/fdf` — all 31 files, and `pdfparser/FDFParser`

`fdfdocument.go` (document and catalogue), `fdfdictionary.go`, `fdffield.go`,
`fdfannotation.go` (the abstract annotation and its two factories),
`annotations.go` and `annotations2.go` (the concrete FDF annotations),
`small.go` (page, template, named page reference, icon fit and its enums,
`FDFOptionElement`, `FDFJavaScript`).

`FDFParser` answers a `*cos.Document` rather than an `FDFDocument`:
`pdfparser` sits below `pdmodel`, exactly as it does for `PDFParser`. The
wrapping is in `pdfbox/loader_fdf.go` — `LoadFDF`, `LoadFDFReader`,
`LoadFDFFrom`, `LoadXFDF` and `LoadXFDFReader`. `COSWriter.write(FDFDocument)`
inverts the same way: `pdfwriter.WriteFDF` takes the COS document, and
`FDFDocument.Save` calls it.

`pdfbox/util/xmlutil.go` is `XMLUtil`, `pdfbox/util/hex.go` the two `Hex`
methods FDF needs, and `go/awt/color.go` is `java.awt.Color` — a colour built
from a packed integer or three components, which is all PDFBox uses of it, plus
the named constants it references.

### `pdmodel/fixup` and `fixup/processor` — all 8 files

`fixup/fixup.go` holds `PDDocumentFixup`, `AbstractFixup`,
`AcroFormDefaultFixup` and `AcroFormOrphanWidgetsFixup`; `fixup/processor/`
holds the four processors.

`AcroFormOrphanWidgetsProcessor.ensureFontResources` found the replacement font
but did not embed it, because Java calls `PDType1Font.load` and the font
embedders were not ported. **`track/font-embedding` closed it** — the method now
loads the mapped font and puts it in the default resources, as Java does, and
`TestEnsureFontResourcesEmbedsTheReplacement` covers it.

### `pdmodel/graphics/optionalcontent` — all 3 files, and `PDPropertyList` with them

`pdoptionalcontentgroup.go`, `pdoptionalcontentmembershipdictionary.go` and
`pdoptionalcontentproperties.go`.

`PDOptionalContentGroup.getRenderState` takes a `RenderDestination`, which lives
in `org.apache.pdfbox.rendering` — slice 9's package. `go/pdfbox/rendering`
holds that one type and nothing else, and its package comment says why.

**This closes the slice 2 `PDResources.getProperties` gap**, and with it the
`BDC` and `DP` operators: a properties operand that names a property list in the
resources is now resolved, which is what `operator/markedcontent` said was
missing. `ResourceCache` gained its three property list members, and
`DefaultResourceCache` the map and stable-cache bookkeeping behind them.

### `pdmodel/common` — what slice 2 left

| Java source | Go source | Status |
| --- | --- | --- |
| `COSArrayList.java` | `common/cosarraylist.go` | done |
| `COSDictionaryMap.java` | `common/cosdictionarymap.go` | done |
| `PDStream.java` | `common/pdstream.go` | done — the writing constructors, the decode parameters, the file specification and the metadata |
| `PDMetadata.java` | `common/pdmetadata.go` | done |
| `PDNameTreeNode.java` | `common/pdnametreenode.go` | done |
| `PDNumberTreeNode.java` | `common/pdnumbertreenode.go` | done |
| `PDObjectStream.java` | `common/pdobjectstream.go` | done |
| `PDPageLabels.java`, `PDPageLabelRange.java` | `common/pdpagelabels.go`, `common/pdpagelabelrange.go` | done |
| `filespecification/*.java` | `common/filespecification/` | done |

`PDEmbeddedFile`'s four date accessors are here too: `DateConverter` is this
slice's, taken in dependency order, and they read through it.

### Top-level `pdmodel` — the holes the interactive types were blocking

- **`PDPage`.** `getThreadBeads`, `setThreadBeads`, `getMetadata`,
  `setMetadata`, `getActions`, `setActions`, `getTransition`, the two
  `setTransition`, `getAnnotations`, `getAnnotations(AnnotationFilter)`,
  `setAnnotations`, `getViewports`, `setViewports`, `getUserUnit` and
  `setUserUnit` are ported. Only `removePageResourceFromCache` was left, on the
  four halves of the resource cache that still had no type; since ported by
  `track/testdata-sources`, with the call `PDFTextStripper.processPage` makes —
  see "`PDPageTree` and the resource cache".
- **`PDDocumentCatalog`.** Every accessor is ported. `getAcroForm` and
  `setAcroForm` are in `interactive/form` (see above); the other 36 are in
  `pddocumentcatalog.go`, kept out of `pddocument.go` so that the file next to
  `PDDocument` names no interactive type.
- **`PageMode`, `PageLayout`** — `pagemode.go`.
- **`PDDestinationNameTreeNode`, `PDEmbeddedFilesNameTreeNode`,
  `PDJavascriptNameTreeNode`, `PDDocumentNameDictionary`,
  `PDDocumentNameDestinationDictionary`** — `nametreenodes.go`.
- **`PDDocumentInformation`.** Its four date accessors were waiting on
  `DateConverter`, which is this slice's; `getTrapped`, `setTrapped`,
  `getMetadataKeys` and `getPropertyStringValue` came with them, and the class
  is complete.
- **`PDAbstractContentStream`, `PDPageContentStream`,
  `PDAppearanceContentStream`, `PDFormContentStream`.** None of the four is
  named in any slice's scope table; they are here because
  `AppearanceGeneratorHelper` cannot exist without them.
- **`PDOutputIntent`** is in `graphics/color`, where Java has it, because the
  catalogue's three output intent accessors name it.

Not ported, and why:

- `shadingFill` names `PDShading` and `PDPatternContentStream` names
  `PDTilingPattern`; both are slice 9's.
- `PDPageContentStream`'s five deprecated `appendRawCommands` methods, which is
  a choice rather than a gap: they write bytes into the stream unchecked, Java
  marks every one `@Deprecated`, and nothing in the main tree calls them.
- `PDStream.createInputStream(DecodeOptions)`. `cos.Stream` has no reader that
  takes decode options, because the `Codec` interface does not; the options
  exist and `filter.DCT` honours them, so closing this needs a change to that
  interface rather than to `PDStream`.
- `PDOutputIntent(PDDocument, InputStream)`. It reads the ICC profile with
  `java.awt.color.ICC_Profile`, and Go has no ICC engine — the same gap
  `PDICCBased` records. It also takes a `PDDocument`, which `graphics/color`
  cannot name.

### Which Java tests are ported, and which are not

Every Java test in this slice's packages is ported except the eight named below.

| Java test | Go test |
| --- | --- |
| `PDActionURITest` | `interactive/action/pdactionuri_test.go` |
| `PDTransitionTest`, `PDTransitionDirectionTest` | `interactive/pagenavigation/pdtransition_test.go` |
| `PDDocumentOutlineTest`, `PDOutlineItemTest`, `PDOutlineItemIteratorTest` | `interactive/documentnavigation/outline/outline_test.go` |
| `PDOutlineNodeTest` | `interactive/documentnavigation/outline/pdoutlinenode_test.go` |
| `PDAnnotationTest`, `PDSquareAnnotationTest`, `PDCircleAnnotationTest` | `interactive/annotation/pdannotation_external_test.go` |
| `AppearanceGenerationTest` | `interactive/annotation/appearancegeneration_external_test.go` |
| `PDAcroFormTest` | `interactive/form/pdacroform_external_test.go` |
| `PDFieldTest`, `PDTextFieldTest`, `PDSignatureFieldTest`, `PDDefaultAppearanceStringTest`, `PlainTextTest`, `TestUtils` | `interactive/form/pdfield_test.go`, `interactive/form/form_test.go` |
| `TestFields`, `PDChoiceTest`, `TestCheckBox`, `HandleDifferentDALevelsTest` | `interactive/form/fields_external_test.go` |
| `ControlCharacterTest` | `interactive/form/controlcharacter_external_test.go` |
| `AlignmentTest`, `CombAlignmentTest`, `AcroFormsRotationTest`, `MultilineFieldsTest` | `interactive/form/multiline_external_test.go` |
| `TestListBox` | `interactive/form/listbox_test.go` |
| `PDButtonTest` | `interactive/form/button_external_test.go` |
| `TestRadioButtons` | `interactive/form/radiobutton_external_test.go` |
| `PDStructureElementTest` | `documentinterchange/logicalstructure/pdstructureelement_external_test.go` |
| `COSArrayListTest` | `common/cosarraylist_external_test.go` |
| `PDStreamTest`, `TestEmbeddedFiles` | `common/pdstream_external_test.go` |
| `TestOptionalContentGroups` | `graphics/optionalcontent/optionalcontent_external_test.go` |
| `TestPDDocumentCatalog` | `pdmodel/pddocumentcatalog_external_test.go` |
| `TestPDPageAnnotationsFiltering`, `TestPDPageTransitions` | `pdmodel/pdpageannotations_external_test.go` |
| `TestPDPageContentStream` | `pdmodel/pdpagecontentstream_external_test.go` |
| `PageModeTest`, `PageLayoutTest` | `pdmodel/pagemode_test.go` |
| `TestFDF` | `pdmodel/fdf/fdf_external_test.go` |
| `TestPDDocumentInformation` | `pdmodel/pddocumentinformation_external_test.go` |

Not ported, each because the Maven build fetches its input or because it
compares rendered images:

| Java test | Why |
| --- | --- |
| `PDFieldTreeTest` | reads PDFs the build downloads into `target/pdfs` |
| `PDAcroFormGenerateAppearancesTest` | the same |
| `PDAcroFormFromAnnotsTest` | the same |
| `PDAcroFormFlattenTest` | downloads its PDFs and compares `PDFRenderer` output pixel by pixel |
| `AppearanceGenerationTest` — the two rendering cases | they compare rendered images; the rest of the class is ported |
| `TestOptionalContentGroups` — two cases | `testOCGsWithSameNameCanHaveDifferentVisibility` and `testOCGGenerationSameNameCanHaveSameVisibilityOff` read pixels out of a `PDFRenderer` image |
| `TestFDF.testPDFBox5894` | reads `target/pdfs/PDFBOX-5894.fdf` |
| `TestPDDocumentCatalog.handleOutputIntents` | builds a `PDOutputIntent` from `sRGB.icc` through the constructor that needs `ICC_Profile`; the half that does not is ported |

`TestCheckBox.testPDFBox6207` reads `target/pdfs` and is not ported either. The
inputs under `target/pdfs` named in both tables have been present since
2026-09-11; see "What the fetch unblocked".

`interactive/digitalsignature`, `interactive/measurement` and
`interactive/viewerpreferences` have no Java test directory at all.

`signverify_test.go` is not a port — PDFBox has no signing test in the `pdfbox`
module — and checks the two things that can be checked without a CMS
implementation, against the output file rather than against the port's own
arithmetic: that the `/ByteRange` in the written file starts at 0, ends at the
file length and leaves a hole whose first byte is `<` and whose last is `>`, so
the hole is the `/Contents` hex string and nothing else; and that the bytes the
signer was handed are byte for byte the written file with that hole removed,
which is also what `PDSignature.getSignedContent` answers on the output. Both
the `SignatureInterface` path and the external signing path pass. It says
nothing about the CMS blob: the port has no signature algorithm.

`AppearanceGenerationTest` asserts no value of its own. It generates appearances
for the annotations of a fixture and compares the token stream against the
appearance PDFBox itself wrote into the same file; numbers are compared to
within a tolerance the Java sets, and operators must match exactly.

### Deviations from Java

- **`PlainText` line breaking.** Java uses `java.text.BreakIterator`, which
  follows the Unicode line breaking algorithm, and Go has no such iterator.
  `lineBreakSegments` breaks before a run of whitespace and after a hyphen,
  which is what those rules come to for the Latin text a form field holds; text
  in a script that breaks by its own rules — Thai, Khmer, Japanese — is broken
  differently. Commented in `interactive/plaintext.go`.
- **The comb field walks runes where Java walks UTF-16 code units.**
  `insertGeneratedCombAppearance` takes each cell with `value.substring(i, i+1)`
  in Java and one rune in the port, and counts the cells with `value.length()`
  against `len([]rune(value))`. The two agree for every character in the basic
  plane and differ for one outside it, where Java splits the surrogate pair
  across two cells. It is unreachable: both sides measure each cell through
  `PDFont.getStringWidth` first, Java asking for the lone surrogate `U+D83D` and
  the port for the whole `U+1F600`, and no font in this repository carries
  either, so both refuse the value rather than laying it out.
  `TestCombFieldRefusesASupplementaryCharacter` asserts that the port refuses,
  so the claim fails loudly if the encoder ever starts accepting such a
  character.
- **The fixups can only be linked in by the program itself.** `form` cannot
  import `fixup` — `fixup` names `PDAcroForm` — so `fixup`'s `init` sets
  `form.NewAcroFormDefaultFixup`, and a program that wants the default fixup
  applied must blank-import the package:

      import _ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fixup"

  Without it, `form.AcroFormOfCatalog` reads the form with no fixup applied,
  which is what `getAcroForm(null)` of Java does. The same call therefore has
  two behaviours depending on the import graph, where Java has one. The package
  comment says so, and so does every test that needs the fixup; `JAVA-BUGS.md`
  48 records it.
- **`SignatureOptions.close` reports the first of two close failures**, where
  Java's `finally` replaces the exception in flight and surfaces the later one.
  Both are closed either way. `JAVA-BUGS.md` 47.
- **`FDFDocument.saveXFDF` closes the writer it is given and the port does
  not**, because a Go `io.Writer` has nothing to close. The doc comment says so.
- **Reading XFDF needs a DOM, and Go has none.** `go/w3c/dom` is a small reading
  DOM built for this — `Parse(reader, namespaceAware)`, `TextContent`,
  `FirstElementByTagName`, `ElementsByPath`. `encoding/xml` erases the
  difference between a CDATA section and ordinary text, which
  `FDFAnnotationFreeText` and `FDFAnnotationText` depend on, so the parser
  records byte offsets and looks back at the source to tell them apart.
  PDFBox's four XPath expressions are replaced by the two child-element helpers,
  because every one of them is a direct child lookup.
- **`PDPageDestination` is an abstract class in Java.** The port keeps its state
  in a struct the five concrete destinations embed, and declares the
  `PageDestination` interface for what `instanceof PDPageDestination` asks.
  `destination` cannot name `PDPage`, which reaches the package through the
  annotations, so `PageLike` names what is used and `pdmodel` sets
  `NewPageFromDictionary` and `IndexOfPageInTree` from its `init`.
- **`PDPropertyList.create` names the two `optionalcontent` subclasses.** Go
  forbids that direction, so `markedcontent` keeps a registry and
  `optionalcontent` fills it from its `init`; `CreatePropertyList` then
  dispatches exactly as Java's chain of `if`s does.
- **`PDArtifactMarkedContent` is folded into `markedcontent/pdmarkedcontent.go`**
  as the constructor `NewPDArtifactMarkedContent`: Java's subclass adds
  accessors for the artifact's own properties, and only its tag reaches the text
  extractor.
- **`PageMode.fromString` and `PageLayout.fromString`** throw
  `IllegalArgumentException` in Java for a value that is not one of the
  constants; the value comes out of a PDF rather than from the library, so the
  port answers an error. `getPageMode` and `getPageLayout` check it exactly
  where Java catches.
- **`SigningSupport` needs the writer, and `pdfwriter` cannot import
  `pdmodel`.** `COSWriterLike` names the two methods it uses and the writer
  satisfies it. `SignatureInterface` itself stays in `pdfwriter`, where slice 7
  declared it.
- **`PDStream.getFile` and `setFile`** are `filespecification.FileOfStream` and
  `SetFileOfStream`: `PDFileSpecification` is in `common/filespecification`,
  which imports `common` for `PDEmbeddedFile`, so `common` cannot name it.
- **Two `panic`s in `fixup` have no Java throw behind them** and say so:
  `PDType1Font(FontName)` declares no exception, so a missing standard 14 metric
  is a library bug there too.
- **Java's `(int)` of an out-of-range double saturates and Go's conversion is
  undefined.** Every narrowing site in this slice is a count derived from a page
  geometry and guarded by the caller; `CloudyBorder`'s two `(int) Math.ceil(...)`
  and every `(float)(double expression)` in the line, free text, polyline and
  strikeout handlers narrow at exactly the same point in the Go.

### Defects found and fixed

The adversarial review swept the slice's roughly 200 Java files for key sets,
`COSArrayList`'s filtered-list guards, missing public and protected methods,
dropped `catch` and `finally` behaviour, unchecked throws against panics,
narrowing casts, test provenance and asserted literals, recorded deferrals and
this slice's Java bugs; nothing came of the sweeps beyond what is listed here.

- **`PDStream.createOutputStream(COSName)` crashed on a null filter**, which
  Java accepts and reads as "no filter": a nil `*cos.Name` widened to a
  `cos.Base` is not `nil`, and `filter.ByName` dereferenced it.
  `common/pdstream.go`; `PDSquareAnnotationTest`.
- **`COSArrayList` compared by Go pointer identity** where Java's `contains`,
  `indexOf`, `lastIndexOf` and `remove` use `equals`, so two wrappers over one
  dictionary were unequal and `retainAll` kept the wrong entries. `equalsAny`
  now asks the element for an `Equals` method first.
  `common/cosarraylist.go`; `COSArrayListTest.testRetainIndirectObject`.
- **`PDFont` had no `Equals`**, which Java compares the font dictionary with;
  added, along with `PDType1Font.getType1Font`. `PDDefaultAppearanceStringTest`.
- **`COSArray.toCOSNameStringList` answered the wrong thing.** It now returns
  `[]string` and panics on an entry that is not a name, which is the
  `ClassCastException` Java's cast throws.
- **Three units of this slice's scope were not ported at all**, found by phase C:
  `PDOptionalContentProperties`, `PDResources.getProperties`, and 36 of
  `PDDocumentCatalog`'s 40 methods. All are ported now, with their Java tests.
- **`PDStream` was missing ten public methods** although this file recorded the
  class as whole: `getDecodeParms`, `getFileDecodeParams`, `setDecodeParms`,
  `setFileDecodeParams`, `getFileFilters`, `setFileFilters`, `getFile`,
  `setFile`, `getMetadata` and `setMetadata`. All ten are ported.
- **`PDAcroForm.getFieldIterator` was missing**, now `FieldIterator`.
- **`PDDocument.addSignature` was not ported at all**, so nothing could reach
  the signature model, `COSWriter.getDataToSign`, `SigningSupport` or
  `SignatureOptions`. The four overloads, their five private helpers and
  `saveIncrementalForExternalSigning` are ported, in
  `interactive/form/addsignature.go`.
- **`PDChoiceTest.getOptionsFromMixed` was missing**, and is ported.
- **`cos/names.go` carried a duplicate `DCTDecodeAbbreviation`** beside the
  `DCT` the port already had; removed, since `DCT` is the convention its six
  sibling filter abbreviations follow.
- **`PDNameTreeNode.getValue` could not tell an absent limit from an empty
  one**, because the accessors answered `""` for both, so a tree whose first
  child covers `["" "a"]` swallowed every lookup. `Value` now reads the
  `/Limits` array through `limitOf`, which reports presence separately.
  `common/pdnametreenode.go`;
  `TestValueSkipsAChildWhoseLowerLimitIsTheEmptyName`.
- **`DateConverter.parseBigEndianDate` accepted a second of 60 or 61**, which
  `time.Date` then normalised — `D:20200101120060Z` was read as 12:01:00. Java
  builds on a `GregorianCalendar` with leniency off, whose maximum for `SECOND`
  is 59; the leap-second range belongs to `java.util.Date`. The bound is 59.
  `TestSecondsBeyond59AreRefused`.
- **`PlainText` kept one paragraph too many**, because `String.split` with a
  limit of zero drops every trailing empty result and the port stopped stripping
  at one, drawing a space where PDFBox draws nothing. The strip now runs to
  zero, and the branch above it answers the value whole where the regex never
  matched. `interactive/plaintext.go`;
  `TestValueOfOnlyLineBreaksHasNoParagraphs` and
  `TestTrailingLineBreaksAreDropped`.
- **Two doc and message typos.** `PDListAttributeObject`'s doc comment named a
  type that does not exist, and `PDSignatureField.SetValue`'s panic message had
  lost its apostrophe; the message otherwise stays as Java writes it, naming
  `setValue(PDSignature value)`.

### Java bugs found, carried, and recorded

Fourteen: `JAVA-BUGS.md` 35 to 48, which records for each what the Java does,
where the Go carries it and whether it was kept.

### Still open

- The deferrals listed above: `shadingFill` and `PDPatternContentStream` wait on
  slice 9's `PDShading` and `PDTilingPattern`;
  `PDStream.createInputStream(DecodeOptions)` waits on an options-taking reader
  on the `Codec` interface; `PDOutputIntent(PDDocument, InputStream)` waits on
  an ICC engine. None is blocked on difficulty.
- This slice's tests do not compare the appearance handlers' output against a
  renderer. That comparison is slice 9's, and `AppearanceGenerationTest`'s two
  rendering cases wait for it.

## Slice 9 — rendering

Branch `slice/9-rendering`. The last slice of the plan: `java.awt.geom.Area`,
`pdmodel/graphics/shading`, `pdmodel/graphics/pattern`, the `contentstream`
graphics engine with its graphics and colour operators, the rest of
`PDResources` and `DefaultResourceCache`, `pdfbox/rendering` and
`pdfbox/printing`. `pdmodel/common/function` and `graphics/color`, which
`PLAN.md` counts here, were already ported by slices 2, 3 and 6.
`slice/8-forms-annotations` is merged into this branch, because eleven types it
ports are named directly by `pdfbox/rendering` and slice 8 had not landed on
`migration-base`.

### The decision

PDFBox draws through `java.awt.Graphics2D`, and Go has nothing equivalent.
Taken as B0: `PLAN.md`'s third option, port the geometry and defer the raster
behind `rendering.Backend`. Everything that computes is ported and runs; only
the last drawing step is behind the interface, and no implementation of that
interface shipped with this slice. `RASTER-PRECEDENT.md` carries the library
comparison behind the choice. `track/raster` has since written the backend, in
`rendering/raster`; see that branch's sections for what it matches.

### What the raster decision cost

Slice 9's record: what the interface was designed against, and what a caller
who brings a backend of their own still gets.

- **Without a backend the port cannot produce a rendered page.**
  `PDFRenderer.RenderImage` and its four siblings answer `ErrNoBackend` — with
  the size, the type and the page they worked out, so the error says what would
  have been made. It is deliberately not a blank image, which would look like a
  rendered page.
- **It therefore could not rasterise, print, or run PDFBox's own image
  comparisons.** `TestPDFToImage`, `TestRendering` and `TestQuality` all compare
  against reference PNGs, and `PDFPrintable` printed as vectors and refused to
  rasterise.
- **Everything above the interface runs.** The whole content stream is walked,
  every operator is processed, the colours are converted, the shadings evaluate,
  the clip is computed, the optional content is resolved, the annotations are
  placed. `Backend` is seventeen methods.
- **What a backend has to do that the port does not describe for it:**
  anti-aliased scan conversion of a path under a winding rule; stroking a path
  into an outline with caps, joins, a miter limit and a dash pattern; sampling an
  image through an arbitrary transform with the two interpolations; compositing
  a layer under a blend mode, an alpha constant and a soft mask; and turning each
  of the four `Paint` descriptions into pixels — which for a tiling pattern
  means calling back into `PageDrawer.DrawTilingPattern` for one tile, and for a
  shading means asking the shading for the colour at a point.
- **What the tests compare instead of pixels** is the A5 decision: `Area`
  against the JDK's documented contract, functions and colour conversions
  against values taken from the Java, and `PageDrawer` against a backend that
  records every call. An image comparison is what this leaves uncovered;
  `track/raster` covers it.

### `java.awt.geom.Area` — the JDK, not PDFBox

| Java source | Go source | Status |
| --- | --- | --- |
| `java.awt.geom.Area` | `awt/geom/area.go` | done, minus curves |

Constructive area geometry — `add`, `subtract`, `intersect`, `exclusiveOr`,
`contains`, `getBounds2D`, `transform` and a `PathIterator` over the result —
which slice 2 recorded as the one thing blocking
`PDGraphicsState.getCurrentClippingPath`. The port splits every boundary edge at
its crossings **and at its T-junctions**, classifies each piece by offsetting
perpendicular from its midpoint, and chains the pieces it keeps into rings. The
T-junction split is not a corner case: `getCurrentClippingPath` starts from the
bounding box of the clipping paths and intersects each path into it, so the
first intersection is always tangent, and a crossing test alone answers the
bounding box instead of the shape.

### `pdmodel/graphics/shading` — the model half of all seven types

| Java source | Go source | Status |
| --- | --- | --- |
| `PDShading.java` | `shading/pdshading.go` | done — `Shading` is the interface, `PDShading` the shared state |
| `PDShadingType1/2/3.java` | `shading/pdshadingtype123.go` | done |
| `PDTriangleBasedShadingType`, `PDShadingType4/5.java` | `shading/pdshadingtype45.go` | done |
| `PDMeshBasedShadingType`, `PDShadingType6/7.java` | `shading/pdshadingtype67.go` | done |
| `Patch`, `CoonsPatch`, `TensorPatch`, `CubicBezierCurve` | `shading/patch.go` | done — unexported, as Java's are package-private |
| `Vertex`, `Line`, `ShadedTriangle`, `CoordinateColorPair` | `shading/triangle.go` | done — unexported |
| the 19 `*Paint` and `*Context` classes | `rendering/raster/shading*.go` | **not ported by name** -- see below; `track/raster` ported their arithmetic |

The nineteen are `AxialShadingPaint`, `AxialShadingContext`,
`RadialShadingPaint`, `RadialShadingContext`, `Type1ShadingPaint`,
`Type1ShadingContext`, `Type4ShadingPaint`, `Type4ShadingContext`,
`Type5ShadingPaint`, `Type5ShadingContext`, `Type6ShadingPaint`,
`Type6ShadingContext`, `Type7ShadingPaint`, `Type7ShadingContext`,
`ShadingPaint`, `ShadingContext`, `TriangleBasedShadingContext`,
`GouraudShadingContext` and `PatchMeshesShadingContext`. Each is a
`java.awt.Paint` or a `java.awt.PaintContext` that fills a raster. The colour
evaluation they call into — the function, the colour space conversion, the patch
subdivision, the triangle interpolation — is here.

### `pdmodel/graphics/pattern`

| Java source | Go source | Status |
| --- | --- | --- |
| `PDAbstractPattern.java`, `PDTilingPattern.java` | `pattern/pattern.go` | done |
| `PDShadingPattern.java` | `pattern/pattern.go` | done |
| `color/PDPattern.java` | `pattern/pdpattern.go` | done — **in this package, not `graphics/color`** |

### `contentstream` — the graphics engine and its operators

| Java source | Go source | Status |
| --- | --- | --- |
| `PDFGraphicsStreamEngine.java` | `contentstream/graphicsstreamengine.go` | done, minus the operator registrations |
| `operator/graphics` — all 23 | `contentstream/operator/graphics/graphics.go` | done |
| `operator/color` — all 13 | `contentstream/operator/color/color.go` | done |
| `operator/DrawObject.java` | `contentstream/drawobject.go` | done |
| `operator/markedcontent/DrawObject.java` | `operator/markedcontent/markedcontent.go` | done |
| `operator/state/SetGraphicsStateParameters.java` | `operator/state/state.go` | done — slice 8 ported it, waiting on `PDExtendedGraphicsState` |

Java has three `DrawObject` processors, one per engine, and slice 3 deferred two
of them on `PDXObject`; all three are here now. `PDFStreamEngine`'s remaining
half came with them: `showForm`, `showTransparencyGroup`, `processSoftMask`,
`processTransparencyGroup`, the two `processTilingPattern` overloads,
`processChildStream`, `showAnnotation`, `getAppearance` and `processAnnotation`.
So did the two arms of `processStreamOperators` that clear
`shouldProcessColorOperators` — an uncoloured tiling pattern, and a Type 3 char
proc whose first operator is `d1` — which slice 2 recorded as unreachable.

### `pdmodel/PDResources` and `DefaultResourceCache` — finished

`getXObject` with `isAllowedCache`, `getShading` and `getPattern` are ported,
and with `getExtGState` and `getProperties` from slice 8 the family is complete.
So are `add` and `put` for a shading and for a pattern, which neither branch had.
`DefaultResourceCache` gains its five remaining halves — XObjects, shadings,
patterns, extended graphics states and property lists — with the stable-cache
bookkeeping Java repeats per kind written once, as a generic map.

### `pdfbox/rendering`

| Java source | Go source | Status |
| --- | --- | --- |
| `ImageType.java` | `rendering/imagetype.go` | done — minus `toBufferedImageType` |
| `RenderDestination.java` | `pdmodel/graphics/optionalcontent/renderdestination.go` | done — **declared there**, aliased in `rendering` |
| `PageDrawerParameters.java` | `rendering/pagedrawerparameters.go` | done |
| `GlyphCache.java` | `rendering/glyphcache.go` | done |
| `PDFRenderer.java` | `rendering/pdfrenderer.go` | done — minus the `BufferedImage` it makes |
| `PageDrawer.java` | `rendering/pagedrawer.go`, `pagedrawer_oc.go` | done — minus four raster pieces |
| `GroupGraphics.java` | `rendering/raster/group.go` | a `Graphics2D` subclass, so slice 9 left it; **`track/raster` ported it** |
| `SoftMask.java` | `rendering/raster/softmask.go` | a `java.awt.Paint`, so slice 9 left it; **`track/raster` ported it**, with the drawer half in `rendering/softmask.go` |
| `TilingPaint.java` | `rendering/raster/tiling.go` | a `java.awt.Paint`, so slice 9 left it; **`track/raster` ported it** |
| `TilingPaintFactory.java` | `rendering/raster/tilingcache.go` | **`track/raster` ported it** |

`PageDrawer` keeps every decision: which paint applies to a colour, what the
stroke is made of, what the clip intersects to, whether a path is rectangular,
how thin a clip may be before it is widened, whether an optional content group is
visible at this destination, whether an annotation is skipped, how far an image
may be subsampled, whether a transparency group needs its backdrop.

Four pieces of it are raster work end to end and were not ported by this slice.
`track/raster` ported all four; the rows above say where each landed. What this
slice deferred:

- **the `TransparencyGroup` inner class**, which makes a `BufferedImage`,
  renders into it and composites it. `Backend.PushGroup` and `PopGroup` stand
  for it, and the box it computes is ported as `transparencyGroupBox`.
- **the pixel work of the stencil-mask-with-pattern arm of `drawImage`** —
  `dilateAlpha`, the inverted lookup table, the per-pixel alpha combine of
  PDFBOX-6077 and PDFBOX-5403. The port decides "this stencil, that paint" and
  `Backend.DrawStencil` stands for the rest.
- **`applySoftMaskToPaint`'s building of the mask raster**, with `adjustImage`.
  `rendering.SoftMaskedPaint` names the mask and the matrix it was installed
  under instead, which is what a backend needs to build the same raster. One
  branch is lost with it: Java answers the parent paint where the group rendered
  to nothing — "Adobe Reader ignores empty softmasks instead of using bc color"
  — which only a backend can tell.
- **`applyTransferFunction`**, which maps an image's pixels through the /TR
  function.

`PDFRenderer.getPageImage` goes with them: it exists so a non-isolated
transparency group can read the page it is being composited onto, which is what
`PushGroup`'s `needsBackdrop` says instead.

### `pdfbox/printing`

| Java source | Go source | Status |
| --- | --- | --- |
| `Orientation.java`, `Scaling.java` | `printing/printing.go` | done |
| `PDFPrintable.java` | `printing/pdfprintable.go` | done |
| `PDFPageable.java` | `printing/pdfpageable.go` | done |
| `java.awt.print.Paper`, `PageFormat` | `printing/pageformat.go` | the state only |

Go has no print system. What is ported is what the two classes compute: the
rotated crop and media boxes, the portrait-normalised paper of the PDFBOX-2922
workaround, the scale-to-fit arithmetic, the centering and its negative-value
guard, and the page border. `java.awt.print.Printable` and `Pageable` become
plain methods taking a `rendering.Backend`. Rasterizing a page to a bitmap
before printing it used to answer `ErrRasterizeUnsupported`; `track/raster`
closed that, and the error is gone.

### The tests

Java's three rendering tests and one printing test do not port as they stand.

- **`TestPDFToImage`** is disabled in Java itself, because different JVMs
  produce different images.
- **`TestRendering`** renders twenty files and asserts that nothing threw. The
  twenty files are not in this repository: the Java build downloads them into
  `target/pdfs`. `track/raster` renders four pages it writes itself instead.
- **`TestQuality`** reads back four pixels of four files from `target/pdfs`,
  which the build downloads. **[input present since 2026-09-11; see "What the fetch unblocked"]**
- **`TestPDFPrintable`** has five cases: three port as they stand — the page
  index, the printer state left unchanged, and the result codes — and two read
  back pixels to see whether the page border came out grey. Those two are asked
  of the recording backend instead, which sees the stroke rather than the
  pixels it leaves.

What each was asking is asked instead of a backend that records every call the
drawer makes, over real content streams through the real engine:
`rendering/pagedrawer_test.go`, `rendering/pdfrenderer_test.go` and
`printing/printing_test.go`. The colour a rectangle is filled in, the stroke
parameters a line carries and the three values `getStroke` invents, the clip `W`
leaves behind, the group a transparency form pushes, the operators a hidden
optional content group swallows, the transform a rotated page installs, the
scale each of the four scaling modes chooses.

### Deviations from Java

- **A shape is flattened when it becomes an `Area`.** The result is a polygon
  where the JDK's would still be a curve, so the visible effect is the
  flattening tolerance rather than a different region; nothing in PDFBox reads
  the curves back out of an `Area`, because every use is a clip. Documented on
  the type in `awt/geom/area.go`.
- **`PDPattern` is declared with the patterns, not in `graphics/color`.** It
  reads a `PDAbstractPattern` out of the resources, so it would make `color`
  import `pattern`, which imports `color` for the underlying colour space.
  `color.Create` reaches it through the `NewPatternColorSpace` hook
  `pattern/pdpattern.go` sets from its `init`.
- **The sixty operators are not registered in the engine's constructor.** Every
  processor holds the engine, so the operator packages import `contentstream`
  and it cannot import them back. `rendering.addAllOperators` is that list,
  called from `NewPageDrawer`, the way `text.NewLegacyPDFStreamEngine` already
  registers its own.
- **The plain `DrawObject` lives in `contentstream`**, where Java has it in
  `contentstream/operator`: the port's `operator` package holds no processors,
  because a processor names the engine and the engine's package imports
  `operator`.
- **`PDFormXObject` and `PDAppearanceStream` do not implement
  `PDContentStream`.** `getResources` answers a `PDResources`, which lives in
  `pdmodel`, and `graphics/form` cannot import it; `contentstream` adapts both,
  the way it already adapts `PDType3CharProc`.
- **The two `IllegalStateException`s `PDFStreamEngine` throws for a child
  stream processed without a page are errors, not panics.** Every caller of
  those methods is an operator, and an operator's errors already travel back
  through `processOperator`.
- **`PDDocument` gains a `CreateStream`**, which Java has no equivalent of —
  everything there goes through `getDocument().createCOSStream()`. It is what
  makes a `*PDDocument` a `common.COSDocumentLike`, so it can be passed where
  `new PDStream(document)` takes one.
- **`RenderDestination` is declared in `pdmodel/graphics/optionalcontent`** and
  aliased back under the Java name in `rendering`. Java's `rendering` imports
  `optionalcontent` for the groups and `optionalcontent` imports `rendering`
  back for `getRenderState(RenderDestination)`; Go forbids the cycle, and slice
  9's `rendering` must import `contentstream` and through it `pdmodel` and
  `optionalcontent`. Same device as `pdmodel.ResourceCache` for
  `pdmodel/font`'s.
- **`adjustClip` does not ask for a transform type bitmask.** Java reads
  `AffineTransform.getType()`, which the port does not have; the two tests it
  makes — "translation and flip only" and "no shear or rotation" — are written
  out against the matrix in `rendering/pagedrawer.go`, with the Java bits named.
- **`ProcessSoftMask` dereferences the soft mask without checking**, as Java
  does. Its only caller, `PageDrawer.applySoftMaskToPaint`, has already tested
  the mask; Java's method is `protected` and the port's is exported, so the
  precondition is written on it.
- **`renderPageToGraphics` mutates the surface it is given**, which is what Java
  does and what Java's own javadoc warns about under PDFBOX-4583.

Java bugs carried rather than fixed: JAVA-BUGS.md 49 and 50, the two restores
`PageDrawer` and `PDFStreamEngine` make without a `finally`, and JAVA-BUGS.md
51, a pattern's underlying colour space built without the resources.

### Defects found and fixed

- **The resource cache keyed two of its kinds on the stable-cache hash rather
  than on the `COSObject`.** With the stable cache disabled that hash is
  unavailable, so
  colour spaces (slice 2) and extended graphics states (slice 8) were not cached
  at all; with it enabled, two objects sharing a hash collided. All eight kinds
  key on the `COSObject` now and use the hash only for the removal bookkeeping,
  as Java does. `pdmodel/resourcecache.go`.
- **Text inside a form XObject was silently lost**, because the plain
  `DrawObject` was one of the two slice 3 deferred, so the text extractor never
  walked into a form. `contentstream/drawobject.go`.
- **`PageDrawer.setClip` was unexported.** Java's is `protected final`, and its
  javadoc says an embedder overriding `showGlyph` may need it; exported now.
- **`Math.round(float)` is `floor(x + 0.5)`**, which rounds a half towards
  positive infinity, where Go's `math.Round` rounds away from zero, so
  `Math.abs(Math.round(x))` differed for a negative half.
- **`getSubsampling`'s `imageWidth * imageHeight` is an `int` product that wraps
  at 2^31**, which a Go `int` does not.
- **`PDFRenderer.transform` built its transform from the identity** instead of
  concatenating onto the one the surface already carries, which discarded
  `PDFPrintable`'s translate to the imageable area and its centering — every
  printed page would have landed in the top left corner of the paper.
  `rendering/pdfrenderer.go`; `TestScalingChoosesTheScale` and
  `TestCenteringTranslatesByHalfTheSlack`.
- **`applySoftMaskToPaint` logged and carried on for a soft mask whose subtype
  is neither `/Alpha` nor `/Luminosity`**, where Java throws; it returns the
  error now. `rendering/pagedrawer.go`.
- **`Area.equals` compared rings pairwise rather than the sets they describe**,
  so a square did not equal the union of its two halves. It is the JDK's
  algorithm now — exclusive-or, then ask whether what is left is empty — over
  the `ExclusiveOr` and `IsEmpty` the port already had. `awt/geom/area.go`;
  `TestEqualsComparesTheSetsNotTheRings`.
- **`renderImage` accumulated the backend's transform.** Java takes a
  `Graphics2D` of a fresh `BufferedImage` on every call, so the transform always
  starts at the identity; the port concatenated onto whatever the caller's
  backend carried, which after `DrawPage` is the previous page's flip, and two
  consecutive `RenderImage` calls compounded. It starts from the identity now
  and puts back what it found. `rendering/pdfrenderer.go`;
  `TestRenderImageStartsFromAFreshSurface`,
  `TestRenderImageLeavesTheBackendAsItFoundIt` and
  `TestRenderImageIgnoresTheBackendsOwnTransform`.
- **`PDFPrintable.print` leaked six kinds of state.** Java's printable draws on
  a copy — `graphics.create()` first, `printerGraphics.dispose()` last — so
  nothing it does reaches the surface the print system handed it; the port
  restored only the transform, and with `showPageBorder` on it was guaranteed to
  leave a grey paint, a hairline stroke and a clip behind. `Backend` gains
  `Create` and `Dispose`, and `Print` works on the copy.
  `printing/pdfprintable.go`; `TestPrintLeavesEveryPieceOfStateAlone`.
- **The page border was drawn at the corner of the paper.** Java captures
  `printerBorderTransform` after translating to the imageable area and centring
  the page; the port captured it before, so on any paper with a margin, or with
  centring on, the border framed the wrong thing. `printing/pdfprintable.go`;
  `TestPageBorderIsDrawnAroundTheRenderedPage`.
- **`PDExtendedGraphicsState.CopyIntoGraphicsState`'s doc comment** still said
  the `/SMask` arm was not applied, three commits after it was.

The review also swept every Java type in the slice's packages, the public and
protected methods of `PDFRenderer`, `PageDrawer`, `PDFPrintable` and
`PDFPageable`, the sixteen `COSName`s against their Java constants, every
`finally`, Java's narrowing conversions and every deferral; what it found is
above, and the rest matched.

### Still open

- **The print spooler.** Go has no `PrinterJob` and no `javax.print`, so
  nothing enumerates the printers on the machine, reads the trays and media
  sizes one offers, shows a dialog or hands it a job, and `PrintPDF` cannot be
  ported. That is a per-platform API and no pure-Go library binds it.

## Track `xmpbox` — the XMP metadata module

Branch `track/xmpbox`. A parallel track rather than a slice: `xmpbox` is a
module of its own that depends on nothing else in the build, so it does not have
to wait for a slice, and nothing waits for it. `pdfbox` hands back the raw
metadata stream and this module parses it. All 74 Java files are ported, and all
27 Java test files, as 396 Go test cases.

### `org.apache.xmpbox` — the root, 3 files

| Java file | Go file | Notes |
| --- | --- | --- |
| `XMPMetadata.java` | `xmpmetadata.go`, `xmpmetadata_schemas.go` | done |
| `XmpConstants.java` | `xmptype/xmpconstants.go`, aliased in `xmpconstants.go` | moved down a layer, see below |
| `DateConverter.java` | `xmptype/dateconverter.go`, aliased in `xmpconstants.go` | moved down a layer, see below |

### `org.apache.xmpbox.type` — all 33 files, as `xmpbox/xmptype`

Renamed because `type` is a Go keyword; `mapping/packages.tsv` records it.

| Java files | Go file |
| --- | --- |
| `Attribute`, `Cardinality`, `AbstractField`, `PropertyType`, `StructuredType`, `PropertiesDescription` | `attribute.go`, `cardinality.go`, `field.go`, `propertytype.go` |
| `AbstractSimpleProperty`, `BooleanType`, `IntegerType`, `RealType`, `TextType`, `DateType` | `simpleproperty.go` |
| The 13 derived text types | `derivedtext.go` |
| `AbstractComplexProperty`, `ComplexPropertyContainer`, `AbstractStructuredType`, `ArrayProperty` | `complexproperty.go` |
| The 17 structured types | `structuredtypes.go`, `structuredtypes2.go` |
| `DefinedStructuredType` | `definedstructuredtype.go` |
| `Types`, `TypeMapping` | `types.go`, `typemapping.go`, `typemappingcreate.go` |
| `BadFieldValueException` | `errors.go` |

### `org.apache.xmpbox.schema` — all 14 files

| Java file | Go file |
| --- | --- |
| `XMPSchema.java` | `xmpschema.go` |
| `XMPSchemaFactory.java`, `XmpSchemaException.java` | `schemafactory.go` |
| `AdobePDFSchema`, `DublinCoreSchema`, `PDFAExtensionSchema`, `PDFAIdentificationSchema`, `XMPRightsManagementSchema`, `XMPPageTextSchema`, `XMPBasicJobTicketSchema` | `schemas.go` |
| `XMPBasicSchema` | `schemas2.go` |
| `PhotoshopSchema`, `XMPMediaManagementSchema` | `schemas3.go` |
| `TiffSchema` | `schemas4.go` |
| `ExifSchema` | `schemas5.go` |

### `org.apache.xmpbox.xml` — all 6 files, plus a DOM

| Java file | Go file |
| --- | --- |
| `DomHelper.java` | `domhelper.go` |
| `DomXmpParser.java` | `domxmpparser.go` |
| `PdfaExtensionHelper.java` | `pdfaextensionhelper.go` |
| `XmpSerializer.java` | `serializer.go` |
| `XmpParsingException.java`, `XmpSerializationException.java` | `errors.go` |
| — (`org.w3c.dom`, `javax.xml.parsers`, `javax.xml.transform`) | `dom.go` |

### The DOM, and how it differs from Xerces

Java reaches the XML through `org.w3c.dom`: `DocumentBuilderFactory` parses into
a namespace-aware `Document` and a `Transformer` writes one back. Go has no DOM,
so `dom.go` is one — the node kinds this module asks for and nothing else.
`encoding/xml` alone will not do, because it resolves a name's prefix away and
`DomXmpParser` reads an element's prefix as often as it reads its namespace. The
parser is built on `encoding/xml`'s `RawToken`, which reports names as written,
with namespace resolution, element nesting and the DOCTYPE refusal done here.

`go/w3c/dom`, which slice 8 added for XFDF, is a second DOM, and the two do not
fold together. That one is read only, because XFDF is written out by hand with a
`Writer`, and it resolves a prefix away when it is namespace aware, because that
is what the FDF reading matches against. This one has to build a document in
order to serialize it, and has to keep the prefix on every name.

Where the two differ, and what the port does:

- **Attribute order.** Xerces holds an element's attributes in a `NamedNodeMap`
  it searches by name, so they are held, walked and written in name order rather
  than document order. Both the parser and the serializer walk that list and the
  order reaches the output, so `Element.addAttribute` keeps it sorted the same
  way.
- **Namespace fixup.** An element built with `createElementNS` carries a
  namespace and no declaration; a DOM serializer emits the missing declaration
  as it writes the start tag, and leaves out one an enclosing element already
  makes with the same binding. `writeElement` does both, and writes an element's
  own declarations before its other attributes, which is the order the JDK's
  serializer uses.
- **Line separators.** The JDK's transformer ends the document with the
  platform's separator, CRLF on Windows. The port always writes LF. Every Java
  caller that compares serialized output normalizes CRLF to LF first —
  `DeserializationTest.checkTransform` does — so this changes nothing they
  check.
- **Whitespace and comments.** `removeCommentsAndBlanks` is ported as written,
  including its early return for an element with one child, so a comment inside
  a single-child element survives in both.
- **Malformed input.** Both parsers refuse a document that is not well formed;
  the message differs, and no test compares it, because Java's comes from
  Xerces. A DOCTYPE declaration is refused here the way
  `disallow-doctype-decl` refuses it there.
- **Encoding.** Java's parser sniffs the encoding declaration; the port reads
  UTF-8, which is what every XMP packet in the corpus is and what the
  specification requires.

The serializer's output was checked against Java's, not against the port's own:
every fixture in `DeserializationTest` serializes byte-for-byte identically once
CRLF is normalized to LF, and the twelve SHA-256 digests that test asserts are
over Java's bytes and are the ported test's assertions too.

### No reflection

Java reads `@StructuredType` and `@PropertyType` off a class at run time, holds
a `Class` in each `Types` constant, and finds a schema by
`clz.getAnnotation(StructuredType.class).namespace()`. None of that is ported as
reflection. Instead:

- each structured type declares its `StructuredTypeInfo` and its
  `PropertiesDescription` as values, and registers them against its `Types`
  constant from its package's `init`;
- each `Types` constant carries a constructor function rather than a `Class`,
  registered the same way, and `ImplementingClassName` answers the simple name
  Java's messages use;
- `getClass().getSimpleName()` becomes a `TypeName() string` method;
- each schema's namespace is an exported constant, and `XMPMetadata`'s typed
  getters ask for the namespace and type-assert the result, which is what Java's
  cast does anyway;
- `TestValidatePermitedMetadata` walks the class for annotated fields; the port
  asks the schema factory's property description, which is what those
  annotations were read into.

### Package cycles

Java's four packages form three cycles that Go forbids. The devices, all of them
already used elsewhere in this port:

- **root ↔ type**: `XmpConstants` and `DateConverter` are declared in
  `xmpbox/xmptype` and aliased back under the Java name in the Java place,
  because the root package holds `XMPMetadata`, which every property points back
  at. Same device as `pdmodel.ResourceCache` for `pdmodel/font`'s.
- **type ↔ schema**: `TypeMapping.initialize` names the twelve default schema
  factories. `xmptype` declares `SchemaFactoryLike` and
  `RegisterDefaultSchema`, and `xmpbox/schema`'s `init` pushes the twelve in.
  `NewDefaultSchemaFactory` is the hook `addNewNameSpace` reaches back through;
  where `xmpbox/schema` is not linked into a binary it is nil and
  `AddNewNameSpace` does nothing, which cannot happen in practice because
  nothing reaches a `TypeMapping` without going through a schema.
- **schema → root**: `XMPSchemaFactory.createXMPSchema` calls
  `metadata.addSchema`. `schema.MetadataHolder` names what is used and
  `xmpbox.XMPMetadata` satisfies it.

`xmpbox/xml` imports the root package directly, because Java's root package
imports nothing from `org.apache.xmpbox.xml`.

### Which Java tests are ported

All 27, as 396 Go test cases.

| Java test | Go test |
| --- | --- |
| `type/AttributeTest` | `xmptype/attribute_test.go` |
| `type/TestSimpleMetadataProperties` | `xmptype/simplemetadataproperties_external_test.go` |
| `type/TestAbstractStructuredType`, `type/TestDerivedType` | `xmptype/structuredtype_external_test.go` |
| `type/TestStructuredType` | `xmptype/teststructuredtype_external_test.go` |
| `type/AbstractTypeTester` | — a reflection helper with no cases of its own |
| `schema/SchemaTester`, `schema/XMPSchemaTester` | `schema/schematester_test.go` |
| `schema/DublinCoreTest` | `schema/dublincore_test.go` |
| `schema/XMPBasicTest` | `schema/xmpbasic_test.go` |
| `schema/AdobePDFTest`, `schema/AdobePDFErrorsTest` | `schema/adobepdf_test.go` |
| `schema/PhotoshopSchemaTest` | `schema/photoshop_test.go` |
| `schema/XMPMediaManagementTest` | `schema/mediamanagement_test.go` |
| `schema/XmpRightsSchemaTest` | `schema/xmprights_test.go` |
| `schema/BasicJobTicketSchemaTest` | `schema/basicjobticket_test.go` |
| `schema/PDFAIdentificationTest`, `schema/PDFAIdentificationOthersTest` | `schema/pdfaidentification_test.go` |
| `schema/TestExifXmp` | `schema/exifxmp_test.go` |
| `schema/XMPSchemaTest` | `schema/xmpschema_test.go` |
| `xml/DomXmpParserTest` | `xml/domxmpparser_test.go`, `domxmpparser2_test.go`, `domxmpparser3_test.go` |
| `parser/DeserializationTest` | `xml/deserialization_test.go` |
| `DateConverterTest` | `dateconverter_test.go` |
| `XMPMetaDataTest`, `DoubleSameTypeSchemaTest`, `TestXMPWithDefinedSchemas`, `TestXMPWithUndefinedSchemas`, `TestValidatePermitedMetadata` | `xmpmetadata_test.go` |

The tests read the Java test resources from `xmpbox/src/test/resources`, the way
the ported tests in the other modules do; nothing is copied.

`SchemaTester` and `XMPSchemaTester` name every accessor by reflection and find
the fields by walking the class. The port's harness takes a table naming them,
and makes the same three assertions: nothing is set on a fresh schema, a value
set through the accessors comes back through them, and setting one field leaves
every other field of the schema alone. `SchemaTester`'s fifty rounds of random
values become the several values a row may carry.

Two Java cases are not assertions about the port and are left out:
`XMPMetaDataTest.testTransformerExceptionMessage` and
`testTransformerExceptionWithCause`, which construct an exception and assert
that it was thrown. Three Go test files are not ports and say so in their own
doc comments: `dateconverter_smart_test.go`, whose values were read by running
Java's `DateConverter`; `nullprefix_test.go`, which pins JAVA-BUGS.md 59; and the
`schema` harness.

This module depends on nothing outside the JDK, so it compiles with `javac` and
runs under `jshell`, and the two implementations were driven over the same
input: every XML fixture in the repository, every simple field of every one of
the twelve schemas, all seventeen structured types, and fifty-nine date strings.
130 of the 132 fixture runs are byte-identical or the identical failure message
once CRLF is normalized to LF; the two that differ are one file,
`PDFBOX-5835.xml`, which Java cannot serialize at all. What the comparison
found is in the two lists below.

### Deviations from Java

- **`XMPSchema.reorganizeAltOrder` and an alternative with no `xml:lang`.**
  `languageOf` answers the empty string and the walk continues, where Java
  raises NullPointerException. JAVA-BUGS.md 52.
- **`DomXmpParser.parseDescriptionInner` and an undeclared property.** The port
  reports the property as `NoType`; the parse fails either way. JAVA-BUGS.md 57.
- **Java's null in a setter.** `setTextPropertyValue(name, null)` and its three
  neighbours remove the property, and `setAboutAsSimple(null)` removes the
  attribute; a Go string cannot be null, so
  `XMPSchema.RemoveUnqualifiedProperty` and `RemoveAttribute` are those paths.
  Without them the branches would be unreachable.
- **`setUnqualifiedLanguagePropertyValue` with an empty value** removes the
  alternative, where Java removes it only for null. An alternative whose value
  is the empty string cannot be set through the port.
- **`DateConverter.toCalendar` of a blank string** answers the zero time where
  Java answers null; `IsBlankDate` is how a caller tells the two apart, and
  `DateType` uses it so that an empty date property stays empty.
- **`QName` gained a `Prefix` field.** `javax.xml.namespace.QName` has one and
  the parser puts it in its messages; the port's had only the two parts
  `getSpecifiedPropertyType` reads.
- **Namespace declarations are written in sorted order** where Java walks a
  `HashMap`, whose order is arbitrary but fixed for a given content. Sorting
  makes the same metadata always write the same bytes.
- **Three unchecked casts carry on rather than raising.**
  `ArrayProperty.getElementsAsString` — which Java's own FIXME flags —
  `XMPSchema.mergeComplexProperty` and
  `removeUnqualifiedArrayValue(String, AbstractField)` cast every element of an
  array without checking, so an array holding a shape they do not expect raises
  `ClassCastException`. The port skips the element, or compares without the
  cast, in `xmpbox/schema/xmpschema.go` and `xmpbox/xmptype/complexproperty.go`.
- **A property whose element had no prefix is written under its local name**,
  where Java's serializer cannot write the packet at all. JAVA-BUGS.md 59;
  `TestSerializingAnUnprefixedPropertyWhereJavaFails` in
  `xmpbox/xml/nullprefix_test.go` pins it.
- **A date before the 1582 cutover is a different instant** from Java's, whose
  `GregorianCalendar` switches to the Julian calendar there; both write the same
  ISO 8601 string. `TestDatesBefore1582DifferFromJava` pins it. Implementing the
  hybrid calendar is out of proportion to a case XMP does not carry.
- **`ErrorType.Configuration` is unreachable**, because the port has no
  `DocumentBuilderFactory` to fail to configure.
- **A sequence holding an empty date can still have an element removed**, where
  Java raises NullPointerException on the empty one. JAVA-BUGS.md 60.
- **A list of sequence dates leaves out an element that holds no date**, where
  Java puts a null in the `List<Calendar>` its javadoc promises. JAVA-BUGS.md
  61; `track/java-bug-fixes` closed the port's earlier zero-time stand-in.
- **The two places Java pops `nsFinder` without a `finally`** — the loop in
  `parseChildrenAsProperties` and the tail of `parseLiDescription` — pop without
  a `defer` here, so the same leak on an early exit is reproduced. The seven
  Java `finally` blocks are all `defer`.
- **The `StringIndexOutOfBoundsException` of JAVA-BUGS.md 56 is a panic**, where
  the checked exceptions are errors.

Ten Java bugs came out of this track: JAVA-BUGS.md 52 to 61.

### Defects found and fixed

- **A getter of a derived text field read back as nothing.** Java's
  `getPropertyAs(name, TextType.class)` answers a `URLType` or an
  `AgentNameType`, because `Class.isInstance` is true of a subclass; a Go type
  assertion to `*TextType` is false for a type that only embeds one, so
  `getBaseURL`, `getCreatorTool` and every other accessor of a derived text
  field answered the empty string. `xmptype.TextValued` and
  `schema.TextPropertyOf` are what that call means here.
- **`DimensionsType.toString`** wrote its two Floats through `%v`, which drops
  the fraction part Java always writes, and answered 0 for a dimension that is
  not there rather than `null`.
- **`instanciateSimpleProperty`'s message** named the type rather than the class
  that implements it — "Date" where Java writes "DateType" — and wrote the cause
  into the message, which Java attaches to the exception without putting it in
  `getMessage`.
- **The serializer** wrote redundant namespace declarations, wrote attributes in
  the order they were set, wrote an empty property as a pair of tags where
  `setTextContent("")` adds no text node, and ended the document without a line
  separator. All four are in the DOM section above.
- **`setAboutAsSimple("")`** removed the attribute where Java sets it to the
  empty string.
- **A `DateType` built from a blank string** held the zero time where Java holds
  no date at all, so PDFBOX-6029's empty date was written as the epoch.
- **A date property that is there and holds no date read back as the epoch.**
  Java's `getCreateDate` and its neighbours answer `getValue()`, which is null
  both when the property is absent and when it is there holding nothing, and the
  port checked only the pointer;
  `AbstractStructuredType.getDatePropertyAsCalendar` had the same shape. Fixed
  at the root: `DateType.Value` answers nil where the property holds no date,
  and `DateType.DateValue` answers `(time.Time, bool)` rather than a bare time,
  so every caller has to say what it does with the null.
  `TestAnEmptyDateReadsBackAsNothing` and
  `TestAnEmptyDateInAStructuredTypeReadsBackAsNothing` in
  `xmpbox/xml/emptydate_test.go` pin it.
- **`fromISO8601`** took any two-digit zone offset, where `java.time.ZoneOffset`
  refuses one beyond eighteen hours.
- **`toISO8601`** wrote the proleptic year where a `Calendar` counts within an
  era, so PDFBOX-6107's "0000-01-01" came back as "0000" rather than "0001".
- **`PdfaExtensionHelper.validateNaming` and `populateSchemaMapping` were
  unexported**, where Java's are public or protected; exported now.
- **`caseName` numbered subtests with `string(rune('1'+i))`**, which stops being
  a digit past nine rounds; `strconv.Itoa` now. No row carries more than two
  values today, so it could not bite yet.

### Still open

Nothing in this track. Every `TODO` and `FIXME` in the Go is Java's own, carried
over with it. Four differences are deliberate and pinned by tests, and are
listed above: the 1582 cutover, the unprefixed property, and JAVA-BUGS.md 60 and
61.

## Track `scratchfile` — the five files slice 0 deferred

Branch `track/scratchfile`. A parallel track rather than a slice: it ported the
five `pdfio` files `slice/0` left unported, three of them because `ScratchFile`
was deferred to phase 2 and two because they needed a decision. With these five,
**phase 0 is done: all 18 files**, and `PDPage.getContentsForStreamParsing` has
the `NonSeekableRandomAccessReadInputStream` it had been doing without since
slice 2. The rows are in the phase 0 table and the ported tests in the slice 0
note. `MemoryUsageSetting` and `ScratchFile` have no Java test at all, so every
expected value in `memoryusagesetting_test.go` was read out of the running Java
under `jshell` rather than reasoned about.

The four exported types say what they promise. `ScratchFile` is safe for
concurrent use, with the two locking hazards named. `ScratchFileBuffer`,
`NonSeekableRead` and `MappedFile` are not safe for concurrent use, which is
Java: none of the three synchronises anything, and each holds a single cursor.
`MemoryUsageSetting` is safe to read once built, and `SetTempDir` writes, so the
directory belongs set before the setting is shared.

### The memory mapping decision — B0

`RandomAccessReadMemoryMappedFile` maps the whole file with `FileChannel.map`,
and Go has no mapping in its standard library.

**Settled on `golang.org/x/exp/mmap`.** It carries the per-platform work the
port would otherwise write twice, once against `syscall.Mmap` and once against
`CreateFileMapping`, and it is reached from one file, so replacing it later
touches nothing else. Its cost is that `x/exp` promises no compatibility. That
trade is written into the header of `mappedfile.go`.
`RandomAccessReadMemoryMappedFileTest.testUnmapping`, which exists because of
JDK-4724038 — Windows refusing to delete a mapped file — passes.

### The flate fast path — B5

`PDPage.getContentsForStreamParsing` now branches the way Java's does. Its fast
path needed `FlateFilterDecoderStream` and
`NonSeekableRandomAccessReadInputStream`, and both are here now:
`NewFlateDecoderReader` in `flate.go` is the class itself, where the buffered
`Decode` had reproduced the class's behaviour and carried no streaming reader at
all. Checked against the corpus with and without the fast path: 37 of 40
documents unsorted and 36 of 40 sorted, identical either way.

### Java bugs found

Seven: [`JAVA-BUGS.md`](JAVA-BUGS.md) 62 to 66, and 72 and 73 from the review
round. JAVA-BUGS 62, 65 and 66 were reproduced by running the Java rather than
argued from it. JAVA-BUGS 66 and 72 are both `ScratchFile` taking its locks in
two orders, and both are named in the type's concurrency contract.

### Deviations from Java
- **`NewMappedFile` stats the file and refuses one over 2 GB before it opens
  anything.** Java opens the channel, then throws, and leaks it. JAVA-BUGS 73,
  said in `mappedfile.go`.
- **`ScratchFile.Close` wraps the cause of its `os.Remove`**, which Java has no
  equivalent of because `File.delete()` answers a boolean.
- **Two overflow guards cannot fire here.** `pageCount + ENLARGE_PAGE_COUNT >
  pageCount` in `enlarge` and `newSize < pageIndexes.length` in `addPage` are
  dead where Go's `int` is 64 bits. Both are kept, because they do fire where
  `int` is 32 bits, and both say so.
- **`MemoryUsageSetting.SetTempDir` takes a string where Java takes a `File`.**
  Nothing in the port needs a directory handle, and `os.Stat` is the check Java
  makes with `isDirectory()`.

### Defects found and fixed
- **`NonSeekableRead.fetch` read `(0, nil)` as the end of the stream.** Correct
  for `InputStream.read(byte[])`, which blocks until it holds a byte or the
  stream ends; `io.Reader` may answer `(0, nil)` at any time, so the port
  truncated a stream silently, mid-content. `pdfio/nonseekablestream.go`;
  `TestNonSeekableReadsPastAZeroByteRead`.
- **`NonSeekableRead.fetch` dropped bytes returned with an error.**
  `InputStream.read` cannot return bytes and throw, so Java sees the bytes from
  one call and the exception from the next; the failure is now kept in
  `pendingErr`. `pdfio/nonseekablestream.go`;
  `TestNonSeekableKeepsBytesReturnedWithAnError`.
- **`NonSeekableRead.Close` marked the source closed when the underlying close
  failed.** Java assigns `isClosed` after `is.close()` returns, so a close that
  throws leaves the source usable. `pdfio/nonseekablestream.go`;
  `TestNonSeekableCloseFailureLeavesTheSourceOpen`.
- **`NewFlateDecoderReader` propagated inflate errors.**
  `FlateFilterDecoderStream` catches the `DataFormatException`, keeps what
  inflated and reports the end, which is the tolerance PDFBOX-1232 inflates raw
  for; without it a truncated content stream failed to parse at all rather than
  parsing up to the damage. `filter/flate.go`;
  `TestFlateDecoderReaderEndsAtDamageInsteadOfFailing`.
- **That tolerance then swallowed the source's own failures.** Java reads its
  source outside the try block and catches `DataFormatException` alone, so an
  IOException propagates; `compress/flate` reports both through one error, and a
  failing disk was reported as the end of the page's content. `isDeflateDamage`
  separates the two. `filter/flate.go`;
  `TestFlateDecoderReaderPassesSourceErrorsOn`.
- **The 2 GB size check ran after the mapping**, so an unsupported file reserved
  the address range before being refused. `pdfio/mappedfile.go`.
- **Two log lines Java writes had been dropped**, and are back.
- **`x/exp` was marked `// indirect`**, and two mapped file tests did not defer
  their close, so a failing assertion left the mapping open for the cases after
  it.

### Deferrals
- `IOUtils.createTempFileOnlyStreamCache` was blocked on `ScratchFile` and is
  ported, with the two `TestIOUtils` cases for it.
- `createProtectedTempDir` is **not** ported and will not be: its only caller in
  the tree is `PDFDebugger`, which is not in the plan, and its substance is a
  JVM shutdown hook. Said in the header of `ioutils.go`.
- Java does not use `File.deleteOnExit` here — `createProtectedTempFile`'s
  javadoc says the caller is responsible, and the only shutdown hook in
  `IOUtils` belongs to `createProtectedTempDir` — so the scratch file is deleted
  in `ScratchFile.close()` and nowhere else, and a crash or a missed close
  leaves it behind. The port does the same, deleting it under `ioLock` in
  `Close`.

### The `slice/0` re-read — D9

All thirteen `slice/0` files were read against their Java, and none of them
mistranslates it. What came out was JAVA-BUGS 67 to 71 and the discovery that
entry 3 is a hang rather than an off-by-one; the account is in the `slice/0`
note, which also carries the one real difference the pass found,
`BufferedFile.Length` checking closed where Java's does not. Two things changed
in the Go: `ReadView` gained the `Rewind` override Java has, which the port had
been getting from the interface default and so was quietly fixing a Java bug,
and `SequenceRead.Read` now moves the cursor backwards on a `-1` the way
`currentPosition += bytesRead` does.

### Observations that are not defects
- `ReadBuffer.CreateView` and `BufferedFile.CreateView` append to a slice of
  clones that is never pruned, so a document with very many views holds a clone
  each until the source is closed. Java's per-thread map is bounded by thread
  count instead. The clones share their chunks or their file, so each costs a
  header rather than a copy.
- `SequenceRead.Seek` searches its sources from index 0; Java searches outward
  from the current index. Same source found, different number of comparisons.
- `RandomAccessInputStream` logs an error on a "should never happen" branch that
  the port answers with a plain `io.EOF`. The branch is unreachable except under
  unsynchronised concurrent access, which the type does not support either way.
- `ScratchFileBuffer.addPage` grows a page early, at `pageCount+1 >=
  len(pageIndexes)`, because `ScratchFileBuffer.java:103` does.

### Still open

- The three-buffer rewind of `NonSeekableRead` is exercised only by the Java
  cases. They are thorough — `testRewindAcrossBuffers2` and PDFBOX-5158 and
  5161 all live in the awkward corners — but the class is new to the port and
  has no corpus behind it yet.

## Track `test-backfill` — the Java tests merged slices missed

Branch `track/test-backfill`. Not a slice: it ported no new Java class. It ran
sixteen Java test classes that already-merged slices left behind, against Go
that already existed. **87 of their 107 cases are ported. Twenty are not, each
for a reason recorded below. Every one of the 87 that failed, failed because of
a defect in the port** — none of them is a Java bug, and `JAVA-BUGS.md` gains no
entry from this branch.

| Java test | `@Test` | Go test |
| --- | --- | --- |
| `pdfparser/TestCOSParser.java` | 21 | `pdfparser/objectparser_test.go` |
| `pdfparser/TestPDFParser.java` | 18 | `loader_test.go`, `testpdfparser_test.go` |
| `pdmodel/graphics/blend/BlendModeTest.java` | 17 | `blend/blendmode_test.go` |
| `pdmodel/fdf/FDFUtilsTest.java` | 13 | `fdf/fdfutils_test.go` |
| `util/StringUtilTest.java` | 7 | `util/stringutil_test.go` |
| `util/TestNumberFormatUtil.java` | 6 | `util/numberformatutil_test.go` |
| `pdfparser/PDFObjectStreamParserTest.java` | 5 | `pdfparser/objectstreamparser_test.go` |
| `util/TestHexUtil.java` | 4 | `util/hex_test.go` |
| `pdmodel/fdf/FDFFieldTest.java` | 4 | `fdf/fdffield_test.go` |
| `cos/TestCOSIncrement.java` | 3 | `pdfbox/increment_test.go` |
| `pdfparser/PDFStreamParserTest.java` | 2 | `pdfparser/streamtokenparser_test.go` |
| `pdfparser/EndstreamFilterStreamTest.java` | 2 | `pdfparser/streamparser_test.go`, `loader_test.go` |
| `pdmodel/fdf/FDFAnnotationTest.java` | 2 | `fdf/fdfannotation_test.go` |
| `cos/TestCOSUpdateInfo.java` | 1 | `cos/updateinfo_test.go` |
| `pdmodel/graphics/PDLineDashPatternTest.java` | 1 | `graphics/pdlinedashpattern_test.go` |
| `pdfparser/TestBaseParser.java` | 1 | `loader_test.go` |

The Go paths are under `go/pdfbox/`. Several Java cases became one Go table —
`TestCOSParser`'s 21 are 8 Go functions, `BlendModeTest`'s 17 are 3,
`FDFUtilsTest`'s 13 are one table of 13 rows.

The branch also ported the three `pdmodel` classes the coverage survey found
unported and unrecorded:

| Java source | Go source | Status |
| --- | --- | --- |
| `pdmodel/ResourceCacheFactory.java`, `ResourceCacheCreateFunction.java`, `DefaultResourceCacheCreateImpl.java` | `pdmodel/resourcecachefactory.go` | done |

They are the process-wide override point `PDDocument` reads its cache from;
without them a document could neither be given a different cache nor be told to
keep none, which the factory's own javadoc offers by setting the function to
null.

### Deviations from Java
- **`ResourceCacheFactory` is a package variable, guarded.** Java is a class of
  statics with a static initialiser; the setter is called from one thread while
  documents open on others. `pdmodel/resourcecachefactory.go`.
- **`PDFStreamParserTest.testNestedBI` asserts the offsets and not the
  wording.** Java asserts its whole message; the port's carries the same two
  offsets in the lower-case package-prefixed form every error in `pdfparser`
  uses. Said where it is.
- **`TestCOSIncrement` ends with `System.out.println(dash)` in
  `PDLineDashPatternTest`**; the port checks `String()` answers something rather
  than printing it, which is all that line proves. Said where it is.

### Defects found and fixed
- **`splitJava` and `splitOnCommaOrSemicolon` broke all three of
  `String.split`'s rules**, whose contract is in
  [`conventions/java-to-go.md`](conventions/java-to-go.md). Trailing empties
  were kept, so the `coords` attribute ending in a comma in
  `xfdf-test-document-annotations.xml` — a file in this repository, which Java
  reads without complaint — handed `parseFloat` an empty string and **panicked**;
  `strings.FieldsFunc` then dropped leading and interior empties too, so `"1,,2"`
  came back as two numbers where Java gives three and then rejects the middle
  one, and **the port silently accepted a coordinate list Java rejects and read
  the remaining numbers into the wrong positions**; and `splitJava("")` answered
  `[]` where Java answers `[""]`. Both helpers now go through one
  `splitJavaFunc`. `pdmodel/fdf/annotations.go`; `split_test.go`, covering
  leading, interior, trailing, all-separator and empty inputs for both
  separators.
- **`StringUtil.SplitOnSpace("   ")` answered `[""]` where Java answers `[]`.**
  `Pattern.split` answers the whole input untrimmed only when the pattern never
  matched; the port's loop stopped at one element, conflating that with a match
  that dropped every trailing empty. `util/stringutil.go`; `TestSplitOnSpace`
  asserts both shapes.
- **A float-to-long cast that saturates in Java and does not in Go.**
  `FormatFloatFast` guards with `value > Long.MAX_VALUE`, and `Long.MAX_VALUE`
  widened to a float is 2^63 exactly, so that value passes the guard and is then
  cast: Java gives 9223372036854775807 and Go gives -9223372036854775808 on
  amd64, so the port wrote one byte where Java writes nineteen. `int64OfFloat`
  is Java's cast. `util/numberformatutil.go`; `TestFormatOfIntegerValues`.
- **Attributes were walked in source order.** Xerces holds a `NamedNodeMap`
  sorted by qualified name, so the `/RC` that `FDFAnnotation.richContentsToString`
  writes is in that order; `w3c/dom` now inserts attributes in name order, and
  its doc comment, which still said "in the order they were written", points at
  `addAttribute` instead. `w3c/dom/dom.go`; `TestLoadXFDFAnnotations` asserts
  the string byte for byte. The `xmpbox` DOM found this independently and
  already did it; the two still cannot be folded together, for the reason the
  `track/xmpbox` section gives.
- **`Document.RemoveXRefOffset` was missing.** Java hands out the live
  cross-reference map, so every map operation is available; the port had put,
  add and clear and no remove.
  `PDFObjectStreamParserTest.testParseAllObjectsIndexed` changes an object's
  stream index by removing the entry and putting a new one, because `HashMap.put`
  keeps the key object it already has and `AddXRefTable` reproduces that
  faithfully. `cos/document.go`; `TestParseAllObjectsIndexed`.
- **`PDDocument.RemovePage` and `RemovePageAt` were missing.** `PDPageTree` had
  both halves; the two document-level methods were simply absent.
  `pdmodel/pddocument.go`; `TestIncrementallyCreateDocument`, which is six
  incremental saves with a reload and a re-check between each, and the first
  thing to exercise slice 7's incremental writer end to end.

### Three test headers that were not true

Each said the Java suite did not exercise something directly, which is why the
Go test had been written from the source instead. Each is now corrected and
followed by the ported cases:

| File | Claimed | Actually |
| --- | --- | --- |
| `pdfparser/objectparser_test.go` | "the Java suite exercises these only through whole documents" | `TestCOSParser` calls `parseCOSName` and `parseCOSLiteralString` directly, 21 times |
| `graphics/blend/blendmode_test.go` | "the Java suite covers the blend functions through rendered images" | `BlendModeTest` calls `blendChannel` with exact values |
| `pdfparser/streamparser_test.go` | (kept — its subject really is only reached through documents) | — |

### The twenty cases not ported (nineteen, since track/font-embedding took one)

| Java case | Why |
| --- | --- |
| ~~`TestPDFParser`, 17 of 18~~ **ported, all 18** | They read from `target/pdfs`, a directory the Maven build fills by downloading PDFs over the network. The port fetched nothing in a test. **[input present since 2026-09-11; see "What the fetch unblocked"]** 16 of the 17 are readable now; `WXMDXCYRWFDCMOSFQJ5OAJIAFXYRZ5OA.pdf` is the one the fetch did not land. `testPDFBox3950` also needs `PDFRenderer`, which is behind `rendering.Backend` |
| `TestCOSIncrement.testConcurrentModification` | Downloads a PDF from `issues.apache.org` |
| `TestCOSIncrement.testSubsetting` | ~~Needs `PDType0Font.load`, which is font embedding~~ — **ported by `track/font-embedding`**, which brought the load. It is `TestSubsetting` in `go/pdfbox/increment_test.go`, so nineteen of the twenty are still out |
| `TestNumberFormatUtil.testFormattingInRange` | A property test comparing against `BigDecimal` with `HALF_UP` rounding. Go has no arbitrary-precision decimal in its standard library, and re-implementing one to check a formatter would be checking the re-implementation. The five example-based cases it is built on are ported, with the exact bytes |

`TestPDFParser.testPDFParserMissingCatalog` is the one of its eighteen whose
fixture is checked in, and it is in `go/pdfbox/loader_test.go`. The other
seventeen are the whole-document recovery suite, and were the largest single
block of Java testing the port had no answer to; all eighteen are ported since
the fetch of 2026-09-11, in `go/pdfbox/testpdfparser_test.go`.

### Three Java files with no Go counterpart, because none of them is a test

| Java file | What it is |
| --- | --- |
| `fontbox/ttf/GSUBTableDebugger` | one `` that asserts nothing. It reads a font and prints the GSUB table; its own javadoc says "to be used mainly for debugging purposes" |
| `fontbox/ttf/gsub/GSUBTablePrintUtil` | no `` at all — the printer the above calls |
| `pdmodel/interactive/annotation/package-info` | a package declaration |

The survey that produced this branch was re-run and found nothing new: of the 54
Java test classes with no trace in the Go tests, 38 have one now, all sixteen of
this branch's are gone, and the remaining 20 are `tools` and `pdfbox-layout`,
which belong to `track/tools` and `track/pdfbox-layout`.

### Still open

- **`maxBinCharTestLength` is unpinned.** `PDFStreamParserTest.testInlineImages`
  carries a comment saying its last eight cases test the boundary of that
  constant, and the 39 cases do not constrain it: setting the port's to 3, 5, 9,
  15 or 40 leaves every one of them passing, because those cases put nothing but
  spaces inside the look-ahead window, so `startOpIdx` never leaves -1 and both
  checks that use the window length are skipped. This is the Java's, not the
  port's: `hasNoFollowingBinData` and `atEndOfInlineImage` are line-for-line
  ports, signed-byte comparison included. A distinguishing input was looked for
  — operators of several lengths at several distances past the `EI` — and none
  of the shapes tried told 9 from 10. Writing a case that pins it would be
  writing a test the Java does not have; it is worth doing by whoever next
  touches the inline image parser.

## Track `font-embedding` — writing a font into a document

Branch `track/font-embedding`. Not a slice: `slice/4` ported the fonts a
document is *read* with and `slice/7` ported writing, and the embedding half
fell between them. **Before it, a Go program could not write a PDF with an
embedded font** — `PDType0Font`'s `load`, `addToSubset` and `subset` panicked
where the half was missing, and that is most of what writing a PDF is for.

Four Java classes, 1,314 lines, plus the halves two ported classes were left
without:

| Java source | Go source | Status |
| --- | --- | --- |
| `pdmodel/font/Subsetter.java` | `truetypeembedder.go`, the `subsetter` interface | done |
| `pdmodel/font/TrueTypeEmbedder.java` | `truetypeembedder.go` | done |
| `pdmodel/font/PDCIDFontType2Embedder.java` | `pdcidfonttype2embedder.go` | done |
| `pdmodel/font/PDTrueTypeFontEmbedder.java` | `pdtruetypefont_embed.go` | done |
| `PDType0Font.load` / `loadVertical` / `addToSubset` / `subset` | `pdtype0font_embed.go`, `pdtype0font.go` | done — the three panics answer |
| `PDTrueTypeFont.load` | `pdtruetypefont_embed.go` | done |

`pdmodel/font` is **39 of 39**. `ToUnicodeWriter`, the fifth file the survey
listed, was ported by slice 7 in `tounicodewriter.go` and the survey missed it —
see "Rows this file had wrong" above.

A subsetting embedder keeps the font program open past its constructor, because
the subset is not built until the document is saved. Java holds it in
`PDDocument.fontsToClose` and closes it in `close()`; the port added
`fontsToClose` and `RegisterTrueTypeFontForClosing` to match, which is the first
time the port has needed a document to own a resource this way.

### Deviations from Java
- **`buildSubset` is a function field.** Java declares it `protected abstract`
  on `TrueTypeEmbedder` and the two concrete embedders implement it, one to
  build a subset and one to refuse. Go has no abstract method, so the base
  carries `buildSubsetFromStream func(...) error` and each constructor fills it
  in. The simple-font one panics with Java's `"use PDType0Font instead"`, which
  is an `UnsupportedOperationException` and so unchecked.
- **The embedders name what they need of a document in an `embeddingDocument`
  interface**, which `*pdmodel.PDDocument` satisfies, because `pdmodel/font`
  cannot import `pdmodel`. It is the third time the port has needed the shape;
  `common.COSDocumentLike` and `pdfwriter.PDDocumentLike` are the others.
- **`buildFontFile2` panics on an OpenType font with `glyf` outlines**, because
  Java calls `getCFF()` unguarded and it throws `UnsupportedOperationException`
  for a font with no PostScript table. Ported as written, said at the site.
- **`PDTrueTypeFont` sets `otf` to nil**, which is Java's own line with Java's
  own comment: "OpenTypeFonts are not fully supported yet".

### The Java bug

**[`JAVA-BUGS.md`](JAVA-BUGS.md) 74 — `TrueTypeEmbedder.getTag` indexes its
alphabet with a negative remainder.** Ported as written, said at the point of
difference in `subsetTag`, and measured. The `/ToUnicode` this branch writes
runs through JAVA-BUGS 33, whose failing case is measured rather than derived
since this branch. Nothing else this branch touched turned out to be the Java
behaving oddly.

### Defects found and fixed
- **Java's `Math.round` rounds half up; Go's `math.Round` rounds half away from
  zero.** Twenty sites. Every `/W2` entry of a vertical font is a negated
  metric, so an advance height of 128 in a 2048-unit em is -62.5, which Java
  writes as -62 and the port wrote as -63. Now one pair of helpers in
  `truetypeembedder.go`; `TestJavaRoundHalfUp` holds what `jshell` prints for
  ten inputs.
- **`ttf instanceof OpenTypeFont` was ported as a Go type assertion, which can
  never hold.** `OpenTypeFont` embeds `*TrueTypeFont` rather than extending it,
  so the field never carries one; `checkForCidGidIdentity` was unreachable and
  its body was a stub whose comment claimed the opposite. `AsOpenType` is this
  port's standing answer to that `instanceof`, as `pdcidfonttype2.go` already
  asks it, and the check is written out now, panicking as Java's unchecked
  `IllegalStateException` does. Not run; see below.
- **Three of Java's thirteen `load` overloads had no Go entry point**: the
  public `load(RandomAccessRead, boolean, boolean)`, `loadVertical(File)` and
  `loadVertical(TrueTypeFont, boolean)`. The four-argument form the others
  funnel into was unexported, and the two `File` loaders copied the whole font
  into memory where Java's `RandomAccessReadBufferedFile` reads through it.
  `pdtype0font_embed.go`; `TestEveryLoadOverload` runs all thirteen.
- **A typed nil in an interface is not `nil`.** `TrueTypeFont.getUnicodeCmapLookup`
  returns `null` where the font has no Unicode cmap and the caller is not
  strict; the Go returned `(*CmapSubtable)(nil)` boxed in a `CmapLookup`, so
  `PDType0Font`'s null check passed and the next call panicked.
  `truetypefont.go` now returns an untyped nil; `TestIndicScripts` found it the
  moment subsetting became real. The trap is in
  [`conventions/java-to-go.md`](conventions/java-to-go.md).
- **`getUnicodeCmapLookup` then skipped the GSUB branch when the cmap was
  null.** That nil check was put at the top of the method rather than on the
  branch Java returns the cmap from: Java reads the GSUB table first and, with a
  feature enabled and a table present, returns a `SubstitutingCmapLookup`
  wrapping the null cmap, which is not null, and raises whatever reading that
  table raised; the port answered nil and swallowed the error. `fontbox/ttf`;
  `TestUnicodeCmapLookupKeepsTheGsubBranch`, which holds the panic the port
  raises where Java raises the NullPointerException.
- **Both embedding sites passed `false` to `getUnicodeCmapLookup`.** The
  no-argument form delegates to `getUnicodeCmapLookup(true)`, the strict one,
  which raises where the font has no Unicode cmap; `PDType0Font:143` and
  `TrueTypeEmbedder:120` both call the no-argument form. Only the PDFBOX-5324
  fallback at `pdtype0font.go:333` takes `false`, and it still does.
- **Two `IllegalArgumentException`s were returning an error**, in `getWidths`
  and `getVerticalMetrics`; unchecked, so they panic now. Everything Java logs
  and swallows is logged and swallowed.
- **`fontsToClose` was a slice documented as a set.** Java's field is
  `Set<TrueTypeFont>`, so the same program registered twice is closed once; the
  port now keys a map on the pointer.
  `TestRegisterTrueTypeFontForClosingIsASet` registers one font twice and
  another once and expects two.
- **`testEmbeddedFontWithZeroWidthChars` was ported with a string of the port's
  own invention and without its second half** — the four assertions that the
  zero-width character has width 0 from `/W` and from the font program, an empty
  path, and an undamaged font, **which is the half that checks the four
  `forceInvisible` calls in `TrueTypeEmbedder.subset`**. Restored; removing
  `forceInvisible(0x200C)` now fails it. The two surrogate cases also took
  Java's font size and offset.

### The tests

`TestFontEmbedding`, 17 cases. **Eleven are ported.** The six that are not each
read a font the Maven build downloads into `target/fonts`, and the port fetches
nothing in a test: **[input present since 2026-09-11; see "What the fetch unblocked"]**

| Java case | Font it needs |
| --- | --- |
| `testCIDFontType2VerticalSubsetMonospace` | `ipag.ttf` |
| `testCIDFontType2VerticalSubsetProportional` | `ipagp.ttf` |
| `testMaxEntries` | `ipag.ttf` |
| `testSurrogatePairCharacter` | `ipag.ttf` |
| `testToUnicodePrefersUsedCodePoint` | `NotoSansCJKkr-VF.ttf` |
| `testToUnicodeCjkAndRadicalLookAlike` | `NotoSansCJKkr-VF.ttf` |

Every case that is ported writes a document, saves it, reads it back with
`Loader` and extracts the text with `PDFTextStripper`, which is what the Java
does. `TestCIDFontType2` and `TestCIDFontType2Subset` embed LiberationSans,
write `Unicode русский язык Tiếng Việt`, and read the same string back.

Three cases were added because the Java has no equivalent to port:

- `TestSimpleTrueTypeFontEmbedding` — `PDTrueTypeFont.load`, which
  `TestFontEmbedding` never exercises. Java's own coverage of the simple path is
  in tests that read downloaded PDFs.
- `TestSubsetting` — `TestCOSIncrement.testSubsetting`, which
  `track/test-backfill` deferred here for `PDType0Font.load`. A subsetted font
  added to an existing document, saved incrementally: the subset is only built
  at save time, so an incremental save that skipped the subsetter would write a
  font dictionary with no `/FontFile2`. Removing the `subsetDesignatedFonts`
  call from `SaveIncremental` makes it fail.
- `TestEnsureFontResourcesEmbedsTheReplacement` — the slice 8 hole this branch
  closed, above. The Java class that covers it downloads its PDFs.

Two cases compare bytes against the running Java rather than against the port's
own reader. `TestSubsetBytesMatchJava` drives `TTFSubsetter` from Java exactly as
`TrueTypeEmbedder.subset` drives it — the same ten tables, the same four
`forceInvisible` calls, the same `getTag` — and gets the same 30 glyphs, the
same map hash 21410, the same tag `AALHKC+`, the same 8332 bytes and the same
SHA-256. `TestEmbeddedFontMatchesJava` writes the two documents
`validateCIDFontType2` writes and compares every entry: `/BaseFont`,
`/FontFile2` and its `/Length1`, `/W`, `/CIDToGIDMap` in both forms, `/CIDSet`,
`/ToUnicode`. All match.

Each function this branch wrote, and the test that covers it:

| Function | The test that covers it |
| --- | --- |
| `newTrueTypeEmbedder`, `createFontDescriptor` | `TestSimpleTrueTypeFontEmbedding` asserts the flags and the widths |
| `isEmbeddingPermitted`, `IsEmbeddingPermittedForFsType` | `TestIsEmbeddingPermittedMultipleVersions`, eight fsType values |
| `subsetTag` | `TestSubsetTagMatchesJava` — Java's tag for an ordinary map, and JAVA-BUGS 74 for the pathological one |
| `javaRound`, `javaRoundLong` | `TestJavaRoundHalfUp` |
| `Subset`, `buildSubset`, `buildFontFile2` | `TestSubsetBytesMatchJava`, `TestEmbeddedFontMatchesJava` |
| `AddToSubset`, `SubsetCodePoints`, `buildToUnicodeCMap` | `TestEmbeddedFontMatchesJava` compares `/ToUnicode` byte for byte |
| `AddGlyphIds`, `AddGlyphsToSubset` | `TestAddGlyphsToSubsetKeepsAGlyphNothingDrew` |
| `addNameTag` | `TestSubsetWritesTheEntriesTheSpecificationAsksFor`, and the tagged `/BaseFont` in `TestEmbeddedFontMatchesJava` |
| `buildCIDToGIDMap` | `TestSubsetKeepsTheGlyphsThatWereAskedFor`, which fails when the map is written off by one |
| `buildCIDSet` | `TestSubsetWritesTheEntriesTheSpecificationAsksFor`, `TestEmbeddedFontMatchesJava` |
| `buildWidths` (both overloads), `getWidths`, `unitsScaling` | `TestWidthArraysMatchJava`, `TestEmbeddedFontMatchesJava` |
| `buildVerticalHeader` | its warning branch, by the vertical rows of `TestEveryLoadOverload` |
| `getVerticalMetrics` | `TestWidthArraysMatchJava` — the only thing that runs it |
| `createCIDFont`, `toCIDSystemInfo`, `CIDFont`, `sortedCIDs` | `TestCIDFontType2`, `TestEmbeddedFontMatchesJava` |
| `NeedsSubset`, `WillBeSubset`, the three panics | `TestSubsettingDisabledPanics`, `TestAWholeFontWritesIdentityCIDToGIDMap` |
| the thirteen `load` factories | `TestEveryLoadOverload` |
| `RegisterTrueTypeFontForClosing`, `PDDocument.Close` | `TestClosingTheDocumentClosesTheFontItRegistered` |
| `ensureFontResources` | `TestEnsureFontResourcesEmbedsTheReplacement` |
| `subsetDesignatedFonts` on the incremental path | `TestSubsetting` |

### Still open

- **`buildVerticalMetrics`, `buildVerticalMetricsOfSubset` and the `/DW2` branch
  of `buildVerticalHeader` cannot be reached through a document.** Every `.ttf`
  and `.otf` in this repository was parsed and asked for a `vhea` table; none
  has one, which is why the Java cases that write vertical metrics download
  `ipag.ttf`. `getVerticalMetrics` is driven directly instead; the two builders
  above it that read `vhea` and `vmtx` are read against the Java and not run.
- **`checkForCidGidIdentity` is written but not run.** It needs an OpenType font
  with a CID-keyed CFF charset. The one `.otf` here, `FoglihtenNo07.otf`, is
  CFF but name-keyed, so the method returns at its second gate.
- **`isSubsettingPermitted` answering false is not reached.** It needs a font
  whose OS/2 `fsType` has the no-subsetting bit; none here has it, and the
  branch cannot be reached without editing a font file, which is a test resource
  and so out of bounds.
- **Six of `TestFontEmbedding`'s seventeen cases** are not ported, each because
  it reads a font the Maven build downloads. Listed above.

## Track `tools` — the command-line utilities

Branch `track/tools`, one of the four the survey found. The Go package is
`go/tools` and the single binary `go/cmd/pdfbox`. It built 17 of Java's 26 main
files, the dispatcher among them, and left nine; the three tracks after it took
eight of those nine.

### A0 — the two decisions, taken before any code

**The flag parser is the standard library's `flag`.** picocli is annotation
driven and has no Go equivalent worth transliterating, so this was always going
to be a substitution; what settled which one is the shape of the flags. picocli
here declares long options with a **single** dash — `-alwaysNext`, `-encoding`,
`-startPage`, `-rotationMagic` — with only a handful of `-i`/`--input` pairs.
Go's `flag` reproduces that exactly: it treats `-name` and `--name` alike and
takes both `-name value` and `-name=value`. A POSIX parser would not: `pflag`
and `cobra` read `-addFileName` as a cluster of ten single-letter flags. It also
needs no new dependency.

What picocli gives and `flag` does not is written once in `command.go`:

| picocli | here |
| --- | --- |
| `mixinStandardHelpOptions = true` | a marker a command embeds, adding `-h`/`--help` and `-V`/`--version` |
| `usageHelp = true` on a command's own `-h` | a second marker, adding only the first pair |
| `required = true` | `requireSet`, checked before `call()` |
| `@Parameters` | `setPositional`, which only `WriteDecodedDoc` needs |
| an option that repeats | a `flag.Value` that appends |
| `subcommandsRepeatable = true` | the dispatcher splits the argument list at each name |
| `ExitCode.OK`/`SOFTWARE`/`USAGE` | 0, 1, 2, with a `recover` turning a panic into 1 |

**The package is `go/tools`, the binary `go/cmd/pdfbox`.** Every Java module
already maps one for one — `pdfbox` to `go/pdfbox`, `fontbox` to `go/fontbox`,
`xmpbox` to `go/xmpbox` — so `tools` to `go/tools` continues it, and the
commands stay a library that a test can call. The one `main` goes where the
empty `cmd/` placeholder had reserved it, which is Go's own convention.

**A command keeps picocli's shape**: a struct with its options as fields, a
`Call` answering the exit code, and `Execute` in place of `CommandLine.execute`.
The two streams are passed in rather than taken from `os`, which is what makes
the exit code and the output testable.

### `tools` cannot be run here

picocli is not in the local Maven cache, so the CLI layer is ported from the
source alone and the assertion values that *are* Java's come from the four
ported test classes. Two of picocli's three exit codes are therefore taken from
its documented `CommandLine.ExitCode` rather than measured; the third, 0, the
Java tests pin down with `assertEquals(0, exitCode)` in `TestExtractText` and
`TestTextToPdf`.

### What was built

| Java | Subcommand | Notes |
| --- | --- | --- |
| `Version` | `version` | and `pdfbox/util/Version`, which was unported |
| `PDFBox` | — | the dispatcher, with repeatable subcommands |
| `DecompressObjectstreams` | — | not a subcommand in Java either |
| `WriteDecodedDoc` | `decode` | the only command with positional `@Parameters` |
| `ExtractText` | `export:text` | the largest: 17 options, embedded PDFs, `-rotationMagic` |
| `PDFText2HTML` | — | extends `PDFTextStripper` |
| `PDFText2Markdown` | — | extends `PDFTextStripper` |
| `Decrypt` | `decrypt` | |
| `Encrypt` | `encrypt` | minus `-certFile`, below |
| `PDFSplit` | `split` | |
| `TextToPDF` | `fromtext` | |
| `ExportFDF` | `export:fdf` | |
| `ExportXFDF` | `export:xfdf` | JAVA-BUGS 75 |
| `ImportFDF` | `import:fdf` | |
| `ImportXFDF` | `import:xfdf` | JAVA-BUGS 76 |
| `ExtractXMP` | `export:xmp` | |
| `ImageToPDF` | `fromimage` | no raster in it after all |

### The flag and exit-code surface

Each command's `Flags` declares the names in the order the Java declares them,
with Java's description text, so the two can be read side by side. The two
commands that are **not** `mixinStandardHelpOptions` have a case each proving
they refuse `-V`. Exit codes are 0 from a command that ran, 1 from one that
reported a problem, 2 from arguments that did not parse, 4 from an
`IOException`, and `ExitCode.SOFTWARE` from a panic. Every error goes to stderr,
and four cases assert that stdout is empty on the failing path.

### Deferrals, and which branch closed each

`NotBuiltCommands` in `go/tools/notbuilt.go` is the list of what is not built
and what each waits for, the dispatcher's help prints it, and
`TestSubcommandNamesAreJavas` checks that every name `PDFBox.main` registers is
either built or recorded — so that list and this file cannot drift apart.

Nine were left here. Seven were recorded as waiting for a raster: `PDFToImage`,
`PrintPDF`, `ExtractImages` and the four `tools/imageio` helpers. Five of the
seven were not waiting for one — `ExtractImages` never imports `rendering`, and
neither do the four `tools/imageio` classes, because writing an image out and
rendering a page are different jobs — and `track/imageio` took them, making the
`javax.imageio` substitution there rather than in `track/raster`; see the
`imageio` section at the end of this file. `track/raster` took `PDFToImage`. Two
waited for `multipdf`: `PDFMerger` needs `PDFMergerUtility` and `OverlayPDF`
needs `Overlay`, which slice 7 deferred to slice 8, slice 8 never took, and the
coverage survey counted as done; `track/multipdf` ported all three and built
both commands.

**`Encrypt -certFile` was not built here.** Public key encryption needs an X.509
certificate and the CMS enveloping around it, and the port's
`PublicKeySecurityHandler` reported at the time that its encryption half was not
ported, so the option is refused by name with what it waits for rather than
silently writing a password-encrypted file instead. `track/stale-deferrals`
found that reason had stopped being true and built it, in `go/tools/encrypt.go`.

### What this branch had to add underneath

**`PDFTextStripper` could not be extended.** Java's is designed to be:
`writeText` calls `startDocument` and `endDocument`, `writePage` calls
`startArticle`, `endArticle`, `writeString` and `writeParagraphEnd`, and
`PDFText2HTML` overrides seven of them. Go has no dispatch from a base into an
embedder, so a struct embedding `*PDFTextStripper` would have overridden
**nothing**: every override would compile, never run, and leave the suite green.
The base now holds a `TextStripperOverrides` and calls that, defaulting to
itself — the shape `PDFStreamEngine` already uses. `writeText(PDDocument,
Writer)` and `getText(PDDocument)` came with it; `GetTextOfPages` had stood in
for them while `Loader` was unported.

**`PDDocument.protect` was unported.** Slice 5 ported the policies, the factory
and both handlers, but not the method that applies one, because nothing in the
port encrypted a document it had built. `encrypt` is the first caller.

### Deviations from Java

- **The error text.** Java's commands end in `catch (IOException ioe)` printing
  `"[" + ioe.getClass().getSimpleName() + "]: " + ioe.getMessage()`; Go has no
  class name for an error, so the port prints the error itself and keeps the
  exit code, which is the part a caller acts on. Said at each site.
- **`ExtractText`'s permission refusal.** Java returns 1 from inside a
  try-with-resources; the port raises a sentinel so the two resources are still
  released, and `Call` turns it back into 1.
- **`pdfbox/util/Version`.** Java reads `pdfbox.version` out of a properties
  resource that Maven fills with `${project.version}` at build time. There is no
  Maven here, so the number is a constant with a comment saying where it comes
  from and what has to move with it.
- **picocli's `arity`** has two shapes with no `flag` counterpart, and both are
  said where they are: `-margins` takes its four numbers separated rather than
  spaced, and a repeatable option like `-certFile` or `ImageToPDF`'s `-i` is
  given again rather than followed by a list.

### The Java bugs

`JAVA-BUGS.md` 75, `ExportXFDF` exiting 0 where its twin `ExportFDF` returns 1,
and `JAVA-BUGS.md` 76, `ImportXFDF` on a document with no form. Each has a test
in this package asserting **Java's** answer rather than the right one.

### The tests

Four of the six Java test classes are ported whole:

| Java test | Cases | Go |
| --- | ---: | --- |
| `TestExtractText` | 7 | `tools/extracttext_test.go` |
| `TestPDFText2HTML` | 2 | `tools/pdftext2html_test.go` |
| `TestTextToPdf` | 4 | `tools/texttopdf_test.go` |
| `PDFBoxHeadlessTest`, `PDFBoxNonHeadlessTest` | 4 | `tools/pdfbox_test.go`, rewritten — see below |
| `imageio/TestImageIOUtils` | — | ported by `track/imageio`, minus the rendering: `tools/imageio/imageio_test.go` and its two companions |

`testOverflow` compares two full pages of laid-out Lorem ipsum against Java's
expected strings, and is the only case in the suite that says the line breaking
and the page breaking are right. The two `PDFBox*Test` classes assert the
subcommand list through picocli's `CommandSpec`, which has no counterpart here;
what they are really asserting is that a name a caller types reaches the command
it should, and that is what the ported cases assert instead.

**A2 — the commands with no Java test.** Most of them. The decision was to test
one where what the command adds over the library call underneath is its own: a
default output name, a filter, an exit code, a page-size table, an orientation
rule. `DecompressObjectstreams`'s `-o` default, `WriteDecodedDoc`'s
`-skipImages` and `_unc.pdf` naming, the encrypt/decrypt round trip and the
eight `-can` permissions, `ImageToPDF`'s page sizes, both JAVA-BUGS and the
whole flag surface all have one; the flag plumbing has `command_test.go`, the
dispatcher five cases, and `PDFText2Markdown`'s escaping and bold/italic rule
their own, because they differ from the HTML ones in three ways. Nothing was
written for a command whose body is one library call with no branch of its own.

### Defects found and fixed

- **`escapeMarkdown` split every surrogate pair.** The Markdown escape walks
  UTF-16 units and, unlike the HTML one, writes the character through in its
  default branch, so a supplementary-plane character came back as two U+FFFD.
  `escapeMarkdown` and `markdownFontState` now keep a `utf16Buffer` and decode
  once at the end, as Java's one `StringBuilder` does;
  `TestMarkdownKeepsASurrogatePair` uses U+1F600.
- **`PDDocument.protect` prepared the handler, and Java's does not.** Java's
  installs the handler and stops, and `COSWriter` prepares on every save; doing
  both ran the password hashing twice and regenerated the revision 6 keys and
  salts. `TestProtectDoesNotPrepareTheHandler`.
- **`-lineSpacing` accepted zero and negatives.** Java routes the parsed value
  through `setLineSpacing`, the one setter of that class that validates, which
  throws `IllegalArgumentException` for anything `<= 0`; the port set the field
  directly. It now panics as the unchecked exception, and `Execute` answers
  `ExitCode.SOFTWARE` as picocli does.
- **A 16 MiB cap on a line.** `bufio.Scanner` has a maximum token size and
  `BufferedReader.readLine` has none, so a longer line failed the conversion
  instead of being wrapped to the page width; `bufio.ScanLines` also leaves a
  lone `\r` inside a line. The port reads lines itself, in `javaLineReader`.
- **The dispatcher split on a subcommand name wherever it appeared.** picocli
  knows each option's arity, so `pdfbox decrypt -i version` gives `-i` the file
  called `version`, and `pdfbox help decrypt`, which the footer advertises,
  printed the global help and then ran `decrypt` with no arguments. `split` now
  asks the command's own flag set which options take a separate value, and
  treats the one argument after `help` as its parameter.

### Still open

- `PrintPDF`, the one command of the nine still not built;
  `go/tools/notbuilt.go` carries what it waits for.

## Track `pdfbox-layout` — A0, and what was built after it

Branch `track/pdfbox-layout`, the last of the four the survey found. Its A0 was
that the backend could not be chosen, because three substitutions have to be
decided together and two of them belong to other work. One of the three turned
out to be work rather than a wall, and the branch went on to write the bidi
algorithm, a GPOS reader and a shaper, and to measure them against the Java.

### What the module actually is

7 main files across two Maven modules, and 13 test classes.

| Java | Lines | What it rests on |
| --- | ---: | --- |
| `awt/GlyphLayoutProcessorAwt` | 284 | `java.awt.Font.layoutGlyphVector`, `GlyphVector` |
| `awt/GlyphLayoutFontLoaderAwt` | 199 | `java.awt.Font.createFont`, `java.awt.font.TextAttribute` |
| `fop/GlyphLayoutProcessorFop` | 354 | `org.apache.fop.fonts.Font`, `GlyphMapping`, `MultiByteFont` |
| `fop/GlyphLayoutFontLoaderFop` | 172 | the same |
| `fop/FopStringTextFragment` | 84 | implements `org.apache.fop.fonts.TextFragment` |
| `examples/GlyphLayoutHelloWorld{AWT,FOP}` | — | examples, which `PLAN.md` puts out of scope |

The core interfaces were already ported by slice 8:
`GlyphLayoutProcessorInterface`, `ContentStreamForGlyphLayoutInterface` and
`GlyphsAndPositions`, in `go/pdfbox/pdmodel/glyphsandpositions.go`. What this
branch was for is the two backends and `AbstractGlyphLayoutProcessor`.

### The decision

**Do not choose a shaper: there is nothing here to port.** PDFBox has none of
its own — it borrows Java's and FOP's, which is why there are two backend
modules and not one implementation. The AWT backend hangs off
`awtFont.layoutGlyphVector(fontRenderContext, chars, 0, chars.length,
localFlags)`, which is GSUB, GPOS and bidi-aware positioning together, answering
per-glyph codes, positions and advances; the FOP backend is the same shape
against FOP's own shaper. Writing `layoutGlyphVector` in Go is a new component,
not a transliteration. What is in the local module cache is
`x/image/font/sfnt`, which gives outlines and advances and does no shaping;
there is no HarfBuzz binding and no `go-text/typesetting` in it. `fontbox` has
GSUB — slice 4 ported it, `fontbox/ttf/gsub` is in the tree — but for
positioning only the old `kern` table's `KerningSubtable`, no GPOS.

So the choice in front of the project is not "which Go shaper", it is "does this
port build one". That is the user's, and it wants the rasteriser decided first:
they are the same family of question, a HarfBuzz binding would answer both, and
`rendering.Backend` already blocks far more.

**A shaper can be measured without a rasteriser.** Every assertion in the 13
test classes that touches the shaping goes through `TestBase.checkRenderIdent`,
which renders both documents and compares pixels, so those classes cannot be
ported as they stand. But the reference PDFs are checked into the repository,
and what they carry is exactly what `showTextUni` wrote — the glyph codes and
the positioning, as `TJ` arrays and `Ts` operators — so a Go shaper can be
compared against the running Java byte for byte with no renderer at all.

**`x/text` cannot supply the embedding levels.** `doBidiSplittingAndReordering`
answers a list of `(text, bidiLevel)` in **visual** order, and both halves are
part of its contract: the level goes to the backend, which reads its parity, and
the order is the order the runs are drawn in. Driving the running Java through
the method by reflection and running `golang.org/x/text/unicode/bidi` over the
same inputs, the run *contents* agree in every case and two things do not:

- **The order is logical, not visual.** Java applies `Bidi.reorderVisually`;
  `x/text`'s `Ordering` hands the runs back in source order.
- **The embedding level is not reachable.** `Ordering` has `NumRuns`, `Run` and
  `Direction`; `Run` has `String`, `Bytes`, `Direction` and `Pos`. There is no
  level accessor anywhere in the package.

Flattening a direction into a level does not work either: the running Java puts
an LTR run at **level 2** inside an RTL paragraph, and `reorderVisually` is
defined over those numbers, so with the levels flattened to 0 and 1 the same
input reorders differently.

### What was built

**`go/javatext/bidi` — UAX#9, written out.** It sits beside `go/awt` and
`go/w3c` for the same reason those do: `java.text.Bidi` is a JDK class the Java
being migrated uses, and Go has no equivalent.
`golang.org/x/text/unicode/bidi` is not one, but its **character data** is
exported, and `bidi.LookupRune(r).Class()` is the Unicode BidiClass property.
The algorithm on top of it is P2/P3, X1–X10, W1–W7, N0–N2, I1–I2, L1 and L2,
including the paired-bracket rule, which the corpus proved is needed rather than
optional.

**`pdmodel/AbstractGlyphLayoutProcessor`** is ported on top of it — the class
the survey counted as the single unported `pdmodel` file. Java's two abstract
methods are function fields, filled in by whatever backend embeds it, the same
shape `TrueTypeEmbedder.buildSubset` took in `track/font-embedding`.

**The bidi is measured, not asserted.** The corpus in
`abstractglyphlayoutprocessor_test.go` was generated by driving the running Java
through `doBidiSplittingAndReordering` by reflection: 65 inputs covering pure
and mixed direction, European and Arabic numbers, brackets nested and
unbalanced, isolates, embeddings, overrides, combining marks, a surrogate pair,
tabs, newlines and segment separators. **63 agree exactly**, on run text,
embedding level and visual order.

**The two backends are not ported by name, and none will be.** Each of
`GlyphLayoutProcessorAwt`, `GlyphLayoutFontLoaderAwt`,
`GlyphLayoutProcessorFop`, `GlyphLayoutFontLoaderFop` and
`FopStringTextFragment` is a shell around a call into a library Go does not
have. What they do is done by `go/pdfbox/glyphlayout` below, and the two
examples `PLAN.md` puts out of scope stay out of scope.

**GPOS — the missing half of a shaper, written from the specification.** GSUB
was ported by slice 4, `fontbox/ttf/gsub/`, from PDFBox's own implementation,
and decides *which* glyph to draw. GPOS decides *where* each glyph goes, and
PDFBox has no reader for it: `OTFParser.readTable` answers a bare `OTLTable` for
the tag, with the comment "todo: this is a stub, a full implementation is
needed", because PDFBox borrows layout from `java.awt.font.TextLayout` or from
Apache FOP — which is why the two `pdfbox-layout-*` modules exist. So
`go/fontbox/ttf/glyphpositioning.go` and its subtables are **written from the
OpenType specification, not ported**, and say so at the top of each file.

| Lookup type | State |
| --- | --- |
| 1 — single adjustment | done |
| 2 — pair adjustment (kerning), formats 1 and 2 | done |
| 4 — mark to base | done |
| 6 — mark to mark | done |
| 9 — extension | done, indirecting to any of the above |
| 3 — cursive attachment | not built; read far enough to skip |
| 5 — mark to ligature | not built; read far enough to skip |
| 7, 8 — contextual and chained contextual | not built; read far enough to skip |

The header — script list, feature list, lookup list, coverage tables — is the
same in GSUB and GPOS. `layoutcommon.go` holds one copy for GPOS to read
through; `GlyphSubstitutionTable` keeps its own, because it is a port of
PDFBox's class and that is how the Java is written.

**How the GPOS reader is checked, with no Java to measure against.** This is the
one piece of the migration with no reference implementation in the repository,
and two things stand in for one.

- **An independent oracle inside the font.** A font that carries both the old
  `kern` table and a GPOS `kern` feature says the same thing twice, and fontbox
  already reads the old one. Over every pair of a 30-character sample alphabet
  in `DejaVuSans.ttf`, **89 of 89 pairs the `kern` table declares agree exactly
  with what the GPOS reader answers**.
- **The table's own meaning.** `A` before `V` comes back with a *negative*
  advance; `II`, which no font kerns, comes back untouched; a feature the caller
  did not ask for does not run; an Arabic fatha after a lam is placed *above*
  it, which is lookup types 4 and 6 working.

All five layout-test fonts parse: `DejaVuSans`, `FiraCode-Regular`,
`Arimo-Regular`, `NotoSansArabic-Regular`, `NotoSansThai-Regular`.

**The backend — `go/pdfbox/glyphlayout`.** This is the substitution the branch
exists for, and it is **not a port**: there is no third Java implementation to
translate. The package is the shaper, assembled from the two halves the port
has, GSUB deciding which glyph and GPOS where. Above it,
`pdmodel.AbstractGlyphLayoutProcessor`, which *is* a port, splits the text into
runs of one direction over `go/javatext/bidi`. Below it, `showTextUni` and
`getStringWidthUni` are close ports: everything after the shaping is arithmetic
that follows the Java line for line, including the unconditional final
`showGlyphsWithPositioning`, which writes an empty `[] TJ` for an empty run as
the Java does. The processor keeps a map of GSUB workers with nothing guarding
it, and carries the Java class's own sentence: use an object of this class only
in one thread.

Two things had to be modelled the way AWT models them rather than the way an
OpenType table states them, and getting either wrong is invisible in a test that
only checks that something was written:

- **A kern reaches the page through the advance, not the placement.** Java
  compares each glyph's laid-out position against the pen plus the *unadjusted*
  advance of the glyph before it — `getGlyphMetrics(i-1).getAdvanceX()` — and
  writes the difference, so a port that reads only the placement writes nothing
  at all for a pure kern. `positionedGlyph` carries both numbers.
- **A mark's anchor is measured from its letter's origin**, which the pen has
  left behind by the time the mark is drawn, and in a right-to-left run has not
  reached yet. `GlyphPosition.AttachedTo` names the glyph a mark hangs off and
  leaves the arithmetic to `resolveAttachments`, which knows the drawing order.
  Folding the pen into the offset inside the subtable put a Thai tone mark two
  letters to the left.

**Right-to-left runs are turned round.** `layoutGlyphVector` is called with
`Font.LAYOUT_RIGHT_TO_LEFT` and answers a vector already in visual order; the
bidi split above only places the run on the line, not the letters inside it. The
port shapes in logical order — a letter takes its form from the letters around
it in the text, not on the page — and reverses at the end.

### How it is measured: the Java's own output

`pdfbox-layout-awt/src/test/resources/pdf/` holds the PDFs the Java tests
compare against, and they are the output of the real AWT backend, checked into
the repository. `go/pdfbox/glyphlayout/testdata/awt-*.txt` is those PDFs read
back — one line per text object, each glyph written as the characters its
ToUnicode gives, with the positioning adjustments and text rises in place. The
tests lay the same pages out with the Go backend and compare. A line that
differs has to be listed as a known deviation with a reason, and **a line listed
there that stops differing fails too**, so neither a new deviation nor a fixed
one can pass unnoticed.

| Java test | Text objects | Agree with AWT |
| --- | ---: | ---: |
| `GlyphLayoutLigaturesAndKerningTest` | 9 | 4 |
| `GlyphLayoutBidiTest` | 2 | 0 |
| `GlyphLayoutSMPTest` | 7 | **7** |
| `GlyphLayoutDin91379Test` | 41 | **40** |

Agreement means every glyph and every number, at a tolerance of 0.02
thousandths of the font size — a fifty-thousandth of an em. The tolerance is not
zero because AWT lays a run out in points at the size asked for and this port in
font design units: a kern the font declares as 9 units comes back out of the
reference PDF as 8.99506.

The four agreeing lines of the ligature test are the four DejaVu lines — plain,
ligatures, kerning, and both — carrying `AVATAR, effective, affiliation, float,
film, affluent`: every ligature AWT formed, the port forms, and every kern AWT
applied, the port applies, to the same value. The seven of the SMP test are the
whole page, every letter on it a surrogate pair and none taken apart. The DIN
91379 page is 41 lines of every letter that can appear in a European name and
then the sequences — a letter with one or two combining marks over it, which is
the widest mark-positioning case there is.

**The reference PDFs are pixel-equal to the Java that renders them, not
byte-equal, and this comparison found where.** Twenty lines of
`GlyphLayoutDIN91379.pdf` end with a space that `LATIN_CHARS_DIN_91379` does not
have: the PDF was rendered from an earlier spelling of the string, and
`checkRenderIdent` never noticed, because a space at the end of a line paints
nothing. Neither side's trailing space is a shaping difference and the
comparison drops it. Nothing was changed in the Java.

### Deviations from Java

- **`supportsFont` is documented as a port and is not one.** Java's is
  `awtFontMap.containsKey(font)` — the AWT backend supports the fonts its own
  loader handed it — and this port has no such loader. It answers on
  `PDType0Font.cmapLookup`, which the constructor that embeds a font program
  sets and the one that reads a font out of a PDF leaves nil, so non-nil is the
  same set of fonts `awtFontMap` holds. It has to be that narrow: the layout
  hands `showTextUni` glyph ids of the font program and `EncodeGlyphID` writes
  each as a two-byte code, which selects the glyph it names only under
  Identity-H over an identity CIDToGIDMap, which is what the embedding
  constructor builds. A font with a predefined CMap, a CIDToGIDMap stream or a
  substitute program would draw an unrelated glyph.
- **`delta` is not the same constant.** Java's is applied to a distance in
  points and this port's `dx` is in thousandths of an em. It cannot come to a
  different answer — every `dx` here is a whole number of design units, four
  orders of magnitude above the threshold — and the comment says so.
- The five shaping deviations from AWT and the two bidi deviations from the JDK,
  both below.

#### Deviations, measured — and the one cause behind nearly all of them

Every deviation but one is a **GSUB** difference, and every GSUB difference but
one has the same cause: **PDFBox's substitution reader implements lookup types
1, 2, 3, 4 and 7, and drops 5 and 6.** `GlyphSubstitutionTable.
readLookupSubtable` says so in a comment — "Other lookup types are not
supported" — and logs each one it throws away. Counting those logs over the
layout fonts:

| Font | Lookups dropped |
| --- | --- |
| `FiraCode-Regular` | 111 of type 6 |
| `NotoSansArabic-Regular` | 7 of type 6, 2 of type 5 |
| `NotoSansThai-Regular` | 5 of type 6, 1 of type 5 |
| `DejaVuSans` | 4 of type 6 |
| `Arimo-Regular` | 2 of type 6 |
| `Lohit-Bengali` | none |

Type 6 is chained contextual substitution: *replace this glyph when it stands
between those glyphs*. It is how a font says almost everything that depends on
what is next to what, so the port asks for the features and the substitutions
are not in the data it was given. Where the glyph run does agree, the
positioning agrees: the Thai line's first eleven glyphs and the Bengali marks
match the AWT reference to the last design unit.

1. **FiraCode's `!=` and `>=`** are drawn with contextual alternates — 111
   type 6 lookups, all dropped. AWT draws the joined forms; the port draws `!`
   and `=`.
2. **Thai contextual forms.** A vowel or tone sign over a tall consonant has a
   lowered variant, and U+0E33 decomposes into U+0E4D and U+0E32. Six dropped
   lookups.
3. **The dotless `j`.** In `j́` the platform puts U+0237 LATIN SMALL LETTER
   DOTLESS J under the accent, so the accent does not land on the dot. Arimo
   spells that as `ccmp`, in two type 6 lookups. It is the only difference on
   the whole DIN 91379 page.
4. **Bengali conjuncts differ** — and this one is not a dropped lookup, because
   Lohit-Bengali is the one layout font whose GSUB the reader reads in full.
   `GsubWorkerForBengali` reorders and substitutes, and the page comes out
   legible: the pre-base vowel moves ahead of its consonant, the conjuncts
   form. It picks a different set of pre-base forms than the platform does, in
   both directions — at one place the port substitutes where AWT does not, at
   another the reverse.
5. **Arabic joining is not applied at all**, and this one is a missing shaper
   rather than a missing lookup type. A letter's initial, medial and final
   forms are the `init`/`medi`/`fina` features, chosen by the Unicode joining
   types of the letters around it, and **PDFBox has no Arabic worker**:
   `GsubWorkerFactory` covers Bengali, Devanagari, Gujarati, Latin and DFLT.
   The port draws the isolated forms. The bidi ordering — which is what
   `GlyphLayoutBidiTest` is named for — is right: the runs are placed by
   `ReorderVisually` and the letters inside each are reversed.

The shape of all five is the same: **the port substitutes exactly what PDFBox
can substitute, and positions exactly what the OpenType specification defines.**

#### The two bidi deviations from the JDK

`TestKnownDeviationsFromTheJDK` pins them rather than hiding them. Both are a
**directional override spanning a whole paragraph** — LRO or RLO, which Unicode
deprecated in favour of the isolates. An override in the middle of a paragraph
agrees exactly: for `abc <RLO>def<PDF> ghi` the running Java answers
`0 0 0 0 | 1 1 1 1 | 0 0 0 0 0`, which is the annex and is what the port gives.
For an override that covers everything the JDK reports the base level for every
character — `0 0 0 0 0 0` where UAX#9 raises to the least greater even — and the
port follows the annex. The consequence for a layout backend is one extra run at
a level of the same parity, and parity is all `GlyphLayoutProcessorAwt` reads
(`bidiLevel % 2 == 0 ? LAYOUT_LEFT_TO_RIGHT : LAYOUT_RIGHT_TO_LEFT`), so the
text lays out the same way.

### Which features the backend asks for, and why

The Java font loader has two switches, `setKerningOn` and `setLigaturesOn`, and
they do not mean "all the shaping" — measuring the reference PDFs says which
features the platform applies regardless:

| Feature | When |
| --- | --- |
| `ccmp`, `calt` | always |
| `liga`, `clig` | with `Ligatures` |
| `kern` (GPOS) | with `Kerning` |
| `mark`, `mkmk`, `abvm`, `blwm` (GPOS) | always |

The evidence for the first row is the FiraCode line written with no options at
all, which carries the contextual forms, and the `j́` of the DIN 91379 page,
which carries the dotless `j`; for the second, the DejaVu line written with no
options has `ffi` in three separate letters where the one written with them has
the ligature. The last row is four features and not two because `mark` and
`mkmk` are the Latin and Arabic spellings of mark positioning and `abvm` and
`blwm` the Indic ones, above-base and below-base: asking only for the first two
leaves a Bengali vowel sign on the baseline, because Lohit-Bengali files its
anchors under the other two.

Java has one GSUB worker per script, each with a feature list fixed in its
class, and no way to ask for some of them and not others. `gsub.
GsubWorkerForFeatures` is that: **not a port**, but not new machinery either --
the same `featureApplier` the ported workers run, over a list the caller
chooses. The script-specific workers are still used unchanged where a script has
one, because they reorder as well as substitute. The script a feature is looked
up under is the font's own GSUB language first and the run's direction second,
so a Bengali font is not asked for its `latn` features and told it has none.

### What the Java tests could not be ported as

Every assertion in the five shared test classes that checks the shaping goes
through `checkRenderIdent`. Those are not ported; the reference comparison above
replaces them. What is ported straight is everything else the Java asserts:

| Java assertion | Go test |
| --- | --- |
| `testMissingGlyph`'s message, character for character | `TestMissingGlyphIsRefused` |
| `assertEquals(f1, f2)`, `f4 < f1`, `f4 < f3` | `TestStringWidthWithAndWithoutKerning` |
| `assertEquals(1, doc.getNumberOfPages())` | implied by the DIN comparison, which reads page 0 of a one-page document |

`GlyphLayoutDin91379Test` **is** ported, as the reference comparison above: its
own assertion is `assertEquals(1, doc.getNumberOfPages())`, and the extracted
text it writes to a file carries a TODO saying the comparison is "Not yet
correct as of 4.7.2026", so there is nothing else in it to port.

`GlyphLayoutDin91379FormTest` is not: it needs `PDAcroForm` field appearances
driven by a layout processor, which is slice 8's `generateAppearance` path over
a backend that did not exist when this branch ran; `track/raster` wrote one
since. The two hello-world classes are examples, which `PLAN.md` puts out of
scope.

### What the GPOS reader does not do

Beyond the lookup types in the table above:

- **Lookup flags are not applied.** `ignoreBaseGlyphs`, `ignoreLigatures`,
  `ignoreMarks`, the mark attachment type in the flag's high byte and the mark
  filtering set all say which glyphs a lookup should skip while matching, and
  none of them is honoured: a kern lookup that asks to ignore marks still sees
  them, so a pair with a mark between it does not kern. Applying them needs the
  GDEF table, which neither PDFBox nor this port reads. `useMarkFilteringSet` is
  the one flag the reader looks at, and only to step over the extra field it
  adds to the lookup header.
- **Device tables are read and dropped.** They carry per-pixel-size corrections
  for a hinted rasteriser, and a PDF is laid out in font design units at no
  particular size.

### Defects found and fixed

- **A surrogate pair was torn in half.** The second unit was hidden from UAX#9
  by giving it class BN, the class of a character X9 removes, and
  `resetSeparators` then gave it the level of the character that follows it, so
  `buildRuns` split the pair and each half decoded to U+FFFD.
  `restoreSurrogatePairs` gives the second unit the level of the first. The
  running JDK never puts a run boundary inside a pair, because it works in code
  points; the eight texts it was measured over are in the corpus.
- **`Bidi.requiresBidi` counts an Arabic-Indic number**, and the port had
  skipped the analysis: Latin text followed by Arabic-Indic digits comes back
  from the running Java as two runs at levels 0 and 2.
- **`new Bidi(String, int)` splits at a paragraph separator**, giving each
  paragraph its own P2 and P3. Hebrew, a newline and then Latin is two
  paragraphs at levels 1 and 0; the port had treated it as one.
- **A character rule X9 removes takes the level of the one after it.** For an
  RLE, `abc` and a PDF the running Java puts the RLE at the level of the text it
  opened rather than the level it was pushed from.
- **Mark-to-base looked back for its letter only over the marks its own subtable
  covers**, and a letter can carry two marks of different classes covered by
  different subtables: `C̨̆` — C, ogonek, breve — lost the breve, because the
  scan stopped on the ogonek. The table now collects every mark glyph any of its
  attachment subtables covers and hands the set to all of them, which is what
  GDEF would say if the port read GDEF.
- **A NULL anchor was read as an anchor at the origin.** A zero offset in a mark
  attachment subtable means this base takes no mark of that class; reading it as
  (0, 0) attached the mark anyway, at the far left of the letter on the
  baseline. Of `NotoSansArabic-Regular`'s 4665 base anchors **3431 are NULL**
  and 6 are genuinely at the origin.
- **A required feature was never run.** A language system's
  `RequiredFeatureIndex` names a feature that applies whether or not the caller
  asked for it. The ported GSUB reader has always handled it, in
  `featureRecords`; the GPOS reader now does too.
- **Every language system was being applied at once.** A script's named language
  systems are alternatives to its default, and unioning them sets the text in a
  language nobody asked for. `Position` has no language argument, so it takes
  the default, which is what a run that names no language gets.
- **Every alias of a script was being applied at once**, `bng2` and `beng`
  together. The tags handed down are now a preference order and the first the
  font carries wins, with the script the font's own GSUB data selected —
  `ActiveScriptName()` — at the head of it.
- **A lookup walked the run once per subtable rather than once per lookup**,
  which lets two subtables of one lookup both adjust the same glyph where the
  specification takes the first that applies. It changed nothing measurable in
  the five test fonts.
- **`ValueRecord` zero was not zero.** `IsZero` gained a field when attachment
  moved onto `GlyphPosition`, so a value record built without setting it
  answered false, which made pair adjustment consume two glyphs where it should
  consume one and dropped every second kern in `AVATAR`. The reference
  comparison caught it; the kerning test did not, because there were still kerns
  in the stream.
- **A damaged positioning table was reported as no positioning table.**
  `position` treated an error from `GPOS()` the same as a font without the
  table, which would drop every kern and every mark and say nothing; it now
  returns the error. The path cannot be reached today, because
  `Parser.parseTables` reads every table of the directory when it parses a font,
  and `TestDamagedGPOSIsReported` asserts it there so that the case moves if the
  reading ever becomes lazy.

Neither the language systems, the script aliases nor the NULL anchors changed a
single glyph or number on the four reference pages.

### Still open

- The five shaping deviations. Closing them means adding contextual substitution
  to a ported reader that deliberately does without it, and writing an Arabic
  shaper PDFBox has never had; both are larger than this branch and are the
  user's call.
- GPOS lookup types 3, 5, 7 and 8; the lookup flags, which need a GDEF reader
  neither PDFBox nor this port has; and the device tables.

## What is left, and the four branches that claim it

Every slice and every earlier track is merged. What `PLAN.md` counts in scope
and the port has not got is below, and each item has a branch. The grouping and
the critical path are in [`BRANCHING.md`](BRANCHING.md); the order was taken
from the imports of the five commands that were missing, and the contents from
the audit recorded at the end of this file.

| Branch | Java | Depends on | Unblocks |
| --- | ---: | --- | --- |
| `track/stale-deferrals` | 3 test classes, 3 methods | nothing | **done** — article beads, `sh`, public-key encryption |
| `track/imageio` | 5 | nothing | **done** — `export:images` |
| `track/multipdf` | 5 + 1 test | nothing | **done** — `merge`, `overlay`, and `Splitter`'s other half |
| `track/raster` | 27 | `track/imageio`, for one task | **done** — `render` and every deferred pixel comparison. Not `print`: see its section |
| `track/java-bug-fixes` | **none — it is not a port** | nothing, and goes last | the entries of `JAVA-BUGS.md` |

`ExtractImages` never imports `rendering` — it walks the content stream with
`PDFGraphicsStreamEngine` and writes what it finds, and `PDImage.Image()`
already answers a Go `image.Image` — so nothing about writing an image out waits
for a rasteriser. `PDFToImage` imports both `rendering` and `imageio`, and that
single command was the only edge between the two branches.

### Still open

- `track/java-bug-fixes`, which is not a port: it is the entries of
  [`JAVA-BUGS.md`](JAVA-BUGS.md).
- `print`, which `track/raster` did not take; see its section.

## The audit that found what the survey missed

The 891-class survey recorded above missed `multipdf` entirely, and its
subtotals do not add up to its own headings (28 and 45 under headings of 24 and
42). This is the audit that replaces it: the method, the commands to re-run it,
and what it found.

### The method: three buckets, not one

A class is **not** ported because its name appears. Sort every Java class into
one of three buckets and read the last two by hand.

```sh
# 1. every in-scope Java main class, as "SimpleName<TAB>fqn"
for m in io fontbox xmpbox pdfbox pdfbox-layout-awt pdfbox-layout-fop tools; do
  find $m/src/main/java -name '*.java'
done | sed 's|^[a-z0-9-]*/src/main/java/||; s|\.java$||; s|/|.|g' \
     | awk -F. '{print $NF"\t"$0}' | sort > java_classes.tsv

# 2. strong evidence: a Go declaration of that name, or an explicit "Port of X"
cd go
rg -o --no-filename '^(type|func) +([A-Z][A-Za-z0-9_]*)' -r '$2' --glob '*.go' . \
  | sort -u > go_decls.txt
rg -o --no-filename '[Pp]ort of ([a-zA-Z0-9_.]*[A-Z][A-Za-z0-9_]*)' -r '$1' --glob '*.go' . \
  | sed 's/.*\.//' | sort -u >> go_decls.txt

# 3. of what is left, split by whether the name reaches any line that is not a
#    comment. Comment-only is the bucket the last survey got wrong.
```

Over 891 classes that gives **100 with no strong evidence**, of which 27 appear
nowhere in the Go tree at all and 51 appear **only in comments**. Every one of
the 51 has to be read: most are ports under a Go name the matcher could not see
— `COSBase` is `cos.Base`, `Hex` is `cos.ParseHexString` — and a few are the
real gaps, indistinguishable from the rest without reading.

Run the same three buckets over `src/test/java` for the 237 test classes. That
pass is what found `TestPDDocument`.

### A class-level audit cannot see a method-level gap, and that is where they hide

`multipdf` was a whole package and the survey still lost it. The gaps below are
smaller than a class and no class-level pass of any quality would have found
them:

```sh
rg -n '//.*\b(is not ported|has not reached|not built|waits for|deferred)\b' \
   --glob '*.go' --glob '!*_test.go' go/
rg -n 'errors\.New\(|panic\(' --glob '*.go' --glob '!*_test.go' go/ \
   | rg -i 'not ported|not built|no backend|not implemented'
```

Then — and this is the step that pays — **check whether each stated reason is
still true.** A deferral records what was missing on the day it was written, and
nothing goes back to look when that thing lands. Four of the ones this port
carries name a dependency that has since been ported.

### What it found

| Found | Where | Now claimed by |
| --- | --- | --- |
| `PDPatternContentStream` unported | `pdmodel` | `track/raster` — **done** |
| `BlendComposite` unported | `graphics/blend` | `track/raster` — **done** |
| `PDFTextStripper.fillBeadRectangles` disabled — `PDThreadBead` is ported now | `text` | `track/stale-deferrals` |
| `PDAbstractContentStream.shadingFill` missing — `PDShading` is ported now | `pdmodel` | `track/stale-deferrals` |
| `PublicKeySecurityHandler` cannot encrypt — the recorded reason is "slice 7", which merged | `encryption` | `track/stale-deferrals`, and its A0 |
| `COSWriterCompressionPoolTest`, `COSDocumentCompressionTest` — every blocker they name is ported | `pdfwriter` | `track/stale-deferrals` |
| `TestPDDocument`, 6 cases — **recorded nowhere at all** | `pdmodel` | `track/stale-deferrals` |
| `PDFCloneUtilityTest` — 2 of its 3 blockers are ported, the third is `PDFMergerUtility` | `multipdf` | `track/multipdf` |
| `ContentStreamWriterTest` | `pdfwriter` | `track/raster` — **done** |
| `contentstream/operator/text` says `Tj`, `TJ`, `'` and `"` are absent; they were ported by slice 3 | doc comment | `track/stale-deferrals` |

The `tools` count — 18 of 26, in this file and in `go/tools/notbuilt.go` — was
wrong the same way: counted against that file's own list, 17 were ported, the
dispatcher among them, and 9 were not. Both were corrected, and the nine are
what the branches below divide between them.

## `track/stale-deferrals` — the deferrals whose reason had stopped being true

Everything below was deferred by a slice that named a dependency, the dependency
landed, and nothing came back. The branch closed them and wrote what each one
was waiting for.

| What | The reason it recorded | What was true by the time it was read |
| --- | --- | --- |
| `PDFTextStripper.fillBeadRectangles` | "PDThreadBead is a slice this port has not reached" | slice 8 ported `PDThreadBead` and `PDPage.ThreadBeads` |
| `PDAbstractContentStream.shadingFill` | "it names PDShading ... and PDResources cannot add one either" | slice 9 ported `PDShading`; `PDResources.AddShading` was already there |
| `PublicKeySecurityHandler.PrepareDocumentForEncryption` | "needs a CMS encoder, which is slice 7" | slice 7 merged, and `rc2.go` plus the CMS structures came in with the decrypting side |
| `tools` `-certFile` | "PublicKeySecurityHandler's encryption half is not ported" | the line above |
| `COSWriterCompressionPoolTest` | needs `PDDocumentOutline`, `PDOutlineItem` | both ported |
| `COSDocumentCompressionTest` | needs `PDAcroForm`, `PDComplexFileSpecification`, `PDPageContentStream`, `PDCheckBox`, `protect` | all five ported |
| `contentstream/operator/text` | "Tj, TJ, ' and " ... need PDFont, which this port has not reached" | slice 3 ported PDFont **and all four operators**; only the sentence was left |
| `TestPDDocument`, 6 cases | **nothing — recorded nowhere** | it was missed, not deferred |

### What each one was

- **Article beads were disabled, and everything above them worked.**
  `fillBeadRectangles` set the list to nil, so every glyph on every page fell
  into one article, while `processTextPosition` divided glyphs by bead
  rectangle, `charactersByArticle` kept a list per division and `writePage`
  walked them in order — over an empty list. Ported in
  `go/pdfbox/text/pdftextstripper.go`; `TestTextIsSortedByArticleBeads` in
  `go/pdfbox/text/beads_test.go` puts two beads on a page and writes the text in
  the other order. The method mutates the rectangle it is handed, which is safe
  because `PDRectangle(COSArray)` copies into a fresh `COSArray` in Java and in
  the port, so extracting text twice gives the same answer; the case asserts it.
- **`sh` could not be written.** `PDResources.AddShading` existed and nothing
  called it. `ShadingFill` is nine lines and needs no rasteriser: what a reader
  does with a shading later is the renderer's business, not the writer's.
  `go/pdfbox/pdmodel/pdabstractcontentstream.go`;
  `go/pdfbox/pdmodel/shadingfill_test.go`.
- **Public-key encryption was half here.** `rc2.go` implements `cipher.Block`,
  so it encrypts as well as it decrypts, and `cms.go` declares every ASN.1
  structure an enveloped-data blob is made of, because it reads one. What was
  missing was the direction, and `go/pdfbox/pdmodel/encryption/cmsencode.go` is
  it: RC2-CBC content encryption under a fresh key, that key wrapped to each
  certificate with RSA PKCS#1 v1.5, and the whole thing wrapped in a
  ContentInfo. `TestPublicKeyEnvelopeRoundTrips` seals a seed and opens it
  again, and `TestPublicKeyVersionNumber` walks all four arms of
  `computeVersionNumber`, both in `publickeyencrypt_test.go`.
- **Three more cases of `TestPublicKeyEncryption`.** `testProtection`,
  `testProtectionError` and `testMultipleRecipients` were deferred on the writer
  of slice 7 and the CMS encoder, which this branch wrote. All three are in
  `go/pdfbox/pdmodel/encryption/publickeyprotect_test.go`, at the three key
  lengths the Java parameterises over: a real document protected, saved and
  opened again from a keystore this port did not write; the wrong certificate
  refused with the message the Java asserts, `serial-#: rid 2 vs. cert 3`; and
  two recipients each getting their own permissions out of one file.
- **The compression tests** are about what a document still says after it has
  been written out compressed — the same pages, the same thirteen fields, the
  same attachment at the same length. Four of `COSDocumentCompressionTest`'s
  five cases came in with this branch; `testPDFBox5927` needed a PDF the Maven
  build downloads and is ported since, in `pdfwriter/compression_test.go` — see
  "What the fetch unblocked".
- **A Java bug on the way past.** `Encrypt` builds one `PublicKeyRecipient`
  outside its loop and adds the same object once per `-certFile`; ported as
  written, **JAVA-BUGS.md 79**.

### Deviations from Java

- **The ContentInfo wrapper is marshalled by hand.** `encoding/asn1` writes a
  `RawValue`'s `FullBytes` verbatim and ignores the field's own tagging
  parameters, so a ContentInfo whose content is `[0] EXPLICIT` comes out
  untagged and nothing reads it back. Commented in `cmsencode.go`.
- **No trailing `0xFF` bytes on the encrypting side.** Java's encrypting path
  does not append the four `0xFF` bytes for unencrypted metadata that its
  decrypting path handles, and neither does the port. Faithful, and written down
  because it looks like an omission.
- **A half-built recipient panics rather than defaulting.**
  `computeRecipientsField` reads `recipient.getPermission().getPermissionBytesForPublicKey()`
  and `recipient.getX509()` with no null check and neither field has a default,
  so Java throws NullPointerException. Failing gracefully or defaulting the
  permissions would make the port accept a policy Java refuses, so the
  convention is pinned instead: `TestRecipientWithoutPermissionPanics` and
  `TestRecipientWithoutCertificatePanics`.
- **`COSDocumentCompressionTest.testAlteredDoc`'s 43 bytes are not asserted.**
  That number is the new page's content stream `/Length`, the stream after it
  was deflated. Over the identical 35 bytes of content `java.util.zip.Deflater`
  answers 43 and Go's `compress/zlib` answers 47, at every compression level
  from 1 to 9; both are valid Flate streams and both inflate to the same bytes.
  The case asserts the content instead, which is what the number stands for.

### Defects found and fixed

- **`computeRecipientInfo` wrote the key algorithm's parameters absent.** It
  takes the whole `AlgorithmIdentifier` off the certificate's
  `SubjectPublicKeyInfo`, which for an RSA key carries an explicit ASN.1 NULL;
  RFC 3370 section 4.2.1 requires it, so a strict reader is entitled to refuse
  what was written. Fixed in `cmsencode.go` and checked by dumping the DER:
  `0500` follows the rsaEncryption OID.

### Still open

- The pixel half of `TestImageIOUtils`, which waits on a rasteriser.

## Track `imageio` — what writes an image out

`ImageIOUtil` is a shell around `javax.imageio`: it asks a registry for a writer
by format name, takes an `ImageWriteParam` and an `IIOMetadata` tree off it,
sets a compression type by string, and edits the metadata as a DOM. Go has none
of that. So this is a **substitution, like `glyphlayout`** -- the calls are
ported, the machinery under them is not, and what it produces is compared with
what Java produces rather than translated from Java's source. Six formats reach
`writeImage` from `PDFToImage` and `ExtractImages`, and the Java test writes all
six.

| Format | What writes it | Resolution |
| --- | --- | --- |
| PNG | `image/png` | a `pHYs` chunk written in |
| JPEG | `image/jpeg` | the JFIF APP0 density patched |
| GIF | `image/gif` | none, and Java writes none either -- "no META data possible for GIF" |
| BMP | written here, ~60 lines | the header's pixels-per-metre fields |
| WBMP | written here, ~20 lines | none, and Java writes none |
| TIFF | written here | the XResolution and YResolution tags |
| JPEG 2000 | **nothing** | — |

### The decision

**TIFF is written uncompressed, and Java compresses it.**
`TIFFUtil.setCompressionType` picks CCITT T.6 for a 1-bit bitonal image and LZW
for everything else. Neither is in Go's standard library and one of them nearly
is: `compress/lzw` implements the LZW of GIF and PDF, and **TIFF's variant
increments the code width one code early**, so feeding a TIFF reader the output
of `compress/lzw` produces a file that some readers accept and others reject,
which is worse than not compressing. CCITT T.6 is a Group 4 fax encoder and is a
piece of work in its own right. So the port writes baseline uncompressed strips:
correct, readable everywhere, and larger. `ExtractImages` converts a bitonal
image to 1-bit-per-pixel before writing it *so that* Java's G4 kicks in, and the
port keeps the conversion -- the file is still 1 bit per pixel, it is simply not
compressed.

**JPEG 2000 cannot be written at all.** `ExtractImages` writes a `.jp2` two
ways: copying the embedded stream out untouched, which needs no encoder and is
ported, and converting an image to JPEG 2000 for a colour space that is not grey
or RGB, which needs one. Go has no JPEG 2000 encoder and this port has no JPX
decoder either -- `PDJPXColorSpace` is recorded as a deliberate non-port for the
same reason. Java has none either without `jai-imageio-jpeg2000`, a test-scope
dependency, which is why writing a `.jp2` is something `pdfbox-tools` cannot do
for a user.

### What the Java actually writes, measured

The four classes of `tools/imageio` compile against nothing but `log4j-api` --
no picocli, no PDFBox core -- so unlike the rest of `tools` they can be run
here. They were, over the same two images the Go test builds: an 8x6
`TYPE_INT_RGB` and an 8x6 `TYPE_BYTE_BINARY`. Everything below came out of that
run and is asserted in `go/tools/imageio/javavalues_test.go`. The JDK has had a
TIFF writer since 9, so the compression figures were measured without the JAI
jars.

| What | The Java | The port |
| --- | --- | --- |
| PNG `pHYs` | 1417, 2835, 11811 pixels per metre at 36, 72, 300 dpi | the same |
| PNG chunk order | IHDR, pHYs, IDAT, IEND | the same |
| JPEG JFIF APP0 | `ffe0 0010 "JFIF\0" 01 02 01 0024 0024 0000` | the same eighteen bytes |
| WBMP | `00 00 08 06 aa 55 aa 55 aa 55` | the same ten bytes |
| TIFF tags | 13 entries; `Software` = "PDFBOX", `RowsPerStrip` = the height, `BitsPerSample` one short per sample | the same 13 |
| TIFF bitonal | `PhotometricInterpretation` 0, WhiteIsZero | the same, with the bits that way round |
| TIFF byte order | big-endian, "MM" | little-endian, "II", which the format allows and the first two bytes say |
| TIFF compression | 4 (CCITT T.6) bitonal, 5 (LZW) otherwise | 1, none — the deviation above |
| BMP resolution | **zero** on a plain JDK — see JAVA-BUGS.md 81 | the pixels per metre |

**The JFIF version is 1.02, not 1.01.** `JPEGUtil.updateMetadata` sets
`majorVersion` 1 and `minorVersion` 2 on the `app0JFIF` node.

**A bitonal TIFF is WhiteIsZero**, which `TIFFUtil.updateMetadata` sets tag 262
to for a one-bit image and nothing else, "because of bug in Windows XP
preview". That decides what the bits mean, so the port inverts them to match: a
clear bit is white. The Java's file round-trips through `ImageIO.read` to the
image it was given, and so does this one.

### What is not ported from `ImageIOUtil`

- **The iCCP chunk.** Java attaches an ICC profile to a PNG when the image's
  colour space is an `ICC_ColorSpace` that is neither sRGB nor the built-in grey
  -- `hasICCProfile`, and `getAsDeflatedBytes` beside it. A Go `image.Image`
  carries no colour space at all: `PDImage.Image()` answers `image.RGBA` or
  `image.Gray`, the profile having been applied on the way. There is nothing to
  attach, and there will be nothing to attach until an image type that carries a
  profile exists. Deferred on absence, not on difficulty.
- **The `compressionType` parameter.** The six-argument `writeImage` takes a
  `javax.imageio` compression name -- "LZW", "JPEG", "None", or null for
  uncompressed -- and hands it to `ImageWriteParam.setCompressionType`. Only the
  TIFF writer has more than one, and this port's TIFF writer has one. The
  five-argument overloads, which are what `ExtractImages` and `PDFToImage` call,
  are ported in full.
- **`MetaUtil.debugLogMetadata`.** It serialises a metadata tree to XML when
  debug logging is on. There is no metadata tree.

Every other method of `ImageIOUtil`, `TIFFUtil`, `JPEGUtil` and `MetaUtil` is
ported, including the two exit codes -- 4 for the `IOException` catch and 1 for
the permission refusal, both literals in the Java rather than picocli constants.

### `-noColorConvert` reaches only one colour space

`ExtractImages` with `-noColorConvert` asks for `pdImage.getRawImage()` and
writes that: a PNG, or a TIFF where the raster has more than three bands,
"that's likely CMYK". `getRawImage` is `getColorSpace().toRawImage(raster)`,
and **six of the port's eight `ToRawImage` implementations answer nil**:

| Colour space | Java | The port |
| --- | --- | --- |
| `PDDeviceGray` | a `TYPE_BYTE_GRAY` image | an `image.Gray` |
| `PDSeparation` | a CS_GRAY colour model over the same samples | delegates to `PDDeviceGray` |
| `PDDeviceRGB`, `PDDeviceCMYK` | null | nil, the same |
| `PDICCBased` | wraps the raster in a colour model carrying the profile | nil: there is no profile to carry |
| `PDIndexed` | an `IndexColorModel` over an sRGB profile | nil: Go has no indexed colour model |
| `PDCIEBasedColorSpace` | null | nil, the same |
| `PDDeviceN` | null | nil, the same |

So `-noColorConvert` writes a PNG for a grey or separation image and otherwise
falls through to the ordinary path, which is what Java does for the four it
answers null for and is not what Java does for `PDICCBased` or `PDIndexed`.
Those two are deferred for want of an ICC engine and of an indexed colour model,
both recorded against slice 6. `channelsOf` in `go/tools/extractimages.go` says
the same at the site: only its first arm can be reached today, so the TIFF half
of the branch waits on those two.

### Defects found and fixed

- **No operators were registered.** `contentstream.NewPDFGraphicsStreamEngine`
  registers none -- every operator package imports `contentstream`, so it cannot
  import them back and the concrete engine registers them instead -- and the
  first version of the engine called `SetOverrides` and stopped, so `Do` was
  never dispatched: the command wrote nothing and exited 0. `addAllOperators` in
  `go/tools/extractimages.go` is the same list `go/pdfbox/rendering/operators.go`
  has; `TestExtractImagesCopiesADeviceRGBJPEG`, over `input/merge/jpegrgb.pdf`.
- **`showGlyph` was missing.** Java's `ImageGraphicsEngine` overrides it to
  process the colour a glyph is painted in, and does not call super, so no glyph
  is drawn: the method is there for the colour. A page whose text is filled with
  a tiling pattern therefore gives up the images inside that pattern, and with
  no override `PDFStreamEngine`'s own ran and the pattern was never walked.
  `go/tools/extractimages.go`; no checked-in PDF paints text that way, so
  `TestExtractImagesFindsAnImageInsidePatternedText` builds one.
- **The JPEG filter list was one name short.** Java's is `DCTDecode` and its
  abbreviation `DCT`; the port had only the first, so a stream filtered `/DCT`
  would have been decoded rather than copied. `go/tools/extractimages.go`.
- **The `jp2` conversion arm returned an error.** Java asks `ImageIOUtil` for a
  "jpeg2000" writer, finds none without the JAI jars, logs two lines and answers
  false -- leaving the file it has already created empty and carrying on. The
  port failed the whole command instead; it now makes the same call and gets the
  same answer.
- **The `tiff` arm compared colour spaces by name.** Java is
  `pdImage.getColorSpace().equals(PDDeviceGray.INSTANCE)`, which `PDDeviceGray`
  does not override, so it is identity. The port compares against the
  `color.DeviceGray` singleton.
- **`WriteImageToFile` wrote nothing for a format it could not write.** Java
  opens the file first and takes the format off the name inside the
  try-with-resources, so an unwritable format leaves an empty file behind; the
  port buffered. `go/tools/imageio/imageio.go`;
  `TestWriteImageToFileLeavesAnEmptyFileForAFormatItCannotWrite`.
- **`channelsOf` answered 4 for an `image.RGBA`.** Java counts the bands of the
  raster the colour space wrapped, and an RGB raster is three; the alpha of a Go
  pixel type is not a fourth, because "we have no alpha information here".
  `go/tools/extractimages.go`.
- **The PNG compression quality was the wrong way round.** `compressionQuality`
  is not a quality for a lossless format; it is the other end of the same dial.
  `ImageWriteParam` documents 0 as "high compression is important", and
  `ImageIOUtil` passes 0 for PNG for exactly that reason -- "PDFBOX-4655:
  prevent huge PNG files on jdk11 / jdk12 / jdk13". The port read 0 as "fastest"
  and mapped it to `png.BestSpeed`, so every PNG it wrote was the *large* one,
  which is the defect PDFBOX-4655 is about.
  `com.sun.imageio.plugins.png.PNGImageWriter`, read off its bytecode, takes
  deflater level 4 by default, 0 for `MODE_DISABLED` and `9 - round(9 *
  quality)` for `MODE_EXPLICIT`; run on a 600x400 gradient it gives 559673 bytes
  at quality 0 against 720846 at quality 1, the second larger than the 720000
  bytes of raw samples because level 0 stores. `pngCompressionLevel` in
  `go/tools/imageio/imageio.go` now maps Java's ten levels onto the four Go has;
  `TestPNGQualityZeroCompressesHardest`.
- **A CMYK image lost its fourth channel.** `ExtractImages` picks a TIFF for a
  raster with more than three bands -- "That's likely CMYK. We use tiff here" --
  and the TIFF writer converted every image that was not grey through `RGBA()`,
  so the separation that `-noColorConvert` exists to keep was thrown away one
  step after being chosen. Java keeps it: measured, by building a
  four-component `ComponentColorModel` over a four-band raster the way
  `PDColorSpace.toRawImage` does, a four-band image comes out with BitsPerSample
  8,8,8,8, SamplesPerPixel 4 and PhotometricInterpretation 5, Separated.
  `tiffSamples` in `go/tools/imageio/tiff.go` now has that arm;
  `TestCMYKTIFFKeepsItsFourSamples`. It cannot be reached through
  `ExtractImages` today, for the reason in the `-noColorConvert` section above,
  but it is reachable through `imageio.WriteImage`, which is public and is what
  `track/raster` calls from `pdfbox render`.
- **`isBitonal` read past the image.** It walked `img.Pix`, and `Pix` is not the
  image: a `SubImage` shares its parent's buffer and stride and its slice runs
  to the end of that buffer, so one grey pixel outside the bounds would send a
  bitonal image out at eight bits per pixel instead of one. It walks the rows
  now. `go/tools/imageio/tiff.go`; `TestBitonalSubImageIsStillBitonal`.

### Still open

- The TIFF is written uncompressed, JPEG 2000 cannot be written,
  `-noColorConvert` reaches one colour space, and the iCCP chunk, the
  six-argument `writeImage` and `MetaUtil.debugLogMetadata` are not ported --
  each with its reason above.
- `jpegWithResolution`'s branch for a JPEG that already carries a JFIF segment
  cannot be reached through this package, because `image/jpeg` writes none. It
  is the port of `JPEGUtil`'s "use the `app0JFIF` node if it is there" and is
  kept for that reason. `go/tools/imageio/metautil.go`.

## Track `multipdf`

The branch ported `PDFMergerUtility`, `Overlay` and `LayerUtility` and took
`Splitter` the rest of the way; the mapping table is in the `pdfbox/multipdf`
section above. It is 37 passing tests across six Java test classes, and the
cases that read `target/pdfs` are ported since the fetch of 2026-09-11 — see
"What the fetch unblocked". Every method of `PDFMergerUtility`, `Overlay` and
`LayerUtility` is ported, in the order `appendDocument` runs them, including the
three the Java gets wrong.

### Defects found and fixed

- **`mergeOpenAction` read both open actions where Java reads one.** Java puts
  both `getOpenAction()` calls inside one `try` and catches an `IOException` out
  of either, so when the *destination's* throws, the source's is never assigned:
  both locals stay null and the block does nothing. Two Go calls that each
  answer an error leave both values in hand, so the port merged a source open
  action into a destination whose own could not be read. It now says what the
  Java's control flow says. `go/pdfbox/multipdf/pdfmergerutility_structure.go`.
- **`GetIDTreeAsMap` walked a nil kids list.** `PDNameTreeNode.getKids()`
  answers nil where Java's returns null and is checked.
  `go/pdfbox/multipdf/pdfmergerutility_structure.go`; `testStructureTreeMerge4`.
- **`mergeThreads` handed a typed nil to `cloneForNewDocument`.** A nil
  `*cos.Array` inside a `cos.Base` is not a nil `cos.Base`, so the guard the
  Java method opens with -- `if (base == null) return null` -- does not fire on
  one. Java reaches it because a Java null is a null whatever its static type;
  the port has to not make the call. `go/pdfbox/multipdf/pdfmergerutility.go`;
  `testClonePDFWithCosArrayStream2`.
- **`PDAnnotationPopup.Parent()` could not answer a subclass.** Java's
  `(PDAnnotationMarkup)` is a cast and every markup annotation satisfies it; the
  port narrowed with a Go type assertion to `*PDAnnotationMarkup`, which a
  `*PDAnnotationText` does not satisfy -- it embeds one. So a popup whose
  `/Parent` was a text annotation answered nil and logged an error.
  `PDAnnotationMarkup` now has `MarkupAnnotation()`, promoted onto every
  subclass, and the assertion is on that. The defect is slice 8's and the test
  is in the annotation package, where it lives:
  `go/pdfbox/pdmodel/interactive/annotation/annotations.go`.
- **`Splitter` was half a port.** See the `pdfbox/multipdf` section above.

### What the port matches

- Every `finally` the Java has, and every place it logs and swallows -- the
  metadata that could not be read (PDFBOX-4227), the invalid open action
  (PDFBOX-4223), the page label index that is not a number, the /IDTree and
  /RoleMap keys that already exist, the orphan annotation -- behaves the same in
  the port, and nothing Java throws became a log.
- `mergeInto`'s exclusion set is compared by pointer, which is sound because
  `cos.Name` is interned.
- The four numbers `PDFMergerUtilityTest` pins -- 104 structure elements
  doubling to 208, 192 IDTree entries, page index 4 for the open action, and the
  126/2/6, 7/4 and six-way ParentTree and RoleMap counts of the splits -- all
  come out of the port unchanged.

### Still open

- The pixel half of `OverlayTest` and of `checkMergeIdentical`. The port
  compares content streams and form XObjects instead, against the same model
  files; what a renderer would add is a second opinion on identical marks.
- Nothing else. The source's /Threads, its /UserProperties and the /PageMode
  went unmerged here because the Java does not merge them, and
  `track/java-bug-fixes` then merged all three in the Go: JAVA-BUGS.md 82, 83
  and 84, each with a **Fixed in the Go** line.

## `track/java-bug-fixes` — the branch that is not a port

The last branch of the migration goes through [`JAVA-BUGS.md`](JAVA-BUGS.md)
entry by entry and fixes the ones worth fixing **in the Go**. It changes no
Java and deletes no entry: a fixed bug is still a bug in the Java, so the entry
gains a **Fixed in the Go** line rather than going away, and the site keeps the
comment naming it. The four triage columns, the four reasons a keep may name,
and the shape of an entry after a fix are in
[`tasks/track-java-bug-fixes.md`](tasks/track-java-bug-fixes.md).
`AGENTS.md`'s rule against fixing Java bugs now names this branch as its one
closed exception, so a divergence carrying a `JAVA-BUGS.md` comment is not a
defect to restore — reverting one puts the bug back.

The branch was taken twice: once for the 84 entries that existed then, and
again for 85 and 86, which `track/raster` added afterwards. Every other branch
is merged, so there is no third pass.

### A0 — the triage

86 entries: **61 fix, 8 keep, 15 not carried, 2 test only.** The two test-only
entries are 4 and 46, whose defect is in a Java test, so the Go test is the
whole of what changes. Before 85 and 86 the fix count was 59.

No verdict is repeated here. Each entry of `JAVA-BUGS.md` carries its own with
the reason in full: a **Fixed in the Go** paragraph, a **Kept in the Go**
paragraph naming one of the four allowed reasons, or a **Where the Go carries
it** line that says the port does not carry it and why.

### Defects found and fixed

- **A tiling pattern's tile was sampled with one texel where Java blends
  four.** `TexturePaintContext.getContext` takes its `filter` flag from
  `KEY_INTERPOLATION` and `PDFRenderer.createDefaultRenderingHints` sets that
  to BICUBIC, so every page PDFBox renders has it on. The four-texel blend is
  `tilingSource.blend` in `go/pdfbox/rendering/raster/tiling.go`, and the flag
  is in the tile cache key because the source bakes it in;
  `TestAScaledTilingPatternRendersAsThePortMeansTo`. This was a port defect in
  merged work rather than an entry of `JAVA-BUGS.md`, taken here because
  `track/stale-deferrals` was already merged.
- **`ScratchFile.isClosed` was a plain `bool` where Java has a `volatile
  boolean`, read by `checkClosed` without a lock.** Found by the race detector;
  it is an `atomic.Bool` in `go/pdfio/scratchfile.go` now, and `JAVA-BUGS.md`
  72 names it.
- **`gen-cos-names.ps1` dropped two names the committed `names.go` has.** The
  line-based match missed `OUTPUT_CONDITION_IDENTIFIER`, whose declaration
  wraps across two lines, and the constant pattern `[A-Z0-9_]+` could not match
  `COSName.Off`, which is declared beside `OFF`.
  `go/migration/scripts/gen-cos-names.ps1` reads the file whole, matches mixed
  case, throws below the 588 names it expects, and breaks ties between
  identifiers differing only in case so the output does not depend on parse
  order; it reproduces the committed file byte for byte, plus the `/Bead`
  override.

### The scaled tiling fixture

`go/pdfbox/rendering/raster/testdata/patternscale.pdf` carries a `/Matrix` that
scales by 1.37, so a 10-unit step is 13.7 device pixels and the tile does not
land on whole ones. `patterns.pdf` cannot show any of that: at 1:1 a tile is
not stretched, a filtered sample lands on a texel corner, and a whole number is
its own ceiling. `JAVA-BUGS.md` 85 is the only fix in this branch that makes a
rendered page differ from PDFBox's, and this is the page that shows it. Of the
page's 7,200 pixels:

| Configuration | Differing | More than a quarter of a channel |
| --- | ---: | ---: |
| before the four-texel blend | 852 | 634 |
| with the blend, `JAVA-BUGS.md` 85 reverted | 550 | 327 |
| both, as merged | 1250 | 600 |

The two halves do not subtract, because the rounding changes the raster the
sampler then reads. What is left is inside `TexturePaintContext.Any`: it walks
the texture with a 16.16 fixed-point accumulator, stepping `xerr` and `yerr`
along each row, and quantises both blend weights to twelve bits before
multiplying them. Float weights from an inverse transform do not land in the
same places. Two readings were ruled out by measurement rather than by reading:
a sweep over every quarter-pixel offset in both axes puts the best fit at
exactly (0, 0) — 550 differing pixels there against 604 a quarter down, 723 a
quarter up and over 1,000 for any offset in x — and both renders put a tile
boundary every 13.7 pixels and start the first at the same place, agreeing
exactly along a row except at the edges of what a tile draws.

### Still open

- `JAVA-BUGS.md` 39 and 55 are each fixed as far as the entry's own "what
  correct would be" goes; the remainder of each is new functionality, and the
  entry says what it would take.
- The 550 pixels above. Matching them means transliterating
  `TexturePaintContext.Any` rather than porting what PDFBox does with it.

## Track `raster` — what draws

`rendering/raster` implements `rendering.Backend` over an in-memory image. It is
a **substitution, not a transliteration**: there is no Java to port, because
Java draws onto a `java.awt.Graphics2D` and Go has no such thing. What is
written is what Graphics2D would have done, against the interface slice 9
defined.

### The decision

`github.com/srwiley/rasterx` draws. [`RASTER-PRECEDENT.md`](RASTER-PRECEDENT.md)
carries the library comparison behind that choice, and what the other
ecosystems do about the same problem. What it does not carry:

- **Weight decided it between the two pure-Go candidates with a real stroke
  model.** `rasterx` needs one module, `golang.org/x/image`. `tdewolff/canvas`
  was rejected on weight and not on capability — it is actively maintained, has
  all four joins and a real `Path.Stroke` — because it needs **24 modules**:
  Fyne, Gio, OpenGL and GLFW (which is cgo), LaTeX, WebP, AVIF, OpenStreetMap.
  `fogleman/gg` needs 2 and `x/image/vector` none.
- **The dependency is pinned at `v0.0.0-20220730225603-2ab79fcdd4ef`**, which is
  its last commit, 2022-07-30. There is no tagged release and upstream will fix
  nothing. It is load-bearing anyway: Fyne's `go.mod` requires it at that same
  pseudo-version, so every SVG icon in every Fyne application is rasterised
  through `oksvg` → `rasterx`. The exposure here is one file —
  `rendering.Backend` is the interface slice 9 defined for exactly this, and
  replacing `rasterx` later touches its implementation and nothing else.
- **The trap, pinned.** `Stroker.SetStroke` takes a gap function *and* a join
  mode, and the gap wins: it defaults from the join mode **only when the gap is
  nil**. Passing `rasterx.FlatGap` explicitly renders a round join as a bevel,
  silently. `rendering/raster/rasterx_test.go` pins this: it drives all three
  joins, all three caps and a dash pattern, and every expected value in it was
  read off an ink map of what `rasterx` actually produces, because pixel
  coverage is area coverage and a pixel counts as inked when any part of it is.

### What draws

| Job | What does it | Why |
| --- | --- | --- |
| stroke a path | `github.com/srwiley/rasterx` | the only pure-Go stroker with miter, a miter limit, all three caps and dashes |
| scan-convert a shape | `github.com/golang/freetype/raster` | both winding rules and native curves; rasterx's own scanner is nonzero-only, its `SetWinding` a documented no-op |
| sample a scaled image | `golang.org/x/image/draw` | nearest-neighbour and bicubic, which are the two `KEY_INTERPOLATION` values PDFBox sets |
| composite, clip, groups, shadings, masks, tiles | written here | the sixteen blend modes of ISO 32000-1 table 136 are not something a general 2D library carries |

### What the numbers are measured against

PDFBox has no rendering test, so `conventions/tdd.md`'s rule that assertion
values are copied verbatim from the Java had nothing to copy from. Four Java
programs are checked in beside the tests they feed, and every number comes from
running one:

| Driver | Reference | What it measures |
| --- | --- | --- |
| `raster/testdata/Java2DDrv.java` | `java2d.txt` | 17 shapes drawn by a JDK 17 Graphics2D under PDFRenderer's own hints |
| `raster/testdata/BlendDrv.java` | `blend.txt` | 340 pixels through PDFBox's own `BlendComposite`, every mode |
| `raster/testdata/RenderDrv.java` | `*-java.png` | three whole pages through PDFBox's `PDFRenderer` |
| `handlers/testdata/SquigglyDrv.java` | `squiggly.pdf` | the appearance PDFBox generates for a squiggly annotation |

The pages the last two render are written by the port's own writer —
`raster/testdata/genpdf.go`, `genpatterns.go`, `genmasks.go` and `genstencil.go`
— and checked in, so both renderers read the same bytes. The first draft of B1
asserted what the operations *mean* rather than what Java produces; five of
those hand-derived tests were deleted rather than kept beside the measured ones.

Every deviation below is pinned in both directions, so one that disappears fails
as loudly as one that appears: `TestAgainstJava2D` and
`TestAgainstJava2DNormalized` pin seventeen differing-pixel counts each, under
the two values of `KEY_STROKE_CONTROL`;
`TestAlphaCompositeRoundsSourceOverDifferently` pins the five rows where Java
disagrees with Java; the page tests pin whole-page counts. Five functions phase
B touched had no test of their own and now have one each: `tilingCeiling`,
`signum`, `isPositiveZero`, the soft mask's `/TR` and the
`NewOffscreen`/`DrawSurface` pair, pinned by `TestTilingCeilingRoundsUp`,
`TestSignumIsJavas`, `TestIsPositiveZeroIsFloatCompare`,
`TestASoftMaskAppliesTheTransferFunction`, `TestDrawSurfacePutsAnOffscreenDown`
and `TestDrawSurfaceHonoursTheClip`.

### What matches exactly

- **Every blend mode**, all 340 rows, including Normal reached through
  `BlendComposite` by reflection.
- **Eleven of the seventeen Java2D shapes under VALUE_STROKE_PURE, and eight
  under VALUE_STROKE_NORMALIZE**: the fills, the clip, the transform, both
  winding rules, the straight-line stroke, and the dashes with and without a
  phase. Three more differ by one unit of one channel and nothing more.
- **Every flat fill on a rendered page**, and every interior. On the two pages
  with strokes, **no pixel anywhere is more than a quarter of a channel out**.
- **A coloured tiling pattern**, 3036 pixels, none of them different.
- **An Alpha soft mask, and a Luminosity one with a /BC backdrop.**
- **The squiggly annotation's appearance**, token for token, across all three
  streams it is made of.
- **`ImageType.GRAY`**: 2182 differing pixels of 40000, none of them more than a
  quarter of a channel — the RGB page's differences, collapsed, because the
  shading's colour drift is worth less once three channels become one.
  **`ImageType.BINARY` is 937**, and every one is a flipped pixel on a stroke: a
  surface with two colours in it has no way to be a little bit out, so the
  single unit of edge coverage the two rasterisers disagree about takes the
  whole pixel. Its fills, shading and rotated square are exact.

### The shading tests are derived, not ported

PDFBox has no shading test: `grep -rli shading` over `pdfbox/src/test` finds two
files and neither tests a shading — one lists operator names, the other checks
that `shadingFill` refuses a shading built from an empty dictionary. Slice 9's
A3 found the same and wrote `graphics/shading/shading_test.go` from the Java
source and the specification. `rendering/raster/shading_test.go` is the same
kind of work: the colour a `ShadingContext` answers at a point, for the axial
type, asserted with no rasteriser and no page, with the derivation in the test.
Two details of `AxialShadingContext` the derivation turns on:

- **The colour is quantised through a table whose size depends on the device
  bounds.** `factor = ceil(the diagonal of the device bounds)`, the table holds
  `factor + 1` colours evaluated at `domain[0] + d1d0 * i / factor`, and the
  raster loop reads `colorTable[(int)(inputValue * factor)]`. The same shading
  over a different sized surface therefore quantises differently. The test uses
  a 300 by 400 surface, whose diagonal is exactly 500, so the quarters fall on
  table entries.
- **`convertToRGB` truncates.** `(int) (rgbValues[0] * 255)` gives 127 for 0.5
  and reaches 255 only at exactly 1.0. It is not rounding, and the difference
  shows on every mid-tone.

### Deviations from Java

- **Stroke normalization — ported, so this one is not a deviation.**
  `PDFRenderer.createDefaultRenderingHints` sets three hints and **not**
  `KEY_STROKE_CONTROL`, so PDFBox renders under the JDK default, which is
  `VALUE_STROKE_NORMALIZE`: Marlin moves each segment endpoint onto a pixel
  centre before stroking — a pixel quarter with anti-aliasing off — so a thin
  line lands on whole pixels instead of straddling two.
  `rendering/raster/normalize.go` is
  `MarlinRenderingEngine.NormalizingPathIterator`, and `SetStrokeNormalization`
  is the hint, on by default because the JDK's default is. The seventeen shapes
  are held to both grids. On the two pages with strokes it took the pixels more
  than a quarter of a channel out from 1002 and 873 to **none**.
- **Anti-aliased coverage quantisation — up to 1 on a straight edge.** Marlin
  samples a pixel on an 8x8 subpixel grid and truncates the count to a byte;
  freetype's rasteriser integrates the area exactly. On an exactly-half-covered
  pixel one says `0x7f` and the other `0x80`. That single unit is all of
  `fillHalfAA` and all of `joinBevel`.
- **Curve flattening, and one pixel of every right-angle join — up to 64.** The
  two flatteners put their line segments in different places, so a round join, a
  round cap and a stroked cubic differ along their edges. The inner corner of a
  right angle is one pixel out by 64: rasterx emits a stroke as separate
  outlines per segment where Marlin emits one, so a scanline rasteriser
  accumulates the two overlapping arms to full coverage where Java has three
  quarters. The ink is in the same place; what differs is how dark its edge is.
- **The JDK's sRGB-to-grey conversion — up to 38, and only for a Luminosity soft
  mask over a non-grey group.** Java draws the group's ARGB image onto a
  `TYPE_BYTE_GRAY` one, which runs the JDK's colour management: an ICC transform
  to `CS_GRAY`, whose grey diagonal measures as `1.055*x^(1/2.4)-0.055` and not
  the weighted sum the name suggests. There is no ICC engine here, so `luma` is
  the standard sRGB luminance. A **grey** group — which is what `isGray` is
  ported for, and what a mask usually is — never converts a colour at all and
  does not go near this.
- **A grey *surface* is the other conversion, and it is BT.601.** A JDK 17
  `renderImage(0, 1, ImageType.GRAY)` of this branch's own page gives 61 for
  (26,51,204), 174 for (230,179,0), 162 for (102,204,102), 53 for (128,0,126)
  and 35 for (32,0,222), and `0.299R + 0.587G + 0.114B` rounded gives every one.
  Those are the coefficients Java2D's ByteGray surface converts with, not the
  ICC transform that drawing an *image* onto a grey surface goes through.
  `quantize.go` is the BT.601 one.
- **The tile cache is a plain map, not a `WeakHashMap`.** Java holds
  `TilingPaintFactory` in a `WeakHashMap` on the PageDrawer, so its entries live
  as long as the page is being drawn; Go has no weak reference, so
  `rendering/raster/tilingcache.go` hangs the cache off the surface and
  `ClearTileCache` empties it. Nothing in a render calls that — a page's
  patterns are wanted for the whole page — and a caller who renders many
  documents through one surface can. Its key leaves out the colour, which is
  Java's behaviour rather than a simplification: JAVA-BUGS.md 86.
- **`awt/geom/area.go` flattens curves to polylines** where the JDK intersects
  them exactly, and the clip is built from it. `rasterx` takes a path of its own
  and the clip reaches it as a mask, so the flattening happens once, in the same
  place, with the same tolerance. Commented in `awt/geom/area.go`.
- **Kept as Java's, with the asymmetry.** `getImage` takes `Math.abs` of the
  pattern matrix's scaling factors and `createContext` does not; `getAnchorRect`
  scales the bbox origin by the *signed* factors while clamping the size by the
  absolute ones. Both are carried as written.
- **Kept as Java's, unguarded.** A tiling pattern whose content stream fills
  with itself recurses in both: `DrawObject` has the level counter and stops at
  50, `processTilingPattern` has nothing, and Java's cache does not help because
  the entry is put in after the constructor returns.
- **Kept as Java's, swallowed.** The soft mask's transfer function throwing is
  ignored and answers the backdrop; a backdrop colour that will not convert to
  RGB keeps the default, which is zero; a singular pattern transform paints
  nothing, which is what `TexturePaint` does with one. `DrawSoftMask` restores
  the drawer's seven fields before it looks at the error, the way
  `DrawTilingPattern` already did, because Java's `finally` runs before the
  exception leaves.
- **`PixelTable`'s inclusive loop is kept**, computing one extra row and column.
  The clamp is `deviceBounds.x + deviceBounds.width` with a `<=` loop, Java
  guards its own indices in `addValueToArray`, the port's table is a map and its
  callers read `x < Max.X`, so the extra entries are never seen. Narrowing the
  clamp would be changing the Java.
- **`ContentStreamWriterTest` asserts against a different file.** Java reads
  `target/pdfs/PDFBOX-4750.pdf`, which the Maven build downloads;
  `pdfwriter/contentstreamwriter_render_test.go` parses, rewrites and renders
  `rendering/raster/testdata/graphics.pdf` instead, with Java's own assertion —
  identical, not close. What is lost is the specific defect PDFBOX-4750 was
  about, and the test says so.

### Defects found and fixed

- **`getAnchorRect`'s zero test was the wrong question.** Java asks
  `Float.compare(xStep, 0) == 0`, which answers zero for `+0.0` alone; Go's
  `xStep == 0` is true for `-0.0` as well, so a pattern whose `/XStep` is
  written `-0` took the bbox width here and kept the negative zero in Java.
  `isPositiveZero` is the test now; `TestIsPositiveZeroIsFloatCompare`, all five
  cases including NaN.
- **`adjustMask` sampled the redraw at the destination pixel's corner rather
  than its centre**, moving every hard mask edge by up to a pixel on a page
  whose transform is not a plain scale. `rendering/raster/softmask.go`;
  `testdata/masksrot.pdf` is the fixture and
  `TestSoftMasksOnARotatedPageRenderAsPDFBoxRendersThem` pins it.
- **The surfaces were `image.RGBA` and held straight colour.** `blendInto` is
  `BlendCompositeContext.compose`, whose components go in and out as straight
  values, so the type was wrong twice over: Go's PNG encoder asks the image what
  it holds and unpremultiplied on the way out, taking blue at half alpha to 253
  rather than 255, and `PopGroup` unpremultiplied pixels that were already
  straight, so a group whose contents were partly transparent came back paler
  than it was drawn. Both were latent for anything but `ImageType.ARGB`, because
  every other type ends opaque. The surfaces are `image.NRGBA` now and the call
  is gone; `TestAnARGBSurfaceExportsItsOwnColours` and
  `TestAPartlyTransparentGroupKeepsItsColour`. `drawSampled`'s scratch buffer
  stays an `image.RGBA` and still unpremultiplies, because what `x/image/draw`
  writes into it really is premultiplied.
- **`DrawSurface` ignored the transform.** `drawImage(image, 0, 0, null)` and
  `clearRect(0, 0, w, h)` both go through the Graphics2D's transform, and
  `PDFPrintable.Print` puts the imageable-area translation, the centring and the
  `scale / dpiScale` rescale on the surface before it calls either, so copying
  pixel to pixel put a rasterized page in the paper's corner at the wrong size.
  `rendering/raster/image.go` goes through `i.transform` now, sampled with the
  interpolation the hints ask for, and the white ground covers the transformed
  rectangle rather than the whole surface.
- **`ImageType.GRAY` and `ImageType.BINARY` were discarded.** `NewImage` used
  the type to choose a white ground and then forgot it, so
  `pdfbox render -color GRAY` produced a colour image. Java draws onto a
  BufferedImage of that type and the image quantizes what is written into it, so
  a later composite reads back what the surface really holds; `quantize.go` does
  the same, at the same moment, and `asImageType` hands back the kind of image
  ImageIO would write as a greyscale or one-bit PNG.
  `TestTheOtherImageTypesRenderAsPDFBoxRendersThem`.
- **`DrawStencil` read the wrong channel.** It asked `ImageOfRegion` for the mask
  and took the coverage from the alpha, but an image mask has no colour space,
  so what comes back is opaque wherever it comes back at all and **every stencil
  painted its whole rectangle**. `rendering/raster/drawimage.go` asks for the
  stencil image now, in an opaque black, and takes the alpha that
  `getStencilImage` sets from the mask's bits and from nothing else. Against
  PDFBox on `testdata/stencil.pdf`, two shapes in four colours, that took the
  count from 3526 pixels of 16000 to **101**, all of them sample boundaries;
  `TestStencilsRenderAsPDFBoxRendersThem`.
- **`PDTilingPattern.COSObject` answered the pattern's dictionary rather than its
  stream**, so `PDResources.AddPattern` wrote a pattern with no content in it.
  Java needs no override — a `COSStream` *is* a `COSDictionary` — but a Go
  `*cos.Stream` carries its dictionary rather than being one.
  `pdmodel/graphics/pattern/pattern.go`; broken since it was written, and
  reached first by `TestSquigglyAppearanceMatchesPDFBox`.
- **JAVA-BUGS.md 85, `TilingPaint.ceiling`, was reproduced rather than fixed** —
  `tilingCeiling` carried it and `TestTilingCeilingIsAFloor` asserted the eight
  JDK-measured wrong answers. `track/java-bug-fixes` fixed it afterwards; the
  test is now `TestTilingCeilingRoundsUp` in `javabug85_test.go`, carrying
  Java's answers beside the corrected ones.

### What this closed elsewhere

| Deferral | Where it was | Now |
| --- | --- | --- |
| `checkRenderIdent` | `pdfbox-layout-awt`'s layout tests | runs, with the pixel counts pinned |
| `PDFPrintable`'s rasterizing | `printing/pdfprintable.go` | ported; `Backend` gained `NewOffscreen` and `DrawSurface`, which are the two Graphics2D calls it needs |
| `PDPatternContentStream` | `pdmodel` | ported |
| the squiggly appearance | `annotation/handlers` | ported |
| `PDFToImage` | `go/tools/notbuilt.go` | ported, registered as `render` |

The three `checkRenderIdent` comparisons are in
`glyphlayout/renderident_test.go`. `TestPDFToImage` is disabled in Java itself
and is not ported. `PDAcroFormFlattenTest` and the six `TestFontEmbedding` cases
were deferred for the files the Maven build downloads and not for the raster;
their inputs arrived with the fetch of 2026-09-11 and both have been ported
since — see "What the fetch unblocked".

### Still open

- **`PrintPDF`**, and the row in `go/tools/notbuilt.go` says why. Everything
  PDFBox computes about where a page lands on a sheet is ported —
  `PDFPrintable`, `PDFPageable`, the rotated boxes, the scale-to-fit, the
  centring, the page border and the rasterizing. What `PrintPDF` needs on top is
  `java.awt.print.PrinterJob` and `javax.print`: enumerating the printers on the
  machine, reading the trays and media sizes one offers, showing the dialog,
  handing it a job. Go's standard library has none of it and no pure-Go library
  does either — printing is a per-platform spooler API.
- **The sRGB-to-`CS_GRAY` difference** in a Luminosity soft mask over a non-grey
  group waits on an ICC engine, which is not going to be written here.

---

## Track `upstream-sync` — what the Apache merge of 2026-09-07 changed

The merge is `3d024173c`, 25 Apache commits over 16 files, and this branch is
what happens to the Go because of them. The branch task file is
[`tasks/track-upstream-sync.md`](tasks/track-upstream-sync.md); it carries the
six out-of-scope files and why, and the ten in-scope rows the table below is
the outcome of.

| Java | Apache | The Go |
| --- | --- | --- |
| `contentstream/operator/state/Concatenate` | PDFBOX-6255 `4a42d294e` | **fixed.** `util.ErrIllegalMatrixValues`, and `Concatenate.Process` recovers it |
| `pdfparser/COSParser` | PDFBOX-5660 `de68eb3e3` | **fixed.** `FileParser.parseObjectDynamically` checks the pool answer |
| `interactive/form/AppearanceGeneratorHelper` | PDFBOX-5660 `ced684bba` | **fixed.** `computeBBox` returns an error |
| `pdfwriter/COSWriter` | PDFBOX-6236 `21661b79f` | **fixed.** `WriteSigned` takes the max of `/Size - 1` and the xref's highest |
| `pdmodel/font/PDTrueTypeFont` | PDFBOX-5960 `a1f50ab4b` | **fixed.** `codeToGIDByName`, `hasContradictorySymbolicFlags`, `isRecognizedBaseEncoding` |
| `util/DateConverter` (test) | PDFBOX-6254 `0079a9cc7` | **assertions ported.** Seven McMurdo dates the Java's own typo had been hiding |
| `fixup/processor/AcroFormOrphanWidgetsProcessor` | PDFBOX-5660 `f7654f001` | **no change.** The port already returned; a test pins it |
| `logicalstructure/PDUserAttributeObject` | PDFBOX-5660 ×3 | **no change.** `JAVA-BUGS.md` 38, fixed in `track/java-bug-fixes` |
| `io/RandomAccessReadBufferedFile` | PDFBOX-5660 `b1d96635f` | **not applicable.** `seek` made `final`; Go has no method overriding, and `OpenBufferedFile` calls the package function `SeekTo` |
| `pdmodel/fdf/FDFUtils` | PDFBOX-5660 `4aa80d810` | **not applicable.** Private constructor on a static-only class; the Go has `escapeXML10` as a package function in `fdf/small.go`, with no type to construct |

### The one `JAVA-BUGS.md` entry the sync closed

Entry 38, `PDUserAttributeObject` reading `/P` without checking: upstream added
the three null checks and landed on the same three answers
`track/java-bug-fixes` had already chosen. The entry gains a **Resolved
upstream** line and stays. No other entry is closed, cross-checked by class name
against every entry and by behaviour against the six methods the sync touched.

### How each row was checked

Every fix has a test at its site that fails with the fix reverted, and every
guard that passes either way was proved load-bearing by a mutation of its own.
`Concatenate.Process` recovers `util.ErrIllegalMatrixValues` and re-panics
anything else, and the error it returns ends the walk, as Java's `IOException`
does: `TestConcatenateOverflowIsAnErrorNotAPanic`,
`TestConcatenateOverflowEndsTheWalk`. The `defer` it adds costs nothing
measurable — 2,000 `cm` operators through `ProcessPage` take 2.77 ms with it
and 2.94 ms without.

### What is still open

Nothing from this sync. The next one starts from `3d024173c`.

