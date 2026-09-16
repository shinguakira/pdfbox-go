# Implementation Plan

Track — the Java bugs. Fixing them **in the Go**, one at a time.

**Branch: `track/java-bug-fixes`** — from and back to `migration-base`.

Depends on nothing and must be taken **last**. Every branch before it adds
entries to the file this one works from, and this is the only branch in the
migration that makes the Go deliberately behave differently from the Java. Doing
it while porting continues would make every later "is this a port defect or is
it the Java?" question harder to answer than it already is.

## This branch inverts one rule, and only one

Every other task file says:

> **NEVER fix a bug that is in the Java.** Port it as written, comment it where
> it occurs, and record it in `migration/JAVA-BUGS.md`.

`JAVA-BUGS.md`'s own header says what that rule bought and why every branch
that ported code kept it.

It is the wrong rule for a port that is finished. Most of the entries
`JAVA-BUGS.md` held when this branch opened were carried in the Go on purpose,
and nothing downstream benefits from those any more.

**So this branch fixes them in the Go.** The rule is suspended for the entries
this branch takes, and for nothing else.

## Rules — do not break these

- **NEVER change the Java.** No `.java`, no `pom.xml`, no test resource, for any
  reason. It is still the reference; a fixed Go is judged against it, and a
  reference that gets edited stops being one. **This branch changes Go and
  documentation. Nothing else.**
- **NEVER delete an entry from `migration/JAVA-BUGS.md`.** A fixed bug is still
  a bug in the Java, and the entry is still the only record that it was found.
  A fixed entry *gains* a line; it does not go away. See "What an entry looks
  like after a fix" below.
- **NEVER fix a bug this branch's A0 put in the "keep" column** without
  reopening A0 and saying why the judgement changed. The point of the triage is that
  it is made once, in writing, before any code moves.
- **One entry, one commit.** Every fix is its own commit, named by its entry
  number, so that any single one can be reverted without the rest. A commit that
  fixes three entries is not reviewable.
- **NEVER commit to `migration-base` directly.**
- **NEVER touch `apache/pdfbox`** — no PR, no pull, fetch, merge or rebase.
- **DO NOT STOP UNTIL PHASE E.** Phases A to D run end to end. Finishing a task
  is not a stopping point; neither is finishing a phase, an entry, or a commit.
  Do not pause to report progress as if it were a result, do not ask whether to
  continue, and do not end a turn with a list of what is left. **E1 is the only
  stop in this file** — that is where the user reviews. Only the user stops the
  work before it.

## How each unit of work runs

Five phases, in this order, never overlapping. A "unit of work" here is **one
entry of `JAVA-BUGS.md`**, not one package.

**A — triage, then write the test.** Decide whether the entry is fixed at all
(A0, once, for all of them). For each entry that is, write a Go test that
asserts the **correct** behaviour and therefore fails. Its expected value does
not come from the Java — that is the whole point — so it comes from the
specification, from arithmetic, or from the entry's own "what correct would be",
and the test says which in a comment.

**B — fix the Go.** Change the smallest amount of code that makes the test pass.
Leave a comment at the site saying what the Java does and that the divergence is
deliberate, pointing at the entry number.

**C — run and fix.** `gofmt -l . && go vet ./... && go test ./...`. **A ported
Java test that now fails is the interesting case** — see "When a ported test
fails" below. It is not automatically wrong and it is not automatically right.

**D — adversarial review.** Green tests are not evidence the fix is right. Go in
assuming it is wrong, and assuming in particular that it is *wider* than the
entry described.

**E — user feedback.** Stop. Wait. Judge each item, and where it is a real
defect, write a strict failing test first and only then fix.

## Scope

**The 84 entries `migration/JAVA-BUGS.md` held when this branch opened.** A0
divides them; nothing else in this file presumes the answer. Entries added
after it are not this branch's, and 85 and 86 came back to it in a second pass.

What is known before A0 starts, from the entries' own "Where the Go carries it"
lines:

| | What A0 does with them |
| --- | --- |
| The Go carries it | judge each: fix or keep |
| The Go already does not carry it | verify the claim still holds, and say so |
| The defect is in a Java **test** | the fix is in the Go test, not the Go code |

The test-only ones this branch opened with are 4, 46 and 78;
`JAVA-BUGS.md`'s "How they group" collects them and each entry says what its
defect is. `STATUS.md` has the count each column ended with.

**The entry numbers are the unit of work.** `JAVA-BUGS.md` says why they never
move.

---

# Phase A — Triage, then write the tests

- [x] A0. **Triage all 84 entries, in writing, before any code moves.** Put the
      table in `STATUS.md`. Every entry lands in exactly one column:

  - **Fix** — the Go carries it, what correct would be is a fact rather than a
    judgement, and a caller can tell the difference. This is the default.
  - **Keep** — one of the four reasons below, named per entry. Not "it looked
    risky".
  - **Not carried** — the entry already says so. Verify it: the claim was true
    when it was written and the code has moved since.
  - **Test only** — the defect is in a Java test; the Go test is what changes.

  The four reasons an entry may be **kept**, and there are no others:

  1. **A reader depends on it.** The bug is in something the port *writes*, and
     files written by PDFBox for twenty years carry it, so readers have been
     built to expect it. Changing the writer makes the port produce files that
     PDFBox itself reads differently. Entry 43 is the shape to watch for.
  2. **"Correct" is a judgement, not a fact.** Where the entry's own "what
     correct would be" is a guess about intent rather than a reading of the
     specification, there is nothing to fix *to*.
  3. **Fixing it is new functionality.** Entry 22 is the example: making the
     branch reachable means implementing a kerning table format, which is a
     port task and not a fix.
  4. **It is unobservable.** A dead branch, a computed value that is never
     used, a log line. Entry 11 is one. Record it as unobservable rather than
     fixing it, so the entry stays honest.

  Record the count in each column and the reason for every **keep**. A0 is the
  document this branch is judged on; the code is downstream of it.

