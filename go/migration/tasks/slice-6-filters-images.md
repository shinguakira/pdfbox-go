# Implementation Plan

Slice 6 — the rest of the filters, and images.

**Branch: `slice/6-<name>`** — from and back to `migration-base`.
Depends on `slice/1` only. Independent of slices 2, 3, 4, 5, 7 and 8.

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
| `pdfbox/filter` — what slice 1 left | ~19 of 23 | 2 |
| `pdmodel/graphics/image` | 9 | 7 |
| `pdmodel/graphics` root — `PDXObject`, `PDPostScriptXObject` | 2 of 4 | — |
| `pdfbox/util/filetypedetector` | 3 | — |

`PDXObject` is the base class of both the image XObject here and the form
XObject in slice 9. It belongs to whichever of the two lands first; if slice 9
goes first, drop it from here.

Slice 1 ported `Filter`, `FilterFactory`, `Predictor`, `FlateFilter`,
`IdentityFilter`. Everything else in that package is here.

Go covers part of it: `compress/lzw`, `image/jpeg`. CCITT, JBIG2 and the ASCII
filters need porting. `image/jpeg` does not read CMYK JPEGs — check that before
relying on it.

---

# Phase A — Write the tests

- [ ] A1. `pdfbox/filter` — port `TestFilters` in full
  - Slice 1 ported the round-trip generator only. `testPDFBOX1977` needs LZW
    and `testRLE` needs RunLength; both are in scope here.
- [ ] A2. `pdfbox/filter` — port the second test in that package
- [ ] A3. `pdmodel/graphics/image` — port all 7 Java tests
- [ ] A4. Write from source for any filter the Java tests do not reach
  - Name which ones before writing

---

# Phase B — Port the implementation

- [ ] B1. The ASCII filters — `ASCIIHexFilter`, `ASCII85Filter`
- [ ] B2. `RunLengthDecodeFilter`
- [ ] B3. `LZWFilter` — check `compress/lzw` against the PDF variant first;
      PDF uses early change, which the stdlib may not
- [ ] B4. `DCTFilter` — JPEG. Check CMYK and YCCK handling
- [ ] B5. `CCITTFaxFilter` and its decoder — Group 3 and Group 4
- [ ] B6. `JBIG2Filter` — decide whether it is ported or left declared and
      unsupported, as Java does when the optional jar is missing
- [ ] B7. `DecodeOptions` — image subsampling, which slice 1 left out
- [ ] B8. `pdmodel/graphics` root — `PDXObject`, `PDPostScriptXObject`
- [ ] B9. `pdmodel/graphics/image` — `PDImageXObject`, `PDInlineImage`,
      `PDImage`, `SampledImageReader` and the rest of the 9
- [ ] B10. `pdfbox/util/filetypedetector` — 3 files, sniffing an image's type
      from its bytes

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

- [ ] D7. A filter that produces *nearly* the right bytes is a failed port
  - Compare byte for byte against the Java output, not visually.

- [ ] D8. Check the damage tolerance
  - Every PDFBox filter is written to return what it decoded when the input is
    corrupt, rather than failing. Slice 1 already found one place the Go got
    this wrong. Check each new filter for the same.

- [ ] D9. Check the image types the Go stdlib does not cover
  - Where `image/jpeg` refuses a file Java reads, that is a port gap, not a
    Java bug. Record it.

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

- [x] `pdmodel/graphics/image` needs colour spaces. Slice 2 ported
      `PDColorSpace` as an interface with only `PDDeviceGray` behind it; the
      other 20 are absent. Decide whether they come here or in slice 9.

      **Decided: they come here, minus two.** `PDImageXObject.getColorSpace`
      calls `PDColorSpace.create`, and `SampledImageReader` cannot turn samples
      into a picture without one, so the image work cannot be done without
      them; slice 2's own `colorspace.go` says the create methods and
      `toRGBImage` "belong with the image work of a later slice", which is this
      one. `create` dispatches on the name, so the dispatch has to be complete
      or the port diverges from the Java on a file it should read.

      Two stay out: **`PDPattern`**, which takes a `PDResources` and builds
      pattern dictionaries that only rendering reads, and **`PDJPXColorSpace`**,
      which only `JPXFilter` constructs and that filter needs a JPEG 2000
      decoder Go has not got. Both go to slice 9. `create` reports them the way
      Java reports a colour space it cannot build.

      `PDSeparation` and `PDDeviceN` evaluate a tint transform, so
      `pdmodel/common/function` comes with them — 6 files and the type 4
      subtree. That is scope this branch takes on rather than defers, because
      the two colour spaces are useless without it.
