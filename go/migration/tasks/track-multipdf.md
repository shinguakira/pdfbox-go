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

The five phases of [`TEMPLATE.md`](TEMPLATE.md), unchanged.

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

**How it was missed.** Slice 7 deferred all three to slice 8 and slice 8 never
took them; `STATUS.md`'s "The 891-class survey, and why it was replaced" has
why the coverage survey then counted them as done. Nothing about the work is
hard or blocked — it simply had no branch.

`PDFCloneUtility` is the piece the other three are written against, and it is
in. Read it first.

---

# Phase A — Write the tests

- [x] A1. Port `PDFMergerUtilityTest` — the biggest of the five, and the one
      that carries the merged-document corpus
- [x] A2. Port `MergeAcroFormsTest` and `MergeAnnotationsTest`
  - Both are about what merging does to slice 8's structures: field names that
    collide, annotation appearance streams that move. Slice 8 is merged, so
    there is something to assert against.
- [x] A3. Port `OverlayTest`
- [x] A4. Port `TestLayerUtility`
  - It renders to compare, which needs `track/raster`. Port what it asserts
    about the object graph and record the pixel half.

- [x] A5. Port `PDFCloneUtilityTest`
  - `PDFCloneUtility` was ported by slice 7 and its test was not: its three
    cases needed `PDPageContentStream`, `PDOptionalContentProperties` and
    `PDFMergerUtility`, and only the last was still missing when this branch
    opened, so the whole class can go in. **Port it before B1** -- it tests the
    machinery the other three classes are written against.

---

# Phase B — Port the implementation

**Every function this phase touches needs a test that says it works.**

- [x] B1. `PDFMergerUtility` — the destination document, the source list, and
      what it does with each of the catalog's dictionaries
- [x] B2. `Overlay`, whose `Position` enum the command takes as an option
- [x] B3. `LayerUtility`
- [x] B4. `PDFMerger` and `OverlayPDF`, and their rows out of
      `go/tools/notbuilt.go`

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

- [ ] E1. Stop and wait for the user's review. Do not start the next branch.
- [ ] E2. For each item of feedback, judge it before acting
- [ ] E3. Where it needs fixing, write a **strict** test first
- [ ] E4. Report back

---

# Blocked

- [x] `TestLayerUtility` and any other case that compares rendered pages. Held
      for `track/raster`; port the rest of the class and record the omission
      rather than skipping the file.

  `TestLayerUtility` turned out to have no rendering half at all: it asserts
  about the object graph and all of it is ported. `OverlayTest` and
  `checkMergeIdentical` are the ones that render; `STATUS.md` says what the port
  compares instead and what is still open there.
