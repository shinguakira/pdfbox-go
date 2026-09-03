# Implementation Plan

Track — `pdfbox-layout-*`. Glyph layout, one interface against two backends.

**Branch: `track/<name>`** — from and back to `migration-base`.
**The branch is not decided.** `migration/PLAN.md` gives this a "Parallel track"
section; `migration/BRANCHING.md`'s track table lists only `track/xmpbox` and
`track/scratchfile`, so no branch row exists for it. Settle that before
branching. Do not add a row to the plan without the user saying so.

Depends on `slice/4` — it needs fonts to shape. `PLAN.md` says it is worth
reading before `slice/9` for its backend-interface shape.

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

`PLAN.md` counts 5 main files across two Maven modules.

| Java module | Main files | Java tests |
| --- | ---: | ---: |
| `pdfbox-layout-awt` | 3 | 7 |
| `pdfbox-layout-fop` | 4 | 6 |

Java package is `org.apache.pdfbox.glyphlayout.*`;
`migration/mapping/packages.tsv` maps it to `pdfbox/glyphlayout/awt`.

Two of the main files are examples — `GlyphLayoutHelloWorldAWT` and
`GlyphLayoutHelloWorldFOP`. `PLAN.md` puts `examples` out of scope; decide
whether these two count, since they sit inside an in-scope module.

**The AWT backend is `java.awt.font.TextLayout` and the FOP backend is Apache
FOP.** Go has neither. This track is the clearest case in the project where a
port means choosing a Go equivalent rather than transliterating, and that
choice is the work — not the 7 files around it.

---

# Phase A — Write the tests

- [ ] A0. **Decide what the Go backend is** before writing any test. The tests
      assert shaped glyph runs; without a shaper there is nothing to assert
      against.
- [ ] A1. Port the shared cases both backends run
  - `GlyphLayoutBidiTest`, `GlyphLayoutDin91379Test`,
    `GlyphLayoutDin91379FormTest`, `GlyphLayoutLigaturesAndKerningTest`,
    `GlyphLayoutSMPTest` — each exists twice, once per backend
- [ ] A2. Port `TestBase` — the AWT side's shared fixture
- [ ] A3. Port the hello-world tests, if A0 leaves them meaningful

---

# Phase B — Port the implementation

- [ ] B1. The interface both backends implement
  - `GlyphLayoutProcessor` and `GlyphLayoutFontLoader` in the core, whichever
    slice ported them, and the contract they define
- [ ] B2. One backend, chosen in A0
- [ ] B3. `FopStringTextFragment` and whatever the second backend needs, if a
      second backend is in scope at all

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md`

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

- [ ] D7. This is a substitution, not a transliteration — say so plainly
  - Whatever Go shaper was chosen, it is not `java.awt.font.TextLayout`.
    Record every case where it shapes differently, in `STATUS.md`, as a
    deviation. Do not let "the test passes" stand in for "it shapes the same".

- [ ] D8. Check bidi and the supplementary plane against the Java output
  - `GlyphLayoutBidiTest` and `GlyphLayoutSMPTest` are the two that will expose
    a shaper difference first.

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

- [ ] The branch itself. `PLAN.md` names this track, `BRANCHING.md` gives it no
      branch. Nothing here starts until that is settled.
- [ ] A0. The backend choice blocks every task in this file.
- [ ] `slice/4`. Without fonts there is nothing to shape.
