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
| `pdmodel/interactive` — 8 subpackages | 144 | 35 |
| `pdmodel/documentinterchange` | 24 | — |
| `pdmodel/fdf` | 31 | — |
| `pdfparser/FDFParser` | 1 | — |
| `pdmodel/fixup` and `fixup/processor` | 8 | — |
| `pdmodel/graphics/optionalcontent` | 3 | — |
| `pdmodel/common` — what slice 2 left | ~10 of 16 | 1 |

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

- [x] B1. `pdmodel/common` — `COSArrayList`, `COSDictionaryMap`, `PDStream`,
      `PDMetadata`, `common/filespecification`. Also `PDNameTreeNode`,
      `PDNumberTreeNode`, `PDObjectStream`, `PDPageLabels` and
      `PDPageLabelRange`. `PDEmbeddedFile`s four date accessors wait for
      `DateConverter`, which is this slice's and is taken in dependency order
  - `migration/STATUS.md` records these as blocking `PDPage.getContentStreams`
    and `setContents` since slice 2
- [ ] B2. `interactive/form` — AcroForms and the field hierarchy
- [ ] B3. `interactive/annotation` — annotations and their appearance handlers
- [x] B4. `interactive/action`
- [ ] B5. `interactive/documentnavigation`, `interactive/pagenavigation`
- [ ] B6. `interactive/digitalsignature`
- [x] B7. `interactive/measurement`, `interactive/viewerpreferences`
  - both subpackages are done: the five viewer preference enums, and the
    measure, number format, rectlinear measure and viewport dictionaries
- [ ] B8. `documentinterchange` — logical structure, marked content, tagged PDF,
      prepress
- [x] B9. `pdmodel/graphics/optionalcontent` — 3 files, and `PDPropertyList`
      with them, since the two name each other. `rendering.RenderDestination`
      came too, because `getRenderState` takes one
  - `PDPropertyList` lives here, and `PDResources.getProperties` returns it.
    Slice 2 recorded that lookup as absent, and BDC and DP in
    `operator/markedcontent` cannot resolve a named property list without it.
- [ ] B10. `pdmodel/fdf` — 31 files, and `pdfparser/FDFParser`
  - Forms Data Format: the import and export half of AcroForms
- [ ] B11. `pdmodel/fixup` and `fixup/processor` — 8 files
  - The document fixups AcroForm reading applies before it trusts a file
- [ ] B12. Close the `PDPage` holes slice 2 left: annotations, thread beads,
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

- [x] Appearance generation for form fields and annotations draws content
      streams, which needs the writer from slice 7. Decide whether this branch
      ports only the reading half, or waits.

      **Decided: the writing half is in scope, and it pulls three files in with
      it.** Slice 7 is merged, so the question is no longer whether to wait.
      What appearance generation actually draws through is
      `PDAppearanceContentStream`, which extends `PDPageContentStream`, which
      extends `PDAbstractContentStream` --- all three in top-level `pdmodel`,
      and none of them named in any slice's scope table. They are ported here
      because this is the slice that needs them; `AppearanceGeneratorHelper`
      cannot exist without them.

      `PDAbstractContentStream` reaches `PDFormXObject`, `PDShading` and
      `PDPattern`, which are slice 9's. Those methods are the ones left out,
      named where they occur and in `migration/STATUS.md`; the text, colour,
      path and image methods appearance generation uses are all here.
