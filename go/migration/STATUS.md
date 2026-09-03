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
| 1 | `pdfbox/cos` | 24 | in progress — 15 of 24; 4 deferred to slice 7, 4 blocked on filter |
| 2 | `filter`, `pdfparser`, `pdfwriter` | 48 | not started |
| 3 | `pdfbox/pdmodel` | 433 | not started |
| 4 | `fontbox` | 143 | not started |
| 5 | `contentstream`, `text` | 85 | not started |
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

15 of 24 files ported.

| Java source | Go source | Status |
| --- | --- | --- |
| `COSBase.java` | `base.go` | done |
| `ICOSVisitor.java` | `visitor.go` | partial — 8 of 11 methods; grows with each type |
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
| `COSStream.java` | — | blocked on `pdfbox/filter` |
| `COSInputStream.java` | — | blocked on `pdfbox/filter` |
| `COSOutputStream.java` | — | blocked on `pdfbox/filter` |
| `COSDocument.java` | — | blocked on `COSStream` |
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
- `Integer.Equals` compares the full `int64`. Java compares `intValue()`, which
  truncates to 32 bits.
- `ParseHexString` always fails on malformed hex. Java has a `FORCE_PARSING`
  mode, read from a JVM system property, that substitutes `'?'` instead.
- `cosEqual` dispatches to the `Equals` method of the types that define one.
  Java relies on every class overriding `equals`; Go has no single such hook,
  and every container comparison goes through this one function.
- `assertBytesEqual` compares lengths properly. The Java `testByteArrays` helper
  compares `byteArr1.length` against itself and so never checks them.
- `Visitor` and the ported types are deliberately smaller than the Java
  originals where a dependency is not ported yet; each says so in its doc
  comment.

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
