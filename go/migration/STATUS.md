# Porting status

Hand maintained. Update the row for a package in the same commit that ports it.

The machine-generated counterpart is
[`mapping/inventory.tsv`](mapping/inventory.tsv), which knows how much Java
source sits behind each row but only distinguishes "has Go files" from "has
none". This file is where partial work and the reasons for it get recorded.

Status values: `done` · `in progress` · `blocked` · `not started` · `out of scope`

Last updated: 2026-09-04

## Summary

| Phase | Area | Java files | Status |
| --- | --- | ---: | --- |
| 0 | `pdfio` | 18 | in progress — 13 of 18 ported |
| 1 | `pdfbox/cos` | 24 | **19 of 24 — every file slice 1 needs**; the remaining 4 are slice 7 incremental-save machinery, plus 1 folded away |
| 2 | `filter`, `pdfparser`, `pdfwriter` | 48 | in progress — `filter` has the slice 1 subset, `pdfparser` 17 of 18; `pdfwriter` not started |
| 3 | `pdfbox/pdmodel` | 433 | in progress — slice 2 subset, `pdmodel/font` at 34 of 39 (the 5 left are the slice 7 embedders), all 12 encodings, and `pdmodel/encryption` at 17 of 19 |
| 4 | `fontbox` | 143 | **done — all 143 files**, finished by slice 4 |
| 5 | `contentstream`, `text` | 85 | in progress — the engine and every text operator; `text` has all 6 files, minus what needs a document |
| — | `awt/geom` (the JDK, not PDFBox) | — | in progress — `Point2D`, `AffineTransform`, `Path2D`, `Rectangle2D` |
| 6 | `rendering`, `printing`, `shading` | 60 | not started — needs a rasteriser decision |
| — | `pdfbox` root (`Loader`) | 1 | done — the reading entry points |
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
| `COSStream.java` | `stream.go` | done — minus the update state; **written before its test**, see the method note |
| `COSInputStream.java` | — | not ported — `Stream.CreateReader` returns a plain `io.Reader`; the class exists in Java only to carry a `DecodeResult` |
| `COSOutputStream.java` | — | not ported — folded into `streamWriter` in `stream.go` |
| `COSDocument.java` | `document.go` | done — minus the document state |
| `COSDocumentState.java` | — | deferred to slice 7 — incremental save |
| `COSUpdateInfo.java` | — | deferred to slice 7 — incremental save |
| `COSUpdateState.java` | — | deferred to slice 7 — incremental save |
| `COSIncrement.java` | — | deferred to slice 7 — incremental save |

### Ported tests — `cos`

| Java test | Go test | Notes |
| --- | --- | --- |
| `TestCOSBase` | `base_test.go` | abstract in Java; becomes `assertBaseContract`, called per type |
| `TestCOSBoolean` | `boolean_test.go` | complete except the COSWriter byte assertions |
| `COSObjectKeyTest` | `objectkey_test.go` | `testPDFBox5742` not ported — needs parser, writer, multipdf and a renderer |
| `TestCOSName` | `name_test.go` | all three Java tests drive documents; their real assertions (`/m#E4nnlich`, `/m#00nnlich`, PDFBOX-4076) are asserted against `WritePDF` directly |
| `TestCOSNumber` | `number_test.go` | complete |
| `TestCOSInteger` | `integer_test.go` | complete |
| `TestCOSFloat` | `float_test.go` | complete |
| `TestCOSString` | `string_test.go` | COSWriter serialisation assertions replaced by checks on `Bytes`, `ToHexString` and `ForceHexForm` |
| `PDFDocEncodingTest` | `pdfdocencoding_test.go` | complete, including PDFBOX-3864 |
| `TestCOSArray` | `array_test.go` | complete |
| `COSDictionaryTest` | `dictionary_test.go` | `testCOSDictionaryNotEqualsCOSStream` needs `COSStream`; the identity semantics it guards are covered |
| `UnmodifiableCOSDictionaryTest` | `unmodifiable_test.go` | every assertion is a compile error in Go, so nothing is left to assert at run time |
| — | `null_test.go`, `object_test.go` | Java has no test for these; written from the source per the tdd rule |

