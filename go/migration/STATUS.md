# Porting status

Hand maintained. Update the row for a package in the same commit that ports it.

The machine-generated counterpart is
[`mapping/inventory.tsv`](mapping/inventory.tsv), which knows how much Java
source sits behind each row but only distinguishes "has Go files" from "has
none". This file is where partial work and the reasons for it get recorded.

Status values: `done` · `in progress` · `blocked` · `not started` · `out of scope`

Last updated: 2026-09-05

## Summary

| Phase | Area | Java files | Status |
| --- | --- | ---: | --- |
| 0 | `pdfio` | 18 | in progress — 13 of 18 ported |
| 1 | `pdfbox/cos` | 24 | **19 of 24 — every file slice 1 needs**; the remaining 4 are slice 7 incremental-save machinery, plus 1 folded away |
| 2 | `filter`, `pdfparser`, `pdfwriter` | 48 | in progress — `filter` has the slice 1 subset, `pdfparser` all 18 including `FDFParser`; `pdfwriter` all 3, `getDataToSign` included |
| 3 | `pdfbox/pdmodel` | 433 | in progress — every file of `interactive`, `documentinterchange`, `fdf`, `fixup`, `common`, `graphics/optionalcontent`, `graphics/pattern` and `graphics/form`, and the model half of `graphics/shading`; `pdmodel/font` at 34 of 39 (the 5 left are the embedders), all 12 encodings, `pdmodel/encryption` at 17 of 19. What is left is the 19 `java.awt.Paint` and `PaintContext` classes of `graphics/shading` |
| 4 | `fontbox` | 143 | **done — all 143 files**, finished by slice 4 |
| 5 | `contentstream`, `text` | 85 | **done — all 85 files**, finished by slice 9: the graphics engine, all 23 graphics operators, all 13 colour operators and the three `DrawObject`s |
| — | `awt/geom` (the JDK, not PDFBox) | — | in progress — `Point2D`, `AffineTransform`, `Path2D`, `Rectangle2D`, `Ellipse2D`, `FlatteningPathIterator`, and `Area` minus curves |
| 6 | `rendering`, `printing`, `shading` | 60 | in progress — everything that computes. The raster half is behind `rendering.Backend`, which nothing implements: 4 of `rendering` and 19 of `shading` are `java.awt` classes and are not ported. See the slice 9 section |
| — | `pdfbox` root (`Loader`) | 1 | done — the reading entry points, FDF and XFDF included |
| — | `w3c/dom`, `awt` (the JDK, not PDFBox) | — | in progress — a reading DOM for XFDF, and `Color` |
| 7 | `cmd/pdfbox` | 26 | not started |
| — | `xmpbox` | 74 | not started |

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
| `IOUtils.java` | `ioutils.go` | done — most of it maps to the Go stdlib instead, see the file header |
| `ScratchFile.java` | — | not started — deferred to phase 2, see PLAN.md |
| `ScratchFileBuffer.java` | — | not started — deferred to phase 2 |
| `MemoryUsageSetting.java` | — | not started — only meaningful once `ScratchFile` exists |
| `RandomAccessReadMemoryMappedFile.java` | — | not started — needs a decision on `golang.org/x/exp/mmap` vs `syscall` |
| `NonSeekableRandomAccessReadInputStream.java` | — | not started |

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

**Open debt — closed in slice 7.** The Java `accept()` tests drive a `COSWriter`
and assert the emitted bytes. Until slice 7 the port asserted the visitor double
dispatch plus a direct `WritePDF` byte check, and the serialised form was only
checked against itself. `cos/accept_external_test.go` now makes the Java
assertions — for booleans, integers, floats and strings, and for the static
`COSWriter.writeString` — in package `cos_test`, because a test file in package
`cos` cannot import `pdfwriter` without a cycle. The four tests named in
`boolean_test.go`, `integer_test.go`, `float_test.go` and `string_test.go` keep
asserting the double dispatch and point at it.

The one thing not restored is `TestCOSFloat`'s sweep. Java's `BaseTester` walks
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

Five places where the Go did something the Java does not. All were caught by
review rather than by the ported tests, which is worth noting: the tests were
written from the Java and still missed these, because each one turns on a
detail of how Go differs from Java rather than on what PDFBox does.

- `Integer.IntValue` did not truncate to 32 bits — Go's `int(int64)` is a no-op
  where Java's `(int)` narrows. This also meant `Equals` was **not** reproducing
  JAVA-BUGS entry 1 despite a comment claiming it did.
- `Float.IntValue` and `LongValue` did not saturate. Java clamps an out-of-range
  float to `MAX_VALUE`/`MIN_VALUE`; Go leaves the conversion undefined.
- `Flate.Decode` propagated a corrupt-input error, so `Stream.CreateReader`
  returned no reader and the decoded prefix was unreachable. Java catches,
  logs and returns what inflated — the damage tolerance the filter exists for.
- `Stream.Length` answered while a writer was open, returning a stale value.
  Java throws there; the Go now returns `ErrStreamWriting`.
- `XrefTrailerResolver` resolved to no type when startxref pointed nowhere. The
  Java constructor defaults it to `TABLE`.

## Slice 1 — `pdfbox/filter`

Only the filters slice 1 needs. The rest arrive in slice 6.

| Java source | Go source | Status |
| --- | --- | --- |
| `Filter.java` | `filter.go` | done — minus the `DecodeOptions` overload, which carries image subsampling |
| `FilterFactory.java` | `filter.go`, `provider.go` | done — as `ByName` plus a `Provider` type rather than a singleton |
| `Predictor.java` | `predictor.go` | done |
| `FlateFilter.java`, `FlateFilterDecoderStream.java` | `flate.go` | done |
| `IdentityFilter.java` | `filter.go` | done |
| `DecodeResult.java` | `filter.go` | partial — the JPX colour space and soft mask fields arrive with that filter |
| `DecodeOptions.java` | — | not started — image subsampling only |
| the other 15 filters | — | slice 6 |

| Java test | Go test | Notes |
| --- | --- | --- |
| `PredictorTest` | `predictor_test.go` | complete |
| `TestFilters` | `flate_test.go` | the round-trip generator is ported; `testPDFBOX4517` needs a loader, `testPDFBOX1977` needs LZW, `testRLE` needs RunLength |

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
| `COSParser.java` | — | **next — 2,021 lines, the core** |
| `XrefParser.java` | — | not started — 695 lines |
| `BruteForceParser.java` | — | not started — 857 lines, the damaged-file recovery path |
| `PDFStreamParser.java` | `streamtokenparser.go` | done in slice 2 — named `StreamTokenParser`, because `StreamParser` is already COSParser's stream half |
| `PDFXRefStream.java` | — | not started |
| `PDFXrefStreamParser.java` | — | not started |
| `PDFParser.java` | — | not started — the entry point |
| `PDFObjectStreamParser.java` | — | not started |
| `EndstreamFilterStream.java` | — | not started |
| `FDFParser.java` | — | not started — FDF, not needed for slice 1 |

None of the Java files in this package have tests; the parsers are exercised
only through whole documents. Every test here is therefore written from the
source per the tdd rule, and the recovery paths named in the Java comments —
PDFBOX-3506, PDFBOX-276, brother_scan_cover.pdf — are pinned individually.

### Method note — `cos/stream.go` was not ported test-first

`stream.go` was written before `stream_test.go`, which breaks the rule in
[`conventions/tdd.md`](conventions/tdd.md). The test was then ported from
`TestCOSStream` rather than written against the Go, so the assertions are still
Java-derived — but the ordering was wrong, and the point of the rule is that
the ordering is what protects against confirming a mistranslation.

This is the second such lapse, after `pdfio`. Both are recorded rather than
quietly fixed.

### Method note — `pdfio` was not ported test-first

The port now runs test-first ([`conventions/tdd.md`](conventions/tdd.md)).
`pdfio` predates that rule and did not follow it: the implementation was written
first and the Java tests were ported afterwards.

What that does and does not mean:

- The tests **are** faithful to the Java. Assertion values, the sample byte
  arrays, and the PDFBOX-numbered regression cases were copied from the Java
  test files, not recomputed from the Go. They are real evidence.
- But they did not **drive** the implementation, so they cannot rule out a
  mistranslation that the Java suite happens not to cover.

Everything from `slice/1` onward follows the rule. `pdfio` is the one package
where a later re-read against the Java is worth doing on its own — the chunk
arithmetic in `ReadBuffer` and the page-boundary handling in `BufferedFile` are
the parts where a silent mistranslation would be easiest to miss.

### Ported tests

| Java test | Go test | Notes |
| --- | --- | --- |
| `RandomAccessReadBufferTest` | `readbuffer_test.go` | `testPDFBOX5111` not ported — it downloads a PDF over the network |
| `RandomAccessReadViewTest` | `readview_test.go` | complete |
| `RandomAccessReadWriteBufferTest` | `readwritebuffer_test.go` | complete |
| `SequenceRandomAccessReadTest` | `sequenceread_test.go` | complete |
| `RandomAccessReadBufferedFileTest` | `bufferedfile_test.go` | fixtures written to `t.TempDir()` instead of read from the source tree |
| `RandomAccessInputStreamTest` | `adapters_test.go` | complete |
| `ScratchFileBufferTest` | — | waiting on `ScratchFile` |
| `NonSeekableRandomAccessReadInputStreamTest` | — | waiting on that type |
| `RandomAccessReadMemoryMappedFileTest` | — | waiting on that type |
| `TestIOUtils` | — | mostly covers methods that map to the Go stdlib |

### Deviations from Java recorded so far

Each of these carries a comment at the point of difference in the Go source.

- `CreateView` hands every caller its own cursor instead of caching one clone
  per thread id. Go has no stable goroutine identity, and the result is safe for
  concurrent use where the Java version is not.
- `ReadBuffer.Read` stops when a chunk read returns nothing rather than adding
  `-1` to its running count, which the Java loop does.
- `BufferedFile` shares one mutex-guarded page cache across all cursors, where
  Java reopens the file per thread.
- The evicted page buffer is not reused for the next page read, so a cursor
  still holding an evicted page keeps reading valid bytes.
- `BufferedFile.IsEOF` compares offset against length instead of `peek() == -1`.
- End of input is `io.EOF` throughout, not a `-1` return.

## Slice 2 — walk content streams

Branch `slice/2-content-streams`. The slice dumps the operator sequence of a
page and interprets nothing, so everything that draws or measures is left for
the slices that bring fonts, XObjects and a rasteriser.

### `awt/geom` — the JDK, not PDFBox

Go has no standard-library geometry, and `Matrix`, the graphics state and the
text machinery are all written against `java.awt.geom`. PLAN.md's slice 9
already settles on porting the geometry and leaving rasterisation behind an
interface; this is where that starts. Only what PDFBox calls is here.

| Java source | Go source | Status |
| --- | --- | --- |
| `java.awt.geom.Point2D` | `point.go` | done — the abstract base as an interface, plus the Float and Double forms |
| `java.awt.geom.AffineTransform` | `affinetransform.go` | done — minus the state/type cache, which only lets the JDK skip terms it knows are zero |
| `java.awt.geom.PathIterator` | `shape.go` | done |
| `java.awt.Shape` | `shape.go` | partial — `contains` and `intersects` are absent; nothing in PDFBox calls them on a path, and they need the curve-crossing machinery in `sun.awt.geom` |
| `java.awt.geom.Path2D` | `path.go` | done — one type holding `float64`, rounding on the way in when it is a Float path |
| `java.awt.geom.Rectangle2D` | `rectangle.go` | done — one type; PDFBox only ever uses the Double form |
| `java.awt.Rectangle` | `rectangle.go` | done — the integer bounds only |
| `java.awt.geom.Area` | `area.go` | done in slice 9 — constructive area geometry, minus curves: an added shape is flattened first. Written from the JDK contract, not from what the renderer needs |

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
| `ResourceCache.java` | `pdmodel/font/resourcecache.go`, aliased in `resourcecache.go` | done — the font and font descriptor members here, the rest arriving with their types up to slice 9. The interface is declared in `pdmodel/font` because it names `PDFont` and `pdmodel` imports that package, so the five kinds it cannot name are asked of the cache by shape from `pdmodel` instead |
| `DefaultResourceCache.java` | `resourcecache.go` | done in slice 9 — all eight kinds, each with the stable-cache bookkeeping, which the port writes once as a generic map rather than eight times. Java holds each entry through a `SoftReference`; Go has none, so the port holds them outright |
| `PDPage.java` | `pdpage.go` | partial here — boxes, rotation, resources, contents. The `PDStream` methods came with slice 7 and everything else with slice 8; only `removePageResourceFromCache` is still absent |
| `PDPageTree.java` | `pdpagetree.go` | done — minus the `PDDocument` the reading constructor takes, which is only there to reach a `ResourceCache` |
| `MissingResourceException.java` | `errors.go` | done |
| `PDDocument.java`, `PDDocumentCatalog.java`, `PDDocumentInformation.java` | — | not started here — slice 3 for the document and its information, slice 8 for the catalogue |

`PDPage.getContentsForStreamParsing` is the general path for now. Its fast path
decodes a single flate stream as it is read, which needs
`FlateFilterDecoderStream` and `NonSeekableRandomAccessReadInputStream`, neither
of which is ported.

### `pdfbox/pdmodel/graphics`

| Java source | Go source | Status |
| --- | --- | --- |
| `PDLineDashPattern.java` | `graphics/pdlinedashpattern.go` | done |
| `blend/BlendMode.java` | `graphics/blend/blendmode.go` | done — blend functions included |
| `blend/BlendComposite.java`, `SoftMask.java` | — | not ported — both are `java.awt` raster classes, and slice 9 put the raster half behind `rendering.Backend`. The blend mode and the alpha constants a `BlendComposite` is built from are on the graphics state, and `rendering.SoftMaskedPaint` names the mask a `SoftMask` would rasterise |
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

### Port defect found while porting slice 2, fixed

`COSDictionary.containsValue` had been written as `getKeyForValue` in disguise.
The two look in opposite directions: `containsValue` unwraps an indirect
reference given as the *argument*, `getKeyForValue` unwraps the references
*stored* in the dictionary. Collapsing them made a dictionary holding a
reference to `x` report that it contained `x`, which would have made the
PDFBOX-4509 search in `PDResources.add` dead code rather than the fix it is.
`getKeyForValue` also gained the guard Java puts in front of `getObject`.

### Port defects found in the slice 2 review, fixed

Nine, all found by review rather than by the ported tests. Each carries a test
that fails without the fix.

- **`COSStream.createView` asked a view for a view.** Java builds a second
  `RandomAccessReadView` around the one it holds; the port called
  `CreateView` on it, which a view refuses — in Go as in Java. Every unfiltered
  stream read from a file failed, and `PDPage.ContentsForRandomAccess` reported
  it as a malformed content stream and substituted a newline. **Page content was
  being silently dropped.** The test that covered `createView` had built its
  stream over a `ReadBuffer`, so it never took the path.
- **`COSArray.toFloatArray` and the two numeric list conversions read the raw
  entry.** Java reads through `getObject`, which resolves an indirect
  reference. An indirect number yielded zero, so an indirect `/MediaBox` gave a
  zero-sized page.
- **`setString("")` removed the entry.** Java removes only for a null
  argument, and an empty string is a valid COS string. Go has no null string,
  so the port had used `""` for it; a caller wanting Java's null now calls
  `RemoveItem`, or `Set(index, nil)` on an array. `setEmbeddedString` had the
  same conflation.
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