- [x] A1. For every entry in the **fix** column, write the failing test.
  - It asserts what correct is, and says in a comment where that value came
    from: the specification, the arithmetic, or the entry's own analysis.
  - It fails before the fix. Run it and see it fail. A test that passes before
    the fix is testing something else.
  - Where the Java has a test over the same code, the Go test sits beside it and
    the two disagree on purpose. Say so in both.

- [x] A2. For the **test only** entries, make the Go test stricter.
  - The Java test is weak; the Go one does not have to be. Assert what the Java
    forgot to.

---

# Phase B — Fix the Go

**Every entry fixed in this phase needs a test from A1 that failed without it.**

- [x] B1. Fix them, one entry per commit, in entry order.
  - The commit message names the entry: `pdfbox: JAVA-BUGS 47, ...`.
  - The comment at the site says what the Java does, that the Go does not, and
    the entry number. A future reader comparing the two files needs to find
    that sentence without leaving the file.

- [x] B2. Update the entry in `migration/JAVA-BUGS.md` in the same commit.
      See below for what an entry looks like afterwards.

- [x] B3. Where a fix changes what the port **writes** into a PDF rather than
      what it reads, say so in the entry. That is the half of a fix that can
      reach someone else's software.

---

# Phase C — Run and fix

- [x] C1. `gofmt -l .` clean
- [x] C2. `go vet ./...` clean
- [x] C3. `go test ./...` green
- [x] C4. Update `migration/JAVA-BUGS.md`: every fixed entry carries its new
      line, and the file's header says what the file now means
- [x] C5. Update `migration/STATUS.md` with the A0 table and its outcome

## When a ported test fails

A test ported from the Java asserts what the Java does. Some of them assert the
bug. When one goes red after a fix, there are three possibilities and they are
told apart by reading, not by guessing:

1. **The test asserts the bug.** Then the expected value changes, and this is
   the one place in the whole migration where an assertion value is not the
   Java's. Keep the Java's value in the comment beside the new one, with the
   entry number, so a reader can see both and why they differ. Do not delete
   the old value.
2. **The fix is wider than the entry.** The failing test is about something the
   entry did not mention, which means the change reached further than it was
   meant to. Narrow the fix.
3. **The fix is wrong.** The entry's "what correct would be" was itself wrong.
   Then the entry moves to the **keep** column with the reason, and A0's table
   is corrected. A0 being wrong about one entry is a normal outcome; hiding it
   is not.

**A knock-on failure is never resolved by loosening the failing test.**

---

# Phase D — Adversarial review

敵対的レビュー. Green tests prove the fixes pass the tests, not that they are
right. Go in assuming each one is wrong, and in particular that it is wider than
the entry it came from.

- [x] D1. Read every fix against the Java it diverges from
  - Is the divergence exactly what the entry described, and nothing more?
  - Does the comment at the site say what the Java does, and the entry number?
  - Would a reader who knows only the Java understand why the two differ?

- [x] D2. Hunt for what the fix reached that it should not have
  - Who else calls the function that changed? A fix to `equals` reaches every
    `indexOf` in the tree.
  - Does the fix change what the port **writes**, and is that said?
  - Is there a caller that was compensating for the bug, which now
    double-corrects?

- [x] D3. Check every expected value that changed
  - For each ported test whose expectation moved: is the new value derived from
    the specification or the arithmetic, and does the comment say which?
  - Is the Java's old value still there to be read?

- [x] D4. Check every fix has a test that failed without it
  - Name the test. Not "the suite is green"
  - Re-run each one with the fix reverted and confirm it fails. A fix whose
    test passes without it fixed nothing

- [x] D5. Check the **keep** column
  - Every kept entry: is its reason one of the four, and is it written down?
  - Has any of them been fixed by accident, as a side effect of another fix?

- [x] D6. Check `JAVA-BUGS.md` is still true
  - Every entry, fixed or kept, still describes the Java correctly
  - Every fixed entry says where the Go now differs
  - No entry was deleted

- [x] D7. Write the review down
  - What was checked, what was found, what was fixed, what is still open

---

# Phase E — User feedback

- [x] E1. Stop and wait for the user's review. Do not start the next branch.

- [x] E2. For each item of feedback, judge it before acting
  - Is it a defect in the fix, a fix that should not have been made, or an
    entry that should have been fixed and was not?
  - A fix the user rejects goes back to the **keep** column with the reason,
    and the entry says so.

- [x] E3. Where it needs fixing, write a **strict** test first
  - Strict: it fails before the fix, takes the real path with the real types,
    and asserts what correct is
  - Then fix the Go
  - Then `gofmt`, `go vet`, `go test ./...` again

- [x] E4. Report back
  - What was changed, what was not, and why for each

---

# What an entry looks like after a fix

The entry keeps every line it had, **Where the Go carries it** included —
`JAVA-BUGS.md`'s header says why that line is never rewritten. It gains one
line, after that one and before **Confidence**:

- **Fixed in the Go** — this branch, the entry number, what the Go does instead
  of what the Java does, what was deliberately left narrowing or unchanged, and
  the test that pins it. Entry 1 is the worked example.
- **Kept in the Go** — this branch, and which one of the four reasons above,
  with what makes it that one. Entry 11 is the worked example.

---

# Blocked

Nothing. Every entry in `JAVA-BUGS.md` names code that is in the tree.
