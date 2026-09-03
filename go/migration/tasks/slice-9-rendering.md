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
- **Do not stop while work in this branch's scope remains.**

## How each unit of work runs

Five phases, in this order, never overlapping. See
[`TEMPLATE.md`](TEMPLATE.md) for what each phase means.

A — write the test · B — port the implementation · C — run and fix ·
D — adversarial review · E — user feedback

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

敵対的レビュー. See [`TEMPLATE.md`](TEMPLATE.md) for D1–D6. Additionally:

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

See [`TEMPLATE.md`](TEMPLATE.md) for E1–E4.

---

# Blocked

- [ ] B0. The raster backend decision blocks everything from B6 onward.
      `PLAN.md` says to take it before starting, not during.