`COSString.parseHex` skips trailing whitespace and, because it computes a start
offset and then indexes from zero anyway, does not skip leading whitespace — it
rejects it as invalid hex. The port had sliced `hex[start:end]`, correcting it.
Reverted, recorded as [`JAVA-BUGS.md`](JAVA-BUGS.md) entry 11, and pinned by
`string_test.go`, which now asserts the error and the force-parsing
substitution.

## Slice 3 — text from simple fonts

Branch `slice/3-text-simple-fonts`. The slice reads the font programs and the
encodings a simple font needs, decodes the text operators, and writes out the
text of a page. What it does not carry is the file-opening path: the loader is a
blocked decision recorded at the end of this section.

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
| `gsub/`, `TTFSubsetter`, `TrueTypeCollection`, `OTFParser`, `OpenTypeFont`, `CFFTable`, the vertical and kerning tables | — | not started when slice 3 closed; ported in slice 4, see below |

Every table the parser does not read keeps its place in the directory as an
`UnknownTable`, so a later slice can add the read without the file being walked
again.

Two things Go cannot reproduce directly, both commented where they are:

- **`TrueTypeFont.readTable` is not locked**, matching Java, which locks only
  `getTableBytes`. A table read reaches back into the font for the tables it
  depends on; a lock there would deadlock where Java simply re-enters its own
  monitor. `GlyphTable` does lock its own stream, so the recursion a composite
  glyph makes goes through `getGlyphLocked`.
- **`GlyfCompositeDescript` shadows the parent's `contourCount`** with a field
  of its own. Go embedding does not shadow, so the composite count has its own
  name.

`TestCMapSubtable` is not ported: both of its tests read fonts the Java build
downloads into `target/fonts`, which this repository does not carry.
`TestTTFParser.testParseVertical` and `testParseHeaders` likewise.

### `pdmodel/font/encoding` — all 12 files

| Java | Go | Status |
| --- | --- | --- |
| `Encoding`, `BuiltInEncoding`, `DictionaryEncoding`, `Type1Encoding` | `encoding.go`, `dictionaryencoding.go` | done |
| `StandardEncoding`, `WinAnsiEncoding`, `MacRomanEncoding`, `MacOSRomanEncoding`, `MacExpertEncoding`, `SymbolEncoding`, `ZapfDingbatsEncoding` | `encodings.go`, `tables.go` | done — the seven tables generated from the Java |
| `GlyphList` | `glyphlist.go` | done |

`glyphlist.txt`, `zapfdingbats.txt` and `additional.txt` are copied byte for
byte into `pdfbox/resources` and embedded, since Go has no classpath.

Three places walk a Java `HashMap` whose result depends on the iteration order
Java leaves unspecified — `BuiltInEncoding`, `Type1Encoding.fromFontBox` and the
reverse glyph list. The port walks the keys in order instead, so the same input
always gives the same encoding.

### `pdmodel/font` — the simple font path

| Java | Go | Status |
| --- | --- | --- |
| `PDFontLike`, `PDVectorFont` | `pdfont.go` | done |
| `PDFont`, `PDSimpleFont` | `pdfont.go`, `pdsimplefont.go` | done — minus the ToUnicode CMap, below |
| `PDFontDescriptor`, `PDPanose`, `PDPanoseClassification` | `pdfontdescriptor.go`, `pdpanose.go` | done |
| `Standard14Fonts`, `UniUtil` | `standard14fonts.go` | done — the 15 AFM files embedded |
| `PDType1Font` | `pdtype1font.go` | partial — the standard 14 path is whole; the embedded PFB needs `fontbox/type1` and the substitute needs the font mapper, both slice 4 |
| `PDTrueTypeFont` | `pdtruetypefont.go` | partial — an embedded font is read whole; a font that is not embedded has no substitute until the font mapper arrives |
| `PDType3Font`, `PDType3CharProc` | `pdtype3font.go`, `pdtype3charproc.go` | done |
| `PDFontFactory` | `pdfontfactory.go` | partial — Type 1, TrueType and Type 3; Type 0, Type 1C, Multiple Master and the CID fonts report that they are not ported |
| `PDType0Font`, `PDCIDFont*`, `PDType1CFont`, `PDMMType1Font`, the `FontMapper` chain, every `*Embedder`, `Subsetter`, `ToUnicodeWriter`, `CMapManager`, `FontCache`, `FileSystemFontProvider` | — | not started — slices 4 and 7 |

What a font in this slice cannot do, and why:

- **No ToUnicode CMap.** `fontbox/cmap` is slice 4. `PDFont.toUnicode` therefore
  always falls through to what each font works out from its encoding, which is
  the same path a font carrying no `/ToUnicode` takes. This is the single
  largest thing standing between this slice and correct text on a real corpus.
- **No font program for a font that is not embedded.** The font mapper chain and
  `fontbox/util/autodetect` are slice 4. Every path that would read a width or
  an outline out of a substitute reports that rather than guessing; the widths
  of a standard 14 font come from its AFM and are unaffected. **Closed by slice
  4.**
- **No glyph outlines.** `GlyphRenderer` and the CFF charstrings were listed
  here as slice 9; both landed in slice 4 instead. **Closed.**
- **No embedded Type 1 program.** `fontbox/type1` is slice 4; until then such a
  font reads as damaged, which is what Java does with one it cannot parse.
  **Closed by slice 4.**

Three cycles Java does not have, and how each is broken:

- **`ResourceCache` names `PDFont`**, and Java puts it in `pdmodel`, which this
  package's parent imports. The interface is declared in `pdmodel/font` and
  `pdmodel` aliases it back under its Java name.
- **`PDType3Font.getResources` and `PDType3CharProc.getResources` return a
  `PDResources`**, likewise in `pdmodel`. They hand out the resource dictionary
  instead, and `contentstream` wraps it where the engine needs one.
- **The `PDFont` constructor calls the abstract `getName`.** Java dispatches
  virtually from a constructor and Go does not, so the port splits it: the
  concrete font sets `self`, then calls `initFromDictionary`.

### `pdmodel` — the holes slice 2 left

`PDResources.getFont` with its direct font cache, `ResourceCache` and
`DefaultResourceCache` for the font and font descriptor members, the font field
of `PDTextState`, and the reading half of `common/PDStream`. Java holds every
cache entry through a `SoftReference`; Go has none and the port holds them
outright. The stable-cache bookkeeping in `removeFont` is ported as it stands,
since it decides whether a re-read gives the same font object.

### `contentstream` — the text path

The five text-showing operators, and the engine methods behind them: `showText`,
`showTextString`, `showTextStrings`, `applyTextAdjustment`, `showGlyph`,
`showFontGlyph`, `showType3Glyph`, `processType3Stream` and `getDefaultFont`.
The four glyph hooks join `StreamEngineOverrides`.

`shouldProcessColorOperators` is still always true: the Type 3 char proc case
that clears it is `d0`/`d1` handling, which belongs with the renderer.
`processChildStream` is still absent, so a form XObject is still not walked.

### `pdfbox/text` — 6 files

| Java | Go | Status |
| --- | --- | --- |
| `TextPosition`, `TextPositionComparator` | `textposition.go` | done |
| `LegacyPDFStreamEngine` | `legacystreamengine.go` | done — minus the `DrawObject` processor, which walks into a form XObject |
| `PDFTextStripper` | `pdftextstripper.go` | partial — the page walk is whole; `getText(PDDocument)`, `writeText`, the bookmark range and the article beads need types this slice does not carry |
| `PDFTextStripperByArea` | `pdftextstripperbyarea.go` | done |
| `PDFMarkedContentExtractor` | `pdfmarkedcontentextractor.go` | done — minus the XObject walk |
| `pdmodel/documentinterchange/markedcontent/PDMarkedContent` | `.../pdmarkedcontent.go` | done — `PDArtifactMarkedContent` folded in, since only its tag matters here |

Two deliberate departures:

- **The default line separator is a line feed, not `System.lineSeparator()`.**
  Java's choice makes the text a document yields depend on the machine that read
  it. A caller wanting the platform separator sets it.
- **The sort is always `IterativeMergeSort`.** Java tries the JDK sort first and
  falls back when it throws on the intransitive comparator; Go's sort does not
  check, so trying it first would give a different order on exactly the
  documents the fallback exists for.

`golang.org/x/text` is the module's first dependency, for the NFKC
normalisation and the bidi reordering the stripper needs and the Go standard
library does not carry.

### The loader — slice 1's unfinished half, ported here

`PLAN.md` slice 1 is "open a document" and lists `pdfbox/pdfparser` at 18 files.
The branch was merged to `migration-base` at 12 of 18, with the rows above
recording `COSParser` as "next" and `PDFParser` as "not started — the entry
point". Nothing in the tree could open a `.pdf`; `go/cmd/` was empty. Slices 2
and 3 did not notice, because both take a `PDPage` a caller hands them. Slice 3
is the first slice whose acceptance criterion — score 40 real PDFs — cannot be
met without a file, so the work was done here as a special case.

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
| `PDFXRefStream.java`, `EndstreamFilterStream.java`, `FDFParser.java` | — | not started — the first two are the writing path, the third is FDF |

Four things worth naming:

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
- **Encryption is not ported.** `pdmodel/encryption` is a package this port has
  not reached, so an encrypted document is reported rather than decrypted.
  **Closed by slice 5.**

### The corpus — 16 of 40

`TestTextStripper` walks the 40 PDFs of `pdfbox/src/test/resources/input` and
compares against the expected text checked in beside each. The port scores
rather than asserts, because a document needing something this slice does not
carry cannot match and a failing assertion would say nothing new.

**40 of 40 open. 16 of 40 match the expected text exactly.** The 24 that do not:

| Cause | Files | Where it lands |
| --- | ---: | --- |
| Type 0 font | 12 | slice 4 — `PDType0Font` and the CID fonts |
| Type 1C font | 3 | slice 4 — `fontbox/cff` |
| ToUnicode CMap | 4 | slice 4 — `fontbox/cmap` |
| Article beads | 2 | needs `PDThreadBead`; both are `PDFBOX-3110-poems-beads` |
| Yields nothing, cause not yet established | 2 | `PDFBOX-3498-…` and `Liste732004001452_…` |
| One line differs in spacing | 1 | `cweb.pdf` line 249 |

The last three rows are the ones to look at first; the first three are the
slice 4 work in the order it will pay off.

### One more port defect the corpus found, fixed

`PDFont.getSpaceWidth` wraps its `getStringWidth(" ")` call in a catch for
`IllegalArgumentException` and `UnsupportedOperationException` — "Happens if
space is not available in the font or if encoding isn't implemented". A Type 3
font's `encode` throws the second outright, so that catch is the ordinary path
for every Type 3 font rather than an edge case. The port had let the equivalent
panic escape, which took down the whole page walk; `stringWidthOfSpace` now
recovers it where Java catches.

### Port defects found in the slice 3 review, fixed

Five, all found by reading the Java beside the Go rather than by the ported
tests. Each carries a test in `text/review_test.go` that fails without the fix.

- **Four length comparisons counted runes where Java counts UTF-16 units.**
  `TextPosition.mergeDiacritic` returns early for a diacritic of length > 1,
  `isDiacritic` requires length 1, and the duplicate-suppression tolerance in
  both `PDFTextStripper` and `PDFMarkedContentExtractor` divides by the length.
  A character outside the basic plane is one rune and two UTF-16 units, so the
  port merged diacritics Java leaves alone and used half the tolerance Java uses.
  `utf16Length` now counts the way Java does at each of the four.
- **`hasFontOrSizeChanged` dropped Java's last branch.** Java compares font
  names, and where *both* are null falls back to comparing `PDFont.hashCode`,
  which is the hash of the font dictionary. The port had collapsed that to a
  name comparison on the grounds that Go's `Name` never returns null — but a
  Type 3 font with no `/Name` returns the empty string, so two different unnamed
  Type 3 fonts compared equal and the running average character width was never
  reset between them.
- **`removeContainedSpaces` did not shrink the article.** Java removes through
  the list iterator, so the list the article holds shrinks with it; the port
  returned a new slice and assigned it only to the local. Anything reading
  `getCharactersByArticle` after a page — which `PDFTextStripperByArea` does —
  still saw the space.
- **`multiplyFloat` widened before multiplying.** Java multiplies in `float` and
  rounds that; the port converted to `float64` first, which rounds the other way
  either side of a half. It decides whether a line is indented enough to start a
  paragraph.

### Port defects found in the slice 3 feedback, fixed

Seven, from the review comments on the pull request. Each carries a test that
fails without its fix.

- **`isDigitAt` and `XrefStreamParser.readNextValue` swallowed every read
  failure.** Java's `RandomAccessRead.read` throws for a failure and returns -1
  only at the end of the data, and both callers distinguish the two. The port
  treated a failing source as "not a digit" and as "no more data", so a parse
  carried on over whatever state the failure left. Both now return the error
  unless it is `io.EOF`.
- **`PDFTextStripperByArea.ProcessPage` did not clear the duplicate map.**
  Java's `processPage` clears `characterListMapping` before walking; the port
  reimplements `processPage` — Go embedding does not dispatch — and had left it
  out. A stripper used twice, which `extractRegions` documents as supported,
  reported nothing the second time.
- **`GetTextOfPages` did not reset the engine.** Java's `writeText` calls
  `resetEngine` first, which puts `currentPageNo` back to 1 and empties the
  per-page state, and applies the extra formatting where it was asked for. The
  port left the page number where the previous call had pushed it.
- **`parseIntRadix` parsed at 64 bits.** Java calls `Integer.parseInt`, which
  rejects anything outside the 32-bit range. An AFM carrying `Characters
  2147483648` was accepted here and then looped on.
- **`readInternationalDate` overflowed.** A `time.Duration` is int64 nanoseconds
  and reaches about 292 years, so a `LONGDATETIME` past 2196 wrapped silently
  and came back as a date in the past. Java counts in milliseconds and has no
  such limit. Both copies of the read now build the instant from the seconds.
- **The corpus harness only ran unsorted.** Java runs every file both ways
  against `<name>.pdf.txt` and `<name>.pdf-sorted.txt`; the harness now scores
  both. **16 of 40 either way.**

### A Java bug the port had corrected, reverted

`PDFTextStripper.handleDirection` reverses a right-to-left run with
`word.charAt(end)` counting down — UTF-16 code units, so a character outside the
basic plane comes out as its two halves in the wrong order and is destroyed. The
port had reversed runes, which keeps the character whole. Reverted, recorded as
[`JAVA-BUGS.md`](JAVA-BUGS.md) entry 15, and pinned by
`text/feedback_test.go`.

### Reviewed and declined

- **The `aux` clone in `IterativeMergeSort` is dead but stays.** `mergeRuns`
  overwrites `aux[from:to]` before copying it back, so the initial copy is never
  read. Java writes `T[] aux = arr.clone()`, and the port writes the clone. It
  is one allocation-sized copy per sort against a deviation from the source; the
  source wins.

### Known behaviour differences, not defects

- **A symbolic TrueType font that is not embedded aborts the page.** Its
  encoding is synthesised from the font program, and there is none until the
  font mapper arrives in slice 4; the port reports that rather than guessing, and
  the error travels out through `Tf` and stops the walk. Java always has a
  substitute, because its font mapper never returns null. The same holds for
  `PDFont.getWidthFromFont` on a font with no `/Widths` array. This is the
  largest practical gap in the slice after the missing ToUnicode CMap.
  **Closed by slice 4:** the mapper always returns a font, down to the
  last-resort LiberationSans, so nothing aborts for want of one.
