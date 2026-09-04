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
