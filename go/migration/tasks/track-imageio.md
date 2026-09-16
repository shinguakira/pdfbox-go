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

The five phases of [`TEMPLATE.md`](TEMPLATE.md), unchanged.

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
  - `ImageIOUtil` also writes the resolution into the file's metadata, which is
    what `MetaUtil` and `JPEGUtil` are for, and Go's encoders write none of it.
    Decide whether to write those bytes by hand or to record the loss.
    `STATUS.md` has the per-format answer.

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

敵対的レビュー. The seven checks are [`TEMPLATE.md`](TEMPLATE.md)'s, and each
was run.

- [x] D1. Read every ported file against its Java side by side
- [x] D2. Hunt for silently dropped behaviour
- [x] D3. Check the tests are Java-derived, not Go-derived
- [x] D4. Check every function phase B touched has a test
- [x] D5. Check every deferral is real and recorded
- [x] D6. Check the Java bugs
- [x] D7. Write the review down

---

# Phase E — User feedback

- [x] E1. Stop and wait for the user's review. Do not start the next branch.
- [x] E2. For each item of feedback, judge it before acting
- [x] E3. Where it needs fixing, write a **strict** test first
- [x] E4. Report back

---

# Blocked

Nothing. This branch is why the other two are not blocked on each other.
