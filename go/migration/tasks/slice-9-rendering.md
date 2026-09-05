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
- [x] A5. Rendered output needs a comparison strategy — decided, see Blocked
  - Java's tests compare against reference images. Decide what the Go compares
    against before writing anything that draws.

---

# Phase B — Port the implementation

- [x] B0. **Take the raster backend decision first.** `PLAN.md` lists the three
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

- [x] B0. The raster backend decision blocks everything from B6 onward.
      `PLAN.md` says to take it before starting, not during.

      **Decided: `PLAN.md`'s third option — port the geometry, defer the raster
      backend behind an interface.** That is the default it names, and the two
      precedents it cites hold up: PdfBox-Android vendored Harmony's
      `AffineTransform` and delegated rasterisation to `android.graphics`, and
      PDFBox itself already runs one glyph-layout interface against two
      backends in `pdfbox-layout-awt` and `pdfbox-layout-fop`.

      What that means concretely, because "defer it" on its own does not say
      enough to write code against:

      **Everything that computes is ported.** Constructive area geometry,
      the PDF functions, the colour spaces and their conversions, the seven
      shading types' colour evaluation, the graphics state, the blend mode and
      soft mask arithmetic, the path and colour operators, and `PageDrawer`'s
      decisions about what to draw. None of that is Java2D; all of it is
      arithmetic the port can carry exactly.

      **Only the last step is behind the interface** — filling a path with a
      winding rule, stroking one with a stroke spec, intersecting the clip,
      drawing a raster with a transform, and pushing and popping a compositing
      layer. That is the boundary `PageDrawer` actually crosses into Java2D,
      and it is small.

      **No raster backend ships in this slice.** Option 1's
      `golang.org/x/image/vector` plus hand-written compositing is a slice of
      its own; taking it here would mean writing an anti-aliasing rasteriser
      and a blend-mode compositor before a single PDF operator was ported, and
      D7 warns exactly about how convincing a wrong one looks. What ships is
      the interface, everything above it, and a recording backend for the tests
      (see A5). `PDFRenderer.renderImage` therefore answers an error saying no
      backend is installed, rather than a blank image, and `pdfbox/printing`
      goes the same way.

      **What this costs is written up under D9**, not left implicit: the port
      cannot produce a rendered page, so it cannot be used to rasterise, to
      print, or to run PDFBox's own image-comparison tests. Every slice that
      deferred something to "the rendering slice" gets the computing half of it
      here and keeps waiting for the drawing half.

- [x] A5. Rendered output needs a comparison strategy.

      **Decided: compare numbers and call sequences, never pixels.** Java's
      rendering tests compare against reference images, which needs a
      rasteriser to produce one and tolerates being subtly wrong when it has
      one — the failure D7 names. With no backend there is no image, and the
      strategy that follows is stronger rather than weaker:

      - `Area` against the JDK's documented contract, not against what the
        renderer happens to need (D8).
      - Functions, colour space conversions and shading colour evaluation
        against values taken from the Java, at chosen inputs. These are pure
        functions of their arguments and compare exactly.
      - Blend modes and soft masks against the arithmetic in the
        specification and in Java's composites, per channel.
      - `PageDrawer` against a **recording backend**: the test installs a
        backend that records every call the drawer makes, runs a real content
        stream through the real engine, and asserts the sequence. That says
        what the drawer decided to draw, in order, with what state — which is
        what a rendering test is actually for, and it does not depend on a
        rasteriser existing.

      An image comparison against PDFBox's reference PNGs stays possible later,
      once a backend exists, and is recorded as the thing this strategy does
      not cover.

## A note on where this branch started

Branched from `migration-base` at `820ac96bf`, which is what this file says to
do. `slice/8-forms-annotations` had not merged at that point, and it holds two
things this slice's scope table also names: `pdmodel/graphics/form`, which slice
8 ported because an appearance stream is a form XObject, and
`rendering/RenderDestination`, which it ported because an optional content group
takes one.

Neither is re-ported here. B5 and B8 merge `migration-base` in once slice 8 has
landed and build on what is there. `pdfbox/rendering` also imports ten slice 8
types directly — `PDAnnotation`, `AnnotationFilter`, `PDAnnotationUnknown`,
`PDAppearanceDictionary`, `PDAppearanceEntry`, `PDPropertyList`,
`PDOptionalContentProperties`, `PDOptionalContentGroup` and its `RenderState`,
and `PDOptionalContentMembershipDictionary` — so B8 cannot start before that
merge either. B0 through B7 do not need any of it.