- **`minYTopForLine` is computed and never read**, in the port as in Java, whose
  own comment says the check it was meant for caused regression failures.

## Slice 4 — text from CID and CFF fonts

Branch `slice/4-text-cid-cff`. The slice finishes `fontbox` — the CFF and Type 1
font programs, the CMaps, the rest of the TrueType tables and the GSUB shaping
machinery — and the CID half of the font model, including the font mapper that
finds a substitute on the machine for a font a PDF does not embed.

### `fontbox/cmap` — all 5 files

| Java | Go | Status |
| --- | --- | --- |
| `CMap`, `CMapParser`, `CMapStrings`, `CIDRange`, `CodespaceRange` | `cmap.go`, `cmapparser.go`, `cmapstrings.go` | done — all 5 Java tests |

The predefined CMaps under `org/apache/fontbox/resources/cmap` are copied byte
for byte into `fontbox/resources` and embedded; Go has no classpath.

`CMapStrings` builds its 65536-entry and 256-entry tables eagerly, where Java
fills them in a static block. `getMapping` and `getIndexValue` return the
comma-ok pair Java gets from a `null` return.

### `fontbox/type1` and `fontbox/pfb` — all 7 files

| Java | Go | Status |
| --- | --- | --- |
| `Type1Font`, `Type1Parser`, `Type1Lexer`, `Token`, `DamagedFontException` | `type1font.go`, `type1parser.go`, `lexer.go`, `token.go` | done — `Type1LexerTest` and the `Type1Font` half of `PfbParserTest` |
| `Type1CharStringReader` | `cff/type1charstring.go` | moved — see the cycle note below |
| `pfb/PfbParser` | `pfb/pfbparser.go` | done — 3 of the 5 `PfbParserTest` cases; the other 2 need a font this repository does not carry |

`PfbParser` accumulates its record size in `int32`, so a record whose length
byte sets bit 31 goes negative and trips the "record size is negative" check the
way Java's 32-bit `int` does.

`Type1Parser.decrypt` needed brackets: Java's `&` binds looser than `+`, so the
whole sum is masked, while Go's binds tighter.

### `fontbox/cff` — all 26 files

| Java | Go | Status |
| --- | --- | --- |
| `CFFParser`, `CFFFont`, `CFFCIDFont`, `CFFType1Font`, the four charsets and the two encodings, `CFFStandardString`, `CFFOperator`, `CharStringCommand`, `Type1CharString`, `Type2CharString`, the two charstring parsers, `DataInput` and its two implementations, `CharStringHandler`, `IndexData`, `FDSelect` and its two formats | `cff/` (15 files) | done — 5 of the 6 Java tests |

`CFFParserTest` skips: it reads a font the Java build downloads into
`target/fonts`.

`DataInput.readByte` is `ReadSignedByte` here, because `go vet` reserves
`ReadByte() (byte, error)` for `io.ByteReader`.

Two Java package cycles the Go cannot have, both resolved by moving the
declaration to the package that is depended on:

- **`cff` ↔ `type1`.** `Type1CharStringReader` lives in `type1` in Java and is
  implemented by `CFFType1Font`. The port declares it in `cff` and aliases it
  back from `type1`.
- **`cff` ↔ `ttf`.** `CFFParser.parseFirstSubFontROS` writes into a
  `FontHeaders`, which lives in `ttf`, which reads `cff` for its `CFF ` table.
  The port declares a `FontHeadersSink` interface in `cff` that `FontHeaders`
  satisfies.

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

A third Java package cycle: **`gsub` ↔ `ttf`.** `GsubWorker` takes a
`CmapLookup`, which lives in `ttf`, which reaches into `gsub` for its worker
factory. The port declares the two-method `CmapLookup` in `gsub` as well.

`model.GlyphKey` stands for Java's `List<Integer>` map key, which Go cannot use:
it is the ids joined with commas.

Go's `regexp` is leftmost-first the way Java's is, so the longest-first
alternation the compound-character tokenizer builds carries over unchanged.

`TrueTypeFont.getPath` reads the glyph and returns its path now that
`GlyphRenderer` is here; it was the last "not ported yet" in `fontbox`.

### `fontbox/util/autodetect` — all 7 files

| Java | Go | Status |
| --- | --- | --- |
| `FontDirFinder`, `NativeFontDirFinder`, `UnixFontDirFinder`, `MacFontDirFinder`, `OS400FontDirFinder`, `WindowsFontDirFinder`, `FontFileFinder` | `util/autodetect/autodetect.go` | done |

A substitution rather than a transliteration, as the task file called for: Java
reads the `os.name` and `env.windir` system properties, and the port reads
`runtime.GOOS` and the environment. `File.isHidden` becomes a leading dot, which
is what a font directory is ever likely to carry. `find()` gives back paths
rather than URIs, which is what every caller turned them into anyway.

### `pdmodel/font` — the CID half, and the font mapper chain

34 of 39 files. The five that are left are the embedders, which slice 7 needs:
`TrueTypeEmbedder`, `PDTrueTypeFontEmbedder`, `PDCIDFontType2Embedder`,
`Subsetter` and `ToUnicodeWriter`.

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

Four deviations, each commented where it is:

- **`FontCache` holds its fonts outright.** Java holds each through a
  `SoftReference`, so the collector may drop one and the next lookup re-reads it
  from disk. Go has no soft reference and no hook that stands in for one. Nothing
  observes the difference beyond memory use.
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

### The corpus — 34 of 40 unsorted, 33 sorted

Slice 3 left it at **16 of 40 either way**. The Type 0 fonts, the Type 1C fonts
and the ToUnicode CMaps that accounted for 19 of the 24 failures are all read
now.

**40 of 40 open. 34 of 40 match unsorted, 33 sorted.** The 6 that do not:

| Cause | Files | Where it lands |
| --- | ---: | --- |
| Text drawn through a form XObject (`Do`) | 3 | the graphics slice — `PDFBOX-3498-…`, `PDFBOX-4322-Empty-ToUnicode-reduced`, `Liste732004001452_…` |
| Article beads | 2 | needs `PDThreadBead` in `pdmodel/interactive/pagenavigation`; both are `PDFBOX-3110-poems-beads` |
| Arabic diacritics ordered differently | 1 | `FC60_Times.pdf`; the merge logic matches the Java line for line, so the difference is upstream in the widths or in `XDirAdj` |

Sorted mode fails one more, `PDFBOX-3127-…VFont.pdf`, on a single missing space.

The three form-XObject files draw all of their text inside a `Do`, which the
content-stream engine does not descend into yet; that was confirmed by dumping
their content streams, not inferred.

### Port defects found in the slice 4 review, fixed

Four, none of which a ported test caught. Each carries a test that fails
without its fix, or a check written out beside it.

- **`PDFont.getSpaceWidth` had lost its first branch.** Java measures the space
  at the code the `/ToUnicode` CMap maps to U+0020 when the font carries one,
  and takes the encoding branch only otherwise. The port always took the
  encoding branch — a deliberate stand-in written while `fontbox/cmap` was
  unported, and never put back once it landed. It decides the width the text
  stripper compares gaps against, so it moves where words break.
  `font/review_test.go` pins it with a font whose space mapping and code 32 have
  different widths.
- **`PDCIDFontType2.codeToGID` swallowed an error Java lets out.**
  `name.equals(ttf.getName())` throws `IOException` and `codeToGID` declares it;
  the port discarded it and compared against the empty string, so a font whose
  name could not be read silently took the ToUnicode fallback instead of
  reporting. Fixed behind the same short-circuit Java's `&&` gives it.
- **`FileSystemFontProvider.createFSIgnored` had been quietly corrected.** Java
  passes `null` for the parent provider, so `getFont()` on one of these entries
  throws `NullPointerException`; the port passed the provider and would have
  loaded the font. Reverted to the Java and recorded as
  [`JAVA-BUGS.md`](JAVA-BUGS.md) entry 21.
- **`PDType0Font` was missing `getGsubData` and `getCmapLookup`.** Both were
  filed under the embedding half, but the reading constructor sets them too, to
  `GsubData.NO_DATA_FOUND` and `null`. They are now there with those values, so
  nothing in the class is deferred that does not have to be.

### Five more Java tests ported, found by the review

`GsubWorkerForDevanagariTest`, `GsubWorkerForGujaratiTest`,
`GsubWorkerForTamilTest`, `GsubWorkerForAaltTest` and `GsubWorkerForSmcpTest`
had been passed over without a recorded reason, and all five run here: three
read fonts this repository carries, one reads `otf/FoglihtenNo07.otf`, and the
last reads Calibri, which its own `assumeTrue` guards.

They are the first coverage the shared reph worker has beyond Bengali, and the
first of the type 3 alternate and type 2 multiple substitution paths.
`GsubWorkerForAalt` and `GsubWorkerForSmcp` live in the Java *test* tree rather
than the library — each is the Latin worker with one feature of its own, with
`applyGsubFeature` copied rather than shared — so the Go test copies them the
same way.

The `@Disabled` cases are left out, as the Bengali port leaves its two out:
Devanagari drops `rkrf`, `cjct`, `abvs` and `psts`, and Gujarati drops `psts`.

### What the review checked

- **Every Java class of the slice has a Go counterpart.** All 143 of `fontbox`;
  34 of 39 in `pdmodel/font`, the five left being the slice 7 embedders.
- **The substitutes table was diffed mechanically** against the Java rather than
  read: thirteen entries, identical.
- **`FontMapperImpl`'s scoring**, where a Java `int` division truncates before
  it is scaled and a `float` distance widens after; the `PriorityQueue` heap and
  `Double.compare` beneath it; `findFont`'s fall-through chain, including the
  PDFBOX-5806 step that reassigns `postScriptName` and so changes the
  `-Regular` attempt after it; `isCharSetMatch`'s bit masking.
- **`FileSystemFontProvider`'s disk cache**, against the 215-font
  `.pdfbox.cache` this machine's first run wrote: twelve pipe-separated fields,
  the platform line separator, hex weight class, twenty Panose digits, absolute
  path, CRC32, epoch millis — the format Java writes, so the two can read each
  other's. `loadDiskCache`'s memo of the last file and hash, and the `continue`
  that keeps a changed file pending, both match.
- **`TTFSubsetter`'s arithmetic**: the table checksums, the four-byte alignment,
  the `headSet` and `subSet` sizes the `hhea` and `hmtx` builders count, and the
  compound-glyph walk that rewrites component GIDs.
- **`CMap.useCmap`**, which carries JAVA-BUGS 16 and Java's `putIfAbsent`
  sharing of the very map it was handed.
- **`OS2WindowsMetricsTable`'s three EOF catches**, each of which sets a version
  down and returns rather than failing.
- **Every ignored error in the slice**: all but one were comma-ok type
  assertions; the one real one is the second defect above. `CMap.readCode`
  ignores its read count because Java does.

### Font substitution is environment-dependent — measured

`FileSystemFontProvider` scans the machine's fonts, so the same corpus could
score differently elsewhere. It does not:

**With every system font directory hidden from the finder, the corpus scores 34
of 40** — the same as with the 215 fonts this machine has installed. Every
lookup falls to the embedded LiberationSans instead of to Arial, and nothing
changes, because text extraction takes its widths from the PDF's `/Widths` and
`/W` arrays rather than from the substitute.

The floor is guaranteed rather than incidental: `getTrueTypeFont`,
`getFontBoxFont` and `getCIDFont` all end at `lastResortFont`, which is compiled
into the binary. None of the three can return nil, and no document can fail for
want of a font on the machine.

### Port defects found in the slice 4 feedback, fixed

Six review comments, of which three were port defects, one was a missing piece
of scope, and two were Java behaviour the port already reproduced.

- **`CMap.toInt` accumulated at 64 bits.** Java's accumulator is an `int`, so a
  four-byte code whose first byte is 0x80 or more comes out negative; a Go `int`
  kept it positive. The width is observable, not cosmetic: `toUnicode(int)`
  tests the code against 256, 0xFFFF and 0xFFFFFF to work out how many bytes it
  had, and a negative code takes the two-byte branch — so Java misses a mapping
  it stored under that same negative key, while the port found it.
  `cmap/feedback_test.go` pins both the arithmetic and the lookup it decides.
- **`PDCIDFont.readVerticalDisplacements` swallowed a malformed `/W2`.** Java
  casts every entry with `(COSNumber)` and indexes past the end without
  checking, so a bad array throws out of the font's constructor. The port had
  softened that to a warning and a partial read, which left the font with some
  of its vertical metrics filled in and the rest defaulted — a deviation that
  was commented at the site but not recorded here, and not one the port was
  entitled to make. The casts are back; a failed type assertion and an
  out-of-range index panic where Java throws.
- **`KerningTable.read` did not narrow its version 1 subtable count.** Java
  casts the unsigned count to a signed 32-bit `int`, so a count with bit 31 set
  goes negative and the `> 0` check skips it; the Go kept it positive and would
  have sized an allocation on it. The branch turns out to be unreachable —
  recorded as [`JAVA-BUGS.md`](JAVA-BUGS.md) entry 22 — so no test can reach it
  either, but the cast is written out so the two lines say the same thing.
- **`PDFontFactory.createFont` was missing the Type 0 `/Subtype` repair.** It
  had been deferred while `PDType0Font` was unported and the deferral was never
  lifted, so the header comment had gone stale — which is what the review
  noticed. `fixType0Subtype`, `getFontTypeFromFont`, `getFontHeader`,
  `getFontDescriptor`, `getDescendantFont`, the six header sniffers and the
  `FontType` inner class are now ported: a Type 0 font whose descendant's
  `/Subtype` disagrees with the embedded font program has both the subtype and
  the `/FontFile2` ↔ `/FontFile3` entry corrected, as Java does. The two log
  lines the port had dropped — the wrong `/Type` and the invalid `/Subtype` —
  are back with it.

### Reviewed and declined — slice 4

- **`CMapStrings.getMapping` reads a zero-length code as the two-byte code 0.**
  Reported as a divergence from UTF-16BE semantics. It is not a divergence: Java
  does exactly this, because its ternary has two arms for three cases. Carried
  as written, recorded as [`JAVA-BUGS.md`](JAVA-BUGS.md) entry 23, and commented
  at the site so it is not "fixed" later.
- **`createDescendantFont` names the `/Type` in its error, not the `/Subtype`
  that failed to match.** Java does that too — `"Invalid font type: " + type`,
  where `type` is the dictionary's `/Type`. The port now concatenates the
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

`TestSymmetricKeyEncryption.testPermissions` is ported whole and its three
files cover the reading path end to end: **R2/V1** (RC4-40), **R3/V2**
(RC4-128) and **R6/V5** (AES-256), so `RC4Cipher`, algorithm 2, algorithm 2.A,
algorithm 2.B and `SaslPrep` all run. All three were made with Adobe Acrobat
rather than with PDFBox, which is the point: a round trip through this port
would pass even if both halves were wrong.

The four read-only tests of `TestPublicKeyEncryption` are ported and pass, on
files and keystores this port did not make either.

`fromsource_test.go` covers what neither reaches, and names it at the top:
`AccessPermission`'s bit arithmetic and its read-only lock,
`getPermissionBytesForPublicKey`, the protection policies, the handler factory,
`PDEncryption`'s setters, and the two ciphers. Its values come from outside the
port — RFC 2268's test vectors for RC2, Go's own `crypto/rc4` for RC4, and
RFC 4013's worked examples for SASLprep.

