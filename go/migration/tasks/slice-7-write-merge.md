# Implementation Plan

Slice 7 — write and manipulate. The first slice that produces a PDF rather than
consuming one.

**Branch: `slice/7-<name>`** — from and back to `migration-base`.
Depends on `slice/1` only. Independent of slices 2, 3, 4, 5, 6 and 8.

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
| `pdfbox/pdfwriter` | 3 | 5 |
| `pdfbox/multipdf` | 6 | 7 |
| `pdfbox/cos` — the update state slice 1 deferred | 4 | — |
| `pdfparser/PDFXRefStream` | 1 | — |

The four `cos` files are `COSUpdateInfo`, `COSUpdateState`, `COSDocumentState`
and `COSIncrement`. `migration/STATUS.md` records them as deferred to this
slice; they are the incremental-save machinery.

---

# Phase A — Write the tests

- [ ] A1. `pdfbox/pdfwriter` — port all 5 Java tests
- [ ] A2. `pdfbox/multipdf` — port all 7 Java tests
- [ ] A3. `pdmodel/font` — port `TestFontEmbedding` and `TestToUnicodeWriter`,
      which slice 3 deferred here because they write PDFs
- [ ] A4. **Close the slice 1 open debt.** `migration/STATUS.md` records it:
      the `accept()` tests in `cos` assert the visitor and a direct `WritePDF`
      call, because `COSWriter` did not exist. Now it does. Restore the byte
      assertions the Java tests make, in `boolean_test.go`, `integer_test.go`,
      `float_test.go` and `string_test.go`.

---

# Phase B — Port the implementation

- [ ] B1. `cos` — the update state: `COSUpdateInfo`, `COSUpdateState`,
      `COSDocumentState`, `COSIncrement`, and the fields the slice 1 types
      left out
- [ ] B2. `pdfbox/pdfwriter` — `COSWriter`, `COSStandardOutputStream`,
      `COSWriterCompressionPool`
- [ ] B3. `pdfparser/PDFXRefStream` — writing the cross-reference stream
- [ ] B4. `pdfbox/multipdf` — `PDFMergerUtility`, `Splitter`, `PageExtractor`,
      `LayerUtility`, `Overlay`
- [ ] B5. `PDDocument.save` and the incremental save path

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md` — including removing the open-debt note
      in the slice 1 `cos` section, once A4 is done

---

# Phase D — Adversarial review

敵対的レビュー. See [`TEMPLATE.md`](TEMPLATE.md) for D1–D6. Additionally:

- [ ] D7. Round-tripping proves nothing on its own
  - Writing with the port and reading with the port passes even when both are
    wrong. Compare the emitted bytes against what Java emits for the same
    document.

- [ ] D8. Check the incremental save does not rewrite what it should not
  - The point of an incremental save is that the original bytes are untouched
    and the update is appended. Verify that, not just that the result opens.

- [ ] D9. Check every deferral slice 1 made here was actually closed
  - `STATUS.md` names four `cos` files and one open debt. All five, or say why
    not.

---

# Phase E — User feedback

See [`TEMPLATE.md`](TEMPLATE.md) for E1–E4.

---

# Blocked

- [ ] Nothing outside this branch.
