# Implementation Plan

Slice 8 — forms, annotations, interactive features.

**Branch: `slice/8-<name>`** — from and back to `migration-base`.
Depends on `slice/1` only. Independent of slices 2, 3, 4, 5, 6 and 7.

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
| `pdmodel/interactive` — 8 subpackages | 144 | 35 |
| `pdmodel/documentinterchange` | 24 | — |

The largest slice by file count. `PLAN.md` notes each subtree is independent of
the others, so the eight subpackages of `interactive` can be taken one at a
time and each finished before the next starts.

Subpackages: `action`, `annotation`, `digitalsignature`,
`documentnavigation`, `form`, `measurement`, `pagenavigation`,
`viewerpreferences`.

---

# Phase A — Write the tests

Take one subpackage at a time. For each: port its Java tests first, then its
implementation, then move on. Do not open all eight at once.

- [ ] A1. `interactive/form` — port its Java tests
- [ ] A2. `interactive/annotation` — port its Java tests
- [ ] A3. `interactive/action` — port its Java tests
- [ ] A4. `interactive/documentnavigation` — port its Java tests
- [ ] A5. `interactive/pagenavigation` — port its Java tests
- [ ] A6. `interactive/digitalsignature` — port its Java tests
- [ ] A7. `interactive/measurement`, `interactive/viewerpreferences`
- [ ] A8. `documentinterchange` — port its Java tests
- [ ] A9. `pdmodel/common` — port `COSArrayListTest`, which slice 2 could not
      port because it needs annotations

---

# Phase B — Port the implementation

- [ ] B1. `pdmodel/common` — `COSArrayList`, `COSDictionaryMap`, `PDStream`,
      `PDMetadata`, `common/filespecification`
  - `migration/STATUS.md` records these as blocking `PDPage.getContentStreams`
    and `setContents` since slice 2
- [ ] B2. `interactive/form` — AcroForms and the field hierarchy
- [ ] B3. `interactive/annotation` — annotations and their appearance handlers
- [ ] B4. `interactive/action`
- [ ] B5. `interactive/documentnavigation`, `interactive/pagenavigation`
- [ ] B6. `interactive/digitalsignature`
- [ ] B7. `interactive/measurement`, `interactive/viewerpreferences`
- [ ] B8. `documentinterchange` — logical structure, marked content, tagged PDF,
      prepress
- [ ] B9. Close the `PDPage` holes slice 2 left: annotations, thread beads,
      transitions, additional actions, viewports, metadata

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md` — including the slice 2 `PDPage` and
      `pdmodel/common` rows this slice closes

---

# Phase D — Adversarial review

敵対的レビュー. See [`TEMPLATE.md`](TEMPLATE.md) for D1–D6. Additionally:

- [ ] D7. This slice is mostly dictionary wrappers — check the keys
  - A wrong `COSName` constant compiles, passes a shallow test, and reads the
    wrong entry from every real file. Check each key against the Java, not
    against the specification.

- [ ] D8. Check the `COSArrayList` semantics
  - It syncs a Go slice to a `COSArray`. Java's is a `List` view with a
    filtered mode that refuses writes. Whatever the Go does instead, check that
    a filtered list still refuses.

- [ ] D9. Digital signatures verify or they do not
  - A signature path that "mostly works" is worse than one that is absent.

---

# Phase E — User feedback

See [`TEMPLATE.md`](TEMPLATE.md) for E1–E4.

---

# Blocked

- [ ] Appearance generation for form fields and annotations draws content
      streams, which needs the writer from slice 7. Decide whether this branch
      ports only the reading half, or waits.
