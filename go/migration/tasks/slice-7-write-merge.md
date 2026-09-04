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
| `pdfbox/pdfwriter` | 3 | 5 |
| `pdfbox/pdfwriter/compress` | 4 | — |
| `pdfbox/multipdf` | 6 | 7 |
| `pdfbox/cos` — the update state slice 1 deferred | 4 | — |
| `pdfparser/PDFXRefStream` | 1 | — |

The four `cos` files are `COSUpdateInfo`, `COSUpdateState`, `COSDocumentState`
and `COSIncrement`. `migration/STATUS.md` records them as deferred to this
slice; they are the incremental-save machinery.

---

# Phase A — Write the tests

- [x] A1. `pdfbox/pdfwriter` — 2 of the 5 Java tests are portable now and are
      ported; `OperatorNameTest` in full, `COSWriterTest` for the 2 of its 4
      that do not need slice 8 or the network.
      `COSWriterCompressionPoolTest`, `COSDocumentCompressionTest` and
      `ContentStreamWriterTest` need slice 8 or a renderer. Each is named with
      its reason in `migration/STATUS.md`
- [x] A2. `pdfbox/multipdf` — `PageExtractorTest` ported in full. The other 6
      need `PDFMergerUtility`, `LayerUtility`, `PDPageContentStream` or a
      renderer, which slice 8 brings; each is named with its reason in
      `migration/STATUS.md`
- [x] A3. `pdmodel/font` — port `TestFontEmbedding` and `TestToUnicodeWriter`,
      which slice 3 deferred here because they write PDFs.
      `TestToUnicodeWriter` is ported in full, with `ToUnicodeWriter` beside it.
      `TestFontEmbedding` needs `PDPageContentStream` and `TestPDFToImage`
- [x] A4. **Close the slice 1 open debt.** `migration/STATUS.md` records it:
      the `accept()` tests in `cos` assert the visitor and a direct `WritePDF`
      call, because `COSWriter` did not exist. Now it does. Restore the byte
      assertions the Java tests make, in `boolean_test.go`, `integer_test.go`,
      `float_test.go` and `string_test.go`.

---

# Phase B — Port the implementation

- [x] B1. `cos` — the update state: `COSUpdateInfo`, `COSUpdateState`,
      `COSDocumentState`, `COSIncrement`, and the fields the slice 1 types
      left out
- [x] B2. `pdfbox/pdfwriter` — `COSWriter`, `COSStandardOutputStream`, and the
      4 files of `pdfwriter/compress`, which hold the object-stream
      compression pool
- [x] B3. `pdfparser/PDFXRefStream` — writing the cross-reference stream
- [x] B4. `pdfbox/multipdf` — `PageExtractor` and `PDFCloneUtility` in full,
      `Splitter` as far as slice 8 allows. `PDFMergerUtility`, `LayerUtility`
      and `Overlay` are deferred to slice 8; the reason for each, and the seven
      `Splitter` methods left out, are in `migration/STATUS.md`.
      The five Java names this line listed are `PDFMergerUtility`, `Splitter`,
      `PageExtractor`, `LayerUtility` and `Overlay`
- [x] B5. `PDDocument.save` and the incremental save path

---

# Phase C — Run and fix

- [x] C1. `gofmt -l .` clean
- [x] C2. `go vet ./...` clean
- [x] C3. `go test ./...` green — 35 packages with tests
- [x] C4. Record every Java bug found in `migration/JAVA-BUGS.md` — entry 33
- [x] C5. Update `migration/STATUS.md` — including removing the open-debt note
      in the slice 1 `cos` section, once A4 is done

---

# Phase D — Adversarial review

敵対的レビュー. Green tests prove the port passes the tests, not that it is a
faithful migration. Go in assuming it is wrong. Every check below is a question
the ported tests cannot answer.

- [x] D1. Read every ported file against its Java side by side
  - Is any method missing? Any branch of an `if`, any `case`, any `catch`?
  - Is any loop bound, any off-by-one, any `<` that should be `<=` different?
  - Java `int` narrows on cast and `float` saturates; Go does neither. Is every
    such conversion written out?

- [x] D2. Hunt for silently dropped behaviour
  - Anything Java does in a `finally` — is it still done on the Go error path?
  - Anything Java logs and swallows — does the Go swallow it too, or does it
    return an error the Java would not have?
  - Anything Java throws — is it an error, or a panic, and is that the right one?

- [x] D3. Check the tests are Java-derived, not Go-derived
  - For each assertion: is that value in the Java test, or did it come from
    running the Go? A value read off the port proves nothing.
  - Does each test take the real path, with the real types? A test over a
    stand-in can pass while the path it stands for is broken.
  - Which Java test cases were dropped, and is each one recorded with a reason?

- [x] D4. Check every deferral is real and recorded
  - Every "not ported yet" in a doc comment — is it in `migration/STATUS.md`?
  - Every deferral — is it deferred because the type is absent, or because it
    was hard? The second is not a deferral.

- [x] D5. Check the Java bugs
  - Every bug found — is it in `migration/JAVA-BUGS.md` with where, what,
    what correct would be, where the Go carries it, and how confident?
  - Was any of them "fixed" on the way past? Revert it.

- [x] D6. Write the review down
  - What was checked, what was found, what was fixed, what is still open

And for this branch in particular:

- [x] D7. Round-tripping proves nothing on its own
  - Writing with the port and reading with the port passes even when both are
    wrong. Compare the emitted bytes against what Java emits for the same
    document.

- [x] D8. Check the incremental save does not rewrite what it should not
  - The point of an incremental save is that the original bytes are untouched
    and the update is appended. Verify that, not just that the result opens.

- [x] D9. Check every deferral slice 1 made here was actually closed
  - `STATUS.md` names four `cos` files and one open debt. All five, or say why
    not.
  - All five are closed. `COSUpdateInfo`, `COSUpdateState`, `COSDocumentState`
    and `COSIncrement` are ported and wired in, and the `accept()` byte
    assertions are restored in `cos/accept_external_test.go`. The one thing not
    restored is `TestCOSFloat`'s `java.util.Random` sweep, which cannot be
    reproduced in Go without porting the generator; the reason and the sweep
    used instead are in `migration/STATUS.md`.

---

# Phase E — User feedback

- [x] E1. Stop and wait for the user's review. Do not start the next branch.

- [x] E2. For each item of feedback, judge it before acting — 4 items: 2 port
      defects (`COSDictionary.addAll` routed through `setItem`, the trailer /ID
      digest hashing UTF-8), 1 documentation defect (`SetTrailer`), 1 Java
      difference (`PageExtractor.Extract` panicking past the last page), which
      is JAVA-BUGS 34
  - Is it a port defect, a missing piece of scope, or a difference the Java
    itself has?
  - A Java difference is not fixed — it is recorded in `JAVA-BUGS.md` and the
    user is told why it stays.

- [x] E3. Where it needs fixing, write a **strict** test first —
      `TestDictionaryAddAllIsARawPut` and `TestDocumentIDDigestUsesISO88591`
      both failed before their fix; `TestExtractBeyondTheDocumentPanics` pins
      the Java bug that stays
  - Strict: it fails before the fix, takes the real path with the real types,
    and asserts what the Java does
  - Then fix the Go
  - Then `gofmt`, `go vet`, `go test ./...` again

- [x] E4. Report back
  - What was changed, what was not, and why for each

---

# Blocked

- [ ] Nothing outside this branch.
