# Implementation Plan

Track — deferrals whose reason no longer holds, and what was never recorded.

**Branch: `track/stale-deferrals`** — from and back to `migration-base`.

Depends on **nothing**. **Take it first**, for the reason `track/test-backfill`
was taken first: it is the only one of the four that can find a defect in work
already merged, rather than adding more work on top of it.

A deferral records what was missing on the day it was written. Nothing goes back
to look when that thing lands, so a port accumulates work that is blocked by
something which is no longer there. This branch is the sweep for that, and it
exists because an audit found four of them at once.

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

No new Java package. Everything here is a piece of an already-merged slice that
was put off, plus one class of test that was missed outright.

| Item | Recorded reason | Why it is stale |
| --- | --- | --- |
| `PDFTextStripper.fillBeadRectangles` | "PDThreadBead is a slice this port has not reached" | slice 8 ported `PDThreadBead`; the method still sets `beadRectangles = nil`, so every glyph falls into one article |
| `PDAbstractContentStream.shadingFill` | "it names PDShading, which belongs to the rendering this port has not reached" | slice 9 ported `PDShading`; `sh` cannot be written to a content stream |
| `PublicKeySecurityHandler.PrepareDocumentForEncryption` | "nothing can save a document until the writer lands in slice 7" | slice 7 merged. The real reason is a CMS encoder, which Go's standard library has not got — see A0 |
| `COSWriterCompressionPoolTest` | needs `PDDocumentOutline`, `PDOutlineItem` | both ported |
| `COSDocumentCompressionTest` | needs `PDAcroForm`, `PDComplexFileSpecification`, `PDPageContentStream`, `PDCheckBox`, `protect` | all five ported |
| `TestPDDocument`, 6 cases | **none — recorded nowhere at all** | it was missed, not deferred |
| `contentstream/operator/text` package comment | says `Tj`, `TJ`, `'` and `"` are not here | slice 3 ported all four; the comment is wrong |

`STATUS.md` carries the audit that found them and the commands to re-run it.

**What this branch is not.** It is not a licence to tidy. Every item above is
either behaviour the port does not have or a record that is false. A comment
that is merely terse is not in scope.

---

# Phase A — Write the tests

- [x] A0. **Decide what to do about public-key encryption.** Java builds a CMS
      enveloped-data blob per recipient through BouncyCastle. Go's standard
      library has `crypto/x509` and no CMS encoder, and there is no network to
      add one.
  - Either write the enveloped-data encoder this needs — it is a narrow subset
    of RFC 5652, one recipient info per certificate — or record the capability
    as permanently absent and correct the reason, which today names a slice
    that merged.
  - **Taken: write it here.** It is not the size the task feared. `rc2.go`
    implements `cipher.Block` -- it encrypts as well as it decrypts -- and
    `cms.go` declares every ASN.1 structure an enveloped-data blob is made of,
    because it reads one. What was missing was the direction, and that is
    `cmsencode.go`: 160 lines, no new dependency, no cgo. The recorded reason
    was stale twice over.

- [x] A1. Port `TestPDDocument` — 6 cases, and nothing in this repository has
      ever run them
- [x] A2. Port `COSWriterCompressionPoolTest` and `COSDocumentCompressionTest`
- [x] A3. Write the case for article beads. `PDFTextStripper` sorts by article
      when a page has thread beads; the Java corpus has pages that do
- [x] A4. Write the case for `shadingFill` — the `sh` operator, written and read
      back

---

# Phase B — Port the implementation

**Every function this phase touches needs a test that says it works.**

- [x] B1. `PDFTextStripper.fillBeadRectangles`, over the `PDThreadBead` that is
      now there
- [x] B2. `PDAbstractContentStream.shadingFill`, and `PDResources` adding a
      shading, which the same comment says it cannot
- [x] B3. Public-key encryption, as A0 decided
- [x] B4. Correct the `contentstream/operator/text` package comment, and every
      other comment this branch proves false

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
  - Which Java test cases were dropped, and is each one recorded with a reason?

- [x] D4. Check every function phase B touched has a test
  - Name the test that covers it. Not "the suite is green"
  - Where there is none, the function was changed on an argument rather than on
    evidence. Write the test, and take whatever it says

- [x] D5. Check every deferral is real and recorded
  - Every "not ported yet" in a doc comment — is it in `migration/STATUS.md`?
  - Every deferral — is it deferred because the type is absent, or because it
    was hard? The second is not a deferral.

- [x] D6. Check the Java bugs
  - Every bug found — is it in `migration/JAVA-BUGS.md` with where, what,
    what correct would be, where the Go carries it, and how confident?
  - Was any of them "fixed" on the way past? Revert it.

- [x] D7. Write the review down
  - What was checked, what was found, what was fixed, what is still open

And for this branch in particular:

- [x] D8. Re-run the audit, and leave it re-runnable
  - The three buckets over `src/main/java` and over `src/test/java`, then the
    two comment sweeps, then the step that pays: **check whether each stated
    reason is still true.** The commands are in `STATUS.md`.
  - Every deferral this branch leaves standing must name a reason that is true
    on the day the branch merges, not on the day it was written.

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


# Blocked

Nothing, unless A0 decides that public-key encryption needs a branch of its
own — in which case B3 is that branch's, not this one's, and this file records
the decision and moves on.