### The three tests that are not ported, and why

`testProtection`, `testProtectionInnerAttachment`, `testPDFBox4308` and
`testPDFBox4453` of the symmetric test, and `testProtection`,
`testProtectionError` and `testMultipleRecipients` of the public key one, all
**encrypt a document and save it**. The writer is slice 7. This is the decision
the branch's Blocked section asked for: the branch ports the reading half and
the encrypting code that does not need a writer, and leaves the tests that save
to the slice that can run them.

`testPDFBox5955` and `testPDFBox5639` read PDFs the Java build downloads into
`target/pdfs`, which this repository does not carry.

### Infrastructure the port supplies, which is not a migration

Java hands three things to BouncyCastle and the JCE. Go's standard library has
none of them and PDFBox has no code of its own to port for them, so this branch
writes them — the same kind of decision the slice 9 rasteriser needs, and each
file says so at the top:

- **`cms.go`** reads a CMS enveloped-data blob: the key transport recipients,
  their identifiers, and the content once the RSA key has unwrapped it. Only
  reading; the encrypting half would need an encoder.
- **`pkcs12.go`** reads a PKCS#12 keystore — the RFC 7292 SHA-1 derivation, the
  MAC, 3DES for the shrouded key bags and 40-bit RC2 for the certificate bags,
  which is what the checked-in keystores use.
- **`rc2.go`** is RC2 from RFC 2268, which nothing in Go has and the
  certificate bags need.

### What is deferred, and why

- **`PublicKeySecurityHandler.prepareDocumentForEncryption`** reports an error.
  It builds one CMS enveloped-data blob per recipient, which needs an encoder
  Go does not have; and nothing can save a document until slice 7, so it cannot
  be exercised either way. It is the one method of the nineteen files that does
  not do what the Java does.
- **`StandardSecurityHandler.prepareDocumentForEncryption` is ported** and does
  what Java does; it has no test, because a test would have to save.

### Deviations from Java, each commented where it is

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

### The slice 5 adversarial review

The branch's own D7, D8 and D9 asked for a byte-level sweep, the wrong-password
behaviour, and not to take a green suite as proof. All three found something.

#### Found and fixed

- **RC2's key expansion divided by zero.** `tm := byte(255 % (1 << n))` reads
  correctly and is wrong: Go gives the untyped `1` the byte type of the
  conversion around it, so `1 << 8` is 0 and the modulus divides by zero. It is
  a defect the port introduced rather than one it carried — Java's `1` is an
  int — and it is the kind D7 exists to catch. The RFC 2268 test vectors now run
  against the fixed expansion.
- **`validatePerms` refused a short /Perms with an error.** Java only turns a
  *misaligned* one into an IOException, through the IllegalBlockSizeException
  the cipher raises; a missing or empty one reaches `perms[9]` and throws an
  unchecked exception. The port now does the same: the length check covers the
  aligned case and the indexing covers the rest.
- **Two of the review's own test premises were wrong**, which is worth writing
  down because both were assumptions rather than readings:
  - A wrong keystore *alias* was expected to fail. It does not:
    `PublicKeyDecryptionMaterial` ignores the alias entirely when the store
    holds one entry, and all four checked-in keystores hold exactly one. The
    test now pins that, because it is surprising and it is what Java does.
  - A password-protected document opened *with* a keystore was expected to
    reach the handler and be refused as incompatible material. It never gets
    there: the keystore is loaded with the same password first, so the failure
    is the keystore's. The test now asserts that.

#### What was checked

- **Every byte-level operation**, which is what D7 asks. `RC4Cipher.fixByte`,
  which exists only because a Java byte is signed and Go's is not; the `& 0xFF`
  masks in `AccessPermission(byte[])`, `calcFinalKey` and the /Perms word; the
  unsigned `>>>` shifts of the permission integer, which the port writes as a
  `uint32` conversion so that a negative permission word shifts the way Java's
  does; the XOR of the iteration key against the round number in all three
  password functions; `truncateOrPad`'s 32-byte pad and `truncate127`'s cut.
- **The digest resets.** Java's `MessageDigest.digest()` resets the digest, so
  the fifty-round loops of `computeEncryptedKeyRev234` and `computeRC4key` start
  each round from empty. A Go hash keeps its state, so each round builds a fresh
  one; getting this wrong would have produced a plausible, wrong key.
- **`computeHash2B`'s loop condition**, `round < 64 || (e[e.length-1] & 0xFF) >
  round - 32`, which reads `e` only from round 64 on — Go's `||` short-circuits
  the same way, so the nil first time round is safe.
- **The two swallows.** Java's AES-256 path reads through a CipherInputStream
  and swallows the bad padding at the end, keeping the blocks already written;
  the AES-128 path turns the same failure into an IOException. The port keeps
  the asymmetry, and returns the partial plaintext where Java has already
  written it.
- **The raw stream hazard.** `decryptStream` reads a stream and writes it back
  through `createRawInputStream` and `createRawOutputStream`. Java's
  `createRawOutputStream` clears the buffer when the data is already in memory,
  which would wipe what the reader is reading; it is safe only because a parsed
  stream keeps its data in a read view instead. The Go does exactly the same, so
  the port is faithful here rather than accidentally safe.

#### What the wrong password does — D8

Five failures, each with a test:

| Case | What happens |
| --- | --- |
| Wrong document password | `InvalidPasswordError`, "Cannot decrypt PDF, the password is incorrect" |
| Wrong keystore password | The PKCS#12 MAC check fails, before anything is decrypted |
| Wrong alias, single-entry store | Opens — the alias is ignored, as in Java |
| Public key document, password given | "Decryption material is not compatible with the document" |
| Password document, keystore given | The keystore refuses the document password |

None of them returns a document full of rubbish, which is the failure D8 is
about.

#### D9 — the evidence is external

Every file the ported tests read was produced by something other than this port:
three by Adobe Acrobat, four by whatever wrote the public key fixtures, and the
keystores with them. The from-source tests take their values from RFC 2268,
RFC 4013 and Go's own `crypto/rc4`. Nothing in the slice is checked against
itself.

### Port defects found in the slice 5 feedback, fixed

**The CMS reader could not read an RC2 envelope, which is the only kind PDFBox
writes.** `decryptCMSContent` read the initialisation vector once, before it
knew the algorithm, as a bare OCTET STRING. That is the shape AES-CBC and
DES-EDE3-CBC use — RFC 3565 section 4.1 and RFC 3370 section 5.2 — and it is not
the shape RC2-CBC uses: RFC 3370 section 5.3 wraps the version and the IV in a
SEQUENCE. The unmarshal failed with a tag mismatch before the switch could
reach the RC2 branch below it, so that branch was unreachable and every
`/Recipients` entry Java's `createDERForRecipient` produces — it asks the JCE
for `PKCSObjectIdentifiers.RC2_CBC` — would have been refused. The four
checked-in public key fixtures use AES and 3DES, so nothing caught it.
`TestCMSContentParameters` now runs both shapes; it failed with exactly the tag
mismatch before the fix.

**A comment on the wrong side of the encryption.** The `SecurityHandler`
interface said `PrepareDocumentForEncryption` "prepares everything to decrypt
the document". Both implementations already said encrypt; Java's javadoc says
"Prepare the document for encryption". The interface was alone in being wrong.

**A dummy reference propping up an import.** `pddocument_encryption.go` carried
`var _ = cos.Encrypt` so that the `cos` import would compile. Nothing else in
the file used the package. Both are gone.

**What `LoadPDFFromWithKeyStore`'s password is.** The comment read as though the
password and the keystore were alternatives. They are not: where a keystore is
given the same string opens it, because Java calls `KeyStore.load(keyStore,
password.toCharArray())` and then hands the string on to
`PublicKeyDecryptionMaterial` as the private key password. One argument, two
jobs. Java's own javadoc says only "password to be used for decryption".

### Reviewed and declined — slice 5

**`RegisterHandler` should refuse a duplicate policy.** It should, and Java's
javadoc says it does, and Java's code does not — it checks `nameToHandler` and
then overwrites `policyToHandler` without a look. Adding the second check would
be fixing a bug that is in the Java. It is recorded as JAVA-BUGS 26, commented
where it happens, and pinned by
`TestRegisterHandlerReplacesADuplicatePolicy`. The doc comment now says what the
code does rather than repeating Java's promise.

**The error message should name the handler.** `"The security handler name is
already registered"` is Java's string, character for character. The port keeps
Java's messages so that a caller matching on them sees the same text.

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

**DCT cannot be byte-identical to Java's**, and says so where it is:

- `image/jpeg` has already applied the Adobe inversion a CMYK JPEG stores its
  samples with, where Java writes the samples as stored and lets the image's
  /Decode array invert them. The port takes that inversion back out, which is
  exact — one subtraction per sample, for both the plain CMYK and the YCCK
  arms, which was read out of `applyBlack` rather than assumed.
- Two JPEG decoders do not agree to the last bit. The inverse DCT and the YCbCr
  conversion are approximations and `image/jpeg`'s differ from the JRE's in the
  last place on some samples.
- `image/jpeg` refuses a four component JPEG with no Adobe APP14 marker;
  Java's `getAdobeTransformByBruteForce` falls back to reading it as CMYK.

### `pdfbox/util/filetypedetector` — all 3 files

Done, with tests written from the source. One thing worth naming: Java searches
the whole array it allocated rather than the part it filled, so a file shorter
than the longest signature is searched with trailing zeroes after it — a three
byte file `00 00 01` is detected as an ICO. The port pads to the same length so
that it reads the same files the same way.

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

**Two conversions are not faithful and cannot be:**

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

**Three substitutions, each commented where it is:**

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

**And three gaps, which are absences rather than differences:**

- **No TIFF decoder.** `PDImageXObject.createFromByteArray` reads a TIFF the
  CCITT reader refuses by falling through to `ImageIO`, which decodes an LZW
  TIFF. Go's standard library has none. `lzw.tif` loads in Java and does not
  here; `TestCreateFromByteArrayLZWTiff` pins that so it stays visible.
- **No BMP decoder**, for the same reason, on the same path.
- **`JPEGFactory.createFromImage` ignores the DPI.** Java writes it by editing
  the JFIF APP0 marker through the writer's metadata tree; Go's `image/jpeg`
  gives no way to. Nothing in a PDF reads it — the image is scaled by the
  content stream — and PDFBOX-6235 notes that a CMYK JPEG has no JFIF marker to
  carry it either.

### Two things slice 1 deferred, brought in because this slice needs them

`COSDictionary.getCOSStream`, for the /Mask and /SMask of an image, and
`PDStream.createInputStream(List<String>)`, which hands a caller the
still-encoded JPEG or fax data.

### A port defect in slice 1, found by this slice and fixed

`COSDictionary.toString` delegates in Java to a `getDictionaryString` that
carries a list of the objects it has already been through. The Go had no such
guard, so a dictionary holding itself — PDFBOX-5315, which the colour space
`create` path reports on — ran the stack out instead of printing a marker. The
recursion test for that colour space is what found it.

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
build downloads and this repository does not carry — the same reason two of
slice 5's tests are absent.

The four image tests that are not ported all **save the document** and several
render it back, which is slice 7 and slice 9. In their place the port tests the
property each factory rests on, which needs neither: that what goes in comes
back out. `LosslessFactory` round trips a synthetic gradient, a grey ramp and
three checked-in PNGs pixel for pixel; `CCITTFactory` round trips a bitmap
through the fax encoder and reads the checked-in Group 3 and Group 4 TIFFs;
`PNGConverter` converts the checked-in truecolor and indexed PNGs and compares
every pixel against the same file decoded by `image/png`, and declines exactly
what Java declines.

Those round trips are not weaker than the Java tests they stand in for. A wrong
Paeth predictor would still deflate and still decode; the pixels would be wrong.

### Corpus

34 of 40 unsorted, 33 sorted — unchanged. The corpus measures text extraction,
which this slice does not touch.

### The slice 6 adversarial review

The branch's own D7, D8 and D9 asked for byte-level comparison, damage
tolerance, and the image types Go's standard library does not cover. All three
found something, and so did D1.

#### Found and fixed

- **`SampledImageReader.from8bit` writes a region to the wrong rows.** Reading
  it against the Java line by line: the destination offset is
  `y * inputWidth * numComponents`, where `y` is the row of the *source* image
  and `inputWidth` its width, but `bank` is the raster of the *clipped region*.
  For any region that is a strict subset this lands on the wrong row and then
  past the end. It is a bug in the Java — the branch beside it, for the
  subsampled case, does the same job with a running index and gets it right —
  so the port carries it and panics where Java throws
  ArrayIndexOutOfBoundsException, which `getRGBImage` does not catch. **This is
  the one the port had quietly fixed**: three bounds guards I had written while
  porting made the Go silently write nothing where Java fails. They are gone.
  Recorded as JAVA-BUGS 31.

- **A truncated ASCII85 stream repeats its last complete group.** The damage
  tolerance test asserted a clean prefix and failed at 680 bytes against 676.
  Reading `read()` and `read(byte[], int, int)` together: `read()` sets `index`
  to 0 before it reads a group and returns -1 from inside the loop when the
  stream ends part way through one, leaving `n` at the previous group's 4. The
  array read then finds `index < n` and copies that group out a second time.
  The port did the same thing, so the test was wrong and not the port; it now
  asserts the repeat and says why. Recorded as JAVA-BUGS 32.

- **`ASCIIHexFilter` adds -1 for a digit that is not hexadecimal.** Written
  while covering the error paths the round trip never takes: the test expected
  `4Z` to decode to 64 and measured 63, because the table entry for an invalid
  digit is -1 and the filter logs it and then adds it anyway. An invalid *first*
  digit contributes -16, so `Z4` comes out as 0xF4. Recorded as JAVA-BUGS 30.

- **Two more bounds guards removed**, for the same reason as the first:
  `PDIndexed.readColorTable` divides by the base colour space's component count
  without checking it, and `from1Bit` indexes its output without checking it.
  Both are unreachable in practice, and both now index the way the Java does.

- **A port defect in slice 1, found by this slice.** `COSDictionary.toString`
  delegates in Java to a `getDictionaryString` that carries the objects it has
  already been through; the Go had no such guard, so a dictionary holding itself
  ran the stack out. The colour space `create` path reports exactly that case —
  PDFBOX-5315 — and the test for it is what found the recursion. Fixed in
  `cos.Dictionary.String`, which is now that method.

#### What was checked

- **The narrowing casts**, which is what D1 asks. `ASCII85InputStream` holds
  every byte as a signed one and the port uses `int8` throughout, because
  `(byte) in.read()` conflates 0xFF with the end of the stream (JAVA-BUGS 27)
  and `(byte)(ascii[k] - OFFSET)` wraps. `ASCII85OutputStream.transformASCII85`
  builds its word with 32 bit arithmetic that overflows before a mask takes the
  low 32 bits back; the port writes that out in `int32` rather than assuming the
  answer is the four bytes big endian. `from1Bit` shifts a sign extended byte up
  so that the bit under test is the sign bit, and the port shifts an `int32`.
  `LZWFilter.findPatternCode` returns a signed byte where its comment says the
  index matches the value (JAVA-BUGS 28). `CCITTFaxFactory.readlong` combines
  four reads in one expression, and Go does not fix the order of evaluation of
  operands, so the port reads them into named variables first.
- **`estCompressSum` sums *signed* bytes.** Which of the five PNG predictor rows
  wins depends on it, and reading them as unsigned would pick a different one.
