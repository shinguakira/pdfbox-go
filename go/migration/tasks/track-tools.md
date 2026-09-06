# Implementation Plan

Track — `tools`, the command-line utilities.

**Branch: `track/tools`** — from and back to `migration-base`.

This file used to be `tools-unassigned.md`, and existed to make a gap visible
rather than to close it: `PLAN.md` counted `tools` in scope, deliberately kept
it out of the out-of-scope list, and then never mentioned it again — no slice,
no track, no branch. The gap is closed now. The argument the old file made is
kept below, because it is still the reason this is a track and not a slice.

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

## Why a track, and why now

**Why a track.** `tools` is not one unit of work — every command lands
naturally in the slice that gives it its library, which is exactly why no slice
claimed it. Making it a late slice would have meant one branch that touches
nothing and depends on everything.

**Why now.** The reason it could not be a track before was that its libraries
did not exist. They do: every slice 1 through 9 and both earlier tracks are
merged into `migration-base`. The old file listed three options for where
`tools` should live; the third — "a track of its own, once the libraries under
it exist" — is the one that became true.

| Command | Needs | State |
| --- | --- | --- |
| `Version`, `PDFBox` (the dispatcher) | nothing | ready |
| `DecompressObjectstreams`, `WriteDecodedDoc` | slice 1 | ready |
| `ExtractText`, `PDFText2HTML`, `PDFText2Markdown` | slice 3 | ready |
| `Decrypt`, `Encrypt` | slice 5 | ready |
| `ExtractImages`, `ImageToPDF` | slice 6 | ready, but see the raster note |
| `PDFMerger`, `PDFSplit`, `OverlayPDF`, `TextToPDF` | slice 7 | ready |
| `ExportFDF`, `ImportFDF`, `ExportXFDF`, `ImportXFDF` | slice 8 | ready |
| `PDFToImage`, `PrintPDF` | slice 9 | **blocked** — see below |
| `ExtractXMP` | `track/xmpbox` | ready |

## Scope

26 Java main files, 6 Java test classes.

**Not all 26 are portable today.** Eight of them reach `javax.imageio` or
`java.awt`, and `rendering.Backend` — the interface slice 9 put the raster half
behind — has no implementation:

| Java | Why it is held back |
| --- | --- |
| `PDFToImage`, `PrintPDF` | render a page to a raster. Nothing implements `rendering.Backend` |
| `imageio/ImageIOUtil`, `TIFFUtil`, `JPEGUtil`, `MetaUtil` | `javax.imageio` writers, and its metadata trees. Go's `image/png`, `image/jpeg` and a TIFF library are a substitution, not a transliteration, and the substitution is only worth choosing once there is a raster to write |
| `ExtractImages` | writes the images it extracts through `ImageIOUtil` |
| `PDFBox` (the dispatcher) | lists every command, so it can only be finished last |

So the branch is roughly **18 commands portable now, 7 held for the raster
backend, and the dispatcher last.** Do not port a weakened `PDFToImage` that
writes nothing; record it as held and say what it is waiting for.

### Two things specific to this module

- **The command-line surface is not Java's.** `picocli` has no Go equivalent
  worth transliterating; `flag` or a Go CLI library is a substitution, and the
  flags and their names are the compatibility surface that matters. Decide once,
  in A0, and apply it to all 26.
- **The 6 Java tests are end-to-end.** They run a command against a fixture and
  compare output. They are the best evidence in the project that the library
  underneath actually works, and the worst evidence about the tool itself — a
  passing `ExtractText` test says the text stripper is right, not that the flag
  parsing is.

### Where it goes in the Go tree

`STATUS.md` has been carrying a row for this as "phase 7 `cmd/pdfbox`", which
`PLAN.md` does not say. `go/cmd/` does not exist yet. Settle the directory in
A0 and make `STATUS.md` agree with whatever is chosen.

---

# Phase A — Write the tests

- [ ] A0. **Take the two decisions first**, before any test
  - The flag library, and how a Java `picocli` annotation maps to it
  - The Go directory, and whether `STATUS.md`'s invented `cmd/pdfbox` row
    stands or is replaced
