# Implementation Plan

<!--
Template. Copy to `slice-N-<name>.md` or `track-<name>.md` and fill in.
One file per branch. Nothing here is migration-wide; the migration-wide
documents are PLAN.md, BRANCHING.md, STATUS.md and JAVA-BUGS.md.

Do not change this template to suit one branch. Change the copy.
-->

Slice N — <what the slice delivers, from `migration/PLAN.md`>.

**Branch: `slice/N-<name>`** — from and back to `migration-base`.

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

<Java packages this branch covers, with file counts. Say what is explicitly
left to a later slice and which slice.>

---

# Phase A — Write the tests

<One task per package. Name the Java test files. Where Java has no test, say so
and write from source.>

- [ ] A1.

---

# Phase B — Port the implementation

<One task per package, in dependency order. Name the Java classes.>

- [ ] B1.

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found on the way in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md` — this slice's section, and any rows an
      earlier slice left deferred that this one closes

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

---

# Phase E — User feedback

- [ ] E1. Stop and wait for the user's review. Do not start the next branch.

- [ ] E2. For each item of feedback, judge it before acting
  - Is it a port defect, a missing piece of scope, or a difference the Java
    itself has?
  - A Java difference is not fixed — it is recorded in `JAVA-BUGS.md` and the
    user is told why it stays.

- [ ] E3. Where it needs fixing, write a **strict** test first

  Strict means all four. A test that misses any of them is why the defect got
  past the first time.

  - **It fails before the fix.** Run it against the unfixed code and watch it
    fail, for the reason the feedback names. A test written after the fix
    proves only that the fix is self-consistent.
  - **It exercises the real path, with the real types.** The slice 2 review
    found a broken `COSStream.createView` that an existing passing test
    covered — the test built its stream over a `ReadBuffer`, so it never took
    the path that was wrong.
  - **It asserts what the Java does, not what the Go does.** Read the Java and
    copy the value. If the two differ, the Java wins.
  - **It stays.** The test is the record that this was once wrong. Do not fold
    it into a broader one afterwards.

  Then fix the Go. Then `gofmt`, `go vet`, `go test ./...` again.

  If the feedback turns out to name Java behaviour rather than a port defect,
  there is no fix: write the test that pins the Java behaviour, add the
  `JAVA-BUGS.md` entry, and say so in E4.

- [ ] E4. Report back
  - What was changed, what was not, and why for each

---

# Blocked

<Anything this branch cannot do until something outside it exists. Name what is
missing and which task it blocks. Delete the section if nothing is blocked.>