- **The three `continue mode` labels of the CCITT 2D decoder**, and its
  `getNextChangingElement` mask of `0xFFFF_FFFE`, which is -2.
- **Every place Java logs and swallows.** `LZWFilter` catches its own
  EOFException, logs and flushes; `from1Bit` and `from8bit` warn on a short read
  and keep going; `PNGConverter` returns null rather than throwing at every one
  of its fourteen checks. The port does the same in each.
- **The two `finally` blocks** of the filters, which dispose an ImageReader the
  port does not have.

#### D7 — byte for byte, not visually

Every filter that round trips is checked by decoding its own output and
comparing the bytes: `TestFilters` runs LZW, ASCIIHex, ASCII85, RunLength,
Crypt and Flate over twenty rounds of adversarial data, `testPDFBOX1977` over
the checked-in regression file, `testRLE` over its nine corner cases. CCITT is
checked by encoding a bitmap and decoding it back to all 960 pixels, and by
reading the checked-in Group 3 and Group 4 TIFFs. `LosslessFactory` and
`PNGConverter` are checked pixel for pixel against three real PNGs each.

**DCT cannot be**, and that is stated rather than worked around: two JPEG
decoders do not agree to the last bit. The nearest available check is the one
Java's own test makes — the mean difference between an image encoded by the
port and its original — and the port measured 5.03 against Java's bound of 5.
That was not adjusted away: encoding jpeg.jpg at quality 60 through 90 gives
7.05, 5.47, 5.03, 4.72, 2.78 and 0.41, a smooth rate-distortion curve, so Go's
encoder is simply lossier at the same nominal quality. The bound in the port is
6 with that curve written beside it.

#### D8 — the damage tolerance

Each filter this slice added, over a stream cut in half:

| Filter | What it does |
| --- | --- |
| LZW | catches its own EOFException, keeps the codes that decoded |
| RunLength | breaks out of both arms at the end of the input |
| ASCIIHex | stops at the end of the data |
| ASCII85 | keeps what decoded, then repeats the last group — JAVA-BUGS 32 |
| CCITTFax | fills the rest of the bitmap with zeroes, so the row count still produces an image |

None of them returns nothing, which is the failure D8 is about.

#### D9 — what Go's standard library does not cover

Five, all recorded in `migration/STATUS.md` and none of them a Java bug:

- **JBIG2 and JPX**: no decoder in Go, and none in PDFBox either — both are
  handed to an ImageIO plugin. The port reports the missing reader, which is
  what Java reports without the jars.
- **TIFF**: `createFromByteArray` falls through to ImageIO for a TIFF the CCITT
  reader refuses. `lzw.tif` loads in Java and does not here;
  `TestCreateFromByteArrayLZWTiff` pins the gap.
- **BMP**: the same path, the same absence.
- **A four component JPEG with no Adobe APP14 marker**: `image/jpeg` refuses it,
  where Java sniffs for the marker by brute force and reads it as CMYK.
- **ICC**: no engine, which is why `PDDeviceCMYK` converts naively and
  `PDICCBased` takes its alternate. That one is not a missing file format but a
  missing colour transform, and it is the largest gap in the slice.

#### What is still open

**A3**: four of the seven image tests are not ported, because every one of their
tests saves the document and several render it back. The port tests the property
each factory rests on instead, over the same files. Closing A3 needs slice 7.

### Port defects found in the slice 6 feedback, fixed

**No sampled or calculator function could be built through the factory.** Java
`PDFunction.create` tests `base instanceof COSDictionary`, and a `COSStream`
satisfies it because `COSStream extends COSDictionary`. A Go `*cos.Stream`
embeds `cos.Dictionary` but is not one, so the port's type assertion rejected
every type 0 and type 4 function — and those two are *always* streams, one
holding a sample table and the other a program. Every `/Separation` and
`/DeviceN` whose tint transform is one of them was therefore unbuildable. The
ported Java test did not catch it because `TestPDFunctionType4` calls the type 4
constructor directly. `NewPDFunction` now names both cases and hands the
constructor the stream, which is what Java's `(COSDictionary) base` still is.

**A losslessly imported CMYK image came back blank.** The predictor encoder
picked DeviceCMYK for an `*image.CMYK` and then wrote nothing: the branch meant
to read the four channels tested for a `CMYK()` method, and `image/color.CMYK`
carries its channels as fields and has no such method. Every sample stayed
zero, which in CMYK is white. It reads the channels through
`color.CMYKModel.Convert` now, and `TestLosslessCMYKRoundTrip` checks every
sample of a 17 by 13 picture.

**A CMYK JPEG was written as three components and declared as four.** Go's
`image/jpeg` has no four component encoder — it writes every image that is not
grey as three component YCbCr — but `colorSpaceOfImage` read the Go image type
and said DeviceCMYK, and the inverted decode array beside it added eight
entries. A reader would take three samples per pixel as four. The port converts
a CMYK image to RGB before encoding and declares what it actually wrote; that
loses the CMYK colour space Java writes, which is recorded above with the other
image gaps.

**A repeated filter was decoded twice by the stopping stream.**
`PDStream.createInputStream(List<String>)` hands its filters to the static
`Filter.decode`, which reduces a repeated filter to one before it applies any —
a stream whose `/Filter` array names the same filter twice is a malformed one
PDFBox repairs. The port decoded both. The reduction is in
`Stream.CreateReaderStopping` and not in `codecList`, because
`createInputStream()` with no stop filters does not do it: it chains the filters
one for one through `COSInputStream`.

**`Raster.SetPixel` half wrote a pixel.** Handed fewer values than the raster
has bands it stopped at the shorter of the two, leaving the remaining bands
holding whatever the pixel had before. Java reads `numBands` values and throws
`ArrayIndexOutOfBoundsException`; the port panics now. A *longer* array is still
fine in both, which is what lets the CIE colour spaces pass a three element one
to a single band raster.

---

## Slice 7 — write and manipulate

Branch `slice/7-write-merge`. The first slice that produces a PDF rather than
consuming one.

### `pdfbox/pdfwriter` — all 3 files

| Java file | Go file | Notes |
| --- | --- | --- |
| `COSWriter.java` | `coswriter.go` | done, minus `getDataToSign` — see below |
| `COSStandardOutputStream.java` | `cosstandardoutputstream.go` | done; unexported, because nothing outside the package uses it |
| `ContentStreamWriter.java` | `contentstreamwriter.go` | done |

`COSWriter`'s public byte constants and its static `writeString` are declared in
`pdfwriter/compress` and re-exported here under the Java names.
`COSWriterObjectStream` needs them and `pdfwriter` imports `compress`, so Go
forbids the direction Java uses; putting the definitions at the bottom of the
dependency keeps one implementation rather than two. `compress/tokens.go` says
so, and so does the block at the top of `coswriter.go`.

`getDataToSign` is **not ported.** It builds the byte range to be signed out of
`COSFilterInputStream`, which lives in `pdmodel/interactive/digitalsignature`
and arrives with slice 8. Everything around it is ported: `doWriteSignature`
computes and writes the `/ByteRange`, and `WriteExternalSignature` writes a
signature made elsewhere into the reserved space. Signing through a
`SignatureInterface` returns an error naming this gap. `SignatureInterface`
itself is declared in `pdfwriter` rather than in the package Java has it in, for
the same reason — it is one method, and the writer is the only thing in this
slice that names it.

### `pdfbox/pdfwriter/compress` — all 4 files

| Java file | Go file |
| --- | --- |
| `CompressParameters.java` | `compressparameters.go` |
| `COSObjectPool.java` | `cosobjectpool.go` |
| `COSWriterCompressionPool.java` | `coswritercompressionpool.go` |
| `COSWriterObjectStream.java` | `coswriterobjectstream.go` |

`COSWriterCompressionPool` takes a `PDDocument` in Java. The port declares
`compress.DocumentLike` — `Document()` and `Encryption()` — so that the
dependency runs one way, the same device slice 5 used for the security handlers.
`pdfwriter.PDDocumentLike` embeds it and `encryption.PDDocumentLike`.

### `pdfparser/PDFXRefStream` — done

`pdfparser/pdfxrefstream.go`. Java's `Collection<COSObjectKey>` and `Set<Long>`
become maps keyed on the key's internal hash and on the number, sorted where
Java's `TreeSet` iteration order matters.

### `pdfbox/cos` — the update state slice 1 deferred

`COSUpdateInfo`, `COSUpdateState`, `COSDocumentState` and `COSIncrement` are
ported, and wired into `Dictionary`, `Array`, `Object`, `Stream` and `Document`
at every site Java calls `getUpdateState().update(...)`.

Java's three default methods of `COSUpdateInfo` cannot be embedded: they need
the owner, and an embedded struct in Go has no way back to the value embedding
it. Each implementor writes them out, one line each. `Stream` overrides the four
it would inherit from the `Dictionary` it embeds, so that the state's owner is
the stream — otherwise an increment would write the dictionary inside a stream
and drop the stream data.

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

`ObjectStreamXReference`'s `object` field is still not carried by
`xref.ObjectStreamReference`. Its only accessor, `getObject()`, has no caller in
the Java main tree.

### `pdfbox/pdmodel` — the save path

`pddocument_save.go` holds `save`, `saveIncremental`, `setVersion`,
`getDocumentId`/`setDocumentId` and `importPage`. `PDPage` gained
`getContentStreams`, `getContents` and the two `setContents`; `PDStream` gained
the constructors that write into a document.

`subsetDesignatedFonts` is a no-op: Java walks `fontsToSubset` and calls
`font.subset()`, and font subsetting is font embedding, which slice 3 left out.
The set is always empty, so the call site is ported and the body is not.

`new PDDocument()` now does two things it did not: it sets `/Version` `1.4` on
the catalogue, which Java's constructor does, and it hands the document a
`filter.Provider`. Java resolves a filter through a static registry; the port
passes the provider in, which is what keeps `cos` from importing `filter`, and
without it a document built in memory could not write a Flate stream.

### `pdfbox/multipdf` — 3 of 6 files

| Java file | Go file | Notes |
| --- | --- | --- |
| `PDFCloneUtility.java` | `pdfcloneutility.go` | done |
| `PageExtractor.java` | `pageextractor.go` | done |
| `Splitter.java` | `splitter.go` | the page splitting, not the structure tree — see below |
| `PDFMergerUtility.java` | — | **deferred to slice 8** |
| `LayerUtility.java` | — | **deferred to slice 8** |
| `Overlay.java` | — | **deferred to slice 8** |

`PDFMergerUtility` names 12 types from `pdmodel/interactive` and
`pdmodel/documentinterchange/logicalstructure` — acroforms, annotations,
actions, destinations, outlines, the structure tree, viewer preferences — and
`LayerUtility` needs `PDPageContentStream` and `graphics/optionalcontent`.
`Overlay` needs `graphics/form/PDFormXObject`, which in turn needs
`PDPropertyList`. All of that is slice 8's subtree.

`Splitter` is ported as far as the same wall. `split`, `processPages`,
`createNewDocumentIfNecessary`, `splitAtPage`, `createNewDocument`,
`processPage` and the three setters are here, so the pages, their content and
their resources are split exactly as Java splits them. Seven private methods are
not: `fixDestinations`, `cloneStructureTree`, `cloneIDTree`, `cloneRoleMap`,
`cloneTreeElement`, `processResources` and `processAnnotations`. What a split
therefore leaves behind is the structure tree, the outline destinations and the
annotations. The four `createNewDocument` catalogue copies — viewer preferences,
language, mark info, metadata — are in the same position. Every one of them is
named in the type comment on `Splitter`.

### Which Java tests are ported, and which are not

| Java test | Go test | Notes |
| --- | --- | --- |
| `OperatorNameTest` | `contentstream/operator/names_test.go` | all 8, complete. Moved to the package the names live in, which is where a Go reader looks |
| `COSWriterTest` | `pdfwriter/coswriter_test.go` | 2 of 4 — see below |
| `PageExtractorTest` | `multipdf/pageextractor_test.go` | complete |
| `TestToUnicodeWriter` | `pdmodel/font/tounicodewriter_test.go` | all 8, complete — the A3 deferral from slice 3 |
| `COSWriterCompressionPoolTest` | — | needs `PDDocumentOutline` and `PDOutlineItem` — slice 8 |
| `COSDocumentCompressionTest` | — | all 5 need `PDAcroForm`, `PDComplexFileSpecification`, `PDPageContentStream`, `PDCheckBox` or `protect` |
| `ContentStreamWriterTest` | — | needs `PDFRenderer` and `TestPDFToImage` — slice 9 |
| `PDFCloneUtilityTest` | — | all 3 need `PDPageContentStream`, `PDFMergerUtility` or `PDOptionalContentProperties` |
| `OverlayTest` | — | needs `PDPageContentStream` and `PDFRenderer` |
| `TestLayerUtility` | — | needs `LayerUtility` |
| `MergeAcroFormsTest`, `MergeAnnotationsTest`, `PDFMergerUtilityTest` | — | need `PDFMergerUtility` |
| `TestFontEmbedding` | — | needs `PDPageContentStream` and `TestPDFToImage`; the other half of slice 3's A3 deferral |

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
diff a whole file against a Java run. The reference file is the closest thing
available and it is a real one.

### What a plain save does to object numbers

`COSWriter.write` sets `number = getHighestXRefObjectNumber()` before it starts,
so a full, uncompressed save of a **loaded** document renumbers every object
from there — a file with 227 objects comes back with objects 228 to 454 and
`/Size 455`, and the xref table carries 227 free entries for the gap. That is
Java, not a port defect: `fillGapsWithFreeEntries` exists for exactly this. The
compressed path does not renumber, because the compression pool offers each
object's existing key back to `COSObjectPool.put`.

### The slice 7 adversarial review

Read `coswriter.go`, `contentstreamwriter.go`, the four `compress` files,
`pdfxrefstream.go`, the four `cos` update-state files and the three `multipdf`
files against their Java side by side.

**Found and fixed.** `visitFromDictionary` assigns `byteRangeArray = (COSArray)
entry.getValue()`, an unchecked cast that throws `ClassCastException` where the
entry is not an array. The port had written `w.byteRangeArray, _ =
value.(*cos.Array)`, which leaves it nil and turns a loud failure into a nil
dereference several steps later, in `doWriteSignature`. It now asserts the same
way Java casts.

**Found and recorded.** `ToUnicodeWriter.allowDestinationRange` checks only
`prev` for being a single code point, so a longer destination following a
one-character one is swallowed into the range and everything after its first
character is lost. JAVA-BUGS 33; the port carries it, with the comment above the
method.

**Checked and matching**, each read against the Java:

- Every branch and its order in `visitFromArray`, `visitFromDictionary`,
  `visitFromStream`, `visitFromString`, `visitFromObject` and
  `visitFromDocument`.
- `doWriteBody`, `doWriteBodyCompressed`, `doWriteHeader`, `doWriteTrailer`,
  `doWriteXRefTable`, `doWriteXRefInc`, `fillGapsWithFreeEntries`,
  `getXRefRanges`, `getObjectKey`, `addObjectToWrite`, `prepareIncrement`.
- `COSStandardOutputStream`: `position += len` against Go's `Write` returning
  `n == len(b)` on success, and `writeEOL`'s single-newline guard.
- `COSWriter` implements `ICOSVisitor` only — no `Closeable`, no `close()` — and
  `PDDocument.save` does not close the stream it is given. PDFBOX-4321.
