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
- **DO NOT STOP UNTIL PHASE E.** Phases A to D run end to end. Finishing a
  task is not a stopping point; neither is finishing a phase, a package, or a
  commit. Do not pause to report progress as if it were a result, do not ask
  whether to continue, and do not end a turn with a list of what is left.
  **E1 is the only stop in this file** — that is where the user reviews. Only
  the user stops the work before it.

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

- [x] A1. `fontbox/afm` — port all 8 Java tests
  - `AFMParserTest`, `FontMetricsTest`, `CharMetricTest`, `KernPairTest`,
    `TrackKernTest`, `LigatureTest`, `CompositeTest`, `CompositePartTest`

- [x] A2. `fontbox/encoding` — port `EncodingTest`

- [x] A3. `fontbox/ttf` — port the 5 that text extraction needs
  - `TestTTFParser`, `TestCMapSubtable`, `WGL4NamesTest`,
    `GlyfCompositeDescriptTest`, `RandomAccessReadBufferDataStreamTest`
  - Skip `GlyphSubstitutionTable*Test`, `TTFSubsetterTest`,
    `TrueTypeFontCollectionTest` — slice 4
  - `TestCMapSubtable` is **not** ported: both of its tests read fonts that the
    Java build downloads into `target/fonts` (`NotoSansSC-Regular.otf`,
    `ipag00303/ipag.ttf`), which this repository does not carry. Its subject,
    `CmapSubtable.getCharCodes` with several codes for one glyph, is covered by
    the format 4 read that `TestPostTable` exercises.
  - `TestTTFParser.testParseVertical` and `testParseHeaders` are not ported
    either: the first reads the same downloaded font, the second goes through
    `FontHeaders`, which slice 4 ports. `testParseMisc` is ported for the part
    this slice covers -- the kerning, vertical and GSUB assertions are slice 4.

- [x] A4. `pdmodel/font/encoding` — port `TestFontEncoding`
  - Write from source for `GlyphList` — Java has no test for it
  - `testPDFBox3884` is **not** ported: it builds a document, saves it, loads it
    back and runs the text stripper over it. Its subject, the reverse mapping
    preferring a name a standard encoding carries, is covered directly in
    `glyphlist_test.go`.

- [x] A5. `pdmodel/font` — port `PDFontTest`
  - Write from source for `PDFontDescriptor`, `PDFontFactory`, `Standard14Fonts`
    — Java has no test for these
  - Skip `CIDCharSetMatchTest`, `PDCIDFontType0SubstituteTest` — slice 4
  - Skip `TestFontEmbedding`, `TestToUnicodeWriter` — slice 7, they write PDFs
  - Only `testPDFox4318` of `PDFontTest` is ported. Every other test in that
    class builds a `PDDocument`, saves it, loads it back, or runs the text
    stripper. The rest of the file is written from source, per the line above.

- [x] A6. `contentstream` — extend `streamengine_test.go`
  - No Java test exists; extend the slice 2 test for `Tf`, `Tj`, `TJ`, `'`, `"`
    and for the glyph hooks

- [x] A7. `pdfbox/text` — port `PDFTextStripperByAreaTest` and `BidiTest`
  - Write from source for `TextPosition` and `TextPositionComparator` — Java
    tests them only through the stripper
  - Neither Java test is portable: both open a PDF with `Loader`. The tests
    written instead build the page in memory and run the same walk over it,
    covering what each Java test covers plus the geometry of `TextPosition`.

- [x] A8. `pdfbox/text` — port `TestTextStripper`, the corpus harness
  - Table test over the 40 `.pdf` files in `pdfbox/src/test/resources/input/`
  - Compare against the checked-in expected text, sorted and unsorted
  - **See Blocked below — this one cannot run yet**

---

# Phase B — Port the implementation

Written from the Java source, in dependency order.

- [x] B0. `fontbox` root — 2 files
  - `FontBoxFont`, `EncodedFont`. `PDFont` is written against `FontBoxFont`, so
    nothing below compiles without it.

- [x] B1. `fontbox/afm` — 8 files
  - `AFMParser`, `FontMetrics`, `CharMetric`, `KernPair`, `TrackKern`,
    `Ligature`, `Composite`, `CompositePart`

- [x] B2. `fontbox/encoding` — 4 files
  - `Encoding`, `BuiltInEncoding`, `MacRomanEncoding`, `StandardEncoding`

- [x] B3. `fontbox/ttf` — the reading path only, roughly 15 of 44 files
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

- [x] B4. `pdmodel/font/encoding` — 12 files
  - `Encoding`, `BuiltInEncoding`, `DictionaryEncoding`, `GlyphList`,
    `StandardEncoding`, `WinAnsiEncoding`, `MacRomanEncoding`,
    `MacOSRomanEncoding`, `MacExpertEncoding`, `SymbolEncoding`,
    `ZapfDingbatsEncoding`, `Type1Encoding`

- [x] B5. `pdmodel/font` — the simple font path
  - `PDFontLike`, `PDFont`, `PDSimpleFont`, `PDFontDescriptor`, `PDFontFactory`
  - `Standard14Fonts`, `PDType1Font`, `PDTrueTypeFont`, `PDType3Font`,
    `PDType3CharProc`, `PDVectorFont`, `UniUtil`
  - Leave `PDType0Font`, `PDCIDFont*`, `PDType1CFont`, `PDMMType1Font`,
    the `FontMapper` chain and every `*Embedder` to slices 4 and 7

- [x] B6. Close the holes slice 2 recorded in `migration/STATUS.md`
  - `PDResources.GetFont` and the direct font cache
  - `ResourceCache`
  - the font field of `PDTextState`
  - update the STATUS.md rows that call these deferred