- [ ] A1. Port the 6 Java test classes in `tools/src/test`
  - Name them here once read, with their case counts
- [ ] A2. For every command with no Java test — most of them — decide whether a
      Go test is worth writing, and say so either way. A command whose whole
      body is "parse flags, call one library method, write a file" is tested by
      the library's own tests; a command that transforms output is not

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

- [ ] B1. `Version` and the flag plumbing chosen in A0
- [ ] B2. The slice 1 commands — `DecompressObjectstreams`, `WriteDecodedDoc`
- [ ] B3. The text commands — `ExtractText`, `PDFText2HTML`, `PDFText2Markdown`
- [ ] B4. The encryption commands — `Decrypt`, `Encrypt`
- [ ] B5. The write commands — `PDFMerger`, `PDFSplit`, `OverlayPDF`,
      `TextToPDF`
- [ ] B6. The FDF commands — `ExportFDF`, `ImportFDF`, `ExportXFDF`,
      `ImportXFDF`
- [ ] B7. `ExtractXMP`
- [ ] B8. `ImageToPDF` — check how much of it is raster before starting
- [ ] B9. `PDFBox`, the dispatcher, listing what was actually built
- [ ] B10. Record every command **not** built, and what it waits for

---

# Phase C — Run and fix

- [ ] C1. `gofmt -l .` clean
- [ ] C2. `go vet ./...` clean
- [ ] C3. `go test ./...` green
- [ ] C4. Record every Java bug found on the way in `migration/JAVA-BUGS.md`
- [ ] C5. Update `migration/STATUS.md` — this branch's section, and the phase 7
      row, which currently says `not started` for a directory the plan never
      named

---

# Phase D — Adversarial review

敵対的レビュー. Green tests prove the port passes the tests, not that it is a
faithful migration. Go in assuming it is wrong.

- [ ] D1. Read every ported file against its Java side by side
  - Is any method missing? Any branch of an `if`, any `case`, any `catch`?
  - Java `int` narrows on cast and `float` saturates; Go does neither. Is every
    such conversion written out?

- [ ] D2. Hunt for silently dropped behaviour
  - Anything Java does in a `finally` — is it still done on the Go error path?
  - Anything Java logs and swallows — does the Go swallow it too?

- [ ] D3. Check the tests are Java-derived, not Go-derived

- [ ] D4. Check every function phase B touched has a test
  - Name the test that covers it. Not "the suite is green" -- green says the
    code is not broken in a way something already checks, which is a different
    claim from "this works"
  - Where there is none, the function was changed on an argument rather than on
    evidence. Write the test, and take whatever it says

- [ ] D5. Check every deferral is real and recorded

- [ ] D6. Check the Java bugs

- [ ] D7. Write the review down

And for this branch in particular:

- [ ] D8. Check every flag, one at a time
  - The flag names, their defaults, and what an unrecognised flag does are the
    compatibility surface. A command that does the right thing under different
    flag names is a different command
  - Java's `picocli` gives some flags an arity and some a negatable form. Both
    are easy to lose in translation and neither shows up in a test

- [ ] D9. Check the exit codes and the streams
  - Which failures exit non-zero, and what goes to stdout versus stderr. A tool
    that prints its error to stdout breaks every pipeline that uses it

---

# Phase E — User feedback

- [ ] E1. Stop and wait for the user's review. Do not start the next branch.

- [ ] E2. For each item of feedback, judge it before acting

- [ ] E3. Where it needs fixing, write a **strict** test first

- [ ] E4. Report back
  - What was changed, what was not, and why for each

---

# Blocked

- [ ] `PDFToImage`, `PrintPDF`, `ExtractImages` and the four `imageio` helpers
      need a raster. `rendering.Backend` is the interface slice 9 defined for
      it and nothing implements it. **That is a design decision outside this
      branch** — what Go draws with — and this branch must not take it in
      passing. Port the other 18, record these 7 as held, and name the decision.
