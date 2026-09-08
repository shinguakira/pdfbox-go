# Java bugs found while porting

Defects and surprising behaviour noticed in the Java source during the port.

**Nothing here is ever fixed in the Java.** The Java is the reference and this
repository does not edit it.

**Nothing here was fixed in the Go while the port was being written**, either.
Every branch that ported code reproduced these deliberately, because a silently
corrected bug makes the Go behave differently from the thing it exists to
reproduce, and because "the Java does the same thing" is the answer that made
every odd behaviour cheap to investigate. See
[`conventions/java-to-go.md`](conventions/java-to-go.md).

**One branch changes that, once the port is finished:
[`track/java-bug-fixes`](tasks/track-java-bug-fixes.md).** It goes through this
file entry by entry and fixes in the Go the ones worth fixing. It changes no
Java. It deletes no entry.

So an entry here is in one of three states, and says which:

| State | The line that says so |
| --- | --- |
| carried in the Go, on purpose | **Where the Go carries it** — every entry has this |
| fixed in the Go, deliberately differing from the Java | **Fixed in the Go** |
| carried on purpose after the fix branch judged it | **Kept in the Go** |

**An entry is never removed, and its "Where the Go carries it" line is never
rewritten.** That line is the record of what the port did while it was a port;
past-tensing it would lose the fact that the reproduction was deliberate. A fix
adds a line, it does not edit one.

This file exists so the knowledge is not lost. A reader who later finds the Go
behaving oddly can look here and see whether the Java does the same thing, on
purpose, and that it was noticed rather than missed — and, where the two now
differ, why.

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

**Keep the numbering stable.** Do not renumber, compact or reorder. A later
reader's only handle on any of this is "JAVA-BUGS.md 47", and the task files,
the commit messages and the comments in the Go all use it.

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

**Fixed in the Go** `track/java-bug-fixes`, entry 1. `Integer.Equals` compares
the int64 values, so `GetInteger(0)` and `GetInteger(4294967296)` are not equal
where the Java says they are. `Integer.IntValue` still narrows, because that is
`intValue()` and a caller who asks for it wants Java's answer. Tested by
`TestIntegerEqualsDoesNotTruncate` in `go/pdfbox/cos/javabugfixes_test.go`;
`TestIntegerEqualsIsTruncating` in `integer_test.go` used to assert the defect
and is now `TestIntegerEqualsIsNotTruncating`, with the Java's answers kept in
its comment.

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

**Kept in the Go** `track/java-bug-fixes`: unobservable. A0 marked this **fix**;
it is right about the arithmetic and wrong about the reach. `readFromChunk`
answers -1 in exactly two cases -- the cursor is past the end, or the chunk is
spent -- and the loop that would add it checks `remaining() > 0` and moves to
the next chunk before every call, so neither can happen there. A `ReadBuffer`
owns its own bytes, so nothing can tell it there are more than it has; entry 3
is the same arithmetic and **is** reachable, because a view can. See its
**Fixed in the Go**, and `TestSequenceReadCannotAccumulateMinusOne` in
`go/pdfio/javabugfixes_test.go` for the case that stays correct here.

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

**Why it matters** **the read can never return.** The `track/scratchfile` D9
re-read found the loop is not merely wrong but non-terminating: `bytesRead`
oscillates. Where an inner source hands back fewer bytes than its `length()`
promises, one pass reads some bytes, the next adds a `-1` and puts `bytesRead`
back below where it was, the pass after that reads them again, and the loop
never reaches `maxAvailBytes` and never falls to `-1` either. Beyond that, the
returned count is one low per `-1` and `currentPosition += bytesRead` moves the
cursor backwards.

A `RandomAccessReadView` whose `streamLength` is longer than its source is such
a source, and it needs no corruption to build — the view takes the length it is
told.

**What correct would be** as above: stop on a non-positive inner read.

**Where the Go carries it** `go/pdfio/sequenceread.go`, `SequenceRead.Read`.
The helper `readOrMinusOne` stands in for Java's read() returning -1, so the
same accumulation, the same backwards cursor and the same spin all happen. An
earlier draft stopped on a non-positive read; that was corrected once this rule
was adopted, and the backwards cursor was added by the D9 re-read, which found
the port had been keeping the position where Java loses a byte from it.

No test pins the spin. A test that hangs when it succeeds is worse than no test;
this entry is the record.

**Fixed in the Go** `track/java-bug-fixes`, entry 3. `SequenceRead.Read` stops
on a non-positive inner read rather than adding it to the total. Without it the
port lost the data outright: a sequence over a `ReadView(source, 0, 100)` whose
source holds ten bytes answered **0 bytes and EOF**, because the -1 was added
until the count went negative. That case is the one this entry's Confidence
line names, and it is why entry 2 is kept and this one is not -- a view can
declare a length its source cannot supply, and a `ReadBuffer` has no such
second party. Tested by `TestSequenceReadOverALyingView` in
`go/pdfio/javabugfixes_test.go`.

**Confidence** reproduced. A `SequenceRandomAccessRead` over a single
`RandomAccessReadView(source, 0, 100)` whose source holds 10 bytes hung on the
first `read(b, 0, 20)`, and the thread dump named the loop:

```
at org.apache.pdfbox.io.SequenceRandomAccessRead.read(SequenceRandomAccessRead.java:132)
```

which is `bytesRead += randomAccessRead.read(b, offset + bytesRead, maxAvailBytes - bytesRead);`.

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

**Fixed in the Go** `track/java-bug-fixes`, entry 5. `Name.Bytes` answers a
copy. Names are interned, so the array Java hands out is shared by every holder
of that name and a caller who writes through it renames all of them. Nothing in
either tree does, which is why the entry's confidence separates the hazard from
the defect — and why removing it costs nothing: `Bytes` has no caller inside
the port. Tested by `TestNameBytesIsACopy` in
`go/pdfbox/cos/javabugfixes_test.go`.

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

**Fixed in the Go** `track/java-bug-fixes`, entry 8. `ParseCOSName` writes the
`#`, and the digit it had if there was one, before it breaks: `/A#2` at the end
of input parses as `A#2` where the Java answers `A`. That is what the branch
below it already does for a malformed escape, and what a `#` not followed by two
hex digits is. Tested by `TestParseCOSNameKeepsAPrematureHash` in
`go/pdfbox/pdfparser/javabugfixes_test.go`; the `/A#2` case of
`TestParseCOSName` asserted the Java's answer and now asserts this one.

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

**Fixed in the Go** `track/java-bug-fixes`, entry 9. The decrement moved out of
the branch that handles a complete inline image, so it runs however the image
ended. The counter guards against a `BI` nested inside a `BI` (PDFBOX-6038) and
a malformed one has ended either way; Java leaves it at 1 and refuses every
later `BI` in the stream. Tested by
`TestMalformedInlineImageDoesNotBlockTheNextOne` in
`go/pdfbox/pdfparser/javabug9_test.go`, over a `BI` whose dictionary ends in a
string rather than an `ID`.

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

**Fixed in the Go** `track/java-bug-fixes`, entry 10. `parseInlineImageData`
writes the two bytes it is holding when the input runs out, so an inline image
with no closing `EI` comes out the length it is on disk rather than two bytes
shorter.

The first attempt was **wider than the entry** and the ported Java tests caught
it: an image that ends `...EI` with nothing behind it reaches the same branch --
`atEndOfInlineImage` wants a whitespace after the `EI` and there is none -- and
there the two bytes in hand are the terminator, not data. Java drops them in
both cases and is right in one, so the port drops them only when they are `E`
and `I`. `TestInlineImages` pins the three cases that end that way and passes
unchanged. Tested by `TestTruncatedInlineImageKeepsItsLastTwoBytes` in
`go/pdfbox/pdfparser/javabug10_test.go`.

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

---

## 12. `TrueTypeFont.nameToGID` dereferences a null cmap for a `uniXXXX` name

**Where** `fontbox/src/main/java/org/apache/fontbox/ttf/TrueTypeFont.java`,
`nameToGID`.

```java
int uni = parseUniName(name);
if (uni > -1)
{
    CmapLookup cmap = getUnicodeCmapLookup(false);
    return cmap.getGlyphId(uni);
}
```

**What it does** `getUnicodeCmapLookup(false)` is the lenient form, and its
whole point is that it returns `null` rather than throwing when the font has no
`cmap` table:

```java
CmapTable cmapTable = getCmap();
if (cmapTable == null)
{
    if (isStrict) { throw new IOException(...); }
    else { return null; }
}
```

`nameToGID` then calls `getGlyphId` on it without a null check, so the lenient
path throws a `NullPointerException` instead of the `IOException` it was written
to avoid.

**What correct would be** a null check returning 0, which is what `nameToGID`
returns for every other name it cannot resolve.

**Why it matters** `TTFParser` requires a `cmap` table only when the font is not
embedded — `if (!isEmbedded && font.getCmap() == null) throw` — so an embedded
TrueType font with no `cmap` parses fine, and is exactly the case this branch
was written for. Asking such a font for `getWidth("uni0041")` or
`hasGlyph("uni0041")` — both of which go through `nameToGID` — throws NPE out of
a method declared to throw `IOException`. The name has to survive the `post`
table lookup first, which an embedded subset with no `post` names will.

**Where the Go carries it** `go/fontbox/ttf/truetypefont.go`, `NameToGID`.
`unicodeCmapImpl` returns a nil `*CmapSubtable`, `UnicodeCmapLookup` hands it
back inside a `CmapLookup` interface, and `cmap.GetGlyphID(uni)` dereferences
nil and panics — which is what this port does with an unchecked Java exception.

**Fixed in the Go** `track/java-bug-fixes`, entry 12. `NameToGID` answers 0
where the font has no Unicode cmap, which is what it answers for every other
name it cannot resolve — the branch three lines above and the one three lines
below both do. Without it the port panicked on a subsetted font, which usually
has no cmap at all. Tested by `TestNameToGIDOfAUniNameWithoutACmap` in
`go/fontbox/ttf/javabugfixes_test.go`.

**Confidence** high. The null return is explicit three lines up in the same
class, and no caller of the lenient form checks it.

---

## 13. `GlyphList.loadList` stops at the first stream that is not ready

**Where**
`pdfbox/src/main/java/org/apache/pdfbox/pdmodel/font/encoding/GlyphList.java`,
`loadList`.

```java
try (BufferedReader in = new BufferedReader(new InputStreamReader(input, StandardCharsets.ISO_8859_1)))
{
    while (in.ready())
    {
        String line = in.readLine();
```

**What it does** `BufferedReader.ready()` reports whether the *next read will
not block*, not whether the stream has more data. The loop therefore ends both
at end of input and at any point where the reader has drained its buffer and the
underlying stream has nothing available yet. The glyph list is then silently
short: no exception, no log line, just names that resolve to null from there on.

**What correct would be** the ordinary idiom the class already half-writes,
since it null-checks `line` inside the loop anyway:

```java
String line;
while ((line = in.readLine()) != null)
```

**Why it matters** the two shipped lists come off the classpath, where `ready()`
is true until the end, so the bundled path is unaffected. But both constructors
are public and take an arbitrary `InputStream`: `GlyphList(InputStream, int)`
and `GlyphList(GlyphList, InputStream)`. A caller handing it a socket, a pipe,
or a slow decompressing stream gets a truncated glyph list and no indication of
it. `LegacyPDFStreamEngine` uses the second constructor to add `additional.txt`
on top of the Adobe Glyph List.

**Where the Go carries it** it does not, and cannot: Go has no `ready()`, and
there is nothing to emulate it with -- a `bufio.Scanner` reads to end of input.
`go/pdfbox/pdmodel/font/encoding/glyphlist.go`, `loadList`, therefore reads the
whole stream. For every input this library actually passes -- the embedded
files -- the two behave identically.

**Confidence** high. `ready()` is documented as "Tells whether this stream is
ready to be read", and using it as a loop condition in place of a null check on
`readLine` is a long-standing known misuse.

---

## 14. `PDFontDescriptor.getPanose` dereferences a missing `/Panose` entry

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/font/PDFontDescriptor.java`,
`getPanose`.

```java
public PDPanose getPanose()
{
    COSDictionary style = dic.getCOSDictionary(COSName.STYLE);
    if (style != null)
    {
        COSString panose = (COSString)style.getDictionaryObject(COSName.PANOSE);
        byte[] bytes = panose.getBytes();
        if (bytes.length >= PDPanose.LENGTH)
        {
            return new PDPanose(bytes);
        }
    }
    return null;
}
```

**What it does** the method null-checks `/Style` and then does not null-check
what it reads out of it. A font descriptor carrying a `/Style` dictionary with
no `/Panose` entry -- or one whose `/Panose` is anything other than a string --
makes `getDictionaryObject` return null, or the cast fail, and the method throws
`NullPointerException` or `ClassCastException` out of a getter declared to
return null when it has nothing.

**What correct would be** the same null check the method already applies one
line up:

```java
COSBase base = style.getDictionaryObject(COSName.PANOSE);
if (base instanceof COSString)
{
    byte[] bytes = ((COSString) base).getBytes();
    ...
}
```

**Why it matters** `/Style` is optional and `/Panose` is the only entry the
specification defines inside it, so in practice the two travel together -- but
nothing enforces that, and a font descriptor is read straight out of a file that
may say anything. The specification (ISO 32000-1 table 124) marks `/Panose`
required *within* `/Style`, which is exactly the kind of "required" a malformed
file ignores. Every other getter on this class returns a default for a missing
entry.

**Where the Go carries it** `go/pdfbox/pdmodel/font/pdfontdescriptor.go`,
`Panose`. The type assertion is written without the comma-ok form, so it panics
where Java throws.

**Fixed in the Go** `track/java-bug-fixes`, entry 14. `Panose` answers nil for
a /Style with no /Panose, which is what it answers for a descriptor with no
/Style at all and for a /Panose shorter than twelve bytes. Tested by
`TestPanoseOfAStyleWithoutOne` in
`go/pdfbox/pdmodel/font/javabugfixes_test.go`.

**Confidence** high. The null check on the line above shows the author knew the
dictionary could be absent; the entry inside it is read without one.

---

## 15. `PDFTextStripper.handleDirection` reverses UTF-16 units, breaking any character outside the basic plane

**Where** `pdfbox/src/main/java/org/apache/pdfbox/text/PDFTextStripper.java`,
`handleDirection`.

```java
if ((level & 1) != 0)
{
    while (--end >= start)
    {
        char character = word.charAt(end);
        if (Character.isMirrored(word.codePointAt(end)))
        {
            ...
        }
        else
        {
            result.append(character);
        }
    }
}
```

**What it does** the loop walks a right-to-left run backwards one `char` at a
time, and a `char` is a UTF-16 code unit rather than a character. A character
outside the basic multilingual plane is two of them, so the pair comes out low
half first. The two halves no longer form a pair, and the character is gone:
writing the resulting `String` out as UTF-8 replaces each unpaired half.

The `codePointAt(end)` on the next line shows the author knew the difference —
the mirroring test is done on the code point and the append on the code unit.

**What correct would be** walking the run backwards by code point, which is what
`StringBuilder.reverse` does; its own documentation says "if there are any
surrogate pairs included in the sequence, these are treated as single
characters". `getVisuallyOrderedUnicode`, four hundred lines away in
`TextPosition`, reverses with exactly that and is unaffected.

**Why it matters** every right-to-left script that reaches outside the basic
plane loses its characters when the text is extracted: Arabic Mathematical
Alphabetic Symbols (U+1EE00–U+1EEFF), Cypriot, Phoenician, Old South Arabian,
and the Arabic and Hebrew ranges in the Supplementary Multilingual Plane. The
run has to be right to left for the branch to be taken, so a Latin document is
never affected — which is why it has gone unnoticed.

**Where the Go carries it** `go/pdfbox/text/direction.go`, `handleDirection`.
The port originally reversed runes, which kept the character whole and
corrected the bug. That was reverted: the units are reversed here too, and the
halves that no longer pair become the replacement character, which is what
Java's `String` becomes once it is written out. `feedback_test.go`,
`TestHandleDirectionReversesUTF16Units`, pins it.

**Fixed in the Go** `track/java-bug-fixes`, entry 15. `handleDirection`
reverses a right-to-left run by code point rather than by UTF-16 code unit, so
a character outside the basic plane stays whole and comes out in one piece. The
mirroring lookup is unchanged: every mirrored character is inside the basic
plane, and Java already read the code point for the mirroring test one line
above the append it got wrong. Tested by
`TestHandleDirectionReversesByCodePoint` in
`go/pdfbox/text/javabugfixes_test.go`, whose expected value is the run reversed
by character and not read off the Java;
`TestHandleDirectionStillMirrors` beside it keeps the mirroring. The pin that
held the bug, `TestHandleDirectionReversesUTF16Units` in `feedback_test.go`, is
gone with it.

**Confidence** high. The same method reads the code point for the mirroring
test and appends the code unit, one line apart.

## 16. `CMap.useCmap` builds a one-byte code with `% 0xFF` instead of `& 0xFF`

**Where** `fontbox/src/main/java/org/apache/fontbox/cmap/CMap.java`, `useCmap`.

**What it does** the `usecmap` operator copies one CMap's mappings into
another. The forward maps are copied wholesale; the inverted map,
`unicodeToByteCodes`, is rebuilt from the keys, and for the one-byte table the
key is turned back into a byte with

```java
cmap.charToUnicodeOneByte.forEach((k, v) ->
        unicodeToByteCodes.put(v, new byte[]{(byte) (k % 0xFF)}));
