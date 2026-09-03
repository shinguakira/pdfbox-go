# Implementation Plan

`tools` — the command-line utilities.

**Branch: not decided, and no slice claims this work.**

`migration/PLAN.md` counts `tools` in scope — 26 files, 5.0k lines — and
deliberately leaves it out of the out-of-scope list beside `debugger`,
`debugger-app`, `examples`, `benchmark`, `app` and `parent`. It is then never
mentioned again: no slice covers it, `BRANCHING.md` gives it no branch or track,
and the only home it has is the row `STATUS.md` invented for it, "phase 7
`cmd/pdfbox`", which the plan does not say.

**This file exists to make that gap visible, not to close it.** Do not create a
branch for it and do not add one to the plan. The user decides.

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

## The decision to take first

- [ ] Where does `tools` live?
  - Its own slice, appended to the plan? That is the user's call, not the
    port's.
  - Split across the slices that give each command its library? `ExtractText`
    belongs with slice 3, `Encrypt` and `Decrypt` with slice 5, `PDFMerger` and
    `PDFSplit` with slice 7, `PDFToImage` with slice 9.
  - A track of its own, once the libraries under it exist?
  - Or out of scope after all, which the plan currently says it is not.

Until that is answered, the tasks below are a survey, not a plan.

## Scope

| Java module | Main files | Java tests |
| --- | ---: | ---: |
| `tools` | 26 | 6 |

The 22 top-level commands, and which slice gives each one its library:

| Command | Needs |
| --- | --- |
| `Version`, `PDFBox` (the dispatcher) | nothing |
| `DecompressObjectstreams`, `WriteDecodedDoc` | slice 1 |
| `ExtractText`, `PDFText2HTML`, `PDFText2Markdown` | slice 3 |
| `Decrypt`, `Encrypt` | slice 5 |
| `ExtractImages`, `ImageToPDF` | slice 6 |
| `PDFMerger`, `PDFSplit`, `OverlayPDF`, `TextToPDF` | slice 7 |
| `ExportFDF`, `ImportFDF`, `ExportXFDF`, `ImportXFDF` | slice 8 |
| `PDFToImage`, `PrintPDF` | slice 9 |
| `ExtractXMP` | `track/xmpbox` |

Reading that table is most of the argument: `tools` is not one unit of work.
Every command lands naturally in the slice that gives it its library, which is
why no single slice claims it.

## If it becomes a slice

The five phases apply unchanged — see [`TEMPLATE.md`](TEMPLATE.md).

Two things are specific to this module:

- **The command-line surface is not Java's.** `picocli` has no Go equivalent
  worth transliterating; `flag` or a Go CLI library is a substitution, and the
  flags and their names are the compatibility surface that matters.
- **The 6 Java tests are end-to-end.** They run a command against a fixture and
  compare output. They are the best evidence in the project that the library
  underneath actually works, and the worst evidence about the tool itself.