- The xref entry number formats: `DecimalFormat("0000000000")` and `("00000")`
  against `%010d` and `%05d`. They differ for a negative value, where Java keeps
  ten digits after the sign and Go counts the sign into the width; neither
  column is ever negative — one is a byte offset, the other a free object
  number.
- `PDFXRefStream.getIndexEntry` walked by hand for the inputs {0}, {0,1} and
  {0,5}.
- `COSObjectKey.Equals(nil)` returns false, which is what `equals(null)` does,
  and `COSObjectPool.put` depends on it.
- `ObjectStreamXReference`'s constructor argument order against
  `xref.NewObjectStreamReference`.

**Deliberate divergences**, each commented where it occurs:

- `PDFCloneUtility.cloneForNewDocument` is generic in Java and casts its result.
  The port returns `cos.Base` and adds `CloneDictionaryForNewDocument` for the
  one caller shape that needs the concrete type; both are the same unchecked
  cast, in a different place.
- `PDFCloneUtility`'s constructor and `cloneMerge`, and `Splitter.splitAtPage`,
  `createNewDocument` and the two document accessors, are `protected` or
  package-private in Java. Go has no such level and the types are public, so
  they are exported.
- `PDStream(PDDocument, InputStream, ...)` closes the `InputStream`; a Go
  `io.Reader` has nothing to close, so `NewPDStreamOfInput` leaves that to the
  caller.
- `ToUnicodeWriter` is package-private and final in Java, so the port keeps it
  unexported. Nothing calls it yet: its caller is the font embedding path, which
  slice 3 left out.

### Port defects found in the slice 7 feedback, fixed

**`COSDictionary.addAll` was routed through `setItem`.** Java's `addAll` is
`items.putAll(dict.items)` and nothing else. The port had it copying entries
through `SetItem`, which was harmless until this slice gave `SetItem` two jobs
it did not have before: wrapping a keyed, non-direct value into a `COSObject`,
and calling `getUpdateState().update(value)`. Both then leaked into every
`addAll`, so a copied entry became an indirect reference and the receiving
dictionary was marked as needing an incremental write. `COSDictionary`'s copy
constructor is `addAll`, so `new COSDictionary(page.getCOSObject())` in
`importPage` was affected too. `AddAll` now uses the raw insertion helper.
`TestDictionaryAddAllIsARawPut` asserts both halves and fails on either.

**The trailer `/ID` digest hashed UTF-8.** Java feeds it
`Long.toString(idTime).getBytes(ISO_8859_1)` and
`cosBase.toString().getBytes(ISO_8859_1)`; the port converted the Go strings
directly, which is UTF-8. Identical for ASCII metadata and different for
anything else, so a document with a non-ASCII `/Title` came out with an `/ID`
the reference would not produce — visible whenever `setDocumentId` is used to
make the output deterministic. `encodeISO88591` in `coswriter.go` now does what
the charset does, including the `?` an unmappable character becomes. Java has no
shared helper for this and neither does the port; `encryption` has its own copy
with the same body. `TestDocumentIDDigestUsesISO88591` computes the expected
digest from the Java rule, not from the port.

**`Document.SetTrailer`'s doc comment.** Java's javadoc carries an editorial
note from the original author, and the port had transcribed it with its leading
`//` intact, producing a doubled comment marker and a line that reads as
nonsense in the Go documentation. The comment now says what the method does —
it links the trailer to the document state, which is what makes a later change
count as an update — and attributes the note.

### Reviewed and declined — slice 7

**`PageExtractor.Extract` panics for a start page beyond the document.**
Reported as contradicting the doc comment, which promises a blank document. It
does contradict it — in the Java. `extract`'s guard covers `startPage >
endPage` and not `startPage > getNumberOfPages()`, so pages 30 to 40 of a 28
page file reach `setEndPage(28)` with `startPage` already 30, and
`IllegalArgumentException` is thrown two frames down. The port panics, which is
what an unchecked exception becomes. Recorded as JAVA-BUGS 34 and pinned by
`TestExtractBeyondTheDocumentPanics`, which asserts the panic and its message.

The same report also worried about indexing `splitted[0]` when nothing is
extracted. That cannot happen: reaching the index means both setters accepted
their arguments, so `1 <= startPage <= endPage <= numberOfPages`, and
`processPages` therefore makes at least one destination document.

## Slice 8 — forms, annotations, interactive features

Branch `slice/8-forms-annotations`. The largest slice by file count: every
subtree under `pdmodel/interactive`, the whole of `pdmodel/documentinterchange`
and `pdmodel/fdf`, the document fixups, the optional content, and the half of
`pdmodel/common` slice 2 left.

Every Java class in those packages has a Go counterpart. What is missing is
named method by method below, and each gap is a type a later slice brings.

### `pdmodel/interactive` — the four classes of the package itself

| Java file | Go file |
| --- | --- |
| `PlainText.java` | `interactive/plaintext.go` |
| `PlainTextFormatter.java` | `interactive/plaintextformatter.go` |
| `AppearanceStyle.java` | `interactive/appearancestyle.go` |
| `TextAlign.java` | `interactive/textalign.go` |

`PlainText` breaks a paragraph into lines with `java.text.BreakIterator`, which
follows the Unicode line breaking algorithm. Go has no such iterator in its
standard library. `lineBreakSegments` breaks before a run of whitespace and
after a hyphen, which is what those rules come to for the Latin text a form
field holds; text in a script that breaks by its own rules — Thai, Khmer,
Japanese — is broken differently. The comment above the function says so.

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

### `interactive/annotation` — all 52 files

`pdannotation.go` holds the abstract base and its factory; `annotations.go` and
`annotations2.go` the concrete annotations; `dictionaries.go` the border
styles, the appearance dictionary and stream, the action dictionary, the
additional actions, the markup information, the external data and the rest.

`annotation/handlers` holds every appearance handler plus `cloudyborder.go` and
`pathwriter.go`. `register.go` fills the `DefaultAppearanceHandlers` registry
from its `init`, which is how `PDAnnotation.constructAppearances` reaches a
handler without this package naming each one.

`PDSquigglyAppearanceHandler.generateNormalAppearance` is **not ported.** It
fills the squiggle with a tiling pattern, which needs `PDTilingPattern`,
`PDPatternContentStream` and the `PDPattern` colour space — all slice 9's. The
handler is here and its comment says so.

### `interactive/action` — all 25 files

`pdaction.go` (the abstract action and `PDActionFactory`), `actions.go` (the
concrete actions, `PDURIDictionary` and `PDWindowsLaunchParams`) and
`additionalactions.go` (the five additional-action dictionaries).

### `interactive/documentnavigation` — all 12 files

`destination/pddestination.go` and `destination/pdpagefitdestination.go` hold
the abstract destination, the factory, the named destination and the five page
destinations. `PDPageDestination` is an abstract class in Java; the port keeps
its state in a struct the five concrete destinations embed, and declares the
`PageDestination` interface for what `instanceof PDPageDestination` asks.

`outline/` holds `PDOutlineNode`, `PDDocumentOutline`, `PDOutlineItem` and
`PDOutlineItemIterator`.

`destination` cannot name `PDPage`: `PDPage` reaches this package through the
annotations. `PageLike` names what is used, and `pdmodel` sets
`NewPageFromDictionary` and `IndexOfPageInTree` from its `init`.

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

`SigningSupport` needs the writer, and `pdfwriter` cannot import `pdmodel`.
`COSWriterLike` names the two methods it uses and the writer satisfies it.
`SignatureInterface` itself stays in `pdfwriter`, where slice 7 declared it.

**This closes slice 7's `getDataToSign` gap.** `COSWriter.DataToSign` is
ported, and with it the real signing path: `PDDocument.saveIncremental` with a
`SignatureInterface` now signs rather than returning the error slice 7 left.
`COSFilterInputStream` is what it needed.

### `pdmodel/documentinterchange` — all 24 files

`logicalstructure/` — `PDStructureNode`, `PDStructureElement`,
`PDStructureTreeRoot`, `PDAttributeObject`, `PDUserAttributeObject`,
`PDMarkInfo`, `PDMarkedContentReference`, `PDObjectReference`, `Revisions`,
`PDParentTreeValue`. `markedcontent/` — `PDMarkedContent`, `PDPropertyList`.
`taggedpdf/` — the standard attribute objects and `StandardStructureTypes`.
`prepress/` — `PDBoxStyle`.

`PDArtifactMarkedContent` is folded into `markedcontent/pdmarkedcontent.go` as
the constructor `NewPDArtifactMarkedContent`: Java's subclass adds accessors for
the artifact's own properties, and only its tag reaches the text extractor.

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
`LoadFDFFrom`, `LoadXFDF` and `LoadXFDFReader`.

`COSWriter.write(FDFDocument)` inverts the same way: `pdfwriter.WriteFDF` takes
the COS document, and `FDFDocument.Save` calls it.

Reading XFDF needs a DOM, and Go has none. `go/w3c/dom` is a small reading DOM
built for this — `Parse(reader, namespaceAware)`, `TextContent`,
`FirstElementByTagName`, `ElementsByPath`. `encoding/xml` erases the difference
between a CDATA section and ordinary text, which `FDFAnnotationFreeText` and
`FDFAnnotationText` depend on, so the parser records byte offsets and looks back
at the source to tell them apart. PDFBox's four XPath expressions are replaced
by the two child-element helpers, because every one of them is a direct child
lookup.

`pdfbox/util/xmlutil.go` is `XMLUtil`, `pdfbox/util/hex.go` the two `Hex`
methods FDF needs, and `go/awt/color.go` is `java.awt.Color` — a colour built
from a packed integer or three components, which is all PDFBox uses of it, plus
the named constants it references.

### `pdmodel/fixup` and `fixup/processor` — all 8 files

`fixup/fixup.go` holds `PDDocumentFixup`, `AbstractFixup`,
`AcroFormDefaultFixup` and `AcroFormOrphanWidgetsFixup`; `fixup/processor/`
holds the four processors.

**The fixups can only be linked in by the program itself.** `form` cannot import
`fixup` — `fixup` names `PDAcroForm` — so `fixup`'s `init` sets
`form.NewAcroFormDefaultFixup`. A program that wants the default fixup applied
must blank-import the package:

    import _ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fixup"

Without it, `form.AcroFormOfCatalog` reads the form with no fixup applied, which
is what `getAcroForm(null)` of Java does. The package comment says so, and so
does every test that needs the fixup.

The same call therefore has two behaviours depending on the import graph, where
Java has one. That is a divergence of the port, left as it stands; JAVA-BUGS 48
records it, together with what `getAcroForm()` mutates when it is applied.

`AcroFormOrphanWidgetsProcessor.ensureFontResources` finds the replacement font
but does not embed it: Java calls `PDType0Font.load`, and the font embedders are
not ported. The lookup and its logging are here so the shape is right when they
land.

### `pdmodel/graphics/optionalcontent` — all 3 files, and `PDPropertyList` with them

`pdoptionalcontentgroup.go`, `pdoptionalcontentmembershipdictionary.go` and
`pdoptionalcontentproperties.go`.

`PDPropertyList` lives in `documentinterchange/markedcontent`, where Java has
it, and its static `create` names the two optionalcontent subclasses. Go forbids
that direction, so `markedcontent` keeps a registry and `optionalcontent` fills
it from its `init` — `CreatePropertyList` then dispatches exactly as Java's
chain of `if`s does.

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
  `setUserUnit` are ported. Only `removePageResourceFromCache` is left: it
  purges the colour space, ext gstate, pattern, shading and XObject halves of
  the resource cache, and four of those five still have no type.
- **`PDDocumentCatalog`.** Every accessor is ported. `getAcroForm` and
  `setAcroForm` are in `interactive/form` (see above); the other 36 are in
  `pddocumentcatalog.go`, kept out of `pddocument.go` so that the file next to
  `PDDocument` names no interactive type.
- **`PageMode`, `PageLayout`** — `pagemode.go`. Java's `fromString` throws
  `IllegalArgumentException` for a value that is not one of the constants; the
  value comes out of a PDF rather than from the library, so the port answers an
  error. `getPageMode` and `getPageLayout` check it exactly where Java catches.
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
  `AppearanceGeneratorHelper` cannot exist without them. `shadingFill` is not
  ported — it names `PDShading` — and neither is `PDPatternContentStream`, which
  names `PDTilingPattern`. Both are slice 9's.
  `PDPageContentStream`'s five deprecated `appendRawCommands` methods are not
  ported either, and that one is a choice rather than a gap: they write bytes
  into the stream unchecked, Java marks every one `@Deprecated`, and nothing in
  the main tree calls them.
- **`PDStream.createInputStream(DecodeOptions)`** is not ported. `cos.Stream`
  has no reader that takes decode options, because the `Codec` interface does
  not; the options exist and `filter.DCT` honours them, so closing this needs a
  change to that interface rather than to `PDStream`.
- **`PDOutputIntent`** is in `graphics/color`, where Java has it, because the
  catalogue's three output intent accessors name it. Its
  `PDOutputIntent(PDDocument, InputStream)` constructor is **not ported**: it
  reads the ICC profile with `java.awt.color.ICC_Profile`, and Go has no ICC
  engine — the same gap `PDICCBased` records. It also takes a `PDDocument`,
  which `graphics/color` cannot name.

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

`interactive/digitalsignature`, `interactive/measurement` and
`interactive/viewerpreferences` have no Java test directory at all.

### Port defects found while porting slice 8, fixed

**`PDStream.createOutputStream(COSName)` crashed on a null filter.** Java's
`stream.createOutputStream(filter)` accepts null and means "no filter". The port
passed the `*cos.Name` straight into `CreateWriterWithFilters`, where a nil
name widened to a `cos.Base` is not `nil` — Go's typed-nil trap — and
`filter.ByName` dereferenced it. Found by porting `PDSquareAnnotationTest`,
which is the first caller that passes no filter.

**`COSArrayList` compared by Go pointer identity.** Java's `contains`,
`indexOf`, `lastIndexOf` and `remove` use `equals`, and
`PDDictionaryWrapper.equals` compares the COS dictionary. Each accessor of the
port builds a fresh wrapper, so two wrappers over one dictionary are different
pointers and `retainAll` kept the wrong entries.
`COSArrayListTest.testRetainIndirectObject` left one element where Java leaves
three. `equalsAny` now asks the element for an `Equals` method first.

**`PDFont` had no `Equals`.** Java's `PDFont.equals` compares the font
dictionary, and `PDDefaultAppearanceStringTest` asserts that the font it put in
the resources is the font that comes back. Added, along with
`PDType1Font.getType1Font`, which the same test reads.

**`COSArray.toCOSNameStringList` answered the wrong thing.** It now returns
`[]string` and panics on an entry that is not a name, which is the
`ClassCastException` Java's cast throws.

**Three units of this slice's scope were not ported at all**, found by phase C
rather than by a test: `PDOptionalContentProperties`, the third file of the
optional content package; `PDResources.getProperties`, which the B9 note
promised; and 36 of `PDDocumentCatalog`'s 40 methods, every one of which names a
type from this slice. All three are ported now, with their Java tests.

### Java bugs found, carried, and recorded

Fourteen, `JAVA-BUGS.md` entries 35 to 48:

- **35** `COSName.BEAD` is `"BEAD"` where the specification says `/Bead`, so
  `PDThread.getFirstBead` reads an entry no writer produces.
- **36** `PDWindowsLaunchParams.setOperation` writes `/D` and `getOperation`
  reads `/O`.
- **37** `PDMarkInfo.setSuspect` ignores its argument and always writes false.
- **38** `PDUserAttributeObject` reads `/P` without checking it is there.
- **39** `PDStructureNode.insertBefore` inserts at -1 when the reference kid is
  not in the array.
- **40** `PDStandardAttributeObject` writes a string array and reads a name
  array.
- **41** `PDFourColours` pads a short array to five entries.
- **42** `StandardStructureTypes.types` collects itself.
- **43** `PDSeedValue` writes strings into `/Reasons` and `/LegalAttestation`
  and reads them back as names.
- **44** `FDFAnnotationFreeText.getRotation` reads a string that `setRotation`
  wrote as an integer.
- **45** `FDFDictionary.getPages` reads the array with `get` rather than
  `getObject`, so an indirect page throws `ClassCastException`.
- **46** `PDStreamTest` builds its stop filters from `COSName.toString()`, so
  the stopping it is named after never happens. In the Java test, not the
  library.
- **47** `SignatureOptions.close` loses the first of two close failures, because
  its `finally` replaces the exception in flight.
- **48** `getAcroForm()` changes the document it is asked to read, and the port
  does so only when `pdmodel/fixup` is linked.

### The slice 8 adversarial review

The slice touches about 200 Java files, too many to read one after another and
stay honest about it. So the review was run as a set of sweeps, each of which
asks one question of every file at once and names the files it cannot answer
for. Where a sweep flagged something, that file was read against its Java.

**D7 — the keys.** Every `COSName` constant of `go/pdfbox/cos/names.go` was
compared against `COSName.java` by value. The two sets are identical: 586 Java
declarations, 588 Go constants, every Go value present in the Java and every
Java value present in the Go. The pairs that differ only in case — `CA`/`ca`,
`FL`/`Fl`, `OFF`/`Off`, `OP`/`op` — were then checked one at a time against
their Java declarations, since those are the ones a swap would hide in; all
four are right. `DCTDecodeAbbreviation` was removed: it was added during phase C
next to the `DCT` the port already had, and `DCT` is the convention its six
sibling filter abbreviations follow.

Then, for each slice 8 package, the set of keys the Go reads or writes was
compared against the set its Java counterpart uses, **both directions**. No
package uses a key its Java does not, apart from `interactive/form`'s
`/AcroForm`, which comes from the catalogue accessors demoted into it. No
package fails to use a key its Java does, apart from `documentinterchange`'s
`/OCG` and `/OCMD`, which moved with `PDPropertyList.create` into the
`optionalcontent` registry, and `common`'s five, which found the `PDStream` gap
below.

**D8 — `COSArrayList`.** Java sets `isFiltered` in one place, the
`(List, COSArray)` constructor when the two sizes differ, and refuses in six:
`remove(Object)`, `addAll(Collection)`, `addAll(int, Collection)`,
`set(int, E)`, `add(int, E)` and `remove(int)`. The port sets the flag in the
same place and refuses in the same six, with the same messages, as panics.
`add(E)`, `removeAll`, `retainAll` and `clear` carry no guard on either side —
so a filtered list still accepts a plain append in the Go exactly as it does in
the Java. Both filtered cases of `COSArrayListTest` are ported.

**Missing methods.** Every `public` or `protected` method of every Java class in
this slice was looked for in the Go tree. Twenty-three were not found; twenty-one
are renames the port made deliberately — `getRenderState` to `RenderStateFor`,
`valueOf` to `BaseStateValueOf`, the `getStringOrStream` and `createShortStyles`
kind that Java declares `protected` and the port keeps unexported, the two
`hashCode`s that Go has no contract for. Two were real: `PDStream`'s ten (below)
and `PDAcroForm.getFieldIterator`, now `FieldIterator`.

**Found: `PDStream` was missing ten public methods.** `getDecodeParms`,
`getFileDecodeParams`, `setDecodeParms`, `setFileDecodeParams`, `getFileFilters`,
`setFileFilters`, `getFile`, `setFile`, `getMetadata` and `setMetadata`, although
this file recorded the class as whole. All ten are ported.
`getFile` and `setFile` are `filespecification.FileOfStream` and
`SetFileOfStream`: `PDFileSpecification` is in `common/filespecification`, which
imports `common` for `PDEmbeddedFile`, so `common` cannot name it.

**Found: `PDDocument.addSignature` was not ported at all.** D9 asks whether
digital signatures verify or they do not, and the answer was that they could not
be attached. The signature model, `COSWriter.getDataToSign`, `SigningSupport`
and `SignatureOptions` were all here; the four `addSignature` overloads, their
five private helpers and `saveIncrementalForExternalSigning` were not, so
nothing could reach any of it. All of them are ported, as functions in
`interactive/form` — they name `PDAcroForm`, `PDSignatureField` and
`PDAnnotationWidget`.

`signverify_test.go` answers D9. It is not a port: PDFBox has no signing test in
the `pdfbox` module. It checks the two things that can be checked without a CMS
implementation, and checks them against the output file rather than against the
port's own arithmetic:

- the `/ByteRange` in the written file starts at 0, ends at the file length, and
  leaves a hole whose first byte is `<` and whose last is `>` — that is, the
  hole is the `/Contents` hex string and nothing else;
- the bytes the signer was handed are byte for byte the written file with that
  hole removed, and `PDSignature.getSignedContent` on the output answers the
  same bytes.

Both the `SignatureInterface` path and the external signing path pass. What the
test does not say is anything about the CMS blob: the port has no signature
algorithm and does not pretend to.

**D2 — dropped behaviour.** All 46 `catch` blocks and all three `finally` blocks
in the slice were read against their Go. Every one of the seventeen appearance
handlers logs its `IOException` and carries on in Java; every Go
`GenerateNormalAppearance` logs and returns nil, and none of them propagates an
error the Java would have swallowed. `PDNameTreeNode.calculateLimits`,
`PDStructureTreeRoot.getRoleMap`, `PDObjectReference.getReferencedObject`,
`PDAnnotationPopup.getParent`, `PDButton`'s `NumberFormatException`,
`PDAcroForm.exportFDF`'s close-and-rethrow, the four `FDFDictionary` parses and
the three fixup processors all match. Two divergences, both deliberate:

- `SignatureOptions.close`. Java's `finally` closes `pdfSource` after
  `visualSignature`, and a `finally` that throws replaces the exception in
  flight, so Java surfaces the *later* failure. The port keeps the first, which
  is the convention every other `Close` in the port follows. Both are closed
  either way; only which of two close failures is reported differs. JAVA-BUGS 47.
- `FDFDocument.saveXFDF` closes the writer it is given and the port does not,
  because a Go `io.Writer` has nothing to close. The doc comment says so.

The 71 unchecked `throw` sites were counted per package against the port's
`panic` sites. The Go count is greater than or equal to the Java count in every
package, which is what it should be: the port also panics where Java's implicit
casts throw `ClassCastException`. Two `panic`s in `fixup` have no Java throw
behind them and say why — `PDType1Font(FontName)` declares no exception, so a
missing standard 14 metric is a library bug there too.

Java's narrowing casts were checked where they matter. `CloudyBorder`'s two
`(int) Math.ceil(...)` and every `(float)(double expression)` in the line, free
text, polyline and strikeout handlers narrow at exactly the same point in the Go
— `float32(float64(y) * math.Sin(angle))`, not `float32(y) * float32(...)`. The
one difference that remains is at the edges: Java's `(int)` of an out-of-range
double saturates and Go's conversion is undefined. Every such site here is a
count derived from a page geometry and guarded by the caller.

Java has no inclusive-bound `for` loop anywhere in this slice; the port has one
`for fs <= appearanceDefaultFontSize`, which is
`AppearanceGeneratorHelper.calculateFontSize`'s `while (fs <= DEFAULT_FONT_SIZE)`.

Every `TODO` in the ported code is Java's own, carried across.

**D3 — the tests.** Every test file in this slice names the Java test it came
from in its header, or says that it is not a port. Every `@Test` method of every
Java test class in the slice was looked for in the Go; twenty-three were not
found, and each was accounted for:

- Fourteen belong to the four classes recorded above as not ported.
- Six are covered by a Go table test under one name —
  `PDImmutableRectangleTest`'s four setters, the two tree-node limit pairs.
- `PDOutlineNodeTest.openNodeAndAppend` has an empty `// TODO` body in the Java.
- `PDFieldTest.testHashCode` asserts equal hash codes, and Go has no `hashCode`
  contract; `Equals` is ported and `TestFieldEquals` covers it.
