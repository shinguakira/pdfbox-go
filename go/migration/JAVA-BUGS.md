# Java bugs found while porting

Defects and surprising behaviour noticed in the Java source during the port.

**Nothing here is fixed, in the Java or in the Go.** The port reproduces every
one of these, because the Java is the reference and a silently corrected bug
makes the Go behave differently from the thing it exists to reproduce. See
[`conventions/java-to-go.md`](conventions/java-to-go.md).

This file exists so the knowledge is not lost. A reader who later finds the Go
behaving oddly can look here and see that the Java does the same thing, on
purpose, and that it was noticed rather than missed.

**Do not report any of this upstream.** `AGENTS.md` forbids agents from filing
findings to a public tracker, and this repository has no relationship with
Apache PDFBox — see [`BRANCHING.md`](BRANCHING.md).

## How to add an entry

One entry per defect, with: where it is, what the Java does, what correct would
be, where the Go carries it, and how sure you are. Confidence matters — "looks
wrong to me" and "provably wrong" are different claims and should not be filed
as if they were the same.

Add the entry when you port the code, not later. The point at which you are
reading the Java closely enough to notice is the only point at which it is cheap
to write down.

---

## 1. `COSInteger.equals` truncates to 32 bits

**Where** `pdfbox/src/main/java/org/apache/pdfbox/cos/COSInteger.java`, `equals`

```java
return o instanceof COSInteger && ((COSInteger)o).intValue() == intValue();
```

**What it does** `intValue()` is `(int) value`, so the comparison drops
everything above bit 31. `COSInteger.get(0)` and `COSInteger.get(4294967296)`
compare **equal**.

**What correct would be** compare the `long` values.

**Why it matters** `equals` is what `COSArray.indexOf`, `removeObject` and every
dictionary comparison route through. Object numbers can exceed the `int` range.

**Where the Go carries it** `go/pdfbox/cos/integer.go`, `Integer.Equals`, via
`Integer.IntValue`, which narrows through int32 so that Go reproduces Java's
(int) cast. An earlier draft did not narrow, so the defect was not in fact
reproduced; caught in review.

**Confidence** high. The truncation is unambiguous and there is no comment
suggesting it is deliberate.

---

## 2. `RandomAccessReadBuffer.read` adds a `-1` sentinel to its byte count

**Where** `io/src/main/java/org/apache/pdfbox/io/RandomAccessReadBuffer.java`,
`read(byte[], int, int)`

```java
while (bytesRead < length && available() > 0)
{
    if (currentBufferPointer == chunkSize) { nextBuffer(); }
    bytesRead += readRemainingBytes(b, offset + bytesRead, length - bytesRead);
}
```

**What it does** `readRemainingBytes` returns `-1` when the chunk has nothing
left. That `-1` is added to the running total, so a read that ends this way
reports one byte fewer than it produced.

**What correct would be** stop when `readRemainingBytes` returns a
non-positive value.

**Why it matters** the returned count is wrong, and a caller looping on it reads
the same byte twice. The loop condition bounds the damage to one byte per read.

**Where the Go carries it** `go/pdfio/readbuffer.go`, `ReadBuffer.Read`.

**Confidence** high.

---

## 3. `SequenceRandomAccessRead.read` has the same `-1` accumulation

**Where**
`io/src/main/java/org/apache/pdfbox/io/SequenceRandomAccessRead.java`, `read`

```java
int bytesRead = randomAccessRead.read(b, offset, maxAvailBytes);
while (bytesRead > -1 && bytesRead < maxAvailBytes)
{
    randomAccessRead = getCurrentReader();
    bytesRead += randomAccessRead.read(b, offset + bytesRead, maxAvailBytes - bytesRead);
}
```

**What it does** the same defect as #2, in a loop that can run several times: if
an inner read returns `-1` the total goes **down**, and the loop keeps going
until the total falls to `-1` or below.

**What correct would be** treat a non-positive inner read as the end.

**Why it matters** the returned count can be negative, and the offset arithmetic
inside the loop goes backwards with it.

**Where the Go carries it** `go/pdfio/sequenceread.go`, `SequenceRead.Read`.
The helper `readOrMinusOne` stands in for Java's read() returning -1, so the
same accumulation happens. An earlier draft stopped on a non-positive read;
that was corrected once this rule was adopted.

**Confidence** high on the defect, and it is worse than #2 because the loop can
iterate.

---

## 4. `TestCOSBase.testByteArrays` never checks the lengths

**Where** `pdfbox/src/test/java/org/apache/pdfbox/cos/TestCOSBase.java`

```java
assertEquals(byteArr1.length, byteArr1.length);
```

