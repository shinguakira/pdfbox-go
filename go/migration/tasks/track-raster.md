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

The five phases of [`TEMPLATE.md`](TEMPLATE.md), unchanged.

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
      and slice 9 took a fourth; the three are still the three, and
      [`../RASTER-PRECEDENT.md`](../RASTER-PRECEDENT.md) is what each of them
      costs and what other ecosystems did with the same problem.
  - Whatever is chosen has to get into `go.mod` with no network, so settle that
    before choosing it.
  - Whatever is chosen, `awt/geom.Area` **flattens curves to polylines** where
    the JDK intersects them exactly. That is a recorded approximation and it is
    what the clip is built from. Read `awt/geom/area.go`'s head comment before
    A0, and decide whether the choice makes it better or has to live with it.
  - Record the decision in `STATUS.md` before A1, the way `track/pdfbox-layout`
    recorded its A0.

- [x] A1. Port the pixel comparisons the earlier branches deferred
  - They are the tests this branch exists to make possible. `STATUS.md` records
    them per slice: `ContentStreamWriterTest`, `TestFontEmbedding`,
    `PDAcroFormFlattenTest`, `TestLayerUtility`, `TestImageIOUtils` and every
    `checkRenderIdent` in `pdfbox-layout-awt` and `pdfbox-layout-fop`.
  - `PDAcroFormFlattenTest` also downloads its PDFs, which is a second reason
    and does not go away with this branch. Record what is left.
  - Do this **before** B, not after. A backend written first and compared
    afterwards is a backend written to whatever it happens to produce.

- [x] A2. Port the shading tests
  - `PDShadingTest` and the type-specific cases assert colours at points, which
    is what a `ShadingContext` answers. They can be asserted without a full
    page render.

---

# Phase B — Port the implementation

**Every function this phase touches needs a test that says it works.**

Ordered so that each step is testable before the next needs it.

- [x] B1. The backend itself: fill, stroke, clip, transform, over a solid paint
  - `Fill`, `Draw`, `SetClip`, `SetTransform`, `SetStroke`, `SetPaint` for
    `ColorPaint`, `SetAntiAliasing`. A page of black text on white is the first
    thing that should come out.
- [x] B2. `DrawImage` and `DrawStencil`, with both interpolations
  - `PDImage.Image()` already answers the pixels; this is placement and
    sampling.
- [x] B3. The shading contexts and paints, type by type
  - 1, 2 and 3 first — function, axial, radial. Then the mesh types 4 to 7,
    which share `TriangleBasedShadingContext` and `PatchMeshesShadingContext`.
- [x] B4. `TilingPaint`, `TilingPaintFactory`, `GroupGraphics`, `SoftMask`,
      `BlendComposite`
  - Transparency groups, blend modes and soft masks are `PushGroup`/`PopGroup`
    and `SoftMaskedPaint`. This is the part Java gets from `Graphics2D` for
    free and the part a Go backend has to write.
  - `PDPatternContentStream` goes here too, and the squiggly annotation's
    `generateNormalAppearance` with it: both were deferred for want of
    `PDTilingPattern`, which this task brings. Close them or restate the
    reason. `STATUS.md`'s "What this closed elsewhere" is where the answer goes.
- [x] B5. `PDFToImage` and `PrintPDF`, and their rows out of
      `go/tools/notbuilt.go`
  - **B5 needs `track/imageio` merged.** `PDFToImage` writes through
    `ImageIOUtil`. Everything above this line does not.

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

敵対的レビュー. The seven checks are [`TEMPLATE.md`](TEMPLATE.md)'s, and each
was run.

- [x] D1. Read every ported file against its Java side by side
- [x] D2. Hunt for silently dropped behaviour
- [x] D3. Check the tests are Java-derived, not Go-derived
- [x] D4. Check every function phase B touched has a test
- [x] D5. Check every deferral is real and recorded
- [x] D6. Check the Java bugs
- [x] D7. Write the review down

And for this branch in particular:

- [x] D8. This is a substitution, not a transliteration — say so plainly
  - Whatever was chosen in A0, it is not `java.awt.Graphics2D`. Record every
    case where it draws differently, in `STATUS.md`, as a deviation, and pin
    each one in a test so that a deviation which disappears fails as loudly as
    one that appears. `track/pdfbox-layout` did this and its
    `reference_test.go` is the worked example.

- [x] D9. Close the deferrals, or say why not
  - Every "held for a raster backend" in `STATUS.md` and every
    `rendering.ErrNoBackend` path. Each one either runs now or has a new reason.

---

# Phase E — User feedback

- [ ] E1. Stop and wait for the user's review. Do not start the next branch.
- [x] E2. For each item of feedback, judge it before acting
- [x] E3. Where it needs fixing, write a **strict** test first
- [ ] E4. Report back

---

# Blocked

- [x] B5 — `track/imageio` merged, so it was not blocked. Nothing else in this file is
      blocked by anything outside it.
