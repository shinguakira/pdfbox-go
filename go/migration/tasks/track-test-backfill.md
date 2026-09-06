# Implementation Plan

Track — the Java tests that merged slices never ported.

**Branch: `track/test-backfill`** — from and back to `migration-base`.

Not a slice: it ports no new Java class. It runs sixteen Java test classes that
already-merged slices left behind, against Go that already exists.

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

The five phases, with **B reversed**. Everywhere else B is "port the
implementation"; here the implementation is already merged, so B is "fix the Go
the ported test found wrong". That is the whole point of the branch, and it is
the one place where a failing test is *expected*.

**A — port the test.** Copy the Java test case for case. Every assertion value
comes from the Java file, never from running the Go. Do not look at the Go
first: a test written after reading the port asserts what the port does, which
is exactly the thing under suspicion here.

**B — judge each failure, then fix.** A red test is one of three things and the
three are handled differently:

| The failure is | Do |
| --- | --- |
| a defect in the Go | fix the Go, keep the test as the Java wrote it |
| the Java's own behaviour, faithfully reproduced | keep both, and record it in `JAVA-BUGS.md` with the reproduction |
| the test needing something the port does not have (a corpus file, a network fetch, `java.awt`) | drop the case, and record which and why in `STATUS.md` |

**Never** adjust an assertion to match the Go. If the two disagree, one of them
is wrong and it is the branch's job to say which.

**C — run and fix.** `gofmt -l . && go vet ./... && go test ./...`.

**D — adversarial review.** Green tests are not evidence. Read what each ported
test actually exercises against what the Java one did.

**E — user feedback.** Stop. Wait. Judge each item, and where it is a real
defect, write a strict failing test first and only then fix.

## Scope

Sixteen Java test classes, 107 `@Test` methods, 2,557 lines. Every one of them
sits in a package a merged slice claims as done, and none of them is mentioned
anywhere in `STATUS.md` — they were not deferred with a reason, they were
missed.

| Java test | `@Test` | Package, done by |
| --- | ---: | --- |
| `pdfparser/TestCOSParser.java` | 21 | `pdfparser`, slice 1 |
| `pdfparser/TestPDFParser.java` | 18 | `pdfparser`, slice 1 |
| `pdmodel/graphics/blend/BlendModeTest.java` | 17 | `graphics/blend`, slice 2 |
| `pdmodel/fdf/FDFUtilsTest.java` | 13 | `fdf`, slice 8 |
| `util/StringUtilTest.java` | 7 | `util`, slice 7 |
| `util/TestNumberFormatUtil.java` | 6 | `util`, slice 7 |
| `pdfparser/PDFObjectStreamParserTest.java` | 5 | `pdfparser`, slice 1 |
| `util/TestHexUtil.java` | 4 | `util`, slice 6 |
| `pdmodel/fdf/FDFFieldTest.java` | 4 | `fdf`, slice 8 |
| `cos/TestCOSIncrement.java` | 3 | `cos`, slice 7 |
| `pdfparser/PDFStreamParserTest.java` | 2 | `pdfparser`, slice 1 |
| `pdfparser/EndstreamFilterStreamTest.java` | 2 | `pdfparser`, slice 1 |
| `pdmodel/fdf/FDFAnnotationTest.java` | 2 | `fdf`, slice 8 |
| `cos/TestCOSUpdateInfo.java` | 1 | `cos`, slice 7 |
| `pdmodel/graphics/PDLineDashPatternTest.java` | 1 | `graphics`, slice 2 |
| `pdfparser/TestBaseParser.java` | 1 | `pdfparser`, slice 1 |

**The parser is five of the sixteen and 47 of the 107.** `TestCOSParser` and
`TestPDFParser` are the recovery suite — broken cross-reference tables,
truncated objects, the PDFBOX-numbered regressions. Nothing in the port has run
them.

Three small implementation gaps ride along, because this branch is already in
those files:

| Java | Why here |
| --- | --- |
| `pdmodel/ResourceCacheFactory` | the process-wide cache override point; `PDDocument` reads it |
| `pdmodel/ResourceCacheCreateFunction` | its function type |
| `pdmodel/DefaultResourceCacheCreateImpl` | the default it installs |

Not in scope: anything that needs a corpus file this repository does not carry,
and `java.awt` rasterisation. Where a Java case needs one, drop the case and
record it — do not port a weakened version of it.

---

# Phase A — Port the tests

Order is by risk, not by size. The parser first, because it is the one place
where a silent defect corrupts every document that follows.

- [x] A1. `TestCOSParser` — 21 cases
  - The `COSParser` recovery paths: broken `/Root`, broken xref, object numbers
    that do not match, the brute-force scan
- [x] A2. `TestPDFParser` — 18 cases
  - Whole-document parses, PDFBOX-numbered regressions
- [x] A3. `PDFObjectStreamParserTest`, `PDFStreamParserTest`, `TestBaseParser`,
      `EndstreamFilterStreamTest` — 10 cases between them
- [x] A4. `TestCOSIncrement` and `TestCOSUpdateInfo` — 4 cases
  - The incremental-save state slice 7 ported; these are its own tests
- [x] A5. `BlendModeTest` — 17 cases
  - Every separable and non-separable blend function, value by value
- [x] A6. `FDFUtilsTest`, `FDFFieldTest`, `FDFAnnotationTest` — 19 cases
- [x] A7. `StringUtilTest`, `TestNumberFormatUtil`, `TestHexUtil` — 17 cases
- [x] A8. `PDLineDashPatternTest` — 1 case
- [x] A9. For each Java case **not** ported, write down which and why. A case
      dropped without a reason is indistinguishable from one missed.