**What it does** asserts a value against itself. The helper only compares the
first `byteArr1.length` bytes, so a result that is correct as far as it goes but
too long passes.

**What correct would be** `assertEquals(byteArr1.length, byteArr2.length)`.

**Why it matters** every `accept()` and `writePDF` test in `cos` uses this
helper, so none of them verifies output length.

**Where the Go carries it** `go/pdfbox/cos/base_test.go`, `assertBytesEqual`.
The ported tests inherit exactly the same gap, deliberately — strengthening it
would mean the two suites no longer test the same thing.

**Confidence** high. This is a typo, not a design.

---

## 5. `COSName.getBytes` hands out its internal array

**Where** `pdfbox/src/main/java/org/apache/pdfbox/cos/COSName.java`, `getBytes`

**What it does** returns `nameBytes` directly. Names are interned in a shared
map, so a caller that writes to the returned array corrupts that name for every
holder of it, process-wide.

**What correct would be** return a copy, or document the array as read-only.

**Why it matters** the corruption is silent and global. No caller in PDFBox
writes to it today, which is why it has never bitten.

**Where the Go carries it** `go/pdfbox/cos/name.go`, `Name.Bytes`.

**Confidence** high that it is a hazard; lower that it is a *bug*, since nothing
currently exploits it. Filed because the Go inherits it and callers should know.

---

## 6. A null key is kept in the cross-reference table

**Where** `pdfbox/src/main/java/org/apache/pdfbox/cos/COSDocument.java`,
`addXRefTable`, and `XrefTrailerResolver.setXRef`

**What it does** a `null` `COSObjectKey` goes into the map and every reader is
expected to check for it. PDFBOX-6132 is the bug report from a reader that did
not.

**What correct would be** reject the key where it enters, since a key that
cannot be looked up or resolved carries no information.

**Why it matters** it is a null-check obligation spread across every consumer of
the table, and it has already been missed once.

**Where the Go carries it** `go/pdfbox/cos/document.go`, `AddXRefTable`, which
keeps a nil key in a separate field so `XRefTable` still returns it.

**Confidence** medium. Keeping it may be deliberate — a damaged file's entry is
arguably data — but the shape of PDFBOX-6132 suggests otherwise.

---

## 7. The evicted page buffer is reused while a cursor may still hold it

**Where**
`io/src/main/java/org/apache/pdfbox/io/RandomAccessReadBufferedFile.java`

```java
protected boolean removeEldestEntry(Map.Entry<Long, ByteBuffer> eldest) {
    final boolean doRemove = size() > MAX_CACHED_PAGES;
    if (doRemove) {
        lastRemovedCachePage = eldest.getValue();
        lastRemovedCachePage.clear();
    }
    return doRemove;
}
```

`readPage` then reuses `lastRemovedCachePage` for the next page read.

**What it does** an evicted page's buffer is refilled with different data. If
anything still references it as its current page, that reference now reads the
wrong bytes.

**What correct would be** allocate a fresh buffer, or make sure no cursor can
hold an evicted page.

**Why it matters** it would produce silently wrong bytes rather than an error,
which is the worst failure mode for a parser.

**Where the Go carries it** — **not carried.** `go/pdfio/bufferedfile.go` does
not reuse evicted buffers; Go's garbage collector makes the optimisation
pointless. This is a deliberate deviation recorded in `STATUS.md`, not an
oversight.

**Confidence** low that it is reachable. `curPage` is reassigned by every `seek`,
and `read` re-seeks at a page boundary, so a stale reference may be impossible.
Filed as unproven: it needs a reproducer before anyone treats it as real.

---

## 8. `COSName` parsing drops the `#` on a premature end of input

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdfparser/COSParser.java`,
`parseCOSName`

```java
if (ch2 == -1 || ch1 == -1) {
    LOG.error("Premature EOF in BaseParser#parseCOSName");
    c = -1;
    break;     // breaks before buffer.write(ch)
}
```

**What it does** for a name ending `/A#2` at end of input, the `#` and the `2`
are both discarded and the name is `A`. Every other malformed-escape path keeps
the `#` as a literal character.

**What correct would be** consistent with the branch below it, which does
`buffer.write(ch)` before continuing.

**Why it matters** a truncated file yields a name that silently differs from
what is on disk, rather than an error or a faithful literal.

**Where the Go carries it** `go/pdfbox/pdfparser/objectparser.go`,
`ParseCOSName`.

**Confidence** medium. It only triggers on truncated input, where any answer is
somewhat arbitrary, but the inconsistency with the adjacent branch looks
unintended.

---