### Deviations — `cos`

**Open debt — must be closed when `pdfwriter` lands.** The Java `accept()` tests
drive a `COSWriter` and assert the emitted bytes. `COSWriter` is not ported, so
the port asserts the visitor double dispatch plus a direct `WritePDF` byte
check. **Until then the serialised form is only checked against itself.** This
applies to `boolean_test.go`, `integer_test.go`, `float_test.go` and
`string_test.go`.

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
| `java.awt.geom.Area` | — | **not started** — constructive area geometry, needed only to combine clipping paths, which is the renderer's job |

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
| `common/PDStream.java` | `common/pdstream.go` | partial — the reading path the fonts need; the writing constructors, the decode parameters, the file specification and the metadata still need `COSArrayList`, `COSDictionaryMap`, `PDMetadata` and a file specification |
| `common/COSArrayList.java` | — | not started — its Java test needs annotations |
| `PDResources.java` | `pdresources.go` | partial — the dictionary plumbing and `getFont` with its direct cache; `getColorSpace`, `getExtGState`, `getShading`, `getPattern`, `getProperties` and `getXObject` still wait on the type each returns |
| `ResourceCache.java` | `pdmodel/font/resourcecache.go`, aliased in `resourcecache.go` | partial — the font and font descriptor members; the colour space, graphics state, shading, pattern, property list and XObject members wait on their types. The interface is declared in `pdmodel/font` because it names `PDFont` and `pdmodel` imports that package |
| `DefaultResourceCache.java` | `resourcecache.go` | partial — the font and font descriptor halves, including the stable-cache bookkeeping. Java holds each entry through a `SoftReference`; Go has none, so the port holds them outright |
| `PDPage.java` | `pdpage.go` | partial — boxes, rotation, resources, contents; annotations, thread beads, transitions, actions, viewports, metadata and the `PDStream` methods are absent |
| `PDPageTree.java` | `pdpagetree.go` | done — minus the `PDDocument` the reading constructor takes, which is only there to reach a `ResourceCache` |
| `MissingResourceException.java` | `errors.go` | done |
| `PDDocument.java`, `PDDocumentCatalog.java`, `PDDocumentInformation.java` | — | not started |

`PDPage.getContentsForStreamParsing` is the general path for now. Its fast path
decodes a single flate stream as it is read, which needs
`FlateFilterDecoderStream` and `NonSeekableRandomAccessReadInputStream`, neither
of which is ported.

### `pdfbox/pdmodel/graphics`

| Java source | Go source | Status |
| --- | --- | --- |
| `PDLineDashPattern.java` | `graphics/pdlinedashpattern.go` | done |
| `blend/BlendMode.java` | `graphics/blend/blendmode.go` | done — blend functions included |
| `blend/BlendComposite.java`, `SoftMask.java` | — | not started — rendering |
| `color/PDColor.java` | `graphics/color/pdcolor.go` | done |
| `color/PDColorSpace.java` | `graphics/color/colorspace.go` | partial — as an interface; the static `create` methods and the two `BufferedImage` methods are absent |
| `color/PDDeviceColorSpace.java` | `graphics/color/colorspace.go` | done |
| `color/PDDeviceGray.java` | `graphics/color/devicegray.go` | done — minus `toRGBImage` |
| the other 20 colour spaces | — | not started |
| `state/RenderingIntent.java` | `graphics/state/renderingintent.go` | done — `RenderingIntentTest` ported |
| `state/RenderingMode.java` | `graphics/state/renderingmode.go` | done |
| `state/PDTextState.java` | `graphics/state/pdtextstate.go` | done |
| `state/PDGraphicsState.java` | `graphics/state/pdgraphicsstate.go` | partial — minus the soft mask, `getCurrentClippingPath` and the Area form of `intersectClippingPath`, and the two Java composites |
| `state/PDSoftMask.java` | — | not started — needs a transparency group and a function |
| `state/PDExtendedGraphicsState.java` | — | not started — reads fonts, soft masks and dash patterns out of a dictionary |

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
| `PDDocumentInformation.java` | `pdmodel/pddocument.go` | partial — minus the dates, which need the COS date parsing slice 1 left out |
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
