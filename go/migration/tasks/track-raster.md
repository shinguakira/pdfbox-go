# Implementation Plan

Track — the raster. An implementation of `rendering.Backend`, and everything
that has been waiting for one.

**Branch: `track/raster`** — from and back to `migration-base`.

**Take this one last, and take it knowing what it is.** Slice 9 ported
everything in the renderer that computes and put only the drawing behind an
interface; this is the interface's other side. It is the largest single piece of
work left in the migration and the last decision it has.

Depends on `track/imageio` for one task — B5, the `render` command, which writes
its output through `ImageIOUtil`. Everything before B5 can be worked before
`track/imageio` merges; B5 cannot.

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

25 Java classes, 2 commands.

| Java | Files | What it is |
| --- | ---: | --- |
| `graphics/shading` `*ShadingContext` | 11 | the raster loops of the seven shading types |
| `graphics/shading` `*ShadingPaint` | 8 | the `java.awt.Paint` each one is reached through |
| `rendering/GroupGraphics`, `SoftMask` | 2 | transparency groups and soft masks |
| `rendering/TilingPaint`, `TilingPaintFactory` | 2 | tiling patterns |
| `graphics/blend/BlendComposite` | 1 | the `java.awt.Composite` the blend modes are applied through |
| `pdmodel/PDPatternContentStream` | 1 | writing a tiling pattern's own content stream |
| `tools/PDFToImage`, `tools/PrintPDF` | 2 | the `render` and `print` commands |

The model half of all of these is ported and runs: `shading.Shading` evaluates
colours, `PDTilingPattern` reads a pattern, `PDSoftMask` reads a mask,
`blend.BlendMode` computes a blend. What is missing is the code that turns those
answers into pixels.

`printing` is ported in full — `PrintPDF` waits only on the backend, through
`PDFPageable`.

**This is a substitution, not a transliteration.** `java.awt.Graphics2D` has no
Go equivalent, so what this branch writes is not a translation of Java source.
`track/pdfbox-layout` is the worked precedent: it wrote GPOS from the OpenType
specification, then measured the result against the PDFs the Java tests render,
and recorded every difference as a pinned deviation. Do the same here — the
tooling for it already exists, and this branch is what makes those pixel
comparisons possible for the first time.

---

# Phase A — Write the tests

- [x] A0. **Decide what draws.** `PLAN.md`'s slice 9 section names three ways
      and slice 9 took a fourth. The three are still the three:
  - `golang.org/x/image/vector` plus hand-written compositing — small
    dependency, most work. **It is not in `go.mod` and there is no network**;
    settle how it gets there before choosing it.
  - a Cairo or Skia binding — closest to Java2D, adds cgo.
  - write the rasteriser here, over the geometry `awt/geom` already has.
  - Whatever is chosen, `awt/geom.Area` **flattens curves to polylines** where
    the JDK intersects them exactly. That is a recorded approximation and it is
    what the clip is built from. Read `awt/geom/area.go`'s head comment before
    A0, and decide whether the choice makes it better or has to live with it.
  - Record the decision in `STATUS.md` before A1, the way `track/pdfbox-layout`
    recorded its A0.

- [ ] A1. Port the pixel comparisons the earlier branches deferred
  - They are the tests this branch exists to make possible. `STATUS.md` records
    them per slice: `ContentStreamWriterTest`, `TestFontEmbedding`,
    `PDAcroFormFlattenTest`, `TestLayerUtility`, `TestImageIOUtils` and every
    `checkRenderIdent` in `pdfbox-layout-awt` and `pdfbox-layout-fop`.
  - `PDAcroFormFlattenTest` also downloads its PDFs, which is a second reason
    and does not go away with this branch. Record what is left.
  - Do this **before** B, not after. A backend written first and compared
    afterwards is a backend written to whatever it happens to produce.

- [ ] A2. Port the shading tests
  - `PDShadingTest` and the type-specific cases assert colours at points, which
    is what a `ShadingContext` answers. They can be asserted without a full
    page render.

---

# Phase B — Port the implementation

**Every function this phase touches needs a test that says it works.**

Ordered so that each step is testable before the next needs it.

- [ ] B1. The backend itself: fill, stroke, clip, transform, over a solid paint
  - `Fill`, `Draw`, `SetClip`, `SetTransform`, `SetStroke`, `SetPaint` for
    `ColorPaint`, `SetAntiAliasing`. A page of black text on white is the first
    thing that should come out.
- [ ] B2. `DrawImage` and `DrawStencil`, with both interpolations
  - `PDImage.Image()` already answers the pixels; this is placement and
    sampling.
- [ ] B3. The shading contexts and paints, type by type
  - 1, 2 and 3 first — function, axial, radial. Then the mesh types 4 to 7,
    which share `TriangleBasedShadingContext` and `PatchMeshesShadingContext`.
- [ ] B4. `TilingPaint`, `TilingPaintFactory`, `GroupGraphics`, `SoftMask`,
      `BlendComposite`
  - Transparency groups, blend modes and soft masks are `PushGroup`/`PopGroup`
    and `SoftMaskedPaint`. This is the part Java gets from `Graphics2D` for
    free and the part a Go backend has to write.
  - `PDPatternContentStream` goes here too: `pdformcontentstream.go` says it is
    not ported because it names `PDTilingPattern`, which this task brings.
  - `textmarkuphandlers.go` says `generateNormalAppearance` for the squiggly
    annotation is not ported for the same reason. Close it or restate it.
- [ ] B5. `PDFToImage` and `PrintPDF`, and their rows out of
      `go/tools/notbuilt.go`
  - **B5 needs `track/imageio` merged.** `PDFToImage` writes through
    `ImageIOUtil`. Everything above this line does not.

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

And for this branch in particular:

- [ ] D8. This is a substitution, not a transliteration — say so plainly
  - Whatever was chosen in A0, it is not `java.awt.Graphics2D`. Record every
    case where it draws differently, in `STATUS.md`, as a deviation, and pin
    each one in a test so that a deviation which disappears fails as loudly as
    one that appears. `track/pdfbox-layout` did this and its
    `reference_test.go` is the worked example.

- [ ] D9. Close the deferrals, or say why not
  - Every "held for a raster backend" in `STATUS.md` and every
    `rendering.ErrNoBackend` path. Each one either runs now or has a new reason.

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

- [ ] B5, until `track/imageio` is merged. Nothing else in this file is
      blocked by anything outside it.