## 9. A malformed inline image leaves `inlineImageDepth` stuck at 1

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdfparser/PDFStreamParser.java`,
`parseNextToken`, case `'B'`

```java
if (nextToken instanceof Operator)
{
    Operator imageData = (Operator) nextToken;
    ...
    beginImageOP.setImageData(imageData.getImageData());
    inlineImageDepth--;          // only here
}
else
{
    LOG.warn("nextToken {} at position {}, expected {}?!", ...);
    // no decrement
}
```

**What it does** `inlineImageDepth++` runs for every `BI`, but the matching
decrement sits inside the branch that found an `ID` operator. When the inline
image dictionary ends any other way — end of input, or a token that is neither a
`COSName` nor an `Operator` — the counter stays at 1. Every later `BI` in the
same content stream then trips the PDFBOX-6038 guard and throws
`Nested 'BI' operator not allowed`, even though nothing is nested.

**What correct would be** decrementing unconditionally once the `BI` handling is
over, or tracking the depth with try/finally, so that one broken image does not
poison the images after it.

**Why it matters** one malformed inline image turns every subsequent inline
image in the same stream into a hard parse failure. The PDFBOX-6038 guard was
added to stop runaway recursion; here it fires on a document that has none.

**Where the Go carries it** `go/pdfbox/pdfparser/streamtokenparser.go`,
`parseBeginInlineImage` — the decrement is inside the same `if`.

**Confidence** high that the code does this; it is plain from the placement of
the decrement. Medium that it is unintended rather than a deliberate "give up on
this stream" stance, since the `else` branch only warns and carries on.

---

## 10. A truncated inline image loses its last two bytes

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdfparser/PDFStreamParser.java`,
`parseNextToken`, case `'I'`

```java
int lastByte = source.read();
int currentByte = source.read();
while( !(lastByte == 'E' && currentByte == 'I' && ...) && !isEOF())
{
    imageData.write( lastByte );
    lastByte = currentByte;
    currentByte = source.read();
}
```

**What it does** the two bytes held in `lastByte` and `currentByte` are written
only on the next iteration. When the source runs out — an inline image with no
closing `EI`, or an `EI` at the very end with no whitespace behind it, so
`hasNextSpaceOrReturn` fails — `isEOF()` ends the loop and both are dropped.

**What correct would be** flushing the two pending bytes when the loop ends at
end of input rather than at an `EI`.

**Why it matters** it silently shortens the image data of a truncated file
instead of reporting that the image never terminated.

**Where the Go carries it** `go/pdfbox/pdfparser/streamtokenparser.go`,
`parseInlineImageData`.

**Confidence** medium. The loss is provable from the loop shape, but every
answer on a truncated stream is somewhat arbitrary and this may be a deliberate
"stop at whatever we have" choice.

---

## 11. `COSString.parseHex` computes a whitespace offset and never uses it

**Where** `pdfbox/src/main/java/org/apache/pdfbox/cos/COSString.java`,
`parseHex`

```java
int start = 0;
while (start < end && Character.isWhitespace(hex.charAt(start)))
{
    start++;
}

int length = end - start;
...
for (int i = 0; i < length; i += 2)
{
    int value = 16 * Hex.getHexValue(hex.charAt(i)) + Hex.getHexValue(hex.charAt(i + 1));
```

**What it does** the loop indexes `hex` from zero, not from `start`, so the
leading-whitespace offset is computed and then thrown away. Only the *length* of
the leading whitespace is honoured, by shortening the run. For `"  4142  "` the
loop reads `hex.charAt(0)` and `hex.charAt(1)` — two spaces — and
`Hex.getHexValue` returns a negative for each, so `parseHex` throws
`Invalid hex string` unless `FORCE_PARSING` is set, in which case it emits `?`.
The comment above the block says "skip leading and trailing whitespace"; trailing
whitespace is skipped, leading whitespace is not.

**What correct would be** indexing from `start`: `hex.charAt(start + i)` and
`hex.charAt(start + i + 1)`, and likewise `hex.charAt(start + length)` in the
uneven-length branch.

**Why it matters** a hex string written `< 4142 >` is legal — the PDF
specification allows whitespace inside the angle brackets — and PDFBox rejects
it. The parser never sees this, because `parseCOSHexString` strips whitespace
as it scans and hands `parseHex` a clean run of digits, but every other caller
passes the string through as it stands.

**Where the Go carries it** `go/pdfbox/cos/string.go`, `ParseHexString`.
The port originally sliced `hex[start:end]` and indexed the slice, which
corrected the bug. That was reverted: the offset is computed and unused here
too, and `string_test.go` pins the throwing behaviour.

**Confidence** high. The offset is plainly computed and plainly not used, and
the comment above it states an intent the code does not carry out.
