# Implementation Plan

Slice 3 — text extraction, simple fonts.

**Branch: `slice/3-<name>`** — from and back to `migration-base`.
The literal name is not written in `migration/BRANCHING.md`; only the pattern
`slice/N-name` is. **Decide the name before branching.**

## Rules — do not break these

- **NEVER change the Java.** No `.java`, no `pom.xml`, no test resource, for any
  reason. It is the reference; a reference that gets edited stops being one.
- **NEVER fix a bug that is in the Java.** Port it as written, comment it where
  it occurs, and record it in `migration/JAVA-BUGS.md`.
- **NEVER create a branch that is not in the migration plan**, and never add one
  to the plan's list.
- **NEVER change `migration/PLAN.md`.**
- **NEVER commit to `migration-base` directly.**
- **NEVER touch `apache/pdfbox`** — no PR, no pull, fetch, merge or rebase.
- **Do not stop while work in this branch's scope remains.**

## How each unit of work runs

Five phases, in this order, never overlapping:

**A — write the test.** Port the Java test to Go. Assertion values are copied
from the Java, never read off the Go. The implementation does not exist yet.

**B — port the implementation.** Write the Go from the Java source, line for
line. Do not look at what makes the test pass; look at what the Java does.

**C — run and fix.** `gofmt -l . && go vet ./... && go test ./...`. A failure
is a defect in the port, not in the test. Fix the Go. If the Java itself is
wrong, keep the wrong behaviour and record it in `JAVA-BUGS.md`.

**D — adversarial review.** Green tests are not evidence the port is faithful.
Read the Go against the Java looking for what the tests cannot catch, and
assume the port is wrong until each check says otherwise.

**E — user feedback.** Stop. Wait. Judge each item, and where it is a real
defect, write a strict failing test first and only then fix.

---

# Phase A — Write the tests

Ported from the Java test files. Nothing compiles yet; that is expected.

- [ ] A1. `fontbox/afm` — port all 8 Java tests
  - `AFMParserTest`, `FontMetricsTest`, `CharMetricTest`, `KernPairTest`,
    `TrackKernTest`, `LigatureTest`, `CompositeTest`, `CompositePartTest`

- [ ] A2. `fontbox/encoding` — port `EncodingTest`

- [ ] A3. `fontbox/ttf` — port the 5 that text extraction needs
  - `TestTTFParser`, `TestCMapSubtable`, `WGL4NamesTest`,
    `GlyfCompositeDescriptTest`, `RandomAccessReadBufferDataStreamTest`
  - Skip `GlyphSubstitutionTable*Test`, `TTFSubsetterTest`,
    `TrueTypeFontCollectionTest` — slice 4

- [ ] A4. `pdmodel/font/encoding` — port `TestFontEncoding`
  - Write from source for `GlyphList` — Java has no test for it

- [ ] A5. `pdmodel/font` — port `PDFontTest`
  - Write from source for `PDFontDescriptor`, `PDFontFactory`, `Standard14Fonts`
    — Java has no test for these
  - Skip `CIDCharSetMatchTest`, `PDCIDFontType0SubstituteTest` — slice 4
  - Skip `TestFontEmbedding`, `TestToUnicodeWriter` — slice 7, they write PDFs

- [ ] A6. `contentstream` — extend `streamengine_test.go`
  - No Java test exists; extend the slice 2 test for `Tf`, `Tj`, `TJ`, `'`, `"`
    and for the glyph hooks

- [ ] A7. `pdfbox/text` — port `PDFTextStripperByAreaTest` and `BidiTest`
  - Write from source for `TextPosition` and `TextPositionComparator` — Java
    tests them only through the stripper

- [ ] A8. `pdfbox/text` — port `TestTextStripper`, the corpus harness
  - Table test over the 40 `.pdf` files in `pdfbox/src/test/resources/input/`
  - Compare against the checked-in expected text, sorted and unsorted
  - **See Blocked below — this one cannot run yet**

