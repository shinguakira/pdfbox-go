# Implementation Plan

Track — `pdfbox-layout-*`. Glyph layout, one interface against two backends.

**Branch: `track/pdfbox-layout`** — from and back to `migration-base`.

`BRANCHING.md` carries a row for the branch. Depends on `slice/4` — it needs
fonts to shape, and that is merged. `PLAN.md` says it is worth reading before
`slice/9` for its backend-interface shape.

**Take this one last, of the four open tracks.** A0 is the whole of it, and it
is a bigger decision than the branch: the AWT backend is
`java.awt.font.TextLayout` and the FOP backend is Apache FOP, Go has neither,
and choosing for Go was expected to constrain, or be constrained by, whatever
implements `rendering.Backend`. What A0 settled on is in `STATUS.md` under "The
decision".

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

The five phases of [`TEMPLATE.md`](TEMPLATE.md), unchanged.

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

This track is the clearest case in the project where a port means choosing a Go
equivalent rather than transliterating, and that choice is the work — not the 7
files around it.

---

# Phase A — Write the tests

- [x] A0. **Decide what the Go backend is** before writing any test. The tests
      assert shaped glyph runs; without a shaper there is nothing to assert
      against.
  - Taken, and measured against the running Java. `STATUS.md`, under "The
    decision", carries the three substitutions it turned on and which of them
    was work rather than a wall.
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
  - `go/pdfbox/glyphlayout`, a substitution for both. `STATUS.md`, under "What
    was built", says why neither `*Awt` nor `*Fop` is ported by name
- [x] B3. `FopStringTextFragment` and whatever the second backend needs, if a
      second backend is in scope at all
  - It is not. One backend serves both, and the `Features` on it are what the
    two font loaders configure

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
  - Where a number in a comment was read off the port rather than off the Java,
    that is said too. Where each assertion's value comes from is in `STATUS.md`
    under "How it is measured: the Java's own output" and "What the Java tests
    could not be ported as"

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
  - The backend's pass is in `STATUS.md` under "Defects found and fixed" in this
    branch's chapter: findings from reading the Java, findings from reading the
    OpenType specification, and one the reference comparison caught with every
    test green

And for this branch in particular:

- [x] D8. This is a substitution, not a transliteration — say so plainly
  - Whatever Go shaper was chosen, it is not `java.awt.font.TextLayout`.
    Record every case where it shapes differently, in `STATUS.md`, as a
    deviation. Do not let "the test passes" stand in for "it shapes the same".
  - Each deviation is measured against the AWT backend's own output and pinned
    in the tests, so a deviation that disappears fails the test as loudly as one
    that appears. They are in `STATUS.md` under "Deviations, measured".

- [x] D9. Check bidi and the supplementary plane against the Java output
  - `GlyphLayoutBidiTest` and `GlyphLayoutSMPTest` are the two that will expose
    a shaper difference first.
  - Done by comparing against the reference PDFs the Java tests render, which
    are the AWT backend's own output. `STATUS.md`, under "How it is measured:
    the Java's own output", carries what agrees and what does not.

---

# Phase E — User feedback

- [x] E1. Stop and wait for the user's review. Do not start the next branch.

- [x] E2. For each item of feedback, judge it before acting
  - Eight items. Seven were real defects, and are in `STATUS.md` under "Defects
    found and fixed" with the rest of this branch's.
  - One declined: writing an empty `[] TJ` at the end of a run is what the Java
    does, unconditionally, and skipping it would be a deviation from the
    reference. Recorded in `STATUS.md` with the Java it was checked against.

- [x] E3. Where it needs fixing, write a **strict** test first
  - Strict: it fails before the fix, takes the real path with the real types,
    and asserts what the Java does
  - Then fix the Go
  - Then `gofmt`, `go vet`, `go test ./...` again
  - Eight cases added: the bidi corpus grew by eight texts measured on the
    running JDK, `TestSupportsFont` gained the font read from a document,
    `TestNullBaseAnchorDoesNotAttach`, `TestRequiredFeatureRunsUnasked` over a
    GPOS table written by hand because no layout font declares one, and
    `TestDamagedGPOSIsReported`. Each was run against the code with its fix
    taken out again, and each fails without it.

- [x] E4. Report back
  - What was changed, what was not, and why for each
  - In STATUS.md under "Track `pdfbox-layout`", and in the reply to the user.

---

# Blocked

- [x] The branch itself. `PLAN.md` names this track, `BRANCHING.md` gives it no
      branch. **Settled** -- `track/pdfbox-layout` exists and A0 ran on it.
- [x] A0. The backend choice blocks every task in this file. **Settled** -- the
      backend is `go/pdfbox/glyphlayout`, built on this branch over ported GSUB
      and GPOS written from the specification. No third-party shaper is used,
      and none is needed.
- [x] `slice/4`. Without fonts there is nothing to shape. Merged long since.