---

# Phase B — Judge each failure, then fix

- [x] B1. Sort every failure into the three buckets above before fixing
      anything. A batch of fixes made without sorting will quietly "fix" the
      Java's own behaviour.
- [x] B2. Fix the Go where the defect is the port's
- [x] B3. Reproduce against the running Java where the answer is not obvious
      from the source. `io` needs only `log4j-api`; `pdfbox` needs `fontbox`
      and `io` on the classpath. A measured expected value beats an argued one
- [x] B4. Port the three `ResourceCacheFactory` files
  - Java's is a static with a settable function and a `null` that disables
    caching. Go has no static initialiser; a package-level var set in `init()`
    is the shape the port already uses for this
  - `PDDocument` should read it where Java does, instead of calling
    `NewDefaultResourceCache` directly

---

# Phase C — Run and fix

- [x] C1. `gofmt -l .` clean
- [x] C2. `go vet ./...` clean
- [x] C3. `go test ./...` green
- [x] C4. Record every Java bug found on the way in `migration/JAVA-BUGS.md` — **none**. Every failure was the port's
- [x] C5. Update `migration/STATUS.md`
  - This branch's section
  - The five stale rows the survey found, listed under **Known-stale rows**
    below
  - Every dropped Java case, with its reason

---

# Phase D — Adversarial review

敵対的レビュー. Green tests prove the port passes the tests, not that it is a
faithful migration. Go in assuming it is wrong.

- [x] D1. Read each ported test against its Java file side by side
  - Is every `@Test` accounted for — ported, or dropped with a reason?
  - Is every assertion value the Java's? A value that "looked right" is a value
    read off the Go
  - Does a `@ParameterizedTest` keep all its arguments?

- [x] D2. Check what each test actually reaches
  - Does it take the real path with the real types, or a stand-in that would
    pass while the path it stands for is broken?
  - A test that constructs the Go's own output and asserts on it proves nothing

- [x] D3. Hunt for tests that pass for the wrong reason
  - A case asserting an exception: is it the same failure, or a different one
    that happens to also fail?
  - A case asserting a count: would it still pass if the contents were wrong?

- [x] D4. Check every fix made in phase B
  - Was it a port defect, or did the Java behave that way? The second is a
    `JAVA-BUGS` entry and a reverted fix
  - Does each fix have a test that fails without it?

- [x] D5. Check the Java bugs
  - Every bug found — is it in `migration/JAVA-BUGS.md` with where, what,
    what correct would be, where the Go carries it, and how confident?

- [x] D6. Write the review down
  - What was checked, what was found, what was fixed, what is still open
  - Say plainly how many defects the sixteen classes found. If the answer is
    none, say that too — it is the result, not a failure to find one

And for this branch in particular:

- [x] D7. Re-run the survey that produced this branch
  - Enumerate the Java test classes again and confirm the sixteen are gone from
    the unported list, and that nothing new appeared
  - Whatever is still unported must be in `STATUS.md` with a reason by the end
    of this branch

---

# Phase E — User feedback

- [x] E1. Stop and wait for the user's review. Do not start the next branch.

- [x] E2. For each item of feedback, judge it before acting
  - Is it a port defect, a missing piece of scope, or a difference the Java
    itself has?
  - A Java difference is not fixed — it is recorded in `JAVA-BUGS.md` and the
    user is told why it stays.

- [x] E3. Where it needs fixing, write a **strict** test first
  - Strict: it fails before the fix, takes the real path with the real types,
    and asserts what the Java does
  - Then fix the Go
  - Then `gofmt`, `go vet`, `go test ./...` again

- [x] E4. Report back
  - What was changed, what was not, and why for each

---

# Known-stale rows in `STATUS.md`

Found by the survey that produced this branch. Correct them in C5.

All of them have the same cause: **a later slice closed a deferral and only
wrote it down in its own section.** The summary and the deferring slice's tables
were left saying the work was still outstanding.

| Where | Says | Is |
| --- | --- | --- |
| Summary, phase 1 | `19 of 24`, the remaining 4 deferred to slice 7 | Slice 7 ported `COSIncrement`, `COSUpdateInfo` and `COSUpdateState`. 22 of 24 now; the two left are `COSInputStream` and `COSOutputStream`, both already recorded as deliberately not ported with a reason |
| Summary, phase 2 | `in progress — filter has the slice 1 subset` | `filter` is 23 of 23, `pdfparser` 18 of 18, `pdfwriter` 7 of 7. Slice 6 finished the filters |
| Slice 2 section, the `filter` table | `the other 15 filters \| — \| slice 6`, and `DecodeOptions.java \| — \| not started` | Both done. Slice 6's own section records them correctly, 1,200 lines further down the same file |

**Checked and correct — do not "fix" these:**

| Row | Verified |
| --- | --- |
| `pdmodel/font at 34 of 39`, five embedders left | Right. `TrueTypeEmbedder`, `PDTrueTypeFontEmbedder`, `PDCIDFontType2Embedder`, `Subsetter` and `ToUnicodeWriter` are all still unported; `PDType1FontEmbedder` is ported, in `pdfont.go` |
| `4 of rendering ... are java.awt classes` | Right. `GroupGraphics`, `SoftMask`, `TilingPaint`, `TilingPaintFactory` |

The survey's first pass got both of those wrong, by matching a class name that
appears in a Go comment saying the class is *not* ported. A name in the tree is
not evidence of a port; a `Port of <FQN>` comment or a type is.

---

# Blocked

Nothing. Every package this branch tests is merged into `migration-base`.
