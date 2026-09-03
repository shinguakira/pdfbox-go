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
| 1 | `pdfbox/cos` | 24 | in progress — 4 of 24 ported |
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

| Java source | Go source | Status |
| --- | --- | --- |
| `COSBase.java` | `base.go` | done — minus `getKey`/`setKey`, which need `ObjectKey` |
| `ICOSVisitor.java` | `visitor.go` | partial — 2 of 11 methods, grows with each type |
| `COSBoolean.java` | `boolean.go` | done |
| `COSNull.java` | `null.go` | done |
| `COSObjectKey.java` | — | next |
| `COSName.java` | — | not started |
| `COSInteger.java`, `COSFloat.java`, `COSNumber.java` | — | not started |
| `COSString.java` | — | not started |
| `COSArray.java`, `COSDictionary.java` | — | not started |
| `COSStream.java`, `COSDocument.java`, `COSObject.java` | — | not started — need the io layer and filters |
| the remaining 8 files | — | not started |

### Ported tests — `cos`

| Java test | Go test | Notes |
| --- | --- | --- |
| `TestCOSBase` | `base_test.go` | abstract in Java; becomes `assertBaseContract`, called per type |
| `TestCOSBoolean` | `boolean_test.go` | complete except the COSWriter byte assertions |
| — | `null_test.go` | Java has no `TestCOSNull`; written from `COSNull.java` per the tdd rule |

### Deviations — `cos`

- **`accept()` is tested through a recording visitor, not `COSWriter`.** The
  Java tests drive a `COSWriter` and assert the emitted bytes. `COSWriter` is
  `pdfwriter`, not ported. The port asserts the double dispatch plus a direct
  `WritePDF` byte check. **The COSWriter assertions must be added when
  `pdfwriter` lands** — until then the serialised form is only checked against
  itself.
- `assertBytesEqual` compares lengths properly. The Java `testByteArrays` helper
  compares `byteArr1.length` against itself and so never checks lengths match.
- `Visitor` and `Base` are deliberately smaller than the Java interfaces; both
  say so in their doc comments, and both grow as types land.

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