```

`k` is a one-byte code, so it runs 0 to 255. `k % 0xFF` is `k % 255`, which
maps 255 to 0 and leaves every other value alone. The two-byte and the three /
four byte branches directly below both use `& 0xFF` on every byte, so the
one-byte line is the odd one out.

**What correct would be** `(byte) (k & 0xFF)`, or simply `(byte) (int) k` — the
key is already a single byte's worth.

**Why it matters** after a `usecmap`, `getCodesFromUnicode` for whatever the
inherited CMap mapped from code 0xFF hands back code 0x00. The caller is
`PDType0Font.encode`, which is how text is written into a content stream, so a
document built on such a CMap gets the wrong byte written for that one
character. It needs an inherited one-byte CMap with a mapping at 0xFF to show,
which is why it has gone unnoticed.

**Where the Go carries it** `go/fontbox/cmap/cmap.go`, `useCmap`, with the
`% 0xFF` written out and a comment pointing here.

**Fixed in the Go** `track/java-bug-fixes`, entry 16. `useCmap` masks with
`& 0xFF`, which is what the two branches beside it do. The two agree for every
byte but 255, so a parent CMap that maps the code 0xFF used to lose it and the
reverse map answered the code 0 for that character. Tested by
`TestUseCmapKeepsTheCodeFF` in `go/fontbox/cmap/javabugfixes_test.go`.

**Confidence** high. The two branches beside it mask with `& 0xFF`, and `%` on
a value that is already a byte cannot be deliberate.

## 17. `CFFParser.concatenateMatrix` multiplies one cell by the wrong matrix

**Where** `fontbox/src/main/java/org/apache/fontbox/cff/CFFParser.java`,
`concatenateMatrix`.

**What it does** a CID-keyed CFF font may carry a FontMatrix in its Font DICT
as well as in the Top DICT, and PDFBOX-3579 needs the two multiplied together.
The six cells are written out by hand:

```java
matrixDest.set(0, a1 * a2 + b1 * c2);
matrixDest.set(1, a1 * b2 + b1 * d1);
matrixDest.set(2, c1 * a2 + d1 * c2);
matrixDest.set(3, c1 * b2 + d1 * d2);
matrixDest.set(4, x1 * a2 + y1 * c2 + x2);
matrixDest.set(5, x1 * b2 + y1 * d2 + y2);
```

Row 1 ends `b1 * d1`. Every other cell pairs a value from the destination
matrix with one from the matrix being concatenated; this one pairs `b1` and
`d1`, both from the destination. The matrix product wants `b1 * d2`.

**What correct would be** `matrixDest.set(1, a1 * b2 + b1 * d2);`.

**Why it matters** cell 1 is the y shear. For the overwhelmingly common case
where both matrices are diagonal -- `b1` is 0 -- the term vanishes and the bug
is invisible, which is why it has gone unnoticed. A CID-keyed CFF font whose
Font DICT and Top DICT both carry a sheared or rotated FontMatrix gets the
wrong shear, and every glyph of it is drawn skewed.

**Where the Go carries it** `go/fontbox/cff/cffparser.go`, `concatenateMatrix`,
with `b1*d1` written out and a comment pointing here.

**Fixed in the Go** `track/java-bug-fixes`, entry 17. `matrixDest[1]` reads
`d2`, as the other five cells read the second matrix. Tested by
`TestConcatenateMatrixUsesTheSecondMatrixThroughout` in
`go/fontbox/cff/javabugfixes_test.go`, whose expected values are the product of
the two matrices the comment above the function draws.

**Confidence** high. The five cells around it are a textbook 3x2 matrix
product and this one is not.

## 18. `PDType1CFont.getStringWidth` advances one UTF-16 unit at a time

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/font/PDType1CFont.java`,
`getStringWidth`.

**What it does** it measures a string by walking it:

```java
for (int i = 0; i < string.length(); i++)
{
    int codePoint = string.codePointAt(i);
    String name = getGlyphList().codePointToName(codePoint);
    ...
    width += cffFont.getType1CharString(name).getWidth();
}
```

`String.length()` counts UTF-16 code units and `codePointAt` reads a whole
character, so a character outside the basic plane is read at its first unit and
then again at its second. The second read lands on the low surrogate, which is
not part of a pair from where it starts, and `codePointAt` gives back that bare
unit — a code point in D800–DFFF.

`PDFont.encode`, six hundred lines away, does the same walk correctly:

```java
for (int offset = 0; offset < text.length(); )
{
    int codePoint = text.codePointAt(offset);
    ...
    offset += Character.charCount(codePoint);
}
```

**What correct would be** the `charCount` advance `PDFont.encode` uses.

**Why it matters** the character is measured twice, and the second measurement
is of a lone surrogate. `codePointToName` has no name for one, so the width
comes out as the width of whatever `.notdef`-ish name it produces, or — much
more likely — the `hasGlyph` check just above fails and the whole call throws
`IllegalArgumentException`. Measuring any string with an emoji or a
supplementary-plane character in a Type 1C font is therefore either wrong or
fatal. It needs a Type 1C font that actually has such a glyph to show, which is
why it has gone unnoticed.

**Where the Go carries it** `go/pdfbox/pdmodel/font/pdtype1cfont.go`,
`StringWidth`, which walks `utf16Units` one at a time with `codePointAt`, both
written out beside it.

**Kept in the Go** `track/java-bug-fixes`: unobservable. A0 marked this
**fix**; it is right about the walk and wrong about the reach. The second visit
only happens if the first one succeeded, and the first one cannot: the name the
walk measures by is `getGlyphList().codePointToName(codePoint)`, the Adobe
Glyph List holds no code point outside the basic plane -- `glyphlist.txt`,
`additional.txt` and `zapfdingbats.txt` are all four hex digits -- so every
supplementary code point comes back `.notdef`, and `.notdef` is the one name
`CFFType1Font.hasGlyph` can never answer true for, because its SID is 0 and
`getGIDForSID(0)` is the GID 0 the test rejects. The character therefore throws
on its first unit, before the index that was not advanced can be read. The same
two facts hold in the Java, so this is the Java's own behaviour and not a
divergence.

**Confidence** high. The correct walk is in the same package, in the method
this one exists to complement.

## 19. `FileSystemFontProvider.writeFontInfo` sign-extends a Panose byte before hex

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/font/FileSystemFontProvider.java`,
`writeFontInfo`.

**What it does** it writes the ten Panose bytes into the on-disk font cache as
two hex digits each:

```java
byte[] bytes = fontInfo.panose.getBytes();
for (int i = 0; i < 10; i ++)
{
    String str = Integer.toHexString(bytes[i]);
    if (str.length() == 1)
    {
        writer.write('0');
    }
    writer.write(str);
}
```

`bytes[i]` is a signed `byte`, and `Integer.toHexString` takes an `int`, so the
byte is widened with sign extension first. A value of 0x00–0x7F comes out as one
or two digits and is padded to two; a value of 0x80–0xFF becomes a negative
`int` and comes out as **eight** digits — 0x8A prints as `ffffff8a`.

The reader on the other side assumes exactly two digits per value:

```java
String str = parts[8].substring(i * 2, i * 2 + 2);
```

**What correct would be** `Integer.toHexString(bytes[i] & 0xFF)`, which is the
mask the reader already applies coming back (`panose[i] = (byte)(b & 0xff)`).

**Why it matters** one Panose byte of 0x80 or more shifts every later byte of
the field by six characters, so the ten values read back are garbage — and the
Panose comparison in `FontMapperImpl.getFontMatches` is the *most reliable*
signal it has for picking a substitute, per its own comment. The field is not
long enough to throw, because the field is longer than the twenty characters the
reader slices, so the damage is silent. The ten standard Panose digits are all
in 0–15, which is why it survives: it needs a font whose "OS/2" table carries an
out-of-range Panose value, and those exist but are not common.

**Where the Go carries it** `go/pdfbox/pdmodel/font/filesystemfontprovider.go`,
`writeFontInfo`, which converts through `int8` before `toHexString` so that the
same eight digits come out.

**Fixed in the Go** `track/java-bug-fixes`, entry 19. `writeFontInfo` writes
each Panose byte unsigned, so a value of 0x80 or more takes two hex digits
rather than eight. The reader's own `& 0xff` is what says the writer meant two.
Tested by `TestPanoseHexIsTwoDigits` in
`go/pdfbox/pdmodel/font/javabug19_test.go`.

**Confidence** high. The reader's own `& 0xff` says what the writer meant.

## 20. `FileSystemFontProvider.addTrueTypeFontImpl` ANDs the two halves of a CID supplement

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/font/FileSystemFontProvider.java`,
`addTrueTypeFontImpl`, reading the "gcid" table of an Apple AAT font.

**What it does**

```java
int supplementVersion = bytes[140] << 8 & (bytes[141] & 0xFF);
```

**What correct would be** `bytes[140] << 8 | (bytes[141] & 0xFF)` — the two
bytes of a big-endian 16-bit number are ORed together, not ANDed.

**Why it matters** `bytes[140] << 8` has a zero low byte by construction, and
`bytes[141] & 0xFF` has nothing but a low byte, so the AND is always 0. Every
AAT font read this way gets supplement 0 in its `CIDSystemInfo`, whatever the
table says. Nothing in PDFBox compares supplements — `isCharSetMatch` looks at
registry and ordering only — so the wrong value never changes a substitution,
but it is written into the on-disk cache and handed to anyone reading
`FontInfo.getCIDSystemInfo()`.

**Where the Go carries it** `go/pdfbox/pdmodel/font/filesystemfontprovider.go`,
`addTrueTypeFontImpl`, which writes the same `&` with a comment.

**Fixed in the Go** `track/java-bug-fixes`, entry 20. The two bytes are ORed,
in `cidSupplementVersion`, which the test can reach on its own. Java's `&`
between a value whose low eight bits are zero and one whose high bits are zero
is zero for every input, so the supplement was always 0. Tested by
`TestCIDSupplementIsTheTwoBytesJoined` in
`go/pdfbox/pdmodel/font/javabug20_test.go`.

**Confidence** high. `&` between disjoint byte lanes cannot be what was meant.

## 21. `FileSystemFontProvider.createFSIgnored` builds an entry with a null parent

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/font/FileSystemFontProvider.java`,
`createFSIgnored`.

**What it does** it builds the `FSFontInfo` that stands for a font file the
scan could not read:

```java
return new FSFontInfo(file, format, postScriptName, null, 0, 0, 0, 0, 0, null, null, hash, file.lastModified());
```

Counting the parameters against the constructor, the tenth `null` is `panose`
and the eleventh is `parent` — the `FileSystemFontProvider` the entry belongs
to. Every other call site passes `this`.

`FSFontInfo.getFont()` opens with `parent.cache.getFont(this)`, so calling it on
one of these entries throws `NullPointerException`.

**What correct would be** `this` for the parent, as the other two call sites
pass.

**Why it matters** these entries go into `fontInfoList` under the names
`*skipexception*`, `*skipnoname*` and `*skippipeinname*`, and from there into
`FontMapperImpl.fontInfoByName`. `findFont` looks a name up and calls
`info.getFont()` on whatever it finds, so a PDF whose `/BaseFont` is literally
`*skipexception*` takes down the font lookup. `getFontMatches`, the other caller
of `getFont()`, filters these out first, because an ignored entry has no
CIDSystemInfo and no code page bits, so the fuzzy path is safe. The name has to
be crafted to reach it, which is why it has gone unnoticed.

**Where the Go carries it** `go/pdfbox/pdmodel/font/filesystemfontprovider.go`,
`createFSIgnored`, which passes nil for the parent with a comment; the Go panics
on the nil dereference where Java throws.

**Fixed in the Go** `track/java-bug-fixes`, entry 21. `createFSIgnored` passes
the provider, as the other two call sites of the constructor do. Without it
`Font()` on an entry the provider decided to ignore dereferenced the null
parent on its first line. Tested by `TestIgnoredFontInfoHasItsParent` in
`go/pdfbox/pdmodel/font/javabug21_test.go`.

**Confidence** high. The parameter list is unambiguous and the other two call
sites pass `this`.

## 22. `KerningTable.read` can never take its version 1 branch

**Where** `fontbox/src/main/java/org/apache/fontbox/ttf/KerningTable.java`,
`read`.

**What it does** it reads the table version, then switches on it:

```java
int version = data.readUnsignedShort();
if (version != 0)
{
    version = (version << 16) | data.readUnsignedShort();
}
int numSubtables = 0;
switch (version)
{
    case 0:
        numSubtables = data.readUnsignedShort();
        break;
    case 1:
        numSubtables = (int) data.readUnsignedInt();
        break;
    default:
        LOG.debug("Skipped kerning table due to an unsupported kerning table version: {}",
                version);
        break;
}
```

The two 'kern' formats differ in their header: the Microsoft one begins with a
uint16 version of 0 followed by a uint16 count, and the Apple one with a 16.16
fixed version of 0x00010000 followed by a uint32 count. The read above is built
to tell them apart, and the first half works: a zero first word leaves the
version 0, a non-zero one is shifted up by sixteen and OR'd with the next word,
which turns Apple's `00 01 00 00` into 0x00010000.

Then `case 1` asks for the *decimal* 1. By that point the version is either 0 or
at least 0x10000 — `version << 16` with a non-zero `version` cannot be less than
65536 — so nothing reaches it.

**What correct would be** `case 0x10000`.

**Why it matters** every Apple-format 'kern' table is skipped with "unsupported
kerning table version: 65536", so a font that carries only that format has no
kerning at all. `KerningTable.getHorizontalKerningSubtable` then returns null
and the caller falls back to no kerning, which is silent.

**Where the Go carries it** `go/fontbox/ttf/kerning.go`, which switches on `1`
with a comment saying so. The port also narrows the count to a signed 32-bit
int, as Java's cast does, so that the two would behave the same if the branch
were ever reached — without the narrowing a Go `int` stays positive and the
count is used to size an allocation.

**Confidence** high. It is provable from the two lines above it that the case
label cannot match.

## 23. `CMapStrings.getMapping` reads a zero-length code as the two-byte code 0

**Where** `fontbox/src/main/java/org/apache/fontbox/cmap/CMapStrings.java`,
`getMapping`, reached from `CMapParser.createStringFromBytes`.

**What it does**

```java
public static String getMapping(byte[] bytes)
{
    if (bytes.length > 2)
    {
        return null;
    }
    return bytes.length == 1 ? oneByteMappings.get(CMap.toInt(bytes))
            : twoByteMappings.get(CMap.toInt(bytes));
}
```

The ternary has two arms for three cases. A zero-length array is not length 1,
so it takes the two-byte arm; `CMap.toInt` of no bytes is 0, and
`twoByteMappings.get(0)` is the one-character string U+0000. An empty
destination in a `bfchar` or `bfrange` — written `<>` — therefore maps to a NUL
rather than to the empty string.

The caller makes the inconsistency plain:

```java
private static String createStringFromBytes(byte[] bytes)
{
    if (bytes.length <= 2)
    {
        return CMapStrings.getMapping(bytes);
    }
    return new String(bytes, StandardCharsets.UTF_16BE);
}
```

The same empty array down the other arm would decode as UTF-16BE to the empty
string.

**What correct would be** an empty string for an empty code, which is what the
UTF-16BE arm gives and what the two-byte table is not being asked about.

**Why it matters** a CMap with an empty destination maps its code to U+0000
instead of to nothing, so the extracted text carries a NUL. It needs a
hand-written CMap to reach — no producer writes `<>` on purpose — which is why
it has gone unnoticed.

**Where the Go carries it** `go/fontbox/cmap/cmapstrings.go`, `GetMapping`,
which falls through to the two-byte table for a zero-length code exactly as Java
does, with a comment saying why the length-0 case is not special-cased.

**Fixed in the Go** `track/java-bug-fixes`, entry 23. `GetMapping` answers
nothing for a zero-length code, which is the answer the arm for a code longer
than two bytes gives — the ternary has two arms for three cases and no bytes at
all is the third. Tested by `TestGetMappingOfAnEmptyCode` in
`go/fontbox/cmap/javabug23_test.go`.

**Confidence** high. The two arms of the caller disagree about the same input.

## 24. `PDEncryption.hasSecurityHandler` answers the opposite of its name

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/encryption/PDEncryption.java`.

**What it does**

```java
public boolean hasSecurityHandler()
{
    return securityHandler == null;
}
```

The field is null when the document has no security handler — when
`SecurityHandlerFactory.newSecurityHandlerForFilter` did not recognise the
`/Filter`, or when the dictionary was built empty. So the method returns true
exactly when the answer is no.

**What correct would be** `securityHandler != null`.

**Why it matters** nothing in PDFBox calls it, so the bug is latent; but it is
public API, and a caller checking before `getSecurityHandler` — which is what
the name invites — gets the opposite of what it asked and then the IOException
it was trying to avoid.

**Where the Go carries it** `go/pdfbox/pdmodel/encryption/pdencryption.go`,
`HasSecurityHandler`, which returns `e.securityHandler == nil` with a comment
saying so.

**Fixed in the Go** `track/java-bug-fixes`, entry 24. `HasSecurityHandler`
answers `securityHandler != nil`. Nothing in either tree calls it, so nothing
was compensating for the inversion — the audit for callers found none, which is
also why nobody has noticed. Tested by `TestHasSecurityHandlerAnswersItsName`
in `go/pdfbox/pdmodel/encryption/javabugfixes_test.go`.

**Confidence** high. The method body and the method name cannot both be right.