- `TestCheckBox.testPDFBox6207` reads `target/pdfs`, which the Maven build
  downloads.
- **`PDChoiceTest.getOptionsFromMixed` was simply missing**, and is ported now.

The three that stay unported now say so in the header of the file a reader would
look in.

Then every string literal of five characters or more in every slice 8 test was
looked for in the Java test tree, to catch an assertion whose expected value had
been read off the port rather than copied from the Java. Every literal that is
an asserted value is in the Java; the ones that are not are Go method names used
as labels and assertion messages. `ControlCharacterTest`'s two separators are
the real U+2028 and U+2029, not the spaces they look like.

`AppearanceGenerationTest` deserves its own line, because it is the test in this
slice most able to become a tautology: it does not assert any value at all. It
generates appearances for the annotations of a fixture, and compares the token
stream against the appearance PDFBox itself wrote into the same file. Numbers
are compared to within a tolerance the Java sets; operators must match exactly.

**D4 — the deferrals.** Every "not ported" and "is not here" in the slice's doc
comments was checked against this file. All were recorded except two, now added:
`PDStream.createInputStream(DecodeOptions)`, which needs `cos.Stream` to have an
options-taking reader, and `PDPageContentStream`'s five deprecated
`appendRawCommands`, which is a choice rather than a gap. Every remaining
deferral is blocked on a type a later slice brings — `PDShading`,
`PDTilingPattern`, the font embedders, an ICC engine, the four unported halves
of the resource cache — none on difficulty.

**D5 — the Java bugs.** All of this slice's entries were re-read against the Go
site that carries them. Entries 35 to 45 are carried: the `BEAD` type name, the
`/D`-for-`/O` write, `setSuspect`'s ignored argument, the three unchecked `/P`
reads, the `insertBefore` index of -1, the name-array read of a string array,
the five-entry padding, the two `/Reasons` name reads, `getRotation`'s string
read of an integer and `getPages`' unresolved reference. Three are not, and each
says so in its own "Where the Go carries it" line rather than claiming
otherwise: entry 42, because Java's `StandardStructureTypes` fills its list by
reflection over its own fields and picks the list itself up, and Go has no such
reflection to reproduce; entry 47, where the port keeps the first of two close
failures; and entry 48, where the port applies the mutating fixup only when
`pdmodel/fixup` is linked. Nothing was fixed on the way past.

**Still open.** Nothing found in this review is unfixed. What remains absent is
the deferral list above, and this slice's tests do not cover the appearance
handlers' output against a renderer — that comparison is slice 9's, and
`AppearanceGenerationTest`'s two rendering cases wait for it.

### The slice 8 feedback

Six items, all from the pull request review. Two were typos in text the port
wrote; four were behaviour. Three of the four were port defects and are fixed,
each behind a test that fails without the fix. The fourth is a deviation that
turns out to be unreachable, and is recorded here rather than changed.

**Fixed — `PDNameTreeNode.getValue` could not tell an absent limit from an
empty one.** Java's `getUpperLimit` and `getLowerLimit` answer `null` where the
child has no `/Limits`, and `getValue` treats a null limit as "this child may
hold anything": it descends and returns whatever the child answers, without
looking at the children after it. The port's accessors answer `""` for an
absent limit *and* for the empty name, which is a legal limit and sorts before
every other. A tree whose first child covers `["" "a"]` therefore swallowed
every lookup, so a name in a later child came back as not found. `Value` now
reads the `/Limits` array through `limitOf`, which reports presence separately.
`TestValueSkipsAChildWhoseLowerLimitIsTheEmptyName`.

**Fixed — `DateConverter` accepted a second of 60 or 61.** `parseBigEndianDate`
builds its result on a `GregorianCalendar` with leniency off, whose maximum for
`SECOND` is 59; the 0-to-61 leap-second range belongs to `java.util.Date`, not
to `Calendar`. The port's bound was 61, and `time.Date` normalises rather than
refusing, so `D:20200101120060Z` was read as 12:01:00 — a malformed date
silently becoming a different valid instant. The bound is 59.
`TestSecondsBeyond59AreRefused`.

**Fixed — `PlainText` kept one paragraph too many.** The constructor is
`textValue.replace('\t',' ').split("\R")`, and `String.split` with a limit of
zero drops *every* trailing empty result, not all but one: `Pattern.split`
builds `["", ""]` for `"\n"` and strips both, leaving no paragraphs. The port
stopped stripping at one entry and kept an empty paragraph, which the
constructor then turns into a space — drawing a space where PDFBox draws
nothing. The strip now runs to zero, and the branch above it answers the value
whole where the regex never matched, which is what `Pattern.split` returns
before it strips anything. `TestValueOfOnlyLineBreaksHasNoParagraphs` and
`TestTrailingLineBreaksAreDropped`.

**Recorded, not changed — the comb field walks runes where Java walks UTF-16
code units.** `insertGeneratedCombAppearance` takes each cell with
`value.substring(i, i+1)` in Java and one rune in the port, and counts the cells
with `value.length()` against `len([]rune(value))`. The two agree for every
character in the basic plane and differ for one outside it, where Java splits
the surrogate pair across two cells — a different cell count, a different
alignment for a centred or right-aligned field, and two half-glyphs instead of
one.

It is not reachable. Both sides measure each cell through
`PDFont.getStringWidth` before drawing it, Java asking the font for the lone
surrogate `U+D83D` and the port for the whole `U+1F600`, and no font this port
can build has either — the embedders that would produce one are not ported. Both
refuse the value rather than laying it out.
`TestCombFieldRefusesASupplementaryCharacter` asserts that the port refuses, so
that the claim fails loudly rather than silently if the encoder ever starts
accepting such a character. Changing the walk would mean decoding each UTF-16
unit back to a Go string, and a lone surrogate becomes U+FFFD there because a Go
string cannot hold one — a behaviour Java does not have, bought for a case
neither side reaches.

**The two typos.** `PDListAttributeObject`'s doc comment named a type that does
not exist, and `PDSignatureField.SetValue`'s panic message had lost its
apostrophe to an editing slip. The message otherwise stays as Java writes it,
naming `setValue(PDSignature value)`: the port copies Java's exception text, and
the Go name it points at is on the method two lines below.

---

## Slice 9 — rendering

Branch `slice/9-rendering`. The last slice of the plan, and the one `PLAN.md`
left a decision in front of: PDFBox draws through `java.awt.Graphics2D`, and Go
has nothing equivalent.

**The decision, taken as B0, is `PLAN.md`'s third option: port the geometry,
defer the raster backend behind an interface.** Everything that computes is
ported and runs. Only the last drawing step is behind `rendering.Backend`, and
**no implementation of that interface ships**. What that costs is written out
under "What the raster decision costs" below, rather than left implicit.

Slice 8 had not landed on `migration-base` when this branch started, and eleven
types it ports are named directly by `pdfbox/rendering`.
`slice/8-forms-annotations` is merged into this branch, so the content is
byte-identical and the eventual merge has nothing to reconcile.

### `java.awt.geom.Area` — the JDK, not PDFBox

| Java source | Go source | Status |
| --- | --- | --- |
| `java.awt.geom.Area` | `awt/geom/area.go` | done, minus curves |

