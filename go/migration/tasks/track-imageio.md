# Implementation Plan

Track — `tools/imageio`. Writing a raster out to a file.

**Branch: `track/imageio`** — from and back to `migration-base`.

Depends on **nothing**, and is the first of the last three. It is small, and it
is on the critical path: `track/raster` needs it for one task.

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

5 Java classes, 1 Java test class.

| Java | Files | What it is |
| --- | ---: | --- |
| `tools/imageio` | 4 | `ImageIOUtil`, `TIFFUtil`, `JPEGUtil`, `MetaUtil` |
| `tools/ExtractImages` | 1 | the `export:images` command |
| `tools/imageio/TestImageIOUtils` | 1 test | the Java test |

**This branch does not need a rasteriser, and that is the point of taking it
first.** `ExtractImages` walks the content stream with
`PDFGraphicsStreamEngine`, which slice 9 ported, and never imports `rendering`;
the port already decodes an embedded image to pixels, because
`image.PDImage.Image()` answers a `go image.Image`. What is missing is only the
step that writes those pixels to a file.

`PDFToImage` is **not** in scope. It imports `rendering.PDFRenderer` as well as
`ImageIOUtil`, so it belongs to `track/raster` — it is the single edge between
the two branches.

---

# Phase A — Write the tests

- [x] A0. **Decide what writes PNG, JPEG and TIFF.** Java uses `javax.imageio`,
      which is a registry of plugins; Go has `image/png` and `image/jpeg` in
      the standard library and **no TIFF writer at all**.
  - PNG and JPEG are settled by the standard library. TIFF is the decision:
    write a baseline writer here, or record TIFF as unsupported and say so in
    the command's help. Take it before A1, and record it in `STATUS.md`.
  - `ImageIOUtil` also writes the resolution into the file's metadata — a PNG
    `pHYs` chunk, a JPEG JFIF density, a TIFF tag — which is what `MetaUtil`
    and `JPEGUtil` are for. Go's encoders write none of that. Decide whether to
    write those bytes by hand or to record the loss.

- [x] A1. Port `TestImageIOUtils`
  - It renders pages and writes them, which needs a raster this branch does not
    have. Port what it asserts about the *writing* — the formats, the file
    names, the resolution written into the file — over images this branch can
    make without a renderer, and record what is left for `track/raster`.

- [x] A2. `ExtractImages` has no Java test. Write from source, the way
      `track/tools` did for its 18 commands, and assert against images
      extracted from the checked-in PDFs.

---

# Phase B — Port the implementation

**Every function this phase touches needs a test that says it works.**

- [x] B1. `ImageIOUtil` — the entry points `ExtractImages` and `PDFToImage` use
- [x] B2. `JPEGUtil`, `MetaUtil`, `TIFFUtil`, as A0 decided
- [x] B3. `ExtractImages`, and its row out of `go/tools/notbuilt.go`

---

# Phase C — Run and fix

- [x] C1. `gofmt -l .` clean
- [x] C2. `go vet ./...` clean
- [x] C3. `go test ./...` green
- [x] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [x] C5. Update `migration/STATUS.md`, and the rows in `go/tools/notbuilt.go`
      this branch closes — the dispatcher's help reads that list, so a command
      that now exists must come out of it

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

- [x] D4. Check every function phase B touched has a test
  - Name the test that covers it. Not "the suite is green"
  - Where there is none, the function was changed on an argument rather than on
    evidence. Write the test, and take whatever it says

- [x] D5. Check every deferral is real and recorded
  - Every "not ported yet" in a doc comment — is it in `migration/STATUS.md`?
  - Every deferral — is it deferred because the type is absent, or because it
    was hard? The second is not a deferral.

- [x] D6. Check the Java bugs
  - Every bug found — is it in `migration/JAVA-BUGS.md` with where, what,
    what correct would be, where the Go carries it, and how confident?
  - Was any of them "fixed" on the way past? Revert it.

- [x] D7. Write the review down
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

Nothing. This branch is why the other two are not blocked on each other.