## 25. `PDEncryption.getRecipientsLength` dereferences a missing /Recipients

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/encryption/PDEncryption.java`.

**What it does**

```java
public int getRecipientsLength()
{
    COSArray array = (COSArray) dictionary.getItem(COSName.RECIPIENTS);
    return array.size();
}
```

`getItem` returns null where the key is absent, which every password-encrypted
document is: `/Recipients` belongs to the public key handler. The cast of null
succeeds and `array.size()` throws `NullPointerException`.
`getRecipientStringAt` has the same shape.

**What correct would be** returning 0 for a missing array, which is what the
method's own documentation — "the number of recipients contained in the
Recipients field" — implies for a document that has none.

**Why it matters** it is public API on a class every encrypted document has.
PDFBox itself has stopped calling the pair — `PublicKeySecurityHandler` reads
the array directly, with a TODO saying both should be deprecated — so nothing
in the library trips it, but a caller asking how many recipients a document has
gets a NullPointerException rather than zero.

**Where the Go carries it** `go/pdfbox/pdmodel/encryption/pdencryption.go`,
`RecipientsLength` and `RecipientStringAt`, which assert the type without the
comma-ok and so panic where Java throws.

**Fixed in the Go** `track/java-bug-fixes`, entry 25. `RecipientsLength`
answers 0 for a dictionary with no /Recipients, which is what a count of
nothing is and what every other accessor on the class answers for a missing
entry. Tested by `TestRecipientsLengthOfADictionaryWithNone` in
`go/pdfbox/pdmodel/encryption/javabug25_test.go`.

**Confidence** high. `getItem` is documented to return null for an absent key.

## 26. `SecurityHandlerFactory.registerHandler` does not refuse a duplicate policy

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/encryption/SecurityHandlerFactory.java`.

**What it does**

```java
/**
 * ...
 * If another handler was previously registered for the same filter name or
 * for the same policy name, an exception is thrown
 */
public void registerHandler(String name,
                            Class<? extends SecurityHandler> securityHandler,
                            Class<? extends ProtectionPolicy> protectionPolicy)
{
    if (nameToHandler.containsKey(name))
    {
        throw new IllegalStateException("The security handler name is already registered");
    }

    nameToHandler.put(name, securityHandler);
    policyToHandler.put(protectionPolicy, securityHandler);
}
```

The javadoc promises the check on both maps; the code makes it on one. A second
registration under a new filter name but an existing policy class is accepted,
and `policyToHandler.put` replaces the handler the policy had.

**What correct would be** the second `containsKey`, on `policyToHandler`, which
the javadoc already describes.

**Why it matters** it is only reachable through the public `registerHandler`, so
PDFBox's own two registrations are safe. A caller that adds a handler for a
policy already spoken for takes it over silently, and every later
`newSecurityHandlerForPolicy` for that policy builds the wrong handler — a
document is then encrypted by a handler nobody asked for.

**Where the Go carries it**
`go/pdfbox/pdmodel/encryption/securityhandlerfactory.go`, `RegisterHandler`,
which checks `nameToHandler` only and says so above the assignment.
`TestRegisterHandlerReplacesADuplicatePolicy` in `fromsource_test.go` pins it.

**Fixed in the Go** `track/java-bug-fixes`, entry 26. `RegisterHandler` looks
in `policyToHandler` as well as in `nameToHandler` and refuses a policy that is
already spoken for, which is the check the javadoc describes and the only
reading under which the two refusals are one sentence. The expected behaviour
is read off that javadoc and not off the body. Tested by
`TestRegisterHandlerRefusesADuplicatePolicy` in
`go/pdfbox/pdmodel/encryption/javabug26_test.go`, with
`TestRegisterHandlerStillRefusesADuplicateName` beside it so the fix adds a
refusal rather than moving one. The pin that held the bug,
`TestRegisterHandlerReplacesADuplicatePolicy` in `fromsource_test.go`, is gone
with it.

**Confidence** high. The javadoc and the method body contradict each other in
five lines.

## 27. `ASCII85InputStream` reads a 0xFF data byte as the end of the stream

**Where** `pdfbox/src/main/java/org/apache/pdfbox/filter/ASCII85InputStream.java`.

**What it does**

```java
int zz = (byte) in.read();
if (zz == -1)
{
    eof = true;
    return -1;
}
z = (byte) zz;
```

`InputStream.read` returns 0 to 255, or -1 at the end of the stream. The cast
to `byte` narrows before the test, so a data byte 0xFF also becomes -1 and is
taken for the end of the stream. The same three lines appear twice, once for
the first character of a group and once for the rest.

**What correct would be** testing the int the stream returned, before
narrowing it.

**Why it matters** only for a malformed stream: 0xFF is not a character an
ASCII85 stream may contain, so a well-formed one never carries it. On a damaged
one the difference shows — the decoder ends the stream quietly where it should
raise `IOException("Invalid data in Ascii85 stream")`, which is what every
other byte outside the alphabet gets.

**Where the Go carries it** `go/pdfbox/filter/ascii85.go`, `readSignificant`,
which narrows to `int8` and tests for -1 exactly as the Java does, with a
comment saying why.

**Fixed in the Go** `track/java-bug-fixes`, entry 27. `readSignificant` reports
the end of the stream from the read error alone, so 0xFF is handed on as the
data byte it is and reaches the alphabet check, which rejects it with `Invalid
data in Ascii85 stream` — the answer every other byte outside the alphabet
already gets, and the one the expected value is read off. The other narrowing
in the method is kept: `z = (byte)(ascii[k] - OFFSET)` wraps in the Java too,
and the check beside it, `z < 0 || z > 93`, is written for the wrapped value.
Tested by `TestASCII85RejectsAnFFByte` in `go/pdfbox/filter/javabug27_test.go`,
with `TestASCII85StillEndsAtTheEndOfItsSource` beside it so a source that runs
out mid-group still ends quietly.

**Confidence** high. The narrowing is visible in the expression.

## 28. `LZWFilter.findPatternCode` returns a negative code for a high byte

**Where** `pdfbox/src/main/java/org/apache/pdfbox/filter/LZWFilter.java`.

**What it does**

```java
private static int findPatternCode(List<byte[]> codeTable, byte[] pattern)
{
    // for the first 256 entries, index matches value
    if (pattern.length == 1)
    {
        return pattern[0];
    }
```

A Java `byte` is signed, so a pattern holding one byte of 0x80 or above returns
a negative code where the comment says the index matches the value. The other
place the encoder computes a single byte code writes `by & 0xff`, which is the
mask this branch is missing.

**What correct would be** `return pattern[0] & 0xFF;`.

**Why it matters** it does not, today: `encode` is the only caller and only
asks about patterns of two bytes or more, so the branch is unreachable. It is
recorded because the next caller would not know that.

**Where the Go carries it** `go/pdfbox/filter/lzw.go`, `findPatternCode`, which
writes `int(int8(pattern[0]))` to keep the sign, with a comment.

**Fixed in the Go** `track/java-bug-fixes`, entry 28. `findPatternCode` reads
the single byte unsigned, so the code it answers is the index its own comment
names and the encoder's other single-byte code, `by & 0xff`, already uses.
Nothing calls the branch today, so the test calls the function, which is
unexported here and private in the Java; the expected value is the table
`createCodeTable` builds, where entry N is the one-byte pattern N.
`TestFindPatternCodeOfOneByteIsItsIndex` in
`go/pdfbox/filter/javabug28_test.go` checks the answer against that table as
well as against the value.

**Confidence** high, for the arithmetic. That nothing reaches it is from
reading the one caller.

## 29. Type 4 `not` negates an integer instead of complementing it

**Where**
`pdfbox/src/main/java/org/apache/pdfbox/pdmodel/common/function/type4/BitwiseOperators.java`.

**What it does**

```java
else if (op1 instanceof Integer)
{
    int int1 = (Integer)op1;
    int result = -int1;
    stack.push(result);
}
```

ISO 32000-1 table 42 gives `not` as "logical | bitwise not", and the PostScript
Language Reference says of the integer operand that `not` "returns the bitwise
complement (ones complement) of its value". The ones complement of 52 is -53;
this returns -52.

**What correct would be** `int result = ~int1;`.

**Why it matters** a type 4 function that applies `not` to an integer computes
something else, and a tint transform or a shading built on one comes out wrong.
It is not caught by the Java's own tests because they assert the behaviour:
`TestOperators.testNot` expects `52 not` to be -52.

**Where the Go carries it** `go/pdfbox/pdmodel/common/function/type4/bitwiseoperators.go`,
`notOperator`, which writes `-v` with a comment; `TestNot` in
`type4_test.go` keeps the Java's expected values.

**Fixed in the Go** `track/java-bug-fixes`, entry 29. The integer arm of `not`
is `^v`, the bitwise complement PostScript defines it as. Tested by
`TestNotOfAnIntegerIsTheComplement` in
`go/pdfbox/pdmodel/common/function/type4/javabugfixes_test.go`; `TestNot` in
`type4_test.go` carried the Java's answers -- 52 not = -52, -37 not = 37 -- and
now carries the complement's, with the Java's kept in its comment.

**Confidence** high. The specification and the code disagree in one character.

## 30. `ASCIIHexFilter` adds -1 for a digit that is not hexadecimal

**Where** `pdfbox/src/main/java/org/apache/pdfbox/filter/ASCIIHexFilter.java`.

**What it does**

```java
if (REVERSE_HEX[firstByte] == -1)
{
    LOG.error("Invalid hex, int: {} char: {} (1st byte)", firstByte, (char) firstByte);
}
int value = REVERSE_HEX[firstByte] * 16;
...
if (REVERSE_HEX[secondByte] == -1)
{
    LOG.error("Invalid hex, int: {} char: {} (2nd byte)", secondByte, (char) secondByte);
}
value += REVERSE_HEX[secondByte];
decoded.write(value);
```

The table holds -1 for every byte that is not a hexadecimal digit, and the
filter logs that and then uses it. So `4Z` decodes to 4 × 16 + (-1) = 63, one
less than the 64 a reader would expect from "the bad digit is a zero"; and an
invalid *first* digit contributes -16, so `Z4` decodes to -12, which
`decoded.write` narrows to 0xF4.

**What correct would be** treating the entry as zero after logging, or
refusing the stream. The specification says a conforming reader may ignore
characters outside the alphabet, which is neither of these.

**Why it matters** only for a malformed stream, which is exactly when a filter
is asked to be predictable. One wrong digit shifts the byte around it rather
than the byte itself, and the error is silent past the log.

**Where the Go carries it** `go/pdfbox/filter/asciihex.go`, `Decode`, which
multiplies and adds the table entry as Java does; `TestASCIIHexTolerance` in
`fromsource_test.go` pins both cases with the arithmetic written out.

**Fixed in the Go** `track/java-bug-fixes`, entry 30. `Decode` logs a byte that
is not a hexadecimal digit and then reads it as zero, so one bad digit changes
its own nibble and nothing else. Of the two answers the entry names, this is
the one the method already gives for a digit it does not have — "second value
behaves like 0 in case of EOD", four lines above — and refusing the stream
would make the filter stricter than the reader it is, which drops the `>` and
the whitespace without a word. Tested by `TestASCIIHexTreatsABadDigitAsZero` in
`go/pdfbox/filter/javabug30_test.go`; the two rows of `TestASCIIHexTolerance`
in `fromsource_test.go` that held 63 and 0xF4 now hold 0x40 and 0x04, with the
Java's values in the comment.

**Confidence** high. The port's test was written expecting 64 and measured 63.

## 31. `SampledImageReader.from8bit` writes a region to the wrong rows

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/graphics/image/SampledImageReader.java`.

**What it does**

```java
if (currentSubsampling == 1)
{
    // Not the entire region was requested, but if no subsampling should
    // be performed, we can still copy the entire part of this row
    System.arraycopy(tempBytes, startx * numComponents, bank,
            y * inputWidth * numComponents, scanWidth * numComponents);
}
```

`bank` is the destination raster, `width` by `height`, where `width` and
`height` are the *clipped region*. The destination offset is computed from `y`,
which is the row of the *source* image, and from `inputWidth`, which is the
width of the *source* image. Both are wrong for the destination: the row should
be `y - starty` and the stride should be `width`.

For any region that is a strict subset the copy lands on the wrong row and, once
`y` is large enough, past the end of `bank`. The exception is
ArrayIndexOutOfBoundsException, and `getRGBImage` catches only
`NegativeArraySizeException` and `IllegalArgumentException`, so it escapes.

**What correct would be**

```java
System.arraycopy(tempBytes, startx * numComponents, bank,
        (y - starty) * width * numComponents, scanWidth * numComponents);
