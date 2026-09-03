# Porting status

Hand maintained. Update the row for a package in the same commit that ports it.

The machine-generated counterpart is
[`mapping/inventory.tsv`](mapping/inventory.tsv), which knows how much Java
source sits behind each row but only distinguishes "has Go files" from "has
none". This file is where partial work and the reasons for it get recorded.

Status values: `done` · `in progress` · `blocked` · `not started` · `out of scope`

Last updated: 2026-09-03

## Summary

| Phase | Area | Java files | Status |
| --- | --- | ---: | --- |
| 0 | `pdfio` | 18 | in progress — 13 of 18 ported |
| 1 | `pdfbox/cos` | 24 | **19 of 24 — every file slice 1 needs**; the remaining 4 are slice 7 incremental-save machinery, plus 1 folded away |
| 2 | `filter`, `pdfparser`, `pdfwriter` | 48 | in progress — `filter` has the slice 1 subset, `pdfparser` 12 of 18 |
| 3 | `pdfbox/pdmodel` | 433 | in progress — the slice 2 subset: page, page tree, resources, rectangle, graphics state, colour |
| 4 | `fontbox` | 143 | in progress — `BoundingBox` only, pulled in by `PDRectangle` |
| 5 | `contentstream`, `text` | 85 | in progress — the engine, the dispatch, and the operators needing no font |
| — | `awt/geom` (the JDK, not PDFBox) | — | in progress — `Point2D`, `AffineTransform`, `Path2D`, `Rectangle2D` |
| 6 | `rendering`, `printing`, `shading` | 60 | not started — needs a rasteriser decision |
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
| `common/PDStream.java` | — | not started — needs `COSArrayList`, `COSDictionaryMap`, `PDMetadata` and a file specification |
| `common/COSArrayList.java` | — | not started — its Java test needs annotations |
| `PDResources.java` | `pdresources.go` | partial — the dictionary plumbing; every typed getter waits on the type it returns |
| `ResourceCache.java` | — | not started — every method is typed on a font, colour space, shading, pattern or XObject |
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
| `state/PDTextState.java` | `graphics/state/pdtextstate.go` | done — minus the font, which needs `PDFont` |
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
