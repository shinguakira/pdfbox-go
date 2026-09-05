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

**Confidence** high for the code reading; the failing case is derived from the
method, not measured against a Java run, because there is no Maven in this
environment to build PDFBox with. Every ported test of this class passes with
the Java values.

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