```

which is what the subsampled branch below it computes by running an index.

**Why it matters** it is reachable: `getRGBImage(pdImage, region, subsampling,
colorKey)` with a region, a subsampling of 1 and an 8-bit image whose decode
array is the default. The renderer asks for a region when it draws a tiling
pattern, and `PDImageXObject.getImage(Rectangle, int)` is public. The other
three paths through the reader -- `from1Bit`, `fromAny` and the fast copy --
index correctly.

**Where the Go carries it** `go/pdfbox/pdmodel/graphics/image/sampledimagereader.go`,
`from8bit`, which computes the same offset and so panics where Java throws. The
comment there names this entry.

**Fixed in the Go** `track/java-bug-fixes`, entry 31. `from8bit` writes the row
at `(y - starty) * width * numComponents`, the destination row and the
destination stride, which is the offset the entry names and the one the
subsampled branch below reaches by running an index. The filter-subsampled arm
is untouched by construction: it sets `startx` and `starty` to 0 and
`inputWidth` to `width`, so the two expressions are the same there. Tested by
`TestFrom8BitRegionLandsOnTheRightRows` in
`go/pdfbox/pdmodel/graphics/image/javabug31_test.go`, which asks for a
rectangle away from both edges of an eight by six image and compares every
pixel with the source; without the fix it does not merely differ, it panics on
the region's second row, which is the Java's
`ArrayIndexOutOfBoundsException`. `TestFrom8BitWholeImageIsUnchanged` beside it
keeps the whole-image fast path.

**Confidence** high. The line beside it, for the subsampled case, does the same
job with a running index and gets it right.

## 32. A truncated ASCII85 stream repeats its last complete group

**Where** `pdfbox/src/main/java/org/apache/pdfbox/filter/ASCII85InputStream.java`.

**What it does** `read()` sets `index = 0` before it reads a group, and returns
-1 from inside the group loop where the stream ends part way through one:

```java
index = 0;
...
ascii[0] = z;
for (k = 1; k < 5; ++k)
{
    do
    {
        int zz = (byte) in.read();
        if (zz == -1)
        {
            eof = true;
            return -1;      // n is still the previous group's 4
        }
        ...
```

`n` keeps the previous group's value. The array read then starts with

```java
if (eof && index >= n) { return -1; }
```

which is false, because `index` was just reset to 0 and `n` is 4 — so it copies
`b[0..3]` out a second time. `transferTo` calls it again, gets those four bytes,
and only then reaches the end.

**What correct would be** setting `n = 0` beside `eof = true` on that path, or
resetting `index` only once a group has actually been read.

**Why it matters** a truncated ASCII85 stream — which is what a damaged PDF has
— decodes to the right bytes followed by four bytes of the previous four
repeated. Silently, and only at the end, which is where a reader is least
likely to look.

**Where the Go carries it** `go/pdfbox/filter/ascii85.go`, `readByte` and
`Read`, which keep the same order; `TestASCII85DamageTolerance` in
`fromsource_test.go` asserts the repeat and says why.

**Fixed in the Go** `track/java-bug-fixes`, entry 32. Both paths that give up
part way through a group set `n` to 0 beside `eof`, which is the first of the
two answers the entry names; the second — resetting `index` only once a group
has been read — would move a line the terminator path also depends on. A
truncated stream now decodes to a prefix of the original and stops. Tested by
`TestASCII85TruncatedStreamDoesNotRepeatItsLastGroup` in
`go/pdfbox/filter/javabug32_test.go`, with
`TestASCII85WholeStreamIsUnchanged` beside it for the end that is not
truncation. `TestASCII85DamageTolerance` in `fromsource_test.go`, which
asserted the repeat, now asserts the prefix.

**Confidence** high. Measured: the port decoded 680 bytes from a stream whose
first 676 are the original, and the last four repeat bytes 672 to 675.

---

## 33. `ToUnicodeWriter.allowDestinationRange` checks only one of its two strings

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/font/ToUnicodeWriter.java`,
`allowDestinationRange`.

**What it does**

```java
static boolean allowDestinationRange(String prev, String next)
{
    ...
    int prevCode = prev.codePointAt(0);
    int nextCode = next.codePointAt(0);
    return allowCodeRange(prevCode, nextCode) && prev.codePointCount(0, prev.length()) == 1;
}
```

Both strings are destinations of a `bfrange`, and a range is written as one
starting destination that the reader increments. That is only correct if every
destination in the range is a single code point. The method checks `prev` and
not `next`.

**What correct would be** `&& next.codePointCount(0, next.length()) == 1` as
well — or, equivalently, refusing the range whenever either side is longer than
one code point.

**Why it matters** a CID mapped to a one character string followed by a CID
mapped to a longer one extends the range instead of starting a new one, and
everything after the first character of the longer string is dropped from the
CMap. With 0x400 mapped to `a` and 0x401 mapped to `bc`, the writer emits
`<0400> <0401> <0061>`, and a reader decodes 0x401 as `b`. The ligature the
mapping existed for is lost, silently, in a file that otherwise looks correct.
It does not fire in `TestToUnicodeWriter.testCMapLigatures` only because the
ligatures there — `ff`, `fi`, `ffl` — all start with `f`, so `allowCodeRange`
rejects them first.

**Where the Go carries it**
`go/pdfbox/pdmodel/font/tounicodewriter.go`, `allowDestinationRange`, which
checks `utf8.RuneCountInString(prev) == 1` and not `next`. The comment above it
names this entry.

**Confidence** certain, and **measured** since `track/font-embedding`'s D9.
`ToUnicodeWriter` and `util/Hex`/`util/StringUtil` compile on their own with
`javac`, so the case above was run: for `add(0x400, "a")` and
`add(0x401, "bc")` the running Java writes

```
1 beginbfrange
<0400> <0401> <0061>
endbfrange
```

with the `c` nowhere in the CMap, and answers `allowDestinationRange("a","bc")
= true` against `allowDestinationRange("ab","c") = false`. The port writes the
same bytes; `TestCMapDropsTheTailOfALongerDestination` in
`tounicodewriter_test.go` holds both, with the Java output as the wanted value.
(This entry previously said the case was derived rather than measured, "because
there is no Maven in this environment" — Maven is not needed for a class whose
only dependency is two utility classes.)

---

## 34. `PageExtractor.extract` throws where its own javadoc promises a blank document

**Where** `pdfbox/src/main/java/org/apache/pdfbox/multipdf/PageExtractor.java`,
`extract`, together with `Splitter.setEndPage`.

**What it does**

```java
public PDDocument extract() throws IOException
{
    if (endPage - startPage + 1 <= 0)
    {
        return new PDDocument();
    }
    Splitter splitter = new Splitter();
    splitter.setStartPage(Math.max(startPage, 1));
    splitter.setEndPage(Math.min(endPage, sourceDocument.getNumberOfPages()));
    ...
}
```

and its javadoc says

> If startPage is greater than endPage or greater than the number of pages in
> the source document, a blank document will be returned.

The guard covers only the first half of that sentence. A start page beyond the
end of the document with an end page beyond it too — say pages 30 to 40 of a 28
page file — passes the guard, because `40 - 30 + 1` is 11. Then `setStartPage`
is given 30 and `setEndPage` is given `min(40, 28)`, which is 28, and

```java
if (end < startPage)
{
    throw new IllegalArgumentException("End page is smaller than startPage");
}
```

fires. The blank document is never returned.

**What correct would be** clamping the start page against the page count in the
same guard, so that `startPage > getNumberOfPages()` also returns a blank
document — or reordering `setEndPage` before `setStartPage`, which would remove
the cross-check but is not what the javadoc promises either.

**Why it matters** the javadoc is the contract callers read, and it says the
out-of-range case is handled. It is not: the caller gets an unchecked exception
from two frames down, in a class it never named.

**Where the Go carries it** `go/pdfbox/multipdf/pageextractor.go`, `Extract`,
which has the same guard and the same order of the two setters, and
`splitter.go`, whose `SetEndPage` panics with the same message —
`IllegalArgumentException` is unchecked, so the port panics.
`TestExtractBeyondTheDocumentPanics` in `pageextractor_test.go` pins it and
names this entry.

**Confidence** high. Read from the two methods and confirmed by the port
panicking with `End page is smaller than startPage` for pages 30 to 40 of the
28 page `cweb.pdf`, which is the document `PageExtractorTest` uses.

---

## 35. `COSName.BEAD` is `"BEAD"`, and the specification says `/Bead`

**Where** `pdfbox/src/main/java/org/apache/pdfbox/cos/COSName.java` line 100:

```java
public static final COSName BEAD = getPDFName("BEAD");
```

used in one place, `PDThreadBead`'s no-argument constructor:

```java
bead.setItem(COSName.TYPE, COSName.BEAD);
```

**What correct would be** `getPDFName("Bead")`. PDF 32000-1:2008 Table 30 gives
the thread bead dictionary a `/Type` of `Bead`, and PDF names are
case-sensitive, so `/BEAD` is a different name.

**Why it matters** a bead PDFBox creates is written with `/Type /BEAD`. Nothing
in PDFBox reads that entry back --- `PDThreadBead` never tests it, and the
constant has no other use --- so PDFBox round-trips its own output. A conforming
reader looking for `/Type /Bead` does not find one. It only bites a file PDFBox
wrote, which is why it has survived: the reading path never touches it.

**Where the Go carries it** `go/pdfbox/cos/names.go` already had it as
`BEAD = GetPDFName("BEAD")`, transcribed from the Java in slice 1, and
`go/pdfbox/pdmodel/interactive/pagenavigation/pdthread.go`, `NewPDThreadBead`,
writes it. The name is deliberately spelled `cos.BEAD` rather than `cos.Bead` so
that it does not read like the correct one.

**Confidence** high for the code; the specification reading is from Table 30 of
PDF 32000-1:2008. No test resource in the repository carries a bead dictionary,
so there is nothing to measure it against.

---

## 36. `PDWindowsLaunchParams.setOperation` writes the wrong key

**Where**
`pdfbox/src/main/java/org/apache/pdfbox/pdmodel/interactive/action/PDWindowsLaunchParams.java`:

```java
public String getOperation()
{
    return params.getString(COSName.O, OPERATION_OPEN);
}

public void setOperation( String op )
{
    params.setString( COSName.D, op );
}
```

The getter reads `/O` and the setter writes `/D`.

**What correct would be** `params.setString(COSName.O, op)`.

**Why it matters** two things go wrong at once, and neither is visible from the
class. Setting the operation silently overwrites `/D`, which is the working
directory that `setDirectory` wrote and `getDirectory` reads --- so a launch
action given both a directory and an operation loses the directory. And the
operation itself is never stored, so `getOperation` keeps returning its default,
`"open"`, however many times it is set. A launch action built through this class
can never say `"print"`.

**Where the Go carries it**
`go/pdfbox/pdmodel/interactive/action/actions.go`, `SetOperation`, with the
comment above it naming this entry.

**Confidence** high. It is two adjacent methods reading and writing different
constants, and `PDWindowsLaunchParams` has no other use of either key beyond
`getDirectory` and `setDirectory`, which is what makes the collision real rather
than harmless.

---

## 37. `PDMarkInfo.setSuspect` ignores its argument

**Where**
`pdfbox/src/main/java/org/apache/pdfbox/pdmodel/documentinterchange/logicalstructure/PDMarkInfo.java`:

```java
public void setSuspect( boolean suspect )
{
    dictionary.setBoolean( "Suspects", false );
}
```

**What correct would be** `dictionary.setBoolean("Suspects", suspect)`. The
three setters beside it --- `setMarked`, `setUserProperties` --- all pass their
argument through, which is what makes this one stand out as a slip rather than a
decision.

**Why it matters** `/Suspects` can never be set to true through PDFBox.
PDF 32000-1:2008 Table 321 gives it as the flag that says the tagged-PDF
structure may not conform to the standard, so a producer that has reason to
raise it cannot. Reading is unaffected: `isSuspect` returns whatever the file
holds.

**Where the Go carries it**
`go/pdfbox/pdmodel/documentinterchange/logicalstructure/pdmarkinfo.go`,
`SetSuspect`, with the comment above it naming this entry.

**Confidence** high. The parameter is unused and the literal is written in its
place; there is no reading of the method under which it is correct.

## 38. `PDUserAttributeObject` reads `/P` without checking it is there

**Where**
`pdfbox/src/main/java/org/apache/pdfbox/pdmodel/documentinterchange/logicalstructure/PDUserAttributeObject.java`:

```java
public List<PDUserProperty> getOwnerUserProperties()
{
    COSArray p = getCOSObject().getCOSArray(COSName.P);
    List<PDUserProperty> properties = new ArrayList<>(p.size());
    ...
}
```

`addUserProperty` and `removeUserProperty` read `/P` the same way and use it
without a check.

**What correct would be** `getCOSArray` returns null when the entry is absent
or is not an array, so all three need the null guard the rest of the class
gives its entries: an empty list from the getter, and a new `COSArray` put into
`/P` from the two setters.

**Why it matters** A user attribute object with no `/P` --- which is what
`new PDUserAttributeObject()` builds, since its constructor writes only `/O` ---
throws a `NullPointerException` from every one of the three. So the object
cannot be filled in through its own API: `addUserProperty` on a fresh one always
throws, and the only way in is `setUserProperties`, which writes the array
first. PDF 32000-1:2008 Table 328 marks `/P` required, so a malformed file
reaches the same throw on the reading side.

**Where the Go carries it**
`go/pdfbox/pdmodel/documentinterchange/logicalstructure/pdattributeobject.go`,
`OwnerUserProperties`, `AddUserProperty` and `RemoveUserProperty`. A nil
`*cos.Array` panics on the first method call, the way the null does in Java, and
each carries a comment naming this entry.

**Confidence** high. `COSDictionary.getCOSArray` returns null by contract, and
none of the three tests for it.

## 39. `PDStructureNode.insertBefore` inserts at -1 when the reference kid is not found

**Where**
`pdfbox/src/main/java/org/apache/pdfbox/pdmodel/documentinterchange/logicalstructure/PDStructureNode.java`:

```java
protected void insertBefore(COSBase newKid, Object refKid)
{
    ...
    if (k instanceof COSArray)
    {
        COSArray array = (COSArray) k;
        int refIndex = array.indexOfObject(refKidBase);
        array.add(refIndex, newKid.getCOSObject());
    }
    ...
}
```

**What correct would be** `indexOfObject` answers -1 when the kid is not in the
array, and `COSArray.add(int, COSBase)` hands that to `List.add(int, E)`, which
throws `IndexOutOfBoundsException`. The branch needs to check the index before
using it --- the single-kid branch below it does exactly that, doing nothing
when the reference kid does not match.

**Why it matters** Two ordinary calls reach it. One is a `refKid` that is
simply not a kid of this node. The other is any marked-content identifier taken
from `getKids`, which returns those as `java.lang.Integer`: an Integer is not a
`COSObjectable`, so `refKidBase` stays null, `indexOfObject(null)` answers -1,
and the insert throws --- even though `PDStructureElement.insertBefore(COSInteger,
Object)` exists to insert one identifier before another. The javadoc promises
neither exception.

**Where the Go carries it**
`go/pdfbox/pdmodel/documentinterchange/logicalstructure/pdstructurenode.go`,
`InsertBeforeBase`, with the comment above it naming this entry. `Array.AddAt`
at -1 panics on the slice bounds, which is the same failure.

**Confidence** high. Both the -1 and the throw follow from the two library
contracts, and the sibling branch shows the check that is missing.

## 40. `PDStandardAttributeObject` writes a string array and reads a name array

**Where**
`pdfbox/src/main/java/org/apache/pdfbox/pdmodel/documentinterchange/taggedpdf/PDStandardAttributeObject.java`:

```java
protected String[] getArrayOfString(String name)
{
    ...
    strings[i] = ((COSName) array.getObject(i)).getName();
    ...
}

protected void setArrayOfString(String name, String[] values)
{
    ...
    array.add(new COSString(value));
    ...
}
```

**What correct would be** The two halves have to agree. PDF 32000-1:2008
Table 337 gives `/Headers`, the only attribute that uses them, as an array of
byte strings, so the setter is right and the getter should read `COSString`.

**Why it matters** A round trip throws. `setHeaders(new String[] {"h1"})`
followed by `getHeaders()` is a `ClassCastException`, on
`PDTableAttributeObject` and on `PDExportFormatAttributeObject` alike, and
`toString` calls the getter, so printing a table attribute object that was
filled in through its own API throws too. Reading a conforming file throws for
the same reason: the file holds strings.

**Where the Go carries it**
`go/pdfbox/pdmodel/documentinterchange/taggedpdf/pdstandardattributeobject.go`,
`GetArrayOfString`, with the comment above it naming this entry. The type
assertion panics where the cast throws.

**Confidence** high. The two methods are next to each other and disagree on the
element type; only one of them can match the specification, and it is not the
getter.

## 41. `PDFourColours` pads a short array to five entries

**Where**
`pdfbox/src/main/java/org/apache/pdfbox/pdmodel/documentinterchange/taggedpdf/PDFourColours.java`:

```java
public PDFourColours(COSArray array)
{
    this.array = array;
    // ensure that array has 4 items
    if (this.array.size() < 4)
    {
        for (int i = (this.array.size() - 1); i < 4; i++)
        {
            this.array.add(COSNull.NULL);
        }
    }
}
```

**What correct would be** `for (int i = this.array.size(); i < 4; i++)`. The
comment above the loop says four; the loop starts one index early, so it always
runs one time too many.

**Why it matters** Every array shorter than four comes out five long, whatever
its length: an empty one gets five nulls, a three-entry one gets two more. The
extra entry is written back to the file, since the constructor mutates the array
it was handed rather than a copy --- so reading a `/BorderColor` of three
colours through `getColorOrFourColors` cannot happen (that path needs size 3 or
4), but any caller that builds `PDFourColours` over a short array corrupts it.
PDF 32000-1:2008 Table 344 gives `/BorderColor` as one colour or exactly four.

**Where the Go carries it**
`go/pdfbox/pdmodel/documentinterchange/taggedpdf/pdfourcolours.go`,
`NewPDFourColoursOfArray`, with the comment above it naming this entry.

**Confidence** high. The loop bound is off by one against its own comment, and
the arithmetic is not in doubt.

## 42. `StandardStructureTypes.types` collects itself

**Where**
`pdfbox/src/main/java/org/apache/pdfbox/pdmodel/documentinterchange/taggedpdf/StandardStructureTypes.java`:

```java
public static final List<String> types = new ArrayList<>();
static
{
    Field[] fields = StandardStructureTypes.class.getFields();
    for (Field field : fields)
    {
        if (Modifier.isFinal(field.getModifiers()))
        {
            try
            {
                types.add(field.get(null).toString());
            }
            ...
```

**What correct would be** The loop wants the `String` constants, so it has to
skip anything that is not one --- a `field.getType() == String.class` test, or
naming the constants outright.

**Why it matters** `types` is itself a public final field, so `getFields`
answers it too and the loop adds `types.toString()` to `types`. The list
therefore holds one entry that is not a structure type, and which entry that is
depends on where `types` falls in `Class.getFields`, whose order the Java
language does not define: first gives `"[]"`, last gives the whole list printed
as one string, anything between gives a partial one. Nothing in PDFBox reads
`types`, which is why it has gone unnoticed; a caller that does gets a list it
cannot trust to hold only type names.

**Where the Go carries it** It does not: Go has no reflection over constants,
so there is nothing to enumerate and nothing that could pick itself up.
`go/pdfbox/pdmodel/documentinterchange/taggedpdf/standardstructuretypes.go`
names the forty-seven types and sorts them, and the comment above `Types` says
that the self-referential entry is left out and why. This is the one entry in
this file the port does not carry, because carrying it would mean inventing an
entry whose content Java does not define either.

**Confidence** high for the defect. The field is public and final and the loop
tests only for final.

---

## 43. `PDSeedValue` writes strings into `/Reasons` and `/LegalAttestation` and reads them back as names

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/interactive/digitalsignature/PDSeedValue.java`,
`getReasons`/`setReasons` and `getLegalAttestation`/`setLegalAttestation`.

**What** The setters write `COSString`s and the getters read `COSName`s:

```java
public List<String> getReasons()
{
    COSArray fields = dictionary.getCOSArray(COSName.REASONS);
    return fields != null ? fields.toCOSNameStringList() : Collections.emptyList();
}

public void setReasons(List<String> reasons)
{
    dictionary.setItem(COSName.REASONS, COSArray.ofCOSStrings(reasons));
}
```

and `COSArray.toCOSNameStringList` is

```java
return objects.stream().map(o -> ((COSName) o).getName()).collect(Collectors.toList());
```

**What correct would be** Both entries are text strings in the specification
(PDF 32000-1 table 234: `Reasons` is "an array of text strings"; the same for
`LegalAttestation`), so the setters are right and the getters should read
`toCOSStringStringList`, the way `PDSeedValueCertificate.getKeyUsage` reads the
strings it writes.

**Why it matters** `getReasons` on any dictionary `setReasons` wrote throws
`ClassCastException`, because the cast to `COSName` fails on the first entry.
The same holds for `getLegalAttestation`. Reading a real file is no better: a
conforming writer puts text strings there, and the getter throws on those too.
The pair is only usable if the reader and the writer are both PDFBox and the
entry was written by hand as names. Nothing in PDFBox calls either getter, which
is why it has gone unnoticed. `getSubFilter` and `getDigestMethod` on the same
class are correct, because `setSubFilter` and `setDigestMethod` write names.

**Where the Go carries it** `go/pdfbox/pdmodel/interactive/digitalsignature/pdseedvalue.go`,
`Reasons` and `LegalAttestation` read `cos.Array.ToNameStringList`, which panics
on an entry that is not a name — Go's answer to the `ClassCastException`. Both
carry a comment pointing here.

**Confidence** high. The cast is unconditional and the setter is right next to
it.

---

## 44. `FDFAnnotationFreeText.getRotation` reads a string that `setRotation` wrote as an integer

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/fdf/FDFAnnotationFreeText.java`,
`getRotation`/`setRotation`.

**What** The setter writes an integer and the getter reads a string:

```java
public final void setRotation(int rotation)
{
    annot.setInt(COSName.ROTATE, rotation);
}

public String getRotation()
{
    return annot.getString(COSName.ROTATE);
}
```

`COSDictionary.getString(COSName)` answers the value only when the entry is a
`COSString`, and null otherwise.

**What correct would be** `/Rotate` is an integer in the specification (PDF
32000-1 table 30 for a page, and the XFDF `rotation` attribute the constructor
reads is parsed with `Integer.parseInt`), so the setter is right and the getter
should read `annot.getInt(COSName.ROTATE, 0)` — as the sibling `getJustification`
does, which reads `getInt` and formats it.

**Why it matters** `getRotation` answers null for every free text annotation the
FDF import produced, and for every conforming file, because `/Rotate` is a
number in both. It can only answer non-null for a file that wrote `/Rotate` as a
string, which no writer does. Nothing inside PDFBox calls it, which is why it
has gone unnoticed; a caller that reads the rotation of an imported annotation
gets null and has no way to tell "no rotation" from "rotation present".

**Where the Go carries it**
`go/pdfbox/pdmodel/fdf/annotations2.go`, `FDFAnnotationFreeText.Rotation`, which
reads `GetString` and so answers the empty string — the port's null — for
anything `SetRotation` wrote. Its comment points here.

**Confidence** high. The two methods sit five lines apart and the constructor
right above them parses the attribute as an int.

---

## 45. `FDFDictionary.getPages` reads the array with `get` rather than `getObject`

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/fdf/FDFDictionary.java`,
`getPages`.

**What** Every other list accessor of the class resolves indirect references and
this one does not:

```java
pages.add(new FDFPage((COSDictionary) pageArray.get(i)));
```

against `getFields`, three methods above, which reads

```java
fields.add(new FDFField((COSDictionary) fieldArray.getObject(i)));
```

`COSArray.get` answers the entry as it stands, which is a `COSObject` for an
indirect reference; `getObject` dereferences it first.

**What correct would be** `getObject(i)`, as `getFields`, `getAnnotations` and
`FDFField.getKids` all use.

**Why it matters** `/Pages` holds an array of dictionaries, and a writer is free
to make each of them an indirect object — the FDF specification places no
restriction there, and PDFBox's own writer emits indirect objects for anything
the trailer reaches. Reading such a file throws `ClassCastException` from
`getPages`, because a `COSObject` is not a `COSDictionary`. Only a file whose
pages are all direct dictionaries can be read. `getEmbeddedFDFs` on the same
class has the same shape, but there it is harmless: `PDFileSpecification.createFS`
takes a `COSBase` and dereferences it itself.

**Where the Go carries it** `go/pdfbox/pdmodel/fdf/fdfdictionary.go`, `Pages`,
which reads `cos.Array.Get` and panics on an entry that is not a dictionary —
the port's `ClassCastException`. Its comment points here.

**Confidence** high. The two accessors sit in the same file and differ only in
that one call.

---

## 46. `PDStreamTest` builds its stop filters from `COSName.toString()`

**Where** `pdfbox/src/test/java/org/apache/pdfbox/pdmodel/common/PDStreamTest.java`,
`testCreateInputStreamNullFilters` and `testCreateInputStreamEmptyFilters`.

**What** Both build the stop filter list like this:

```java
List<String> stopFilters = new ArrayList<>();
stopFilters.add(COSName.DCT_DECODE.toString());
stopFilters.add(COSName.DCT_DECODE_ABBREVIATION.toString());
```

`COSName.toString()` is `"COSName{" + getName() + "}"`, so the two entries are
`"COSName{DCTDecode}"` and `"COSName{DCT}"`. The method they are handed to
compares them against the filter's plain name:

```java
for (COSName nextFilter : filters)
{
    if (stopFilters.contains(nextFilter.getName()))
```

`"DCTDecode"` is never equal to `"COSName{DCTDecode}"`, so the stop list can
never match anything.

**What correct would be** `COSName.DCT_DECODE.getName()`, which is what every
caller of `createInputStream(List<String>)` in the main tree passes --
`PDInlineImage` and `SampledImageReader` both build their lists out of
`getName()`.

**Why it matters** Only for the test. Both cases run against a stream with no
filters at all, so the loop the stop list guards never runs a second iteration
and the assertions pass either way; what the test does not do is exercise the
stopping it is named after. `testCreateInputStreamNullStopFilters`, the third
case, passes `null` and is the only one whose name matches what it checks.
Nothing in the shipped library is affected.

**Where the Go carries it** `go/pdfbox/pdmodel/common/pdstream_external_test.go`,
`dctStopFilters`, which builds the same two strings through `cos.Name.String()`
and says why. The stopping the Java test does not reach is covered separately by
`TestCreateInputStreamStoppingStops` in the same package, which slice 6 wrote
and which is not a port.

**Confidence** high. Both halves are three lines apart and `COSName.toString`
has carried the braces since the class was written.

---

## 47. `SignatureOptions.close` loses the first of two close failures

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/interactive/digitalsignature/SignatureOptions.java`,
`close`.

```java
public void close() throws IOException
{
    try
    {
        if (visualSignature != null)
        {
            visualSignature.close();
        }
    }
    finally
    {
        if (pdfSource != null)
        {
            pdfSource.close();
        }
    }
}
```

**What** A `finally` that throws replaces the exception already in flight. Where
`visualSignature.close()` fails and `pdfSource.close()` then fails too, the
caller is told about the second and never hears about the first, which is the
one that says the visual signature template was left in a bad state.

**What correct would be** What PDFBox does everywhere else it closes two things:
`IOUtils.closeAndLogException(a, LOG, "a", null)` threaded through both, which
keeps the first failure and logs the rest. `PDDocument.close`, three classes
away, is written that way.

**Why it matters** Little, in practice. Both resources are closed either way, so
nothing leaks; only which of two failures reaches the caller differs, and two
failing closes in one call is already an unusual state. It is here because it is
a difference the port does not reproduce, not because it breaks a file.

**Where the Go carries it** It does not.
`go/pdfbox/pdmodel/interactive/digitalsignature/signing.go`, `Close`, keeps the
first error and closes both, which is the convention every other `Close` in the
port follows and which is what the Java would do if it used its own helper. This
is a deliberate divergence, left as it stands.

**Confidence** high for the behaviour; it is how Java has always treated a
throwing `finally`. Calling it a defect rather than a style is a judgement, and
that is why the entry says what it costs.

---

## 48. `PDDocumentCatalog.getAcroForm()` changes the document it is asked to read

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/PDDocumentCatalog.java`,
`getAcroForm()` and `getAcroForm(PDDocumentFixup)`.

```java
public PDAcroForm getAcroForm()
{
    return getAcroForm(new AcroFormDefaultFixup(document));
}
```

**What** The no-argument accessor applies `AcroFormDefaultFixup`, which is not a
read: it walks the form and rewrites it — generating appearance streams where
`/NeedAppearances` is set, adopting orphan widgets into fields, adding font
resources to `/DR`. Java's own javadoc says so:

> Using `getAcroForm(PDDocumentFixup acroFormFixup)` might change the original
> content and subsequent calls with `getAcroForm(null)` will return the changed
> content.

So a caller that reads the form and then asks for it again with `null`, wanting
the file as it was, gets the repaired version instead. The catalogue remembers
the fixup it applied and never unapplies it.

**What correct would be** A getter that does not mutate, with the repair on a
method that says it repairs. The shape is deliberate — it is what makes PDFBox
read broken forms out of the box — so this is surprising behaviour rather than
a mistake, and it is recorded for the reader who finds a document changed by a
call that looked like a read.

**Where the Go carries it** `go/pdfbox/pdmodel/interactive/form/catalogacroform.go`,
`AcroFormOfCatalog`, applies the same fixup and mutates the same way — **but
only when the program has linked `pdmodel/fixup`.** `fixup` names `PDAcroForm`,
so `form` cannot import it; it sets `form.NewAcroFormDefaultFixup` from its own
`init`, and a program that never blank-imports the package gets no fixup at all.
The same call therefore has two behaviours depending on the import graph, where
Java has one. That is a divergence of the port, not of the Java, and it is left
as it stands: closing it would mean moving the fixups out of the package Java
puts them in. `migration/STATUS.md` says the same under the slice 8 fixup
section.

**Confidence** high. The javadoc states the mutation itself.

---

## 49. `PageDrawer.drawTilingPattern` restores nothing when the pattern stream fails

**Where** `pdfbox/src/main/java/org/apache/pdfbox/rendering/PageDrawer.java`,
the package-private `drawTilingPattern`.

```java
Graphics2D savedGraphics = graphics;
graphics = g;
GeneralPath savedLinePath = linePath;
linePath = new GeneralPath();
...
setRenderingHints();
processTilingPattern(pattern, color, colorSpace, patternMatrix);

flipTG = savedFlipTG;
graphics = savedGraphics;
linePath = savedLinePath;
lastClips = savedLastClips;
initialClip = savedInitialClip;
clipWindingRule = savedClipWindingRule;
```

**What** The method swaps six fields of the drawer, runs the pattern's content
stream, and swaps them back — but the six restores are plain statements, not a
`finally`. `processTilingPattern` declares `throws IOException` and every
operator it runs can raise one. When it does, the exception leaves the drawer
pointing at the tile's `Graphics2D`, with the tile's line path, the tile's clip
bookkeeping and `flipTG` still true. The drawer is a field of `PageDrawer`, and
`TilingPaint` calls this while the page is still being rendered, so the rest of
the page is then drawn onto a disposed tile surface.

The three sibling methods that swap the same fields all use `finally` — the
`TransparencyGroup` constructor, `processTransparencyGroup` and
`processTilingPattern` itself. This one does not, which is what makes it look
like an oversight rather than a choice.

**What correct would be** The same six lines in a `finally`, as its siblings
have.

**Where the Go carries it**
`go/pdfbox/rendering/pagedrawer.go`, `DrawTilingPattern` returns the error
before the restores, with a comment saying why.

**Confidence** high for the shape — the restores are demonstrably outside any
`finally`, and the siblings show the intended pattern. Whether it is reachable
in practice depends on `TilingPaint` swallowing the exception, which is
`PaintContext` territory this port does not reach.

---

## 50. `PageDrawer.showAnnotation` leaves the page rotated when an annotation fails

**Where** `pdfbox/src/main/java/org/apache/pdfbox/rendering/PageDrawer.java`,
`showAnnotation`, the `isNoRotate()` branch.

```java
AffineTransform savedTransform = graphics.getTransform();
graphics.rotate(Math.toRadians(getCurrentPage().getRotation()),
        rect.getLowerLeftX(), rect.getUpperRightY());
super.showAnnotation(annotation);
graphics.setTransform(savedTransform);
annotation.setAppearance(appearance); // restore
```

**What** An annotation with the NoRotate flag on a rotated page is drawn through
a rotation of its own, and the transform is put back afterwards — again with a
plain statement rather than a `finally`. `super.showAnnotation` declares
`throws IOException`. When it raises one, the page's `Graphics2D` keeps the
annotation's rotation, and every annotation after it in `drawPage`'s loop is
drawn through that rotation as well.

The second line is worse than the first. A few lines earlier the method may have
called `annotation.constructAppearances()` to replace the appearance it is about
to draw, having saved the original in `appearance`; the `setAppearance` here is
the undo. Skipping it leaves the document holding a generated appearance in
place of the one the file carried, which outlives the render.

**What correct would be** Both restores in a `finally`.

**Where the Go carries it** `go/pdfbox/rendering/pagedrawer.go`,
`ShowAnnotation` returns the error before the two restores, with a comment
saying why.

**Confidence** high. The document mutation is the same `setAppearance` the
comment in the Java calls "restore".

---

## 51. `PDColorSpace.create` drops the resources when it builds a pattern's underlying colour space

**Where**
`pdfbox/src/main/java/org/apache/pdfbox/pdmodel/graphics/color/PDColorSpace.java`,
the `/Pattern` arm of `create(COSBase, PDResources, boolean)` that handles an
array.

```java
else if (name == COSName.PATTERN)
{
    if (array.size() == 1)
    {
        return new PDPattern(resources);
    }
    else
    {
        return new PDPattern(resources, PDColorSpace.create(array.get(1)));
    }
}
```

**What** An uncoloured tiling pattern names its underlying colour space as the
second entry of `[/Pattern <colourspace>]`. The method is holding a
`PDResources` — it passes it to the `PDPattern` on the very same line — but
builds the underlying colour space with the **one-argument** `create`, which is
`create(colorSpace, null, false)`. So the underlying space is resolved with no
resources at all.

Everything else that recurses in this method passes them on: `PDIndexed`,
`PDSeparation` and `PDDeviceN` all take `resources` and hand them to the base
colour space they build.

What it costs: `[/Pattern /DeviceRGB]` works, because a device name needs no
resources, and `[/Pattern [/ICCBased 5 0 R]]` works, because the array carries
itself. `[/Pattern /CS1]`, naming a colour space in the page's `/ColorSpace`
dictionary, does not — with `resources` null the name falls past every device
arm and out of the bottom of the method, which throws
`MissingResourceException`. The default colour space substitutions
(`/DefaultRGB` and its two siblings) are skipped for the same reason.

**What correct would be** `PDColorSpace.create(array.get(1), resources,
wasDefault)`, as the three sibling recursions do.

**Where the Go carries it** `go/pdfbox/pdmodel/graphics/color/create.go`, the
`cos.Pattern` case of `createFromArray`, calls the one-argument `Create` and
says why above the line.

**Confidence** high for the shape — the one-argument call is right there beside
the `resources` it ignores. Medium for how often it bites: an underlying colour
space written as a bare name rather than inline is legal but unusual, and no
PDFBox test covers it.

---

## 52. `XMPSchema.reorganizeAltOrder` dereferences a missing `xml:lang`

**Where** `xmpbox/src/main/java/org/apache/xmpbox/schema/XMPSchema.java`,
`reorganizeAltOrder`

```java
if (it.hasNext() && it.next().getAttribute(XmpConstants.LANG_NAME).getValue().equals(XmpConstants.X_DEFAULT))
```

and again in the loop below it:

```java
xdefault = it.next();
if (xdefault.getAttribute(XmpConstants.LANG_NAME).getValue().equals(XmpConstants.X_DEFAULT))
```

`getAttribute` answers null for an alternative that carries no `xml:lang`, and
both calls reach straight through it.

**What correct would be** treating a missing qualifier as "not the default
language", which is what every other reader of an alternative does.

**Why it matters** an `rdf:Alt` whose items have no `xml:lang` is malformed but
common, and this is reached from `setUnqualifiedLanguagePropertyValue`, so
adding one language to such an alternative fails with a NullPointerException
rather than doing something.

**Where the Go carries it** it does not, quite: `languageOf` in
`go/xmpbox/schema/xmpschema.go` answers the empty string where the qualifier is
absent, so the comparison is false and the walk continues. Panicking on
malformed input, in a library that exists to read malformed input, was the worse
of the two; the divergence is recorded here and in [`STATUS.md`](STATUS.md).

**Confidence** high. Read from the two dereferences; `getAttribute` is a map
lookup that answers null for a name it does not hold.

---

## 53. `XMPSchema.merge` stops at the first value both schemas already have

**Where** `xmpbox/src/main/java/org/apache/xmpbox/schema/XMPSchema.java`,
`mergeComplexProperty` and its caller `merge`

```java
private boolean mergeComplexProperty(Iterator<AbstractField> itNewValues, ArrayProperty arrayProperty)
{
    while (itNewValues.hasNext())
    {
        TextType tmpNewValue = (TextType) itNewValues.next();
        for (AbstractField abstractField : arrayProperty.getContainer().getAllProperties())
        {
            TextType tmpOldValue = (TextType) abstractField;
            if (tmpOldValue.getStringValue().equals(tmpNewValue.getStringValue()))
            {
                return true;
            }
        }
        arrayProperty.getContainer().addProperty(tmpNewValue);
    }
    return false;
}
```

and

```java
if (mergeComplexProperty(itNewValues, (ArrayProperty) tmpEmbeddedProperty))
{
    return;
}
```

The `true` means "a value was already there", and the caller takes it as a
reason to stop merging the whole schema.

**What correct would be** skipping the duplicate value and carrying on: the loop
over the new values, the loop over this schema's properties, and the loop over
the other schema's properties should all continue.

**Why it matters** merging two Dublin Core schemas that share one creator drops
every property after the array that held the duplicate -- the rest of that
array, every later array, and every later simple property.

**Where the Go carries it** `go/xmpbox/schema/xmpschema.go`, `Merge` and
`mergeComplexProperty`, which return the same way at the same place.

**Confidence** high. Read from the two methods; the `return` is the whole of
`merge`'s remaining work.

---

## 54. `TiffSchema.setArtist` stores a type its own getter cannot see

**Where** `xmpbox/src/main/java/org/apache/xmpbox/schema/TiffSchema.java`,
`setArtist` and `getArtistProperty`

```java
public ProperNameType getArtistProperty()
{
    return getPropertyAs(ARTIST, ProperNameType.class);
}

public void setArtist(String text)
{
    addProperty(createTextType(ARTIST, text));
}
```

`createTextType` builds a plain `TextType`, which is the superclass of
`ProperNameType` and so not an instance of it; `getPropertyAs` answers null for
anything the class is not an instance of.

**What correct would be** `instanciateSimple(ARTIST, text)`, which reads the
field's declared type -- `Types.ProperName` -- and builds the matching class,
the way every other setter of a derived text field does.

**Why it matters** `setArtist("name")` followed by `getArtist()` answers null.
The value is in the schema and is serialized, but no accessor of the class can
read it back.

**Where the Go carries it** `go/xmpbox/schema/schemas4.go`, `SetArtist` and
`ArtistProperty`: the setter goes through `SetTextValue`, which is
`createTextType`, and the getter asks for `*xmptype.ProperNameType`, which a
`*xmptype.TextType` is not.

**Confidence** high. Read from the two methods and from `getPropertyAs`, which
is `type.isInstance(property) ? type.cast(property) : null`.

---

## 55. `XMPMediaManagementSchema.addVersions` writes the wrong kind of array

**Where** `xmpbox/src/main/java/org/apache/xmpbox/schema/XMPMediaManagementSchema.java`,
`addVersions`

```java
@PropertyType(type = Types.Version, card = Cardinality.Seq)
public static final String VERSIONS = "Versions";

public void addVersions(String value)
{
    addQualifiedBagValue(VERSIONS, value);
}
```

The field is declared as an ordered array of the structured `Version` type;
`addQualifiedBagValue` makes an unordered array of text.

**What correct would be** `addUnqualifiedSequenceValue`, which is what
`addHistory` uses for the neighbouring `Seq` field -- and, for the declared
type, adding a `VersionType` rather than a string.

**Why it matters** a document written through this method carries an `rdf:Bag`
of text where the schema says `rdf:Seq` of `stVer:Version`, and reading it back
in strict mode fails with "Invalid array type, expecting Seq and found Bag".

**Where the Go carries it** `go/xmpbox/schema/schemas3.go`, `AddVersions`, which
calls `AddQualifiedBagValue` and says so in its comment.

**Confidence** high. Read from the annotation and the method, and from the
neighbouring `addHistory` that does it the other way.

---

## 56. `DomXmpParser.parseEndPacket` indexes past the end of a short instruction

**Where** `xmpbox/src/main/java/org/apache/xmpbox/xml/DomXmpParser.java`,
`parseEndPacket`

```java
if (xpackData.startsWith("end="))
{
    char end = xpackData.charAt(5);
```

Four characters are checked and the sixth is read.

**What correct would be** checking the length, or matching the whole thing
against `end=['"][rw]['"]` -- the shape the next two lines assume.

**Why it matters** `<?xpacket end=?>` and `<?xpacket end="?>` are malformed
input a parser should refuse with its own exception; instead they raise
StringIndexOutOfBoundsException, which is unchecked and so escapes the
`XmpParsingException` a caller catches.

**Where the Go carries it** `go/xmpbox/xml/domxmpparser.go`, `parseEndPacket`,
which indexes the same byte and panics the same way, with a comment naming this
entry.

**Confidence** high. Read from the method; `"end="` is four characters and
`charAt(5)` needs six.

---

## 57. `DomXmpParser.parseDescriptionInner` reaches through a null type

**Where** `xmpbox/src/main/java/org/apache/xmpbox/xml/DomXmpParser.java`,
`parseDescriptionInner`

```java
PropertyType dtype = checkPropertyDefinition(tm, DomHelper.getQName(property), null);
PropertyType ptype = tm.getStructuredPropMapping(dtype.type()).getPropertyType(name);
```

`checkPropertyDefinition` answers null when the namespace is known but the
property is not declared in it -- that is what its callers elsewhere test for --
and `dtype.type()` is read without the test.

**What correct would be** the treatment `createProperty` gives the same null:
report it in strict mode, and fall back to text in lenient mode.

**Why it matters** a structured value holding a property its type does not
declare fails with a NullPointerException rather than with the parser's own
exception, in both parsing modes.

**Where the Go carries it** it does not, quite: `go/xmpbox/xml/domxmpparser.go`,
`parseDescriptionInner`, reports the property as `NoType` instead. The parse
fails either way, which is the observable behaviour; a panic out of a parser
handed malformed input was the worse of the two. Recorded here and in
[`STATUS.md`](STATUS.md).

**Confidence** high. Read from the method and from `getSpecifiedPropertyType`,
which returns null on the paths its own comment marks as not found.

---

## 58. `XMPMetadata.createAndAddPDFAExtensionSchemaWithNS` ignores its argument

**Where** `xmpbox/src/main/java/org/apache/xmpbox/XMPMetadata.java`,
`createAndAddPDFAExtensionSchemaWithNS`

```java
public PDFAExtensionSchema createAndAddPDFAExtensionSchemaWithNS(Map<String, String> namespaces)
        throws XmpSchemaException
{
    PDFAExtensionSchema pdfAExt = new PDFAExtensionSchema(this);
    pdfAExt.setAboutAsSimple("");
    addSchema(pdfAExt);
    return pdfAExt;
}
```

The javadoc says "This PDFAExtension is created with specified list of
namespaces" and "@throws XmpSchemaException If namespaces list not contains
PDF/A Extension namespace URI". The map is never read and nothing is thrown, so
the method is `createAndAddPDFAExtensionSchemaWithDefaultNS` with an argument.

**What correct would be** either declaring the namespaces on the schema and
checking the extension namespace is among them, or deleting the method.

**Why it matters** a caller that passes a namespace map gets a schema that does
not have it, with nothing to say the argument was dropped.

**Where the Go carries it** `go/xmpbox/xmpmetadata_schemas.go`,
`CreateAndAddPDFAExtensionSchemaWithNS`, which takes the map, ignores it and
says so.

**Confidence** high. The method body is four lines and the parameter appears in
none of them.

---

## 59. `XmpSerializer` cannot write a property whose element had no prefix

**Where** `xmpbox/src/main/java/org/apache/xmpbox/xml/XmpSerializer.java`,
`serializeFields`

```java
// PDFBOX-2378: add namespace declaration to the top
if (!field.getPrefix().isEmpty() && field.getNamespace() != null && !field.getNamespace().isEmpty())
```

`getPrefix()` is null for a property parsed from an element written without one
— which is what PDFBOX-5835 is about, and what the file
`src/test/resources/org/apache/xmpbox/xml/PDFBOX-5835.xml` holds. The guard
null-checks `getNamespace()` on the next line and not `getPrefix()` on this one.

**What correct would be** the same null check the neighbouring term already has.

**Why it matters** the packet parses — `DomXmpParserTest.testPDFBox5835` asserts
what comes out of it — and then cannot be written back:

```
NullPointerException: Cannot invoke "String.isEmpty()" because the return value
of "org.apache.xmpbox.type.AbstractField.getPrefix()" is null
```

Reading a document and writing it out again is the module's main use, and for
this one it throws an unchecked exception rather than the
`XmpSerializationException` its signature declares.

**Where the Go carries it** it does not: `Prefix()` answers the empty string
where Java answers null, so the test is false and the property is written under
its local name. Both halves of the alternative were bad — panicking out of a
serializer handed a document that parsed, or writing a name Java never gets to
write — and writing it keeps the module usable.
`TestSerializingAnUnprefixedPropertyWhereJavaFails` in
`go/xmpbox/xml/nullprefix_test.go` pins what the port writes and names this
entry.

**Confidence** high. Reproduced by compiling `xmpbox` against JDK 17 and running
`DomXmpParser.parse` then `XmpSerializer.serialize` over
`PDFBOX-5835.xml`; the message above is what came out.

---

## 60. `XMPSchema.removeUnqualifiedSequenceDateValue` dereferences an empty date

**Where** `xmpbox/src/main/java/org/apache/xmpbox/schema/XMPSchema.java`,
`removeUnqualifiedSequenceDateValue`

```java
for (AbstractField tmp : seq.getContainer().getAllProperties())
{
    if (tmp instanceof DateType && ((DateType) tmp).getValue().equals(date))
```

`DateType.getValue()` is null for a property built from a blank string — which
is what `<xmp:CreateDate/>` produces, and what PDFBOX-6029 was about — and the
`.equals` goes straight through it.

**What correct would be** `date.equals(((DateType) tmp).getValue())`, which is
the same comparison with the operands the other way round, or a null check.

**Why it matters** a sequence of dates holding one empty element cannot have any
element removed: the walk raises NullPointerException at the empty one, whether
or not the date being removed is there.

**Where the Go carries it** it does not:
`go/xmpbox/schema/xmpschema.go`, `RemoveUnqualifiedSequenceDateValue`, passes
over an element that holds no date. A panic in a remover, over a shape the same
module produces from a legal document, was the worse of the two; the divergence
is recorded here and in [`STATUS.md`](STATUS.md).

**Confidence** high. Reproduced by compiling `xmpbox` against JDK 17: adding an
empty `DateType` to a sequence and calling
`removeUnqualifiedSequenceDateValue` prints `NullPointerException`.

---

## 61. `XMPSchema.getUnqualifiedSequenceDateValueList` puts nulls in the list

**Where** `xmpbox/src/main/java/org/apache/xmpbox/schema/XMPSchema.java`,
`getUnqualifiedSequenceDateValueList`

```java
for (AbstractField child : seq.getContainer().getAllProperties())
{
    if (child instanceof DateType)
    {
        retval.add(((DateType) child).getValue());
    }
}
```

An element holding no date adds a null to the `List<Calendar>` the method
answers.

**What correct would be** skipping such an element, or documenting that the list
may hold nulls. Neither is done, and the method's javadoc says it answers "the
list of Calendar".

**Why it matters** every caller walks the list and reads the dates; the first
one that reaches an empty element raises NullPointerException somewhere else
entirely, with nothing to say where the null came from.

**Where the Go carries it** partly: `go/xmpbox/schema/xmpschema.go`,
`UnqualifiedSequenceDateValueList`, keeps the element so the length matches, and
a `[]time.Time` cannot hold Java's null, so the zero time stands in it. Said
where it is and in [`STATUS.md`](STATUS.md).

**Confidence** high. Reproduced against JDK 17: a sequence holding a date and an
empty date prints `[java.util.GregorianCalendar[...], null]`.

---

## 62. `ScratchFile.markPagesAsFree` walks to `count` rather than `off + count`

**Where** `io/src/main/java/org/apache/pdfbox/io/ScratchFile.java`,
`markPagesAsFree`

```java
void markPagesAsFree(int[] pageIndexes, int off, int count) {
    synchronized (freePages)
    {
        for (int aIdx = off; aIdx < count; aIdx++)
```

The loop starts at `off` and stops at `count`, so it visits `count - off`
entries rather than `count`.

Its two callers are in `ScratchFileBuffer`. `close(boolean)` passes `off` 0, and
`0 + count == count`, so it is right by coincidence. `clear()` passes

```java
pageHandler.markPagesAsFree(pageIndexes, 1, pageCount - 1);
```

which visits indices 1 to `pageCount - 2` and never the last one.

**What correct would be** `aIdx < off + count`, or passing an end index rather
than a count.

**Why it matters** every `clear()` leaks one page, and the pages are a fixed
allowance. A buffer that filled its allowance cannot be refilled after being
cleared: the next write fails with "Maximum allowed scratch file memory
exceeded." Repeated clears leak a page each time, so a long-lived
`ScratchFile` loses a page per clear until it can hand out none.

**Where the Go carries it** `go/pdfio/scratchfile.go`, `markPagesAsFree`, walks
to `count` and says so.
`TestClearLeaksTheLastPage` in `go/pdfio/scratchfilebuffer_test.go` pins it and
names this entry.

**Confidence** high. Reproduced by compiling the `io` module against JDK 17 and
running it: a `ScratchFile` of three pages of main memory, three pages written,
`clear()`, and the second write fails with

```
second write failed: Maximum allowed scratch file memory exceeded.
```

---

## 63. `PDPage.getContentsForStreamParsing` skips the predictor

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/PDPage.java`,
`getContentsForStreamParsing`

```java
COSStream contentStream = page.getCOSStream(COSName.CONTENTS);
if (contentStream != null && COSName.FLATE_DECODE.equals(contentStream.getFilters()))
{
    // for now only streams using a flate filter are supported
    FlateFilterDecoderStream decoderStream = new FlateFilterDecoderStream(
            contentStream.createRawInputStream());
    return new NonSeekableRandomAccessReadInputStream(decoderStream);
}
return getContentsForRandomAccess();
```

The guard tests the filter and says nothing about `/DecodeParms`.
`FlateFilterDecoderStream` is an `Inflater` and eleven lines of buffering: it
carries no predictor. The general path this one skips,
`getContentsForRandomAccess`, goes through `COSStream.createView` and the whole
filter chain, predictor included.

**What correct would be** taking the fast path only where the stream declares no
predictor, which is one more term in the guard.

**Why it matters** a content stream written as
`/Filter /FlateDecode /DecodeParms <</Predictor 12 /Columns n>>` is legal. Read
through `getContentsForStreamParsing` it yields the raw PNG-filtered bytes,
which are not content stream operators; read through
`getContentsForRandomAccess` it yields the right ones. The same page then parses
differently depending on which accessor the caller reached for.

**Where the Go carries it** `go/pdfbox/pdmodel/pdpage.go`,
`ContentsForStreamParsing`, takes the same fast path on the same condition and
says so, and `go/pdfbox/filter/flate.go`'s `NewFlateDecoderReader` says it
applies no predictor.

**Confidence** high for the shape, which is plain in the two methods and in
`FlateFilterDecoderStream`. Not run: no PDF in the test corpus has a predictored
content stream, which is also why no PDFBox test covers it.

---

## 64. `MemoryUsageSetting.setupMixed`'s javadoc names two equivalences, and neither holds

**Where** `io/src/main/java/org/apache/pdfbox/io/MemoryUsageSetting.java`, both
`setupMixed` overloads

```java
 * @param maxMainMemoryBytes maximum number of main-memory to be used; if <code>-1</code> this is the same as
 * {@link #setupMainMemoryOnly()}; if <code>0</code> this is the same as {@link #setupTempFileOnly()}
```

Neither claim survives the private constructor.

`setupMixed(-1)` is `new MemoryUsageSetting(true, true, -1, -1)` and
`setupMainMemoryOnly()` is `new MemoryUsageSetting(true, false, -1, -1)`. The
constructor copies `useTempFile` through untouched, so the first answers `true`
from `useTempFile()` and the second `false`.

`setupMixed(0)` is `new MemoryUsageSetting(true, true, 0, -1)`, and
`locUseMainMemory && locMaxMainMemoryBytes == 0` with a temporary file turns
`useMainMemory` off — but only after `locMaxMainMemoryBytes` has been set from
the argument, so it stays 0. `setupTempFileOnly()` passes `useMainMemory` false
from the start, and `useMainMemory ? maxMainMemoryBytes : -1` makes it -1. So
the first answers 0 from `getMaxMainMemoryBytes()` and `true` from
`isMainMemoryRestricted()`, the second -1 and `false`.

**What correct would be** the javadoc saying what these setups do rather than
naming another setup they are not equal to, or `setupMixed` delegating to the
setup it names.

**Why it matters** the four getters are public API, and the javadoc is what a
caller reads before choosing a setup. A caller who believes the -1 equivalence
and later branches on `useTempFile()` gets the other branch. Inside `ScratchFile`
neither difference changes behaviour — `setupMixed(-1)` leaves
`maxMainMemoryIsRestricted` false so no scratch file is opened either way, and
both zero cases leave `inMemoryMaxPageCount` at 0 — so nothing in PDFBox itself
notices, which is presumably why the javadoc has stood.

**Where the Go carries it** `go/pdfio/memoryusagesetting.go`,
`newMemoryUsageSetting`, is the same arithmetic, and `SetupMixed` says the two
claims are false rather than repeating them.
`TestMemoryUsageSettingSetupMethods` in `go/pdfio/memoryusagesetting_test.go`
holds the four settings apart.

**Confidence** high. Read out of the running Java, `io` compiled against JDK 17:

```
setupMixed(-1)         mainMem=true  tempFile=true  memRestricted=false maxMem=-1 maxStore=-1
setupMainMemoryOnly()  mainMem=true  tempFile=false memRestricted=false maxMem=-1 maxStore=-1
setupMixed(0)          mainMem=false tempFile=true  memRestricted=true  maxMem=0  maxStore=-1
setupTempFileOnly()    mainMem=false tempFile=true  memRestricted=false maxMem=-1 maxStore=-1
```

---

## 65. `RandomAccessReadMemoryMappedFile.createView` checks nothing and reaches through the released buffer

**Where** `io/src/main/java/org/apache/pdfbox/io/RandomAccessReadMemoryMappedFile.java`,
`createView`

```java
@Override
public RandomAccessReadView createView(long startPosition, long streamLength)
{
    return new RandomAccessReadView(new RandomAccessReadMemoryMappedFile(this), startPosition,
            streamLength, true);
}
```

The private copy constructor it calls begins `mappedByteBuffer = parent.mappedByteBuffer.duplicate()`,
and `close()` sets `mappedByteBuffer` to null. So `createView` on a closed
source dereferences null. The method does not even declare `throws IOException`,
which is the tell: every other method of the class calls `checkClosed` first,
and this one has nothing to throw.

**What correct would be** `checkClosed()` as the first line, which is exactly
what the sibling `RandomAccessReadBufferedFile.createView` does.

**Why it matters** the two sources are interchangeable behind `RandomAccessRead`
and a caller cannot tell which it holds. Closing one and then asking for a view
raises `IOException` from one and `NullPointerException` from the other — and an
unchecked exception out of a method that declares no checked one will not be
caught by a caller handling `IOException`.

**Where the Go carries it** it does not: `go/pdfio/mappedfile.go`, `CreateView`,
calls `checkClosed` and answers `ErrClosed`, which is what the sibling answers
and what every other method on a closed `MappedFile` answers. The alternative
was a nil dereference — a panic — for a caller error the rest of the package
reports cleanly. Said where it is, in [`STATUS.md`](STATUS.md), and pinned by
`TestMappedFileViewOfAClosedSource`.

**Confidence** high. Read out of the running Java, JDK 17:

```
mapped createView on closed:       java.lang.NullPointerException: Cannot invoke
    "java.nio.ByteBuffer.duplicate()" because "<parameter1>.mappedByteBuffer" is null
bufferedfile createView on closed: java.io.IOException:
    org.apache.pdfbox.io.RandomAccessReadBufferedFile already closed
```

---

## 66. `ScratchFile` takes its two locks in both orders, and deadlocks

**Where** `io/src/main/java/org/apache/pdfbox/io/ScratchFile.java`. The class
synchronizes on three things: `ioLock`, `freePages` and `buffers`. Two of them
are taken nested, in both orders.

`getNewPage` holds `freePages` and asks for `ioLock`:

```java
int getNewPage() throws IOException {
    synchronized (freePages) {
        ...
        enlarge();          // -> synchronized (ioLock)
```

`close` holds `ioLock` and asks for `freePages`:

```java
public void close() throws IOException {
    synchronized (ioLock) {
        ...
        for (ScratchFileBuffer buffer : buffers) {
            if (buffer != null && !buffer.isClosed()) buffer.close(false);
            // -> ScratchFile.markPagesAsFree -> synchronized (freePages)
```

**What correct would be** one order, kept everywhere. `close` does not need to
hold `ioLock` while it walks the buffers: it could set `isClosed`, take a copy
of the list, release `ioLock`, and close them outside it.

**Why it matters** one thread writing to a buffer while another closes the
scratch file is not an exotic pattern — it is a document being written while
something else decides to shut down, and `ScratchFile` is the class whose whole
job is to be shared. When the two windows overlap, both threads block forever,
and because they are not daemon threads the JVM will not exit either.

**Where the Go carries it** `go/pdfio/scratchfile.go` keeps the same three
locks, taken in the same two orders — `getNewPage` holds `pagesLock` and
`enlarge` takes `ioLock`; `Close` holds `ioLock` and `closeBuffer` reaches
`markPagesAsFree`, which takes `pagesLock`. Ported as written and said at both
sites. Go's mutexes are not reentrant where Java's monitors are, but that
changes nothing here: the two locks are distinct, so this is the same
cross-goroutine hold-and-wait and not a new self-deadlock.

**Confidence** reproduced. A probe that ran a writer and a `close()` against the
same `ScratchFile` deadlocked in round 215 of 400, and the JVM's own detector
named it:

```
Found one Java-level deadlock:
"writer-215":
  waiting to lock monitor (object 0x0000000530e985d0, a java.lang.Object),
  which is held by "closer-215"
"closer-215":
  waiting to lock monitor (object 0x0000000530e985e0, a java.util.BitSet),
  which is held by "writer-215"

  at org.apache.pdfbox.io.ScratchFile.enlarge(ScratchFile.java:244)
  at org.apache.pdfbox.io.ScratchFile.getNewPage(ScratchFile.java:206)
  - locked <0x0000000530e985e0> (a java.util.BitSet)
  ...
  at org.apache.pdfbox.io.ScratchFile.markPagesAsFree(ScratchFile.java:475)
  at org.apache.pdfbox.io.ScratchFileBuffer.close(ScratchFileBuffer.java:431)
  at org.apache.pdfbox.io.ScratchFile.close(ScratchFile.java:517)
  - locked <0x0000000530e985d0> (a java.lang.Object)
```

No test pins this one. A test for it would have to lose a race on purpose, and
a test that hangs when it succeeds is worse than no test; the probe and this
entry are the record.

---

## 67. `RandomAccessReadView.rewind` checks nothing, and reads outside the view

**Where** `io/src/main/java/org/apache/pdfbox/io/RandomAccessReadView.java`,
`rewind`

```java
@Override
public void rewind(int bytes) throws IOException
{
    checkClosed();
    restorePosition();
    randomAccessRead.rewind(bytes);
    currentPosition -= bytes;
}
```

`seek` in the same class refuses a negative offset — `if (newOffset < 0) throw
new IOException("Invalid position " + newOffset)`. `rewind` never asks. It
points the source at the view's position and then rewinds **the source**, so a
rewind longer than the view has read lands the source before `startPosition`.
`currentPosition` goes negative, and every later read comes out of bytes the
view exists to exclude.

**What correct would be** the same check `seek` makes, or `seek(getPosition() -
bytes)` — which is what the interface's own default does, and which the override
exists only to avoid.

**Why it matters** a view is the port's boundary around an embedded stream. Any
parser that rewinds further than it has read silently starts reading its
neighbour's bytes, with nothing to signal it: no exception, and a position that
merely looks odd.

**Where the Go carries it** `go/pdfio/readview.go`, `ReadView.Rewind`, which
exists so the package function `Rewind` dispatches to it the way Java's virtual
call does. Pinned by `TestReadViewRewindPastItsOwnStart`.

**Confidence** reproduced. Read out of the running Java, JDK 17: a view of
`data[4..7]` over the bytes `0..9`, seeked to 2 and read once, so its position
is 3:

```
after seek(2): position=2 read=6
rewind(5) ok: position=-2 read=2 (source now at 0)
```

2 is `data[2]` — two bytes before the view begins.

---

## 68. The default `available()` has no lower bound, and answers a negative count

**Where** `io/src/main/java/org/apache/pdfbox/io/RandomAccessRead.java`,
`available`

```java
default int available() throws IOException
{
    return (int) Math.min(length() - getPosition(), Integer.MAX_VALUE);
}
```

The `min` bounds the value above and nothing bounds it below. A source whose
position has been put past its length answers a negative count, and the `int`
cast narrows rather than clamps.

`RandomAccessReadView` reaches that state through an ordinary seek: its `seek`
clamps the position it passes down to the source but records `newOffset`
verbatim, so `getPosition()` can exceed `length()` by any amount.

**What correct would be** the `Math.max(0, ...)` that
`RandomAccessInputStream.available()`, twelve files away, already has:

```java
return (int) Math.max(0, Math.min(input.length() - position, Integer.MAX_VALUE));
```

Two implementations of the same idea in the same package, one with the bound and
one without.

**Why it matters** it is latent rather than live: the only caller of
`RandomAccessRead.available()` in the whole tree is
`DataInputRandomAccessRead.hasRemaining`, which asks `available() > 0` — false
for a negative and false for a zero. Anything that sized an array by it, which
is what `available()` is for in `java.io`, would get
NegativeArraySizeException.

**Where the Go carries it** `go/pdfio/randomaccess.go`, `Available`, narrows
through int32 so the cast is reproduced too. Pinned by
`TestAvailableGoesNegativePastTheEnd`.

**Confidence** reproduced. Read out of the running Java, JDK 17, a four-byte
view seeked to 20:

```
ReadView seek(20) ok, position=20 length=4 available=-16 isEOF=true
```

---

## 69. A write at an exact chunk boundary lands on the first byte of the chunk

**Where** `io/src/main/java/org/apache/pdfbox/io/RandomAccessReadBuffer.java`,
`seek`, and `RandomAccessReadWriteBuffer.write`

```java
else
{
    // it is allowed to jump beyond the end of the file
    // jump to the end of the buffer
    pointer = size;
    bufferListIndex = bufferListMaxIndex;
    currentBuffer = bufferList.get(bufferListIndex);
    currentBufferPointer = chunkSize > 0 ? (int) (size % chunkSize) : 0;
}
```

`size % chunkSize` is 0 whenever the buffer holds an exact multiple of the chunk
size — which is the normal state of a buffer that has just been filled. So the
"jump to the end" branch parks the cursor at the **start** of the last chunk
rather than past its end.

Reading recovers, because every read path seeks first or checks `pointer >=
size`. Writing does not: `write` uses `currentBufferPointer` as it stands.

**What correct would be** treating a full last chunk as full —
`currentBufferPointer = chunkSize` where `size > 0 && size % chunkSize == 0`,
which the read path already knows how to step past — or deriving the index as
`size / chunkSize` and letting the pointer be 0 in a fresh chunk.

**Why it matters** it corrupts data and then hides the corruption behind a
length that no longer matches the contents. Writing one byte at the end of a
full buffer overwrites byte 0, and `size` counts the byte anyway, so the buffer
claims a length its chunks cannot supply and a full read fails outright.
`RandomAccessReadWriteBuffer` is the default stream cache — this is where
`COSStream` data lives.

**Where the Go carries it** `go/pdfio/readbuffer.go`, `ReadBuffer.Seek`, and
`go/pdfio/readwritebuffer.go`, `ReadWriteBuffer.Write`. Pinned by
`TestWriteAtAnExactChunkBoundaryOverwritesTheFirstByte`.

**Confidence** reproduced. Read out of the running Java, JDK 17, an 8-byte chunk
written full, seeked to 8, and written one more byte:

```
size=9 position=9
first 8 bytes: 99 2 3 4 5 6 7 8 (read 8)
readFully(9) threw java.io.IOException: No more chunks available, end of buffer reached
```

The 99 was written at position 8 and landed on byte 0.

---

## 70. `SequenceRandomAccessRead` checks its list is not empty before emptying it

**Where**
`io/src/main/java/org/apache/pdfbox/io/SequenceRandomAccessRead.java`, the
constructor

```java
if (randomAccessReadList.isEmpty())
{
    throw new IllegalArgumentException("Empty list");
}
readerList = randomAccessReadList.stream()
        .filter(r -> { ... return r.length() > 0; ... })
        .collect(Collectors.toList());
currentRandomAccessRead = readerList.get(currentIndex);
```

The check is made against the argument and the filter is applied afterwards, so
a list that is not empty but holds only zero-length sources passes the check and
then indexes an empty list.

**What correct would be** checking `readerList` after the filter, which is the
list the constructor actually goes on to use.

**Why it matters** the caller gets `IndexOutOfBoundsException` — unchecked, and
about an index — instead of the `IllegalArgumentException("Empty list")` the
constructor plainly means to raise. A list of empty sources is a perfectly
ordinary thing to assemble from a document with empty content streams.

**Where the Go carries it** it does not: `go/pdfio/sequenceread.go`,
`NewSequenceRead`, checks after filtering as well and answers "empty list". An
index panic for an input the constructor already has an error for was the worse
of the two. Said where it is and in [`STATUS.md`](STATUS.md).

**Confidence** reproduced. Read out of the running Java, JDK 17:

```
all-empty list threw java.lang.IndexOutOfBoundsException: Index 0 out of bounds for length 0
empty list threw java.lang.IllegalArgumentException: Empty list
```

---

## 71. `SequenceRandomAccessRead.close` leaks every source after the first failure

**Where**
`io/src/main/java/org/apache/pdfbox/io/SequenceRandomAccessRead.java`, `close`

```java
for (RandomAccessRead randomAccessRead : readerList)
{
    randomAccessRead.close();
}
readerList.clear();
currentRandomAccessRead = null;
isClosed = true;
```

Nothing catches. The first source whose `close` fails ends the loop, so the
sources after it are never closed, the list is never cleared and `isClosed`
stays false.

**What correct would be** closing all of them and reporting the first failure
afterwards — what `IOUtils.closeAndLogException` in the same package exists for.

**Why it matters** the sources are files and scratch buffers. A single failing
close leaks every handle behind it, and the sequence then answers `isClosed()`
false, so a caller retrying the close closes the first ones twice and still
never reaches the rest.

**Where the Go carries it** it does not: `go/pdfio/sequenceread.go`,
`SequenceRead.Close`, closes every source, keeps the first error and marks
itself closed. Leaking handles to reproduce a `finally` that Java does not have
was the worse of the two. Said where it is and in [`STATUS.md`](STATUS.md).

**Confidence** reproduced. Read out of the running Java, JDK 17, a sequence of a
source whose close throws followed by an ordinary one:

```
close threw java.io.IOException: nope | isClosed=false | second reader closed=false
```

---

## 72. `ScratchFile.close` walks the buffer list without the lock the list has

**Where** `io/src/main/java/org/apache/pdfbox/io/ScratchFile.java`. Every other
touch of `buffers` takes the monitor on it:

```java
public RandomAccess createBuffer() throws IOException {
    ScratchFileBuffer newBuffer = new ScratchFileBuffer(this);
    synchronized (buffers) { buffers.add(newBuffer); }
    return newBuffer;
}

void removeBuffer(ScratchFileBuffer buffer) {
    synchronized (buffers) { buffers.remove(buffer); }
}
```

`close` does not:

```java
synchronized (ioLock) {
    ...
    for (ScratchFileBuffer buffer : buffers) { ... buffer.close(false); }
    buffers.clear();
```

It holds `ioLock`, which is a different monitor, and reads and then clears the
list under it.

**What correct would be** `synchronized (buffers)` around the walk and the
clear, like its two neighbours — or a copy taken under that monitor and walked
outside it, which would also settle the lock-order inversion of entry 66.

**Why it matters** `createBuffer` running against `close` is a plain
`ArrayList` being appended to while it is iterated:
`ConcurrentModificationException` out of `close`, or a buffer added just after
the walk and never closed, left reporting itself open over a scratch file that
is gone. Three fields guard this class and the fourth thing it owns is guarded
by whichever lock happened to be held.

**Where the Go carries it** `go/pdfio/scratchfile.go`, `Close`, reads and clears
`s.buffers` under `ioLock` while `CreateBuffer` and `removeBuffer` take
`buffersLock`, so the same window is open. Go has no
ConcurrentModificationException; it is a data race on the slice, which the race
detector would name. Ported as written and said at the site.

**Confidence** high, from the source: the three methods are twenty lines apart
and two of them synchronize on the field that the third does not. Not
reproduced — it needs the two threads to interleave inside the window, and a
test that loses a race on purpose is not a test.

---

## 73. `RandomAccessReadMemoryMappedFile` leaks its channel on a file it refuses

**Where**
`io/src/main/java/org/apache/pdfbox/io/RandomAccessReadMemoryMappedFile.java`,
the `Path` constructor

```java
fileChannel = FileChannel.open(path, EnumSet.of(StandardOpenOption.READ));
size = fileChannel.size();
// TODO only ints are allowed -> implement paging
if (size > Integer.MAX_VALUE)
{
    throw new IOException(getClass().getName() + " doesn't yet support files bigger than "
            + Integer.MAX_VALUE);
}
```

The channel is opened and then the constructor throws without closing it. No
object is returned, so no caller can close it either; the handle is held until
the `FileChannel` is finalised.

**What correct would be** closing the channel on the way out, or asking
`Files.size(path)` before opening anything.

**Why it matters** every attempt at a PDF over 2 GB leaks a file handle, and
the caller's natural response — catch, log, try the next file — leaks one per
attempt. On Windows the file also stays locked against deletion and renaming
for as long as the handle is held.

**Where the Go carries it** it does not: `go/pdfio/mappedfile.go`,
`NewMappedFile`, stats the file and refuses before anything is opened or
mapped, which is also the order Java means to be in — it wants the size first
and only opens the channel because that is how it asks. Nothing is left open to
leak. Said where it is and in [`STATUS.md`](STATUS.md).

**Confidence** high, from the source: the `throw` sits between the open and the
only `close` the class has, and the class has no `finally`. Not reproduced —
observing a leaked handle needs a file over 2 GB.

---

## 74. `TrueTypeEmbedder.getTag` indexes its alphabet with a negative remainder

**Where** `pdfbox/src/main/java/org/apache/pdfbox/pdmodel/font/TrueTypeEmbedder.java`,
`getTag`

```java
public String getTag(Map<Integer, Integer> gidToCid)
{
    // hash might be negative due to an overflow if the map contains lots of values
    long num = Math.abs(gidToCid.hashCode());
    // base25 encode
    StringBuilder sb = new StringBuilder();
    do
    {
        long div = num / 25;
        int mod = (int)(num % 25);
        sb.append(BASE25.charAt(mod));
```

The comment shows the author knew the hash can be negative, and `Math.abs` is
the guard against it. `Math.abs(int)` has one input it does not fix:
`Math.abs(Integer.MIN_VALUE)` is `Integer.MIN_VALUE`, because the positive value
does not exist in the range. Widening the result to `long` afterwards does not
help — the negation has already failed.

`num` is then negative, `num % 25` is negative in Java, and
`BASE25.charAt(negative)` raises StringIndexOutOfBoundsException.

**What correct would be** `Math.abs((long) gidToCid.hashCode())`, which widens
before negating and has no such input. The `long num` is already there; only the
cast is in the wrong place.

**Why it matters** it is one hash value in 2^32, so it is a lottery rather than a
hazard — but the failure is an unchecked exception out of saving a document,
with a message about a string index, in a method whose comment says it is
guarding against exactly this.

**Where the Go carries it** `go/pdfbox/pdmodel/font/truetypeembedder.go`,
`subsetTag`, leaves `math.MinInt32` unnegated for the same reason and then
indexes `base25` with a negative remainder, which panics as Java's unchecked
exception does. Said at the point of difference.

**Confidence** certain, and **measured**. A map whose hash is exactly
`Integer.MIN_VALUE` needs one entry with `key ^ value == 0x80000000`, so
`{0: Integer.MIN_VALUE}` does it, and the running Java answers

```
hashCode(patho)     = -2147483648
getTag(patho) threw java.lang.StringIndexOutOfBoundsException: String index out of range: -23
```

against `getTag({1: 2, 3: 4})` = `AAAAAL+` for an ordinary map. Go's `%` keeps
the sign of the dividend exactly as Java's does, so the port reaches the same
-23 and panics. `TestSubsetTagMatchesJava` in `truetypeembedder_test.go` holds
both, with the Java values as the wanted ones.

---

## 75. `ExportXFDF` reports "this PDF does not contain a form" and exits 0

**Where** `tools/src/main/java/org/apache/pdfbox/tools/ExportXFDF.java`, `call`.

`ExportFDF` and `ExportXFDF` are the same class twice over, differing in the
extension and which save they call. They differ in one more thing:

```java
// ExportFDF
if( form == null )
{
    SYSERR.println( "Error: This PDF does not contain a form." );
    return 1;
}

// ExportXFDF
if( form == null )
{
    SYSERR.println( "Error: This PDF does not contain a form." );
}
```

`ExportXFDF` falls out of the `if`, past the `else`, and reaches the `return 0`
at the end of `call`.

**What correct would be** `return 1`, as its twin does. Nothing else in either
class differs on this path, and the message says it is an error.

**Why it matters** the two commands are used interchangeably — `export:fdf` and
`export:xfdf` differ only in the format the caller wants — and a script that
checks the exit code gets a different answer from each for the same document.
`export:xfdf` reports success while writing no file at all, which is the worst
of the three possible outcomes.

**Where the Go carries it** `go/tools/fdfcommands.go`, `noFormExit`, which is
the one thing `exportForm` does not share between the two callers. Said there.

**Confidence** certain, from the source: it is a missing statement, not a
subtlety. Not measured, because the `tools` module cannot be run in this
environment — picocli is not in the local Maven repository and there is no
network to fetch it.

---

## 76. `ImportXFDF` raises NullPointerException on a document with no form

**Where** `tools/src/main/java/org/apache/pdfbox/tools/ImportXFDF.java`,
`importFDF`.

```java
// ImportFDF
public void importFDF( PDDocument pdfDocument, FDFDocument fdfDocument ) throws IOException
{
    PDDocumentCatalog docCatalog = pdfDocument.getDocumentCatalog();
    PDAcroForm acroForm = docCatalog.getAcroForm();
    if (acroForm == null)
    {
        return;
    }
    acroForm.setCacheFields( true );
    ...

// ImportXFDF
public void importFDF( PDDocument pdfDocument, FDFDocument fdfDocument ) throws IOException
{
    PDDocumentCatalog docCatalog = pdfDocument.getDocumentCatalog();
    PDAcroForm acroForm = docCatalog.getAcroForm();
    acroForm.setCacheFields( true );
    ...
```

`getAcroForm()` answers null for a document with no `/AcroForm`, and the second
one dereferences it.

**What correct would be** the null check its twin has. `call` catches
`IOException`; NullPointerException is unchecked and goes past it, so the
command dies with a stack trace rather than the "Error importing XFDF data"
message every other failure gets.

**Why it matters** importing form data into a document that has no form is an
ordinary mistake, and it is the one case this command handles worst: `importfdf`
saves the document unchanged and answers 0, `importxfdf` crashes.

**Where the Go carries it** `go/tools/fdfcommands.go`,
`ImportXFDF.ImportFDFInto`, which dereferences the nil the same way and panics —
which is what this port renders an unchecked exception as. Said at the site.

**Confidence** certain, from the source. Not measured, for the reason entry 75
gives.

## 77. `GlyphLayoutProcessorAwt.checkMissingGlyphs` prints half a surrogate pair

**Where** `pdfbox-layout-awt/src/main/java/org/apache/pdfbox/glyphlayout/awt/
GlyphLayoutProcessorAwt.java`, `checkMissingGlyphs`.

**What**

```java
int firstMissingCharacter = awtFont.canDisplayUpTo(text);
if (firstMissingCharacter != -1)
{
    char c = text.charAt(firstMissingCharacter);
    int codepoint = text.codePointAt(firstMissingCharacter);

    throw new IllegalArgumentException(
            String.format("Missing glyph in font '%s' for the character '%c', codePoint: %d (U+%04x).",
                    awtFont.getName(), c, codepoint, codepoint));
}
```

`c` is a `char`, which is one UTF-16 code unit; `codepoint` is the whole
character. For anything outside the basic multilingual plane the two disagree:
`charAt` answers the high surrogate, and `'%c'` formats it on its own. The
message then carries an unpaired surrogate — an ill-formed string that prints as
a replacement character or nothing at all — beside the correct code point.

`GlyphLayoutSMPTest` is entirely about characters in that plane, so a font
missing one of them is not a hypothetical case for this class.

**What correct would be** formatting the code point rather than the code unit:
`String.format("...'%s'...", new String(Character.toChars(codepoint)), ...)`.
The `%04x` half of the message is already right.

**Why it matters** the message names the character that could not be drawn, and
for exactly the characters this module has a test class about, it names half of
one.

**Where the Go carries it** `go/pdfbox/glyphlayout/processor.go`,
`missingGlyph`, which takes the first UTF-16 unit of the character for the
`'%c'` position and the whole rune for the code point, the same way. Said at the
site.

**Confidence** certain, from the source. The Java's behaviour was not measured
for a supplementary character: `canDisplayUpTo` has to answer the index of one,
which needs a font missing a supplementary character that the test resources do
not have. The basic-plane message was measured, and matches — see
`TestMissingGlyphIsRefused`, which asserts it character for character.

## 78. `GlyphLayoutDIN91379.pdf` was rendered from a string the test no longer has

**Where** `pdfbox-layout-awt/src/test/resources/pdf/GlyphLayoutDIN91379.pdf`,
against `GlyphLayoutDin91379Test.LATIN_CHARS_DIN_91379`.

**What** twenty of the reference PDF's forty-one lines end with a space glyph.
No line of `LATIN_CHARS_DIN_91379` ends with a space: every one of them ends
`...\n"`, checked byte by byte. The PDF was rendered from an earlier spelling of
the string and has not been regenerated since the spaces were taken out.

`testGlyphLayoutDin91379` still passes, because `TestBase.checkRenderIdent`
renders both documents and compares pixels, and a space at the end of a line
paints nothing. The reference is stale in the one way that comparison cannot
see.

**What correct would be** regenerating the PDF from the current string, which
is what `target/GlyphLayoutDIN91379.pdf` already is on every run — the test
writes it and then compares. Nothing in the Java's behaviour is wrong; the
resource is.

**Where the Go carries it** nowhere: there is nothing to carry. It is recorded
because a Go test does compare against that PDF and had to be told to ignore
it. `go/pdfbox/glyphlayout/reference_test.go`, `movingFields`, drops a trailing
space glyph from both sides, and says why.

**Confidence** certain, and measured both ways: the twenty lines are exactly
the twenty that ended with `G:0020` in the dump of the reference PDF, and the
Java source lines they come from were read as bytes.

## 79. `Encrypt` adds one recipient object N times, so only the last `-certFile` survives

**Where** `tools/src/main/java/org/apache/pdfbox/tools/Encrypt.java`, the
`certFileList` branch of `call()`.

**What**

```java
PublicKeyProtectionPolicy ppp = new PublicKeyProtectionPolicy();
PublicKeyRecipient recip = new PublicKeyRecipient();
recip.setPermission(ap);

CertificateFactory cf = CertificateFactory.getInstance("X.509");

for (File certFile : certFileList)
{
    try (InputStream inStream = new FileInputStream(certFile))
    {
        X509Certificate certificate = (X509Certificate) cf.generateCertificate(inStream);
        recip.setX509(certificate);
    }
    ppp.addRecipient(recip);
}
```

The recipient is built **once, outside the loop**, and the loop overwrites its
certificate and adds the same object again. `-certFile a.cer -certFile b.cer`
gives a policy holding two references to one recipient, and that recipient
carries `b.cer`. The document is then encrypted to `b.cer` twice and not to
`a.cer` at all.

`PublicKeyProtectionPolicy.addRecipient` keeps a `List`, not a `Set`, so the
duplicate is not folded away: the /Recipients array comes out with two entries
that unwrap to the same thing.

**What correct would be** constructing the recipient inside the loop, which is
one line moved.

**Why it matters** `-certFile` is repeatable and documented as repeatable —
"path to a certificate file, can be used multiple times". A user who encrypts a
document for a team gets a document only the last of them can open, and nothing
says so: the command succeeds and the file is written.

**Where the Go carries it** `go/tools/encrypt.go`, in the `certFileList`
branch, which builds the recipient outside the loop the same way. Said at the
site.

**Confidence** certain, from the source. Not measured: `tools` cannot be run
here, because picocli is not in the local Maven repository and there is no
network to fetch it — the same reason `track/tools` gives for its measurements.
The aliasing is plain in the seven lines above and needs no run to see.

## 80. `TIFFUtil.updateMetadata` looks for the IFD in a node it has just built

**Where** `tools/src/main/java/org/apache/pdfbox/tools/imageio/TIFFUtil.java`,
`updateMetadata`.

```java
IIOMetadataNode root = new IIOMetadataNode(metaDataFormat);
IIOMetadataNode ifd;
NodeList nodeListTIFFIFD = root.getElementsByTagName("TIFFIFD");
if (nodeListTIFFIFD.getLength() == 0)
{
    ifd = new IIOMetadataNode("TIFFIFD");
    root.appendChild(ifd);
}
else
{
    ifd = (IIOMetadataNode) nodeListTIFFIFD.item(0);
}
```

`root` is a node constructed on the line above and nothing has been appended to
it, so `getElementsByTagName` can only answer an empty list. The `else` branch
is unreachable, and the code reads as though it were looking for an IFD the
writer had already put there.

**What correct would be** `metadata.getAsTree(metaDataFormat)`, which is what
the sibling method `debugLogMetadata` two lines earlier calls to get a tree
with something in it. The `if`/`else` then does what it looks like it does.

**Why it matters** It does not, in output: the method ends in
`metadata.mergeTree(metaDataFormat, root)`, and merging a tree that carries one
fresh IFD full of new fields has the same effect as merging a tree built from
the existing one. The bug is that the branch is dead, so a reader cannot tell
whether the fields are being added to an IFD or replacing one, and a later
change that depended on the answer would be wrong.

**Where the Go carries it** Nowhere, and it cannot: `go/tools/imageio/tiff.go`
writes the directory itself rather than merging into a plugin's metadata tree,
so there is no tree to read and no branch to leave dead. The five fields
`updateMetadata` adds -- XResolution, YResolution, ResolutionUnit,
RowsPerStrip, Software, plus PhotometricInterpretation for a bitonal image --
are all written, which is the whole of what the method achieves.

**Confidence** certain, from the source, and the output was measured: the four
classes of `tools/imageio` compile against nothing but `log4j-api`, and the
files they write carry exactly those tags.

## 81. The BMP resolution `TestImageIOUtils` asserts is written only by a test-scope dependency

**Where** `tools/src/main/java/org/apache/pdfbox/tools/imageio/ImageIOUtil.java`,
`writeImage`, against `tools/pom.xml` and
`tools/src/test/java/org/apache/pdfbox/tools/imageio/TestImageIOUtils.java`,
`checkBmpResolution`.

The test writes a BMP and reads the two pixels-per-metre fields out of the
header by hand, because "BMP reader doesn't work":

```java
int pixelsPerMeter = Integer.reverseBytes(dis.readInt());
int actualResolution = (int) Math.round(pixelsPerMeter / 100.0 * 2.54);
assertEquals(expectedResolution, actualResolution, "X resolution doesn't match ...");
```

`ImageIOUtil` writes that resolution for a format that is neither TIFF nor
JPEG only through this guard:

```java
if (!metadata.isReadOnly() && metadata.isStandardMetadataFormatSupported())
{
    setDPI(metadata, dpi, formatName);
}
```

and it chooses the writer with a loop whose whole purpose is to find one for
which that guard passes -- "Loop until we get the best driver, i.e. one that
supports setting dpi in the standard metadata format; however we'd also accept
a driver that can't, if a better one can't be found".

**Measured on JDK 17.0.19 with nothing but `log4j-api` on the class path**:
`com.sun.imageio.plugins.bmp.BMPImageWriter` is the only BMP writer, its
`getDefaultImageMetadata` answers `readOnly=true`, `setDPI` is skipped, and
`ImageIOUtil.writeImage(image, "bmp", out, 36)` on an 8x6 `TYPE_INT_RGB` gives
198 bytes whose `biXPelsPerMeter` and `biYPelsPerMeter` -- bytes 38 to 45 --
are all zero. The test's `round(0 / 100.0 * 2.54)` is 0, and it asserts 36.

The assertion nonetheless holds when Maven runs it, and the reason is in
`tools/pom.xml`:

```xml
<dependency>
    <groupId>com.github.jai-imageio</groupId>
    <artifactId>jai-imageio-core</artifactId>
    <version>${jai.version}</version>
    <scope>test</scope>
</dependency>
```

JAI Image I/O Tools registers a BMP writer of its own whose metadata is
writable, the loop prefers it, and `setDPI` runs. **The scope is `test`.**

**What correct would be** either promoting the dependency out of test scope,
or writing the two header fields without going through a plugin's metadata, or
saying in `ImageIOUtil`'s javadoc that a BMP carries its resolution only with
JAI on the class path -- the javadoc says exactly that about TIFF two
paragraphs earlier and says nothing about BMP.

**Why it matters** The test passes and the shipped code does not do what it
tests. `pdfbox-tools` at runtime has neither JAI jar -- they are not compile or
runtime dependencies -- so every BMP `PDFToImage` writes for a user says its
resolution is zero, and the one assertion standing behind "the BMP carries its
resolution" was made in an environment the user does not have.

**Where the Go carries it** Nowhere. `go/tools/imageio/bmp.go` writes the
pixels per metre into the header always, which is what the Java's test asserts
and what its writer loop is reaching for. There is no plugin registry in Go and
so no writer whose metadata is read-only; the port is the JAI-present
behaviour, and `TestWriteImageFormats` asserts the resolution for BMP because
the Java test does.

**Confidence** the JDK half is certain and measured. The JAI half is inferred
from `tools/pom.xml` and from what the writer loop is for: the two
`com.github.jai-imageio` jars are not in the local Maven repository and there
is no network to fetch them, so the run with them present could not be made
here.

## 82. `PDFMergerUtility.appendDocument` merges the destination's /Threads into itself

**Where** `pdfbox/src/main/java/org/apache/pdfbox/multipdf/PDFMergerUtility.java`,
`appendDocument`.

```java
COSArray destThreads = destCatalog.getCOSObject().getCOSArray(COSName.THREADS);
COSArray srcThreads = cloner.cloneForNewDocument(destCatalog.getCOSObject().getCOSArray(
        COSName.THREADS));
if (destThreads == null)
{
    destCatalog.getCOSObject().setItem(COSName.THREADS, srcThreads);
}
else
{
    destThreads.addAll(srcThreads);
}
```

Both lines read `destCatalog`. The variable called `srcThreads` is a clone of
the **destination's** article thread array, not the source's, and the block
below then appends the destination's threads to themselves.

**What correct would be** `srcCatalog.getCOSObject().getCOSArray(COSName.THREADS)`
on the second line. Every other block of `appendDocument` reads the source and
writes the destination; this one reads the destination twice.

**Why it matters** Two things, in opposite directions. A source document's
article threads are silently dropped by every merge -- the reading order they
describe is lost, which is what /Threads is for. And a destination that has
threads gets them **twice**: the clone is a fresh array of fresh dictionaries,
and `addAll` appends it, so a document merged with N others ends up with
2^N copies of its own threads. The second is the one a user would notice.

**Where the Go carries it** `go/pdfbox/multipdf/pdfmergerutility.go`,
`mergeThreads`, which reads the destination twice and says so at the site.

**Confidence** certain, from the source: the two `getCOSArray` calls are on the
same expression, five words apart.

## 83. `PDFMergerUtility.mergeMarkInfo` writes /Suspect twice and /UserProperties never

**Where** `pdfbox/src/main/java/org/apache/pdfbox/multipdf/PDFMergerUtility.java`,
`mergeMarkInfo`.

```java
destMark.setMarked(true);
destMark.setSuspect(srcMark.isSuspect() || destMark.isSuspect());
destMark.setSuspect(srcMark.usesUserProperties() || destMark.usesUserProperties());
destCatalog.setMarkInfo(destMark);
```

The third line reads `usesUserProperties` and writes `setSuspect`. Its setter
should be `setUserProperties`, which `PDMarkInfo` has and nothing in this class
calls.

**What correct would be**
`destMark.setUserProperties(srcMark.usesUserProperties() || destMark.usesUserProperties());`

**Why it matters** The merged document's /MarkInfo never gets a
/UserProperties entry, so a viewer is told that no structure element carries
user properties even when the source said it did. And /Suspect is decided by
the wrong question: the second call overwrites the first, so a document whose
structure is suspect but which has no user properties comes out marked as not
suspect. Both entries are ISO 32000-1 table 321 and both are read by assistive
technology.

**Where the Go carries it** `go/pdfbox/multipdf/pdfmergerutility_structure.go`,
`mergeMarkInfo`, which calls `SetSuspect` twice with the same arguments in the
same order. Said at the site.

**Confidence** certain, from the source.

## 84. `PDFMergerUtility` never merges the page mode, because the branch that would cannot run

**Where** `pdfbox/src/main/java/org/apache/pdfbox/multipdf/PDFMergerUtility.java`,
`appendDocument`, against
`pdfbox/src/main/java/org/apache/pdfbox/pdmodel/PDDocumentCatalog.java`,
`getPageMode`.

```java
PageMode destPageMode = destCatalog.getPageMode();
if (destPageMode == null)
{
    PageMode srcPageMode = srcCatalog.getPageMode();
    destCatalog.setPageMode(srcPageMode);
}
```

`getPageMode` cannot answer null:

```java
String mode = root.getNameAsString(COSName.PAGE_MODE);
if (mode != null)
{
    try { return PageMode.fromString(mode); }
    catch (IllegalArgumentException e) { ...; return PageMode.USE_NONE; }
}
else
{
    return PageMode.USE_NONE;
}
```

so the `if` never fires and the whole block is dead.

**What correct would be** asking whether the entry is there --
`destCatalog.getCOSObject().containsKey(COSName.PAGE_MODE)` -- rather than
whether the accessor answered null. The accessor used to be able to; the
`else` returning USE_NONE is the newer half.

**Why it matters** A merge into an empty destination loses the source's
/PageMode. That is the case the command line tool takes: `pdfbox merge` starts
with `new PDDocument()`, so a source that asks to open with its bookmarks
showing, or in full screen, is merged into a document that asks for nothing.

**Where the Go carries it** `go/pdfbox/multipdf/pdfmergerutility.go`,
`mergePageMode`, which is a function with the comment and no condition,
because the port's `PageMode()` answers `PageModeUseNone` for the same reason
and a condition there would read as though it did something.

**Confidence** certain, from the source, both halves quoted above.
