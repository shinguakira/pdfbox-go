# Implementation Plan

Track — writing a font into a document: TrueType embedding and subsetting.

**Branch: `track/font-embedding`** — from and back to `migration-base`.

Not a slice: `slice/4` ported the fonts a document is *read* with and `slice/7`
ported writing, and the embedding half fell between them. Both are merged, so
this depends on nothing that does not exist.

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

The five files `STATUS.md` records as `pdmodel/font at 34 of 39`, and the half
of two ported classes that was deferred with them.

| Java source | Lines | What it does |
| --- | ---: | --- |
| `pdmodel/font/TrueTypeEmbedder.java` | 401 | reads a TTF, writes `/FontFile2` and the descriptor, drives the subsetter |
| `pdmodel/font/PDCIDFontType2Embedder.java` | 738 | the CID half — `/CIDToGIDMap`, `/W`, the descendant font |
| `pdmodel/font/PDTrueTypeFontEmbedder.java` | 135 | the simple-font half, with its encoding |
| `pdmodel/font/ToUnicodeWriter.java` | 228 | the `/ToUnicode` CMap the embedded font needs |
| `pdmodel/font/Subsetter.java` | 40 | the interface `addToSubset`/`subset` are declared on |

And the embedding halves the read-side port left as holes, which are named in
the Go already:

- `PDType0Font` — `load`, `loadVertical`, `addToSubset`, `subset`. The Go says
  so at `pdtype0font.go:23`, and **three of its methods panic today** rather
  than answer: the comment at line 445 reads "so this always panics until the
  embedding half arrives."
- `PDTrueTypeFont.load` — the simple-font factory.

**`fontbox`'s side is already done.** `TTFSubsetter` is ported at
`go/fontbox/ttf/ttfsubsetter.go`, which is the machinery the whole track leans
on. This branch is the `pdmodel` layer over it.

Java test: `pdmodel/font/TestFontEmbedding.java`, 17 cases, 914 lines.

### Why this is worth a branch of its own

It is not five files of tidying. **A Go program cannot today write a PDF with an
embedded font**, which is most of what writing a PDF is for: `slice/7` can merge
and rewrite documents whose fonts are already embedded, and can write text in
the 14 standard fonts, and nothing else. The Java class that would do it is
reached, in the Go, by a method that panics.

---

# Phase A — Write the tests

- [ ] A1. Port `TestFontEmbedding` — 17 cases
  - It writes documents and reads them back. Where a case needs
    `PDFTextStripper` to verify the round trip, that is ported and available
  - Where a case needs a font file, check `pdfbox/src/test/resources` carries
    it before assuming the case must be dropped
- [ ] A2. Note every case not ported, and why. Font files this repository does
      not have is a reason; "it was awkward" is not

---

# Phase B — Port the implementation

**Every function this phase touches needs a test that says it works.** Where
porting one site makes you notice a second with the same defect, the second one
needs its own test before it is touched: without it nobody can say whether that
code works, before the change or after, and the suite stays green either way.
Either write the test, or leave the site alone and record it.

If the defect is a Java standard-library contract, port the contract once into a
helper rather than patching each call site. See
[`../conventions/java-to-go.md`](../conventions/java-to-go.md) for the ones this
port has already paid for more than once.

In dependency order — the interface, then the shared base, then the two halves.

- [ ] B1. `Subsetter`
  - 40 lines of interface. In Go it is the method set `PDType0Font` and
    `TrueTypeEmbedder` satisfy; check what the port already names before adding
    a second name for it
- [ ] B2. `ToUnicodeWriter`
  - **JAVA-BUGS entry 33 is about this class** — `allowDestinationRange` checks
    only one of its two strings. It was found while reading, from the Java, when
    nothing was ported. Port it as written and check the entry still describes
    what the Go does
- [ ] B3. `TrueTypeEmbedder`
  - The descriptor, `/FontFile2`, the subsetting drive. Leans on
    `fontbox/ttf.TTFSubsetter`, which is ported
- [ ] B4. `PDTrueTypeFontEmbedder`, and `PDTrueTypeFont.load`
- [ ] B5. `PDCIDFontType2Embedder`, and `PDType0Font.load` / `loadVertical`
  - The largest file in the branch. `/CIDToGIDMap`, the `/W` widths array, the
    descendant font
- [ ] B6. Replace the panics
  - `pdtype0font.go` has methods that panic where the embedding half was
    missing. Each one is a promise this branch is here to keep; leaving one is
    leaving the branch unfinished

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found on the way in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md`
  - The `pdmodel/font` row, from `34 of 39` to what it becomes
  - The `PDType0Font` and `PDTrueTypeFont` rows, which say the embedding half is
    deferred
  - The summary phase 3 row

---

# Phase D — Adversarial review

敵対的レビュー. Green tests prove the port passes the tests, not that it is a
faithful migration. Go in assuming it is wrong.

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
    running the Go?
  - Which Java test cases were dropped, and is each one recorded with a reason?

- [ ] D4. Check every function phase B touched has a test
  - Name the test that covers it. Not "the suite is green" -- green says the
    code is not broken in a way something already checks, which is a different
    claim from "this works"
  - Where there is none, the function was changed on an argument rather than on
    evidence. Write the test, and take whatever it says

- [ ] D5. Check every deferral is real and recorded

- [ ] D6. Check the Java bugs
  - Every bug found — with where, what, what correct would be, where the Go
    carries it, and how confident?

- [ ] D7. Write the review down

And for this branch in particular:

- [ ] D8. Check the bytes, not just the structure
  - A subsetted font that parses is not a font that is right. Read a document
    this branch writes back with the port's own `fontbox` parser, and check the
    glyphs the subset kept are the glyphs that were asked for
  - Where practical, write the same document from the running Java and compare
    the two `/FontFile2` streams. Identical is the strong result; a difference
    needs a reason

- [ ] D9. Check `/ToUnicode` against JAVA-BUGS 33
  - The entry says the Java checks one of two strings. Confirm the port
    reproduces that and that the entry's "where the Go carries it" is now true

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

Nothing. `slice/4` gave this branch `fontbox`, including `TTFSubsetter`, and
`slice/7` gave it the writer. Both are in `migration-base`.