---

# Phase B — Port the implementation

Written from the Java source, in dependency order.

- [ ] B0. `fontbox` root — 2 files
  - `FontBoxFont`, `EncodedFont`. `PDFont` is written against `FontBoxFont`, so
    nothing below compiles without it.

- [ ] B1. `fontbox/afm` — 8 files
  - `AFMParser`, `FontMetrics`, `CharMetric`, `KernPair`, `TrackKern`,
    `Ligature`, `Composite`, `CompositePart`

- [ ] B2. `fontbox/encoding` — 4 files
  - `Encoding`, `BuiltInEncoding`, `MacRomanEncoding`, `StandardEncoding`

- [ ] B3. `fontbox/ttf` — the reading path only, roughly 15 of 44 files
  - `TTFDataStream`, `RandomAccessReadDataStream`, `TTFParser`, `TTFTable`,
    `TrueTypeFont`
  - `HeaderTable`, `MaximumProfileTable`, `HorizontalHeaderTable`,
    `HorizontalMetricsTable`, `IndexToLocationTable`, `NamingTable`,
    `NameRecord`, `PostScriptTable`, `OS2WindowsMetricsTable`
  - `CmapTable`, `CmapSubtable`, `CmapLookup`, `WGL4Names`
  - `GlyphTable`, `GlyphData`, `GlyphDescription`, `GlyfDescript`,
    `GlyfSimpleDescript`, `GlyfCompositeDescript`, `GlyfCompositeComp`
  - Leave `gsub/`, `TTFSubsetter`, `TrueTypeCollection`, `OTFParser`,
    `OpenTypeFont`, `CFFTable`, the vertical and kerning tables to slice 4

- [ ] B4. `pdmodel/font/encoding` — 12 files
  - `Encoding`, `BuiltInEncoding`, `DictionaryEncoding`, `GlyphList`,
    `StandardEncoding`, `WinAnsiEncoding`, `MacRomanEncoding`,
    `MacOSRomanEncoding`, `MacExpertEncoding`, `SymbolEncoding`,
    `ZapfDingbatsEncoding`, `Type1Encoding`

- [ ] B5. `pdmodel/font` — the simple font path
  - `PDFontLike`, `PDFont`, `PDSimpleFont`, `PDFontDescriptor`, `PDFontFactory`
  - `Standard14Fonts`, `PDType1Font`, `PDTrueTypeFont`, `PDType3Font`,
    `PDType3CharProc`, `PDVectorFont`, `UniUtil`
  - Leave `PDType0Font`, `PDCIDFont*`, `PDType1CFont`, `PDMMType1Font`,
    the `FontMapper` chain and every `*Embedder` to slices 4 and 7

- [ ] B6. Close the holes slice 2 recorded in `migration/STATUS.md`
  - `PDResources.GetFont` and the direct font cache
  - `ResourceCache`
  - the font field of `PDTextState`
  - update the STATUS.md rows that call these deferred

- [ ] B7. `contentstream/operator/text` — the 5 remaining processors
  - `SetFontAndSize`, `ShowText`, `ShowTextAdjusted`, `ShowTextLine`,
    `ShowTextLineAndSpace`

- [ ] B8. `contentstream/PDFStreamEngine` — the text path
  - `showText`, `showGlyph`, `showFontGlyph`, `showType3Glyph`,
    `applyTextAdjustment`, `getDefaultFont`, `processType3Stream`

- [ ] B9. `pdfbox/text` — 6 files
  - `TextPosition`, `TextPositionComparator`, `LegacyPDFStreamEngine`,
    `PDFTextStripper`, `PDFTextStripperByArea`, `PDFMarkedContentExtractor`
  - Note `PDFTextStripper` extends `LegacyPDFStreamEngine`, not the engine
    directly