Slice 2 recorded `Area` as the one thing blocking
`PDGraphicsState.getCurrentClippingPath`. It is constructive area geometry:
`add`, `subtract`, `intersect`, `exclusiveOr`, `contains`, `getBounds2D`,
`transform` and a `PathIterator` over the result. The port splits every boundary
edge at its crossings **and at its T-junctions**, classifies each piece by
offsetting perpendicular from its midpoint, and chains the pieces it keeps into
rings.

**The one deviation is curves: a shape is flattened when it becomes an `Area`,**
so the result is a polygon where the JDK's would still be a curve. It is
documented on the type. Nothing in PDFBox reads the curves back out of an
`Area` — every use is a clip, which is rasterised — so the visible effect is the
flattening tolerance, not a different region.

The T-junction split is not a corner case.
`PDGraphicsState.getCurrentClippingPath` starts from the bounding box of the
clipping paths and intersects each path into it, so the **first** intersection
is always tangent: the path touches its own bounding box on all four sides. A
crossing test alone cannot see that, and without the split the intersection
answers the bounding box instead of the shape. The tests written from the JDK
contract caught it.

### `pdmodel/common/function` and `graphics/color`

Both were already ported when this branch started — the functions with the file
type detector, the colour spaces across slices 2, 3 and 6 — so B2 and B3 had
nothing left to do. The scope table in `slice-9-rendering.md` counts them
because `PLAN.md` counts them; they are not slice 9's work.

### `pdmodel/graphics/shading` — the model half of all seven types

| Java source | Go source | Status |
| --- | --- | --- |
| `PDShading.java` | `shading/pdshading.go` | done — `Shading` is the interface, `PDShading` the shared state |
| `PDShadingType1/2/3.java` | `shading/pdshadingtype123.go` | done |
| `PDTriangleBasedShadingType`, `PDShadingType4/5.java` | `shading/pdshadingtype45.go` | done |
| `PDMeshBasedShadingType`, `PDShadingType6/7.java` | `shading/pdshadingtype67.go` | done |
| `Patch`, `CoonsPatch`, `TensorPatch`, `CubicBezierCurve` | `shading/patch.go` | done — unexported, as Java's are package-private |
| `Vertex`, `Line`, `ShadedTriangle`, `CoordinateColorPair` | `shading/triangle.go` | done — unexported |
| the 19 `*Paint` and `*Context` classes | — | **not ported** — see below |

The nineteen that are missing are `AxialShadingPaint`, `AxialShadingContext`,
`RadialShadingPaint`, `RadialShadingContext`, `Type1ShadingPaint`,
`Type1ShadingContext`, `Type4ShadingPaint`, `Type4ShadingContext`,
`Type5ShadingPaint`, `Type5ShadingContext`, `Type6ShadingPaint`,
`Type6ShadingContext`, `Type7ShadingPaint`, `Type7ShadingContext`,
`ShadingPaint`, `ShadingContext`, `TriangleBasedShadingContext`,
`GouraudShadingContext` and `PatchMeshesShadingContext`. Each is a
`java.awt.Paint` or a `java.awt.PaintContext` that fills a raster. The colour
evaluation they call into — the function, the colour space conversion, the
patch subdivision, the triangle interpolation — is here.

### `pdmodel/graphics/pattern`

| Java source | Go source | Status |
| --- | --- | --- |
| `PDAbstractPattern.java`, `PDTilingPattern.java` | `pattern/pattern.go` | done |
| `PDShadingPattern.java` | `pattern/pattern.go` | done |
| `color/PDPattern.java` | `pattern/pdpattern.go` | done — **in this package, not `graphics/color`** |

`PDPattern` is a `PDColorSpace` and Java puts it in `graphics/color`. It cannot
go there: it reads a `PDAbstractPattern` out of the resources, so it would make
`color` import `pattern`, which imports `color` for the underlying colour space.
The colour space lives with the patterns it names, and `color.Create` reaches it
through the `NewPatternColorSpace` hook this package sets from its `init`.

### `contentstream` — the graphics engine and its operators

| Java source | Go source | Status |
| --- | --- | --- |
| `PDFGraphicsStreamEngine.java` | `contentstream/graphicsstreamengine.go` | done, minus the operator registrations |
| `operator/graphics` — all 23 | `contentstream/operator/graphics/graphics.go` | done |
| `operator/color` — all 13 | `contentstream/operator/color/color.go` | done |
| `operator/DrawObject.java` | `contentstream/drawobject.go` | done |
| `operator/markedcontent/DrawObject.java` | `operator/markedcontent/markedcontent.go` | done |
| `operator/state/SetGraphicsStateParameters.java` | `operator/state/state.go` | done — slice 8 ported it, waiting on `PDExtendedGraphicsState` |

`PDFGraphicsStreamEngine`'s constructor registers sixty operators by name. The
port's cannot: every processor holds the engine, so the operator packages import
`contentstream` and it cannot import them back. `rendering.addAllOperators` is
that list, called from `NewPageDrawer`, the way `text.NewLegacyPDFStreamEngine`
already registers its own.

Java has **three** `DrawObject` processors, one per engine, and slice 3 deferred
two of them on `PDXObject`. All three are here now. Without the plain one the
text extractor never walked into a form XObject, so text inside one was silently
lost. The plain one lives in `contentstream` rather than
`contentstream/operator`, where Java has it: the port's `operator` package holds
no processors, because a processor names the engine and the engine's package
imports `operator`.

`PDFStreamEngine`'s remaining half came with them: `showForm`,
`showTransparencyGroup`, `processSoftMask`, `processTransparencyGroup`, the two
`processTilingPattern` overloads, `processChildStream`, `showAnnotation`,
`getAppearance` and `processAnnotation`. So did the two arms of
`processStreamOperators` that clear `shouldProcessColorOperators` — an uncoloured
tiling pattern, and a Type 3 char proc whose first operator is `d1` — which slice
2 recorded as unreachable.

`PDFormXObject` and `PDAppearanceStream` implement `PDContentStream` in Java and
cannot here: `getResources` answers a `PDResources`, which lives in `pdmodel`,
and `graphics/form` cannot import it. `contentstream` adapts both, the way it
already adapts `PDType3CharProc`.

The two `IllegalStateException`s `PDFStreamEngine` throws for a child stream
processed without a page are errors here rather than panics: every caller of
those methods is an operator, and an operator's errors already travel back
through `processOperator`, which is where a PDF that asks for a form outside a
page has to be dealt with.

### `pdmodel/PDResources` and `DefaultResourceCache` — finished

`getXObject` with `isAllowedCache`, `getShading` and `getPattern` are ported, and
with `getExtGState` and `getProperties` from slice 8 the family is complete. So
are `add` and `put` for a shading and for a pattern, which neither branch had.

`DefaultResourceCache` gains its five remaining halves — XObjects, shadings,
patterns, extended graphics states and property lists — with the stable-cache
bookkeeping Java repeats per kind written once, as a generic map.

**Two port defects were fixed on the way.** The colour space half (slice 2) and
the extended graphics state half (slice 8) both keyed their maps on the
stable-cache hash rather than on the `COSObject`. With the stable cache disabled
that hash is unavailable, so neither kind was cached at all; with it enabled,
two objects sharing a hash collided. Java keys the map on the `COSObject` and
uses the hash only for the removal bookkeeping. All eight kinds do now.

`PDDocument` gains a `CreateStream`, which is what makes it a
`common.COSDocumentLike`. Java has no such method — everything that wants a
stream goes through `getDocument().createCOSStream()`, and the constructors that
take a `PDDocument` do that themselves — but the port's `COSDocument` answers the
narrow interface a security handler needs, so the one method is what lets a
`*PDDocument` be passed where `new PDStream(document)` takes one.

### `pdfbox/rendering`

| Java source | Go source | Status |
| --- | --- | --- |
| `ImageType.java` | `rendering/imagetype.go` | done — minus `toBufferedImageType` |
| `RenderDestination.java` | `pdmodel/graphics/optionalcontent/renderdestination.go` | done — **declared there**, aliased in `rendering` |
| `PageDrawerParameters.java` | `rendering/pagedrawerparameters.go` | done |
| `GlyphCache.java` | `rendering/glyphcache.go` | done |
| `PDFRenderer.java` | `rendering/pdfrenderer.go` | done — minus the `BufferedImage` it makes |
| `PageDrawer.java` | `rendering/pagedrawer.go`, `pagedrawer_oc.go` | done — minus four raster pieces |
| `GroupGraphics.java` | — | not ported — a `Graphics2D` subclass |
| `SoftMask.java` | — | not ported — a `java.awt.Paint` |
| `TilingPaint.java`, `TilingPaintFactory.java` | — | not ported — a `java.awt.Paint` and its cache |

**`RenderDestination` had to move.** Java's `rendering` imports
`graphics/optionalcontent` for the groups, and `optionalcontent` imports
`rendering` back for `getRenderState(RenderDestination)`. Java allows the cycle
and Go does not. Slice 8 put the enum in `rendering`, which worked while
`rendering` held nothing else; slice 9's `rendering` must import
`contentstream`, and through it `pdmodel` and `optionalcontent`. The enum is
declared in `optionalcontent` and `rendering` aliases it back to the Java name in
the Java place, the way `pdmodel.ResourceCache` aliases `pdmodel/font`'s.

**`PageDrawer` keeps every decision.** Which paint applies to a colour, what the
stroke is made of, what the clip intersects to, whether a path is rectangular,
how thin a clip may be before it is widened, whether an optional content group is
visible at this destination, whether an annotation is skipped, how far an image
may be subsampled, whether a transparency group needs its backdrop.

Four pieces of it are raster work end to end and are not ported:

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

`adjustClip` asks `AffineTransform.getType()` in Java, a bitmask the port does
not have. The two tests it makes — "translation and flip only" and "no shear or
rotation" — are written out against the matrix, with the Java bits named.

### `pdfbox/printing`

| Java source | Go source | Status |
| --- | --- | --- |
| `Orientation.java`, `Scaling.java` | `printing/printing.go` | done |
| `PDFPrintable.java` | `printing/pdfprintable.go` | done — minus rasterizing |
| `PDFPageable.java` | `printing/pdfpageable.go` | done |
| `java.awt.print.Paper`, `PageFormat` | `printing/pageformat.go` | the state only |

Go has no print system. What is ported is what the two classes compute: the
rotated crop and media boxes, the portrait-normalised paper of the PDFBOX-2922
workaround, the scale-to-fit arithmetic, the centering and its negative-value
guard, and the page border. `java.awt.print.Printable` and `Pageable` become
plain methods taking a `rendering.Backend`.

Rasterizing a page to a bitmap before printing it answers
`ErrRasterizeUnsupported`. Java makes a `BufferedImage` of the imageable area,
renders into it and blits it; there is nothing to make that image with.

### What the raster decision costs

Written down here rather than left implicit, which is what D9 asks.

**The port cannot produce a rendered page.** `PDFRenderer.RenderImage` and its
four siblings answer `ErrNoBackend` — with the size, the type and the page they
worked out, so the error says what would have been made. It is deliberately not
a blank image, which would look like a rendered page.

**It therefore cannot rasterise, print, or run PDFBox's own image comparisons.**
`TestPDFToImage`, `TestRendering` and `TestQuality` all compare against
reference PNGs. `PDFPrintable` prints as vectors onto a backend and refuses to
rasterise.

**Everything above the interface runs.** A caller with a backend of their own —
`golang.org/x/image/vector` plus a compositor, a Cairo or Skia binding, an SVG
or PDF writer — gets a complete renderer: the whole content stream is walked,
every operator is processed, the colours are converted, the shadings evaluate,
the clip is computed, the optional content is resolved, the annotations are
placed. `Backend` is fifteen methods.

**What a backend has to do that the port does not describe for it:** anti-aliased
scan conversion of a path under a winding rule; stroking a path into an outline
with caps, joins, a miter limit and a dash pattern; sampling an image through an
arbitrary transform with the two interpolations; compositing a layer under a
blend mode, an alpha constant and a soft mask; and turning each of the four
`Paint` descriptions into pixels — which for a tiling pattern means calling back
into `PageDrawer.DrawTilingPattern` for one tile, and for a shading means asking
the shading for the colour at a point.

**What the tests compare instead of pixels** is the A5 decision: `Area` against
the JDK's documented contract, functions and colour conversions against values
taken from the Java, and `PageDrawer` against a backend that records every call.
An image comparison against PDFBox's reference PNGs stays possible once a
backend exists, and is the thing this strategy does not cover.

### The tests

Java's three rendering tests and one printing test do not port as they stand.

- **`TestPDFToImage`** is disabled in Java itself, because different JVMs
  produce different images.
- **`TestRendering`** renders twenty files and asserts that nothing threw.
  Without a rasteriser there is nothing to render.
- **`TestQuality`** reads back four pixels of four files from `target/pdfs`,
  which the build downloads.
- **`TestPDFPrintable`** has five cases: three port as they stand — the page
  index, the printer state left unchanged, and the result codes — and two read
  back pixels to see whether the page border came out grey.

What each was asking is asked instead of a backend that records every call the
drawer makes, over real content streams through the real engine:
`rendering/pagedrawer_test.go`, `rendering/pdfrenderer_test.go` and
`printing/printing_test.go`. The colour a rectangle is filled in, the stroke
parameters a line carries and the three values `getStroke` invents, the clip `W`
leaves behind, the group a transparency form pushes, the operators a hidden
optional content group swallows, the transform a rotated page installs, the
scale each of the four scaling modes chooses.

### The adversarial review

Mechanical sweeps, each answering a question the ported tests cannot.

- **Every Java type in the slice's packages against a Go counterpart.** Clean
  apart from the raster classes named above; the package-private geometry
  classes of `shading` are all present as unexported Go types.
- **Public and protected method presence**, class by class, for `PDFRenderer`,
  `PageDrawer`, `PDFPrintable` and `PDFPageable`. One gap: `PageDrawer.setClip`
  is `protected final` in Java, and the port had it unexported. Java's javadoc
  says an embedder overriding `showGlyph` may need it, so it is exported now.
- **Every `COSName` used, against the Java constant** rather than against the
  specification. Sixteen names, all matching.
- **Every `finally`.** The five in `PageDrawer` and `PDFStreamEngine` that have
  one are ported as unconditional restores. The two places where Java has **no**
  `finally` and restores with plain statements are ported as written and
  recorded as Java bugs 49 and 50.
- **Java's narrowing conversions.** Two were wrong and are fixed:
  `Math.round(float)` is `floor(x + 0.5)`, which rounds a half towards positive
  infinity, where Go's `math.Round` rounds away from zero — so
  `Math.abs(Math.round(x))` differed for a negative half; and
  `getSubsampling`'s `imageWidth * imageHeight` is an `int` product that wraps
  at 2^31, which a Go `int` does not.
- **Every deferral.** Each is a `java.awt` raster type, and each is named in a
  row above.

Three things found while writing rather than by a sweep:

- **`PDFRenderer.transform` concatenates onto the transform the `Graphics2D`
  already carries.** The port built it from the identity and installed it,
  which discarded `PDFPrintable`'s translate to the imageable area and its
  centering — every printed page would have landed in the top left corner of
  the paper. The scaling and centering tests catch it.
- **`applySoftMaskToPaint` throws for a soft mask whose subtype is neither
  `/Alpha` nor `/Luminosity`.** The port logged and carried on; it returns the
  error now.
- **`PDExtendedGraphicsState.CopyIntoGraphicsState`'s doc comment** still said
  the `/SMask` arm was not applied, three commits after it was.
