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

Five phases, in this order, never overlapping. See
[`TEMPLATE.md`](TEMPLATE.md) for what each phase means.

A — write the test · B — port the implementation · C — run and fix ·
D — adversarial review · E — user feedback

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

敵対的レビュー. See [`TEMPLATE.md`](TEMPLATE.md) for D1–D6. Additionally:

- [ ] D7. This is a substitution, not a transliteration — say so plainly
  - Whatever Go shaper was chosen, it is not `java.awt.font.TextLayout`.
    Record every case where it shapes differently, in `STATUS.md`, as a
    deviation. Do not let "the test passes" stand in for "it shapes the same".

- [ ] D8. Check bidi and the supplementary plane against the Java output
  - `GlyphLayoutBidiTest` and `GlyphLayoutSMPTest` are the two that will expose
    a shaper difference first.

---

# Phase E — User feedback

See [`TEMPLATE.md`](TEMPLATE.md) for E1–E4.

---

# Blocked

- [ ] The branch itself. `PLAN.md` names this track, `BRANCHING.md` gives it no
      branch. Nothing here starts until that is settled.
- [ ] A0. The backend choice blocks every task in this file.
- [ ] `slice/4`. Without fonts there is nothing to shape.
