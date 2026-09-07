# Implementation Plan

Track — `pdfbox-layout-*`. Glyph layout, one interface against two backends.

**Branch: `track/pdfbox-layout`** — from and back to `migration-base`.

The branch is settled; `BRANCHING.md` carries a row for it. What is **not**
settled is A0, and that is a bigger decision than the branch was — see below.

Depends on `slice/4` — it needs fonts to shape, and that is merged. `PLAN.md`
says it is worth reading before `slice/9` for its backend-interface shape.

**Take this one last, of the four open tracks.** The AWT backend is
`java.awt.font.TextLayout` and the FOP backend is Apache FOP; Go has neither, so
A0 is choosing a Go text shaper — a harfbuzz binding, `x/image/font/shaping`, or
something written here. That choice is very likely to constrain, or be
constrained by, whatever eventually implements `rendering.Backend`, which is the
other undecided substitution in the project. Deciding the shaper alone, ahead of
the rasteriser, risks doing both twice.

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

- [x] A0. **Decide what the Go backend is** before writing any test. The tests
      assert shaped glyph runs; without a shaper there is nothing to assert
      against.
  - **Taken, and the answer is that it cannot be chosen yet.** Three
    substitutions have to be decided together and two belong to other work:
    a shaper for layoutGlyphVector, a rasteriser without which no test in this
    module asserts anything about the shaping, and a source of UAX#9 embedding
    levels, which golang.org/x/text/unicode/bidi does not expose. Measured
    against the running Java; see the branch section of migration/STATUS.md.
- [x] A1. Port the shared cases both backends run
  - `GlyphLayoutBidiTest`, `GlyphLayoutDin91379Test`,
    `GlyphLayoutDin91379FormTest`, `GlyphLayoutLigaturesAndKerningTest`,
    `GlyphLayoutSMPTest` — each exists twice, once per backend
- [x] A2. Port `TestBase` — the AWT side's shared fixture
- [x] A3. Port the hello-world tests, if A0 leaves them meaningful

---

# Phase B — Port the implementation

**Every function this phase touches needs a test that says it works.** Where
porting one site makes you notice a second with the same defect, the second one
needs its own test before it is touched: without it nobody can say whether that
code works, before the change or after, and the suite stays green either way.
Either write the test, or leave the site alone and record it.

If the defect is a Java standard-library contract, port the contract once into a
helper rather than patching each call site. See
[`../conventions/java-to-go.md`](../conventions/java-to-go.md) for the ones this
port has already paid for more than once.

- [x] B1. The interface both backends implement
  - `GlyphLayoutProcessor` and `GlyphLayoutFontLoader` in the core, whichever
    slice ported them, and the contract they define
- [x] B2. One backend, chosen in A0
  - `go/pdfbox/glyphlayout`, which is a substitution for both -- see STATUS.md.
    Neither `*Awt` nor `*Fop` is ported by name: each is a shell around a
    library Go has not got
- [x] B3. `FopStringTextFragment` and whatever the second backend needs, if a
      second backend is in scope at all
  - It is not. `FopStringTextFragment` exists to hand a string to FOP, which
    is the library that is absent; there is nothing behind it to port to. One
    backend serves both, and the `Features` on it are what the two font
    loaders configure

---

# Phase C — Run and fix

- [x] C1. `gofmt -l .` clean
- [x] C2. `go vet ./...` clean
- [x] C3. `go test ./...` green
- [x] C4. Record every Java bug found in `migration/JAVA-BUGS.md`
- [x] C5. Update `migration/STATUS.md`

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

- [x] D4. Check every function phase B touched has a test
  - Name the test that covers it. Not "the suite is green" -- green says the
    code is not broken in a way something already checks, which is a different
    claim from "this works"
  - Where there is none, the function was changed on an argument rather than on
    evidence. Write the test, and take whatever it says
  - `supportsFont` had none for its PostScript branch: `TestSupportsFont`,
    which builds such a font from a dictionary because the embedder refuses to
    load one. `resolveAttachments` had none for its right-to-left half, which
    the reference comparison cannot reach -- the Arabic it would compare
    against is shaped by the platform:
    `TestMarkSitsOverItsLetterInBothDirections`. Both were checked against the
    code with the fix taken out again, and both fail without it

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
  - The backend's pass is in STATUS.md under "The adversarial review of the
    backend": three findings from reading the Java, three from reading the
    OpenType specification, and one the reference comparison caught with every
    test green

And for this branch in particular:

- [x] D8. This is a substitution, not a transliteration — say so plainly
  - Whatever Go shaper was chosen, it is not `java.awt.font.TextLayout`.
    Record every case where it shapes differently, in `STATUS.md`, as a
    deviation. Do not let "the test passes" stand in for "it shapes the same".
  - Five deviations, each measured against the AWT backend's own output and
    pinned in the tests: a deviation that disappears fails the test as loudly
    as one that appears.

- [x] D9. Check bidi and the supplementary plane against the Java output
  - `GlyphLayoutBidiTest` and `GlyphLayoutSMPTest` are the two that will expose
    a shaper difference first.
  - Done by comparing against the reference PDFs the Java tests render, which
    are the AWT backend's own output. SMP agrees on all 7 text objects; bidi
    agrees on the run order and differs on Arabic joining, which is recorded.
    `GlyphLayoutDin91379Test` was added to the same comparison and agrees on
    40 of 41.

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

- [x] The branch itself. `PLAN.md` names this track, `BRANCHING.md` gives it no
      branch. **Settled** -- `track/pdfbox-layout` exists and A0 ran on it.
- [x] A0. The backend choice blocks every task in this file. **Settled** -- the
      backend is `go/pdfbox/glyphlayout`, built on this branch over ported GSUB
      and GPOS written from the specification. No third-party shaper is used,
      and none is needed.
- [x] `slice/4`. Without fonts there is nothing to shape. Merged long since.