- [ ] B10. `pdfbox/util/IterativeMergeSort` — 1 file
  - `PDFTextStripper` falls back to it when `TextPositionComparator` turns out
    not to be transitive and the JDK sort throws. Port `TestSort` with it.

- [ ] B11. **Only if the loader decision below said yes** — `pdfbox/Loader`,
      `pdfparser/PDFParser`, `XrefParser`, `BruteForceParser`,
      `PDFObjectStreamParser`, `PDFXrefStreamParser`, and `PDDocument`,
      `PDDocumentCatalog`, `PDDocumentInformation`
  - This is the whole file-opening path. It is not small, and it is not this
    slice's subject. See Blocked.

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found on the way in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md` — the slice 3 section, and the slice 2
      rows this slice closes
- [ ] C6. Report the corpus score as *N of 40*

---

# Phase D — Adversarial review

敵対的レビュー. Green tests prove the port passes the tests, not that it is a
faithful migration. Go in assuming it is wrong. Every check below is a question
the ported tests cannot answer.

- [ ] D1. Read every ported file against its Java side by side
  - Is any method missing? Any branch of an `if`, any `case`, any `catch`?
  - Is any loop bound, any off-by-one, any `<` that should be `<=` different?
  - Java `int` narrows on cast and `float` saturates; Go does neither. Is every
    such conversion written out?

- [ ] D2. Hunt for silently dropped behaviour
  - Anything Java does in a `finally` — is it still done on the Go error path?
  - Anything Java logs and swallows — does the Go swallow it too, or does it
    return an error the Java would not have?
  - Anything Java throws — is it an error, or a panic, and is that the right one?

- [ ] D3. Check the tests are Java-derived, not Go-derived
  - For each assertion: is that value in the Java test, or did it come from
    running the Go? A value read off the port proves nothing.
  - Which Java test cases were dropped, and is each one recorded with a reason?

- [ ] D4. Check every deferral is real and recorded
  - Every "not ported yet" in a doc comment — is it in `migration/STATUS.md`?
  - Every deferral — is it deferred because the type is absent, or because it
    was hard? The second is not a deferral.

- [ ] D5. Check the Java bugs
  - Every bug found — is it in `migration/JAVA-BUGS.md` with where, what,
    what correct would be, where the Go carries it, and how confident?
  - Was any of them "fixed" on the way past? Revert it.

- [ ] D6. Check the corpus honestly
  - Of the 40, which fail and why? Is each failure a port defect, a missing
    font, or a genuine Java difference?
  - Do not tune the Go until a document passes. Find the cause.

- [ ] D7. Write the review down
  - What was checked, what was found, what was fixed, what is still open

---

# Phase E — User feedback

- [ ] E1. Stop and wait for the user's review. Do not start slice 4.

- [ ] E2. For each item of feedback, judge it before acting
  - Is it a port defect, a missing piece of scope, or a difference the Java
    itself has?
  - A Java difference is not fixed — it is recorded in `JAVA-BUGS.md` and the
    user is told why it stays.

- [ ] E3. Where it needs fixing, write a **strict** test first
  - Strict: it fails before the fix, takes the real path with the real types,
    and asserts what the Java does
  - Then fix the Go
  - Then `gofmt`, `go vet`, `go test ./...` again

- [ ] E4. Report back
  - What was changed, what was not, and why for each

---

# Blocked

- [ ] Decide whether the loader is ported in this slice
  - `TestTextStripper.java` imports `org.apache.pdfbox.Loader` and `PDDocument`;
    it opens the 40 files from disk
  - `Loader.java` is not ported — there is no Go file at `go/pdfbox/` root
  - `PDDocument` and `PDDocumentCatalog` are not ported
  - `pdfparser` is 12 of 18: `PDFParser`, `XrefParser`, `BruteForceParser`,
    `PDFObjectStreamParser`, `PDFXrefStreamParser` are all absent
  - **Without them A8 and C6 cannot run and the slice scores nothing.** Every
    other task in this file is unaffected.
