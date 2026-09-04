# Implementation Plan

Slice 9 — rendering. Deferred; `PLAN.md` says a decision comes before the work.

**Branch: `slice/9-<name>`** — from and back to `migration-base`.
Depends on `slice/3` (text) and `slice/6` (images). The only slice with two
parents.

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

| Java package | Files | Java tests |
| --- | ---: | ---: |
| `pdfbox/rendering` | 10 | 3 |
| `pdfbox/printing` | 4 | 1 |
| `pdmodel/graphics/shading` | 37 | 0 |
| `pdmodel/graphics/color` — the 20 slice 2 left | ~20 of 23 | — |
| `pdmodel/graphics/pattern` | 3 | — |
| `pdmodel/graphics/form` | 3 | — |
| `pdmodel/graphics/state` — what slice 2 left | 2 of 6 | — |
| `pdmodel/common/function` and `function/type4` | 17 | — |
| `contentstream/operator/color` | 13 | — |
| `contentstream/operator/graphics` | 23 | — |
| `java.awt.geom.Area` and the raster backend | — | — |

`pdmodel/common/function` is easy to miss. Shadings, soft masks and the
separation and DeviceN colour spaces all evaluate PDF functions, and `type4`
is a small PostScript calculator interpreter — 11 of those 17 files.

Java2D does the drawing in PDFBox. Go has nothing equivalent. `PLAN.md` names
three options and defaults to the third: port the geometry, defer the raster
backend behind an interface. Cutting this slice permanently is a legitimate
outcome — PdfPig shipped no renderer and became the standard .NET choice.

---

# Phase A — Write the tests

- [ ] A1. `pdfbox/rendering` — port its 3 Java tests
- [ ] A2. `pdfbox/printing` — port its 1 Java test
- [ ] A3. `pdmodel/graphics/shading` — Java has no test here; write from source
- [ ] A4. `awt/geom` — write `Area` tests from the JDK contract, not from the
      implementation
- [ ] A5. Rendered output needs a comparison strategy
  - Java's tests compare against reference images. Decide what the Go compares
    against before writing anything that draws.

---

# Phase B — Port the implementation

- [ ] B0. **Take the raster backend decision first.** `PLAN.md` lists the three
      options. Write the decision down before B1.
- [ ] B1. `awt/geom.Area` — constructive area geometry. Slice 2 recorded this
      as the one thing blocking `PDGraphicsState.getCurrentClippingPath`.
- [ ] B2. `pdmodel/common/function` and `function/type4` — 17 files
  - Everything below evaluates these. `type4` is a PostScript calculator
    interpreter in its own right.
- [ ] B3. `pdmodel/graphics/color` — the 20 colour spaces slice 2 left, and
      `PDColorSpace.create`
- [ ] B4. `pdmodel/graphics/state` — `PDSoftMask`,
      `PDExtendedGraphicsState`, the two Java composites, `BlendComposite`
- [ ] B5. `pdmodel/graphics/form` — `PDFormXObject`, `PDTransparencyGroup`,
      `PDTransparencyGroupAttributes`, and `pdmodel/graphics/pattern` —
      `PDAbstractPattern`, `PDTilingPattern`, `PDShadingPattern`
  - These are what let `PDFStreamEngine` process a form, a transparency group
    and a tiling pattern. Slice 2 recorded all three as absent, and with them
    `shouldProcessColorOperators` ever being false.
- [ ] B6. `contentstream` — `PDFGraphicsStreamEngine`, `operator/graphics` (23
      files, the path operators and `DrawObject`), `operator/color` (13 files)
- [ ] B7. `pdmodel/graphics/shading` — 37 files, seven shading types
- [ ] B8. `pdfbox/rendering` — `PDFRenderer`, `PageDrawer`, and the rest
- [ ] B9. `pdfbox/printing`

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md` — including every slice 2 row that named
      `Area`, the composites or the soft mask as deferred

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

- [ ] D7. "Looks right" is not a passing test
  - Anti-aliasing, blend modes and shading are places where a wrong port
    produces an image that looks fine and is not. Compare numerically.

- [ ] D8. Check `Area` against the JDK, not against what the renderer needs
  - It is used for clipping, and a clipping region that is nearly right cuts
    the wrong pixels on documents nobody tests.

- [ ] D9. Check what the raster backend decision cost
  - Whatever was chosen in B0, write down what it makes impossible, and record
    it in `STATUS.md` as a deviation rather than leaving it implicit.

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

- [ ] B0. The raster backend decision blocks everything from B6 onward.
      `PLAN.md` says to take it before starting, not during.
