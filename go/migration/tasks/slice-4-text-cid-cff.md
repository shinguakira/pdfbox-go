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

- [x] A1. `fontbox/cmap` — port all 5 Java tests
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

- [x] B1. `fontbox/cmap` — the CMap parser and the predefined CMaps
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
  - Does each test take the real path, with the real types? A test over a
    stand-in can pass while the path it stands for is broken.
  - Which Java test cases were dropped, and is each one recorded with a reason?

- [ ] D4. Check every deferral is real and recorded
  - Every "not ported yet" in a doc comment — is it in `migration/STATUS.md`?
  - Every deferral — is it deferred because the type is absent, or because it
    was hard? The second is not a deferral.

- [ ] D5. Check the Java bugs
  - Every bug found — is it in `migration/JAVA-BUGS.md` with where, what,
    what correct would be, where the Go carries it, and how confident?
  - Was any of them "fixed" on the way past? Revert it.

- [ ] D6. Write the review down
  - What was checked, what was found, what was fixed, what is still open

And for this branch in particular:

- [ ] D7. Font substitution is environment-dependent
  - `FileSystemFontProvider` scans the machine's fonts. A corpus document that
    passes on one machine and fails on another is not a passing document.
  - Check what the Go does when no substitute is found, against what Java does.

---

# Phase E — User feedback

- [ ] E1. Stop and wait for the user's review. Do not start the next branch.

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

- [ ] Nothing outside this branch, provided `slice/3` has landed. If the loader
      decision in slice 3 went the other way, the corpus still cannot run and
      C6 is unreportable.