- [x] B7. `contentstream/operator/text` — the 5 remaining processors
  - `SetFontAndSize`, `ShowText`, `ShowTextAdjusted`, `ShowTextLine`,
    `ShowTextLineAndSpace`

- [x] B8. `contentstream/PDFStreamEngine` — the text path
  - `showText`, `showGlyph`, `showFontGlyph`, `showType3Glyph`,
    `applyTextAdjustment`, `getDefaultFont`, `processType3Stream`

- [x] B9. `pdfbox/text` — 6 files
  - `TextPosition`, `TextPositionComparator`, `LegacyPDFStreamEngine`,
    `PDFTextStripper`, `PDFTextStripperByArea`, `PDFMarkedContentExtractor`
  - Note `PDFTextStripper` extends `LegacyPDFStreamEngine`, not the engine
    directly

- [x] B10. `pdfbox/util/IterativeMergeSort` — 1 file
  - `PDFTextStripper` falls back to it when `TextPositionComparator` turns out
    not to be transitive and the JDK sort throws. Port `TestSort` with it.

- [x] B11. **Decided: yes.** `pdfparser/COSParser`, `XrefParser`,
      `BruteForceParser`, `PDFXrefStreamParser`, `PDFObjectStreamParser`,
      `PDFParser`, then `pdmodel/PDDocument`, `PDDocumentCatalog`,
      `PDDocumentInformation`, then `pdfbox/Loader`
  - **This is a special case and is not a precedent.** Work outside a branch's
    scope is not allowed. It is allowed here for one reason: this is not new
    scope, it is *slice 1's* scope. `PLAN.md` slice 1 is "open a document" and
    lists `pdfbox/pdfparser` at 18 files; the branch was merged to
    `migration-base` at 12 of 18, with `STATUS.md` recording `COSParser` as
    "next" and `PDFParser` as "not started — the entry point". `go/cmd/` is
    empty and nothing in the tree defines `Load`. Slice 1 did not deliver what
    it says it delivered.
  - Slices 2 and 3 did not notice, because both take a `PDPage` a caller hands
    them. Slice 3 is the first slice whose acceptance criterion — score 40 real
    PDFs — cannot be met without opening a file.
  - Port order, each test-first: `COSParser`, `XrefParser`,
    `PDFXrefStreamParser`, `PDFObjectStreamParser`, `BruteForceParser`,
    `PDFParser`, `PDDocument`, `PDDocumentCatalog`, `PDDocumentInformation`,
    `Loader`.
  - `AGENTS.md` flags parsing and xref recovery as historically bug-prone. Port
    line for line. This is not the place to improve on the original.
  - When it lands, update the slice 1 rows of `STATUS.md`, not just slice 3's.

---

# Phase C — Run and fix

- [x] C1. `gofmt -l .` clean
- [x] C2. `go vet ./...` clean
- [x] C3. `go test ./...` green
- [x] C4. Record every Java bug found on the way in `migration/JAVA-BUGS.md`
- [x] C5. Update `migration/STATUS.md` — the slice 3 section, and the slice 2
      rows this slice closes
- [x] C6. Report the corpus score as *N of 40* — **16 of 40**

---

# Phase D — Adversarial review

敵対的レビュー. Green tests prove the port passes the tests, not that it is a
faithful migration. Go in assuming it is wrong. Every check below is a question
the ported tests cannot answer.

- [x] D1. Read every ported file against its Java side by side
  - Is any method missing? Any branch of an `if`, any `case`, any `catch`?
  - Is any loop bound, any off-by-one, any `<` that should be `<=` different?
  - Java `int` narrows on cast and `float` saturates; Go does neither. Is every
    such conversion written out?

- [x] D2. Hunt for silently dropped behaviour
  - Anything Java does in a `finally` — is it still done on the Go error path?
  - Anything Java logs and swallows — does the Go swallow it too, or does it
    return an error the Java would not have?
  - Anything Java throws — is it an error, or a panic, and is that the right one?

- [x] D3. Check the tests are Java-derived, not Go-derived
  - For each assertion: is that value in the Java test, or did it come from
    running the Go? A value read off the port proves nothing.
  - Which Java test cases were dropped, and is each one recorded with a reason?

- [x] D4. Check every deferral is real and recorded
  - Every "not ported yet" in a doc comment — is it in `migration/STATUS.md`?
  - Every deferral — is it deferred because the type is absent, or because it
    was hard? The second is not a deferral.

- [x] D5. Check the Java bugs
  - Every bug found — is it in `migration/JAVA-BUGS.md` with where, what,
    what correct would be, where the Go carries it, and how confident?
  - Was any of them "fixed" on the way past? Revert it.

- [x] D6. Check the corpus honestly
  - Of the 40, which fail and why? Is each failure a port defect, a missing
    font, or a genuine Java difference?
  - Do not tune the Go until a document passes. Find the cause.

- [x] D7. Write the review down
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

- [x] Decide whether the loader is ported in this slice — **yes**
  - `TestTextStripper.java` imports `org.apache.pdfbox.Loader` and `PDDocument`;
    it opens the 40 files from disk
  - `Loader.java` is not ported — there is no Go file at `go/pdfbox/` root
  - `PDDocument` and `PDDocumentCatalog` are not ported
  - `pdfparser` is 12 of 18: `COSParser`, `XrefParser`, `BruteForceParser`,
    `PDFObjectStreamParser`, `PDFXrefStreamParser`, `PDFParser` are all absent
  - **Without them A8 and C6 cannot run and the slice scores nothing.** Every
    other task in this file is unaffected.
  - Decided yes, as a special case, because the work is slice 1's unfinished
    scope rather than new scope. See B11 for the reasoning and the port order.
