# Implementation Plan

Slice 4 — text extraction, hard fonts. Drives the corpus score up.

**Branch: `slice/4-<name>`** — from and back to `migration-base`.
Depends on `slice/3`.

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

Five phases, in this order, never overlapping. See
[`TEMPLATE.md`](TEMPLATE.md) for what each phase means.

A — write the test · B — port the implementation · C — run and fix ·
D — adversarial review · E — user feedback, strict test first

## Scope

The bulk of `fontbox`, and the CID half of the font model.

| Java package | Files | Java tests |
| --- | ---: | ---: |
| `fontbox/cff` | 26 | 6 |
| `fontbox/type1` | 6 | 1 |
| `fontbox/cmap` | 5 | 5 |
| `fontbox/pfb` | 1 | — |
| `fontbox/ttf` — what slice 3 left | ~29 of 44 | 4 |
| `fontbox/ttf/gsub` | 13 | — |
| `fontbox/ttf/model` | 5 | — |
| `fontbox/ttf/table/common` | 12 | — |
| `fontbox/ttf/table/gsub` | 9 | — |
| `fontbox/util/autodetect` | 7 | — |
| `pdmodel/font` — the CID and Type0 half | ~15 of 39 | 2 |

---

# Phase A — Write the tests

- [ ] A1. `fontbox/cmap` — port all 5 Java tests
- [ ] A2. `fontbox/type1` — port the 1 Java test
- [ ] A3. `fontbox/cff` — port all 6 Java tests
- [ ] A4. `fontbox/ttf` — port `GlyphSubstitutionTableTest`,
      `GlyphSubstitutionTableLiberationFontTest`, `TTFSubsetterTest`,
      `TrueTypeFontCollectionTest`
- [ ] A5. `pdmodel/font` — port `CIDCharSetMatchTest`,
      `PDCIDFontType0SubstituteTest`
- [ ] A6. Extend the slice 3 corpus test — the score should move

---

# Phase B — Port the implementation

- [ ] B1. `fontbox/cmap` — the CMap parser and the predefined CMaps
- [ ] B2. `fontbox/type1` — the Type 1 font parser, and `fontbox/pfb` — the
      PFB container it arrives in
- [ ] B3. `fontbox/cff` — CFF and Type 2 charstrings, 26 files
- [ ] B4. `fontbox/ttf` — the rest of it
  - GSUB: `gsub/` 13 files, `model/` 5, `table/common/` 12, `table/gsub/` 9 —
    39 files, more than the 13 the `gsub` directory alone suggests
  - OpenType (`OTFParser`, `OpenTypeFont`, `OpenTypeScript`, `CFFTable`),
    collections (`TrueTypeCollection`, `TTCDataStream`), the vertical and
    kerning tables, `TTFSubsetter`, `GlyphRenderer`,
    `SubstitutingCmapLookup`, `RandomAccessReadUnbufferedDataStream`,
    `DigitalSignatureTable`, `FontHeaders`
- [ ] B5. `pdmodel/font` — `PDType0Font`, `PDCIDFont`, `PDCIDFontType0`,
      `PDCIDFontType2`, `CIDSystemInfo`, `PDCIDSystemInfo`, `CMapManager`,
      `PDType1CFont`, `PDMMType1Font`
- [ ] B6. `pdmodel/font` — the font mapper chain: `FontMapper`,
      `FontMapperImpl`, `FontMappers`, `FontMapping`, `CIDFontMapping`,
      `FontProvider`, `FileSystemFontProvider`, `FontCache`, `FontInfo`,
      `FontFormat`
  - and `fontbox/util/autodetect` — 7 files, the per-platform font directory
    finders `FileSystemFontProvider` scans. Windows, Mac and Unix each have
    their own; the Go equivalent is a substitution, not a transliteration.
- [ ] B7. `pdmodel/font/encoding` — anything slice 3 left

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md`
- [ ] C6. Report the corpus score as *N of 40*, and the change from slice 3

---

# Phase D — Adversarial review

敵対的レビュー. See [`TEMPLATE.md`](TEMPLATE.md) for D1–D6. Additionally:

- [ ] D7. Font substitution is environment-dependent
  - `FileSystemFontProvider` scans the machine's fonts. A corpus document that
    passes on one machine and fails on another is not a passing document.
  - Check what the Go does when no substitute is found, against what Java does.

---

# Phase E — User feedback

See [`TEMPLATE.md`](TEMPLATE.md) for E1–E4. E3 is the strict-test rule.

---

# Blocked

- [ ] Nothing outside this branch, provided `slice/3` has landed. If the loader
      decision in slice 3 went the other way, the corpus still cannot run and
      C6 is unreportable.
