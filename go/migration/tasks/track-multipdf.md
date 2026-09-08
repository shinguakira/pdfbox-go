# Implementation Plan

Track — `multipdf`. Combining documents: merge, overlay, layers.

**Branch: `track/multipdf`** — from and back to `migration-base`.

Depends on **nothing**, and is **off the critical path**. It can be worked at
any time, in parallel with `track/imageio` and `track/raster`, and merged
whenever it is ready.

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

3 Java classes, 2 commands, 4 Java test classes.

| Java | Files | What it is |
| --- | ---: | --- |
| `multipdf/PDFMergerUtility` | 1 | merges documents, including their AcroForms |
| `multipdf/Overlay` | 1 | stamps one document over another |
| `multipdf/LayerUtility` | 1 | imports a page as an optional content group |
| `tools/PDFMerger`, `tools/OverlayPDF` | 2 | the `merge` and `overlay` commands |
| `PDFMergerUtilityTest`, `MergeAcroFormsTest`, `MergeAnnotationsTest`, `OverlayTest`, `TestLayerUtility`, `PDFCloneUtilityTest` | 6 tests | the Java tests |

`multipdf` is 6 files; `Splitter`, `PageExtractor` and `PDFCloneUtility` are
already ported by slice 7. This is the other half.

**How it was missed.** Slice 7 deferred all three to slice 8, slice 8 never took
them, and the coverage survey counted them as ported because their names appear
in a Go comment saying they are *absent*. `track/tools` then found the same
matcher failing the other way. Nothing about the work is hard or blocked — it
simply had no branch.

`PDFCloneUtility` is the piece the other three are written against, and it is
in. Read it first.

---

# Phase A — Write the tests

- [ ] A1. Port `PDFMergerUtilityTest` — the biggest of the five, and the one
      that carries the merged-document corpus
- [ ] A2. Port `MergeAcroFormsTest` and `MergeAnnotationsTest`
  - Both are about what merging does to slice 8's structures: field names that
    collide, annotation appearance streams that move. Slice 8 is merged, so
    there is something to assert against.
- [ ] A3. Port `OverlayTest`
- [ ] A4. Port `TestLayerUtility`
  - It renders to compare, which needs `track/raster`. Port what it asserts
    about the object graph and record the pixel half.

- [ ] A5. Port `PDFCloneUtilityTest`
  - `PDFCloneUtility` was ported by slice 7 and its test was not: `STATUS.md`
    records all three of its cases as needing `PDPageContentStream`,
    `PDFMergerUtility` or `PDOptionalContentProperties`, and two of those three
    have since been ported. This branch brings the third, so the whole class
    can go in. **Port it before B1** -- it tests the machinery the other three
    classes are written against.

---

# Phase B — Port the implementation

**Every function this phase touches needs a test that says it works.**

- [ ] B1. `PDFMergerUtility` — the destination document, the source list, and
      what it does with each of the catalog's dictionaries
- [ ] B2. `Overlay`, whose `Position` enum the command takes as an option
- [ ] B3. `LayerUtility`
- [ ] B4. `PDFMerger` and `OverlayPDF`, and their rows out of
      `go/tools/notbuilt.go`

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md`, and the rows in `go/tools/notbuilt.go`
      this branch closes — the dispatcher's help reads that list, so a command
      that now exists must come out of it

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

- [ ] D4. Check every function phase B touched has a test
  - Name the test that covers it. Not "the suite is green"
  - Where there is none, the function was changed on an argument rather than on
    evidence. Write the test, and take whatever it says

- [ ] D5. Check every deferral is real and recorded
  - Every "not ported yet" in a doc comment — is it in `migration/STATUS.md`?
  - Every deferral — is it deferred because the type is absent, or because it
    was hard? The second is not a deferral.

- [ ] D6. Check the Java bugs
  - Every bug found — is it in `migration/JAVA-BUGS.md` with where, what,
    what correct would be, where the Go carries it, and how confident?
  - Was any of them "fixed" on the way past? Revert it.

- [ ] D7. Write the review down
  - What was checked, what was found, what was fixed, what is still open

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

- [ ] `TestLayerUtility` and any other case that compares rendered pages. Held
      for `track/raster`; port the rest of the class and record the omission
      rather than skipping the file.
