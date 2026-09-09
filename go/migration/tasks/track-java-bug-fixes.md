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

That rule bought the migration its most valuable property: when the Go behaves
oddly, the answer is always "so does the Java, and here is the entry". It was
right for every branch that ported code.

It is the wrong rule for a port that is finished. `JAVA-BUGS.md` has 84 entries
and roughly seventy of them are carried in the Go on purpose — an `equals` that
truncates to 32 bits, a text extractor that reverses surrogate pairs, a merge
that drops the source's article threads and doubles the destination's. Nothing
downstream benefits from those any more.

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

**84 entries in `migration/JAVA-BUGS.md`.** A0 divides them; nothing else in
this file presumes the answer.

What is known before A0 starts, from the entries' own "Where the Go carries it"
lines:

| | Entries | What A0 does with them |
| --- | ---: | --- |
| The Go carries it | ~70 | judge each: fix or keep |
| The Go already does not carry it | ~12 | verify the claim still holds, and say so |
| The defect is in a Java **test** | 3 | the fix is in the Go test, not the Go code |

The three test-only ones are 4 (`TestCOSBase.testByteArrays` never checks the
lengths), 46 (`PDStreamTest` builds its stop filters from `COSName.toString()`)
and 78 (`GlyphLayoutDIN91379.pdf` was rendered from a string the test no longer
has).

**The entry numbers are the unit of work and they are stable.** Do not
renumber, do not compact, do not reorder. A later reader's only handle on any of
this is "JAVA-BUGS.md 47".

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
     PDFBox itself reads differently. Entry 43 (`PDSeedValue` writes strings and
     reads names) is the shape to watch for.
  2. **"Correct" is a judgement, not a fact.** Where the entry's own "what
     correct would be" is a guess about intent rather than a reading of the
     specification, there is nothing to fix *to*.
  3. **Fixing it is new functionality.** Entry 22 (`KerningTable` can never
     take its version 1 branch) is the example: making the branch reachable
     means implementing version 1 kerning, which is a port task and not a fix.
  4. **It is unobservable.** A dead branch, a computed value that is never
     used, a log line. Entry 11 (`parseHex` computes an offset and never uses
     it) is one. Record it as unobservable rather than fixing it, so the entry
     stays honest.

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

- [ ] E1. Stop and wait for the user's review. Do not start the next branch.

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

- [ ] E4. Report back
  - What was changed, what was not, and why for each

---

# What an entry looks like after a fix

The entry keeps every line it had. It gains one, after **Where the Go carries
it** and before **Confidence**:

```markdown
**Where the Go carries it** `go/pdfbox/cos/integer.go`, `Integer.Equals`, via
`Integer.IntValue`, which narrows through int32 so that Go reproduces Java's
(int) cast.

**Fixed in the Go** `track/java-bug-fixes`, entry 1. `Integer.Equals` compares
the int64 values, so `GetInteger(0)` and `GetInteger(4294967296)` are not equal
where the Java says they are. `Integer.IntValue` still narrows, because that is
`intValue()` and callers of it want Java's answer. Tested by
`TestIntegerEqualsDoesNotTruncate` in `go/pdfbox/cos/integer_test.go`.

**Confidence** high. ...
```

**Where the Go carries it** is left exactly as it was. It is the record of what
the port did while it was a port, and rewriting it into the past tense loses
the fact that the reproduction was deliberate.

An entry that A0 **keeps** gains a line too:

```markdown
**Kept in the Go** `track/java-bug-fixes` A0: unobservable. The offset is
computed and never read, so there is nothing a caller can tell apart.
```

---

# Blocked

Nothing. Every entry in `JAVA-BUGS.md` names code that is in the tree.
