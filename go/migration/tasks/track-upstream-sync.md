# Implementation Plan

Track — the upstream sync of 2026-09-07. Reading what Apache changed, saying
which `JAVA-BUGS.md` entries it closed, and giving the Go the ones it needs.

**Branch: `track/upstream-sync`** — from and back to `migration-base`.

The merge is `3d024173c`, "Merge branch 'apache:trunk' into migration-base",
2026-09-07 13:14 JST. It brought 25 Apache commits and touched 16 files. This
branch is what happens to the Go because of them.

## What the sync actually changed

Sixteen files. Six are out of this port's scope and are listed here so that the
next person does not have to re-derive that:

| File | Why it is out of scope |
| --- | --- |
| `debugger/flagbitspane/PanoseFlag.java` | PDFBOX-2941. `debugger` is not ported; `PLAN.md` puts it outside the migration |
| `examples/.../BengaliPdfGenerationHelloWorld.java` | `examples` is not ported |
| `examples/.../CreateEmbeddedTimeStamp.java` | same |
| `pdfbox/pom.xml` | Maven; downloads the PDFBOX-5960 fixture |
| `fontbox/.../TestTTFParser.java` | PDFBOX-6254, a zone-name typo. See below — the Go test does not depend on it |
| `pdfbox/.../PDFontTest.java` | the PDFBOX-5960 Java test. Its fixture is a JIRA attachment this tree does not carry; the Go writes its own at the same site |

The other ten are in scope and each has a row below.

## The ten, and what the Go does about each

| # | Java | Apache | What it changed | The Go |
| ---: | --- | --- | --- | --- |
| 1 | `contentstream/operator/state/Concatenate` | PDFBOX-6255 `4a42d294e` | wraps `IllegalArgumentException` from `Matrix.concatenate` in `IOException` | **fix.** The port panics through `util.checkFloatValues` |
| 2 | `pdfparser/COSParser` | PDFBOX-5660 `de68eb3e3` | null check on `getObjectFromPool` | **fix.** The port dereferences the nil |
| 3 | `interactive/form/AppearanceGeneratorHelper` | PDFBOX-5660 `ced684bba` | `computeBBox` throws on a missing `/Rect` | **fix.** The port dereferences the nil |
| 4 | `pdfwriter/COSWriter` | PDFBOX-6236 `21661b79f` | an incremental update starts numbering at the origin trailer's `/Size` | **fix.** The port has neither the read nor the max |
| 5 | `pdmodel/font/PDTrueTypeFont` | PDFBOX-5960 `a1f50ab4b` | a contradictory-flags font with a recognized `/BaseEncoding` resolves by glyph name first | **fix.** The only behavioural change in the sync, and the port has none of it |
| 6 | `util/DateConverter` (test) | PDFBOX-6254 `0079a9cc7` | `Antartica/McMurdo` was a typo, so the seven assertions it carried were of GMT | **port the assertions.** The Go test dropped the loop rather than carrying it |
| 7 | `fixup/processor/AcroFormOrphanWidgetsProcessor` | PDFBOX-5660 `f7654f001` | early return on a null `/DA` | **no change.** The port already returns there. A test pins it |
| 8 | `logicalstructure/PDUserAttributeObject` | PDFBOX-5660 `2ff5c9ab3`, `c766e8298`, `d404dd774` | three null checks on `/P` | **no change.** `JAVA-BUGS.md` 38, already fixed in `track/java-bug-fixes` |
| 9 | `io/RandomAccessReadBufferedFile` | PDFBOX-5660 `b1d96635f` | `seek` made `final` | **not applicable.** Go has no method overriding |
| 10 | `pdmodel/fdf/FDFUtils` | PDFBOX-5660 `4aa80d810` | private constructor on a static-only class | **not applicable.** `go/pdfbox/pdmodel/fdf` has package functions, not a type |

## Rules — do not break these

- **NEVER change the Java.** No `.java`, no `pom.xml`, no test resource, for any
  reason. That includes the files this sync brought in: they are the new
  reference, not a draft.
- **NEVER fix a bug that is in the Java.** This branch is not a licence to fix
  anything else. What it changes is the places where **upstream has already
  decided** what correct is, and the Go is behind it. Anything else found on the
  way goes into `JAVA-BUGS.md` and stays there.
- **NEVER delete an entry from `JAVA-BUGS.md`.** An entry upstream has fixed
  gains a **Resolved upstream** line naming the Apache commit and the JIRA
  issue. It does not go away: the entry is the record that the Go carried it,
  and `track/java-bug-fixes` may already have fixed it independently.
- **NEVER touch `apache/pdfbox`** — no PR, no pull, fetch, merge or rebase. The
  sync already happened; this branch reads `3d024173c` out of local history.
- **NEVER commit to `migration-base` directly.**
- **One row, one commit.** Each of the ten above is its own commit so that any
  one can be reverted alone.
- **DO NOT STOP UNTIL PHASE E.** E1 is the only stop.

## How each row runs

The five phases of every other task file, unchanged:

- **A — write the test.** At the exact site, and failing for the right reason
  before any implementation moves. "The exact site" is not the package: it is
  the function the Java diff touched.
- **B — port the change.** What Apache wrote, in Go, with a comment naming the
  JIRA issue as the Java carries it.
- **C — run and fix.** `go build ./...`, `go vet ./...`, the package's tests.
- **D — adversarial review.** Re-run each new test with its fix reverted. A test
  that still passes is not a test.
- **E — user feedback.**

Assertion values are copied verbatim from the Java. Never recomputed.
