# Implementation Plan

Track — the upstream sync of 2026-11-21. Reading what Apache changed, saying
which [`JAVA-BUGS.md`](../JAVA-BUGS.md) entries it closed, and giving the Go the
ones it needs.

**Branch: `update/sync-original-update`** — from and back to `migration-base`.

The merge is `9fb3ce265`, "Merge remote-tracking branch 'origin/trunk' into
migration-base", 2026-11-21 16:09 JST. It brought **320 Apache commits** and
touched **266 files**: 172 Java under `*/src/main`, 69 Java tests, four
`pom.xml`, and the rest test resources. Sixteen main files are new — fifteen of
them the TrueType hinting interpreter in `fontbox/ttf` (PDFBOX-3293) and the
sixteenth `pdfbox/pdfparser/TrailerParser.java` (PDFBOX-5032). Nothing under
`go/` was touched by the merge.

| Module | Main Java files changed |
| --- | ---: |
| `pdfbox` | 101 |
| `fontbox` | 26 |
| `tools` | 16 |
| `xmpbox` | 11 |
| `debugger` | 7 |
| `examples` | 5 |
| `io` | 3 |
| `pdfbox-layout-fop` | 2 |
| `pdfbox-layout-awt` | 1 |

The sync of 2026-09-07 ([`track-upstream-sync.md`](track-upstream-sync.md)) was
25 commits over 16 files, and that branch read every one of them against the
port. This one is thirteen times the size and **that per-file reading has not
been done** — see "What has not been done" below.

## What has been done

The `JAVA-BUGS.md` pass, and only that.

1. Every Java file cited by an entry was intersected with the 266 the merge
   changed: 45 of the 91 entries name a file the sync touched.
2. For each of those 45, the code the entry quotes was looked for in the
   post-merge file and in the pre-merge one, which separates "the site moved"
   from "the entry abridged the quote".
3. Every entry whose site moved, and every entry in a file whose sync commits
   could plausibly reach it, was read at the site in the merged tree and against
   the Apache commit that changed it.

Seventeen entries are closed, four had their site changed without being closed,
and two defects the file had never recorded turned up in a commit read for entry
29. Each of the twenty-three now carries a **Resolved upstream** or a **Not
resolved upstream** line naming the Apache commit, the issue and this sync.

## The seventeen entries the sync closed

| # | What it was | Apache | Issue | The Go |
| ---: | --- | --- | --- | --- |
| 12 | `TrueTypeFont.nameToGID` dereferences a null cmap | `da7095a09` | PDFBOX-6326 | **no change.** Same answer, 0 |
| 16 | `CMap.useCmap` writes `% 0xFF` | `79d6fe945` | PDFBOX-6325 | **no change.** Same answer |
| 17 | `CFFParser.concatenateMatrix` reads `d1` for `d2` | `1de32abdc` | PDFBOX-6324 | **no change.** Same answer |
| 18 | `PDType1CFont.getStringWidth` walks UTF-16 units | `eec7c6cd9` | PDFBOX-6287 | **no change.** The carry was unobservable and still is |
| 20 | CID supplement ANDed instead of ORed | `27c375dda` | PDFBOX-6323 | **no change.** Same join |
| 23 | `CMapStrings.getMapping` reads a zero-length code as 0 | `3c630cd0b` | PDFBOX-6327 | **no change.** Same answer |
| 24 | `PDEncryption.hasSecurityHandler` inverted | `a03cb3cbf` | PDFBOX-6313 | **no change.** Same answer |
| 27 | ASCII85 reads 0xFF as end of stream | `d219d4140` | PDFBOX-6317 | **no change.** Same answer |
| 29 | Type 4 `not` negates an integer | `47d20cb71` | PDFBOX-6331 | **no change.** Same answer |
| 32 | A truncated ASCII85 stream repeats its last group | `d219d4140` | PDFBOX-6317 | **differs.** The Java decodes the partial group; the Go stops at the last complete one |
| 37 | `PDMarkInfo.setSuspect` ignores its argument | `2fab51caa` | PDFBOX-6312 | **no change.** Same answer |
| 40 | `PDStandardAttributeObject` reads a name array | `691e7d100` | PDFBOX-6261 | **differs.** The Java keeps a null per bad entry; the Go leaves it out |
| 56 | `DomXmpParser.parseEndPacket` indexes past the end | `642cbcc0f` | PDFBOX-6334 | **differs by one character.** The message lost its trailing space |
| 82 | The merge takes the destination's `/Threads` | `a0712f23c` | PDFBOX-6295 | **no change.** Same one-word fix |
| 83 | `mergeMarkInfo` writes `/Suspect` twice | `aa9b1c1e6` | PDFBOX-6276 | **no change.** Same three lines |
| 87 | Type 3 glyph recursion is unbounded | `96337d173` | PDFBOX-6266 | **behind.** The Java bounds it at fifty; the port does not |
| 88 | A zero TIFF predictor row length never returns | `e7951e2d2` | PDFBOX-6304 | **no change.** The port refused it from the start |

Eleven of the seventeen were already fixed on
[`track/java-bug-fixes`](track-java-bug-fixes.md) and upstream landed on the
same answer. Entry 88 the port never carried. That leaves five where something
has to be decided or written: 32, 40, 56, 87 and 18 — the last of which needs
nothing but a reader who knows the Java moved.

**All seventeen entries were then removed from
[`JAVA-BUGS.md`](../JAVA-BUGS.md)**, on 2026-11-21, on the repository owner's
instruction. The table above is the record of what each one was; the full text
of all seventeen is in the commit that removed them, and that file's numbering
note names the missing numbers. The numbers are not reused — the Go cites every
one of them, at the site or in the name of a test file, and so do
[`STATUS.md`](../STATUS.md), [`TESTDATA.md`](../TESTDATA.md),
[`BENCHMARK.md`](../BENCHMARK.md) and the other task files. None of those
citations was touched, because they are source or the record of finished work.

## The four whose site the sync changed without closing them

| # | What upstream did at the site | Issue |
| ---: | --- | --- |
| 6 | Replaced the xref `HashMap` with an indexing `XrefTable` subclass; a null key still goes in through `putAll` | PDFBOX-6268 |
| 30 | Skips white space before the second hex digit; the `-1` from the table is still multiplied and added | PDFBOX-6330 |
| 84 | Rewrote much of `PDFMergerUtility` around the dead page-mode branch, which is still dead | PDFBOX-6283, 6296, 6299, 6301 |
| 91 | Wrapped `EmptyStackException` and `ClassCastException` from `execute` as `IOException`, two statements above the `IllegalStateException` the entry is about | PDFBOX-6320 |

Entries 30 and 91 leave the Go behind on behaviour upstream added beside the
defect: the white space between a hex digit pair, and the two unchecked
exceptions now turned into `IOException`.

## The two entries this sync added

`47d20cb71` (PDFBOX-6331) fixed three type 4 defects. One was entry 29. The
other two were in the frozen snapshot, are carried by the port, and had never
been recorded:

- **92.** `roll` takes `j` without a modulo, so `|j| > n` pops elements the
  operator was not given, or off the bottom of the stack.
- **93.** `eq`, `ne`, `ge`, `gt`, `le` and `lt` compare as `float`, so two `int`s
  that differ above 16,777,216 compare equal. Upstream fixed four of the five
  comparison sites; **`Lt` still compares as `float`**.

Both carry a **Resolved upstream** line and both leave the Go behind.

## What the Go needs

One row, one unit of work, one commit. A row says **done** once that commit has
landed on this branch. The `readSignificant` row was missing from the first
pass of this file: PDFBOX-6330 touched both filters, and only the ASCIIHex half
was written down.

| Site | What to do | Comes from |
| --- | --- | --- |
| `go/pdfbox/contentstream/text.go`, `ShowType3Glyph` | **done** — level, limit fifty, released in a `defer`, as `drawobject.go` does it; `pdfbox6266_test.go` | entry 87, PDFBOX-6266 |
| `go/pdfbox/filter/ascii85.go`, `readByte` | Decide whether a group cut short reads as the terminator, as the Java now does, or stops as the port does; whichever way, say so where it happens | entry 32, PDFBOX-6317 |
| `go/pdfbox/filter/ascii85.go`, `readSignificant` | **done** — the white-space test is the six characters of ISO 32000-1 table 1, not the three it had; `pdfbox6330ascii85_test.go` | PDFBOX-6330 |
| `go/pdfbox/filter/asciihex.go`, `Decode` | Skip white space before the second digit | entry 30, PDFBOX-6330 |
| `go/pdfbox/pdmodel/common/function/type4/stackoperators.go`, `rollOperator` | **done** — `n == 0` returns, `j %= n`, a zero `j` returns; `pdfbox6331roll_test.go` | entry 92, PDFBOX-6331 |
| `go/pdfbox/pdmodel/common/function/type4/relationaloperators.go` | **done** — `eq`, `ne`, `ge`, `gt`, `le` compare as `float64`; `lt` narrows back to `float32`, because the Java's `Lt` does, and a test pins it; `pdfbox6331compare_test.go` | entry 93, PDFBOX-6331 |
| `go/pdfbox/pdmodel/common/function/pdfunctiontype4.go`, `Eval` | **done** — `executeType4` recovers the panics wrapping `type4.ErrEmptyStack` and `type4.ErrWrongType` and returns them as errors; the too-few-outputs panic stays, because the Java's `IllegalStateException` does; `pdfbox6320_test.go` | entry 91, PDFBOX-6320 |
| `go/pdfbox/pdmodel/documentinterchange/taggedpdf/pdstandardattributeobject.go` | Decide whether a bad entry is dropped or kept as an empty string, now that the Java keeps a hole | entry 40, PDFBOX-6261 |
| `go/xmpbox/xml/domxmpparser.go` and its two tests | Drop the trailing space from the `XpacketBadEnd` message | entry 56, PDFBOX-6334 |

## What has not been done

**The per-file reading of the sync.** `track/upstream-sync` took sixteen files
and wrote a row for each. The 172 main Java files this merge changed have had no
such pass: what follows is a list of the themes, not an analysis of them, and
none of it has been read against the port, ported or tested.

- **PDFBOX-3293**, TrueType font hinting: fifteen new `fontbox/ttf` classes, a
  `GlyphHinter` on `TrueTypeFont`, the `cvt `, `fpgm`, `prep` and `gasp` tables,
  and a hinting option through `PageDrawer` and the debugger. A capability the
  port does not have at all
- **PDFBOX-5032**, `TrailerParser` split out of `COSParser`
- **PDFBOX-6268**, the xref key cache, and a new public `COSDocument.getXrefKey`
- **PDFBOX-6287**, the code-to-bytes and codepoint caches moved to `PDFont`,
  and `Encoding.getCode`
- **PDFBOX-6283, 6294, 6296, 6299, 6300, 6301**, the merger and the splitter:
  named destinations, `mergePageLabels` rewritten, structure-tree and XObject
  mapping, stack-overflow and NPE guards, page-label numbering
- **PDFBOX-6281, 6310, 6311**, text extraction: nested marked content in
  `/ActualText`, the bidi handling, a crash between articles
- **PDFBOX-5953, 6251, 6292, 6293**, font encoding and substitution paths
- **PDFBOX-6265**, caps on extreme `/DecodeParms`
- **PDFBOX-5079, 6258, 6260, 6264**, colour: clamping, chromatic adaptation,
  the D65 whitepoint hack
- **PDFBOX-6289, 6291**, signing: a password parameter, DER, and the revocation
  fetch hardened
- **PDFBOX-6170**, `addFontForSubsetting` replacing `PDDocument.getFontsToSubset`
- **PDFBOX-6262**, the tool command names: `exportxfdf` is now `export:xfdf`,
  which `go/tools` mirrors
- **PDFBOX-6280**, deprecations and removals; **PDFBOX-5660**, several hundred
  Sonar fixes, javadoc corrections, `throws IOException` clauses dropped from
  methods that cannot throw, and the `instanciate` → `instantiate` rename
  through `xmpbox`
- `StandardStructureTypes` constants renamed to upper case
- **PDFBOX-6315, 6316, 6319, 6321, 6327, 6329, 6333, 6335** and the other
  single-commit fixes not named above

**[`STATUS.md`](../STATUS.md) has no chapter for this sync.** The previous one's
is "Track `upstream-sync`". This one gets its chapter when there is Go work to
record in it.

**Nothing has been checked by running anything.** The entries were read against
the Java in the merged tree and against the Apache commits. No Go test was
written or run on this branch, and `go build`, `go vet` and `go test` were not
run, because no Go file was touched.

## Rules — do not break these

- **NEVER change the Java.** No `.java`, no `pom.xml`, no test resource, for any
  reason. The files this sync brought in are the new reference, not a draft.
- **NEVER fix a bug that is in the Java.** What the rows above change is the
  places where **upstream has already decided** what correct is and the Go is
  behind it. Anything else found on the way goes into
  [`JAVA-BUGS.md`](../JAVA-BUGS.md) and stays there — 92 and 93 are what that
  looks like.
- **An entry upstream has fixed gains a Resolved upstream line** naming the
  Apache commit and the issue, and it does not go away on an agent's judgement:
  the entry is the record that the Go carried the defect, and
  `track/java-bug-fixes` may have fixed it first. The seventeen this sync closed
  were removed from `JAVA-BUGS.md` on 2026-11-21 because the repository owner
  said to remove them, which is the only thing that removes one.
- **NEVER touch `apache/pdfbox`** — no pull request, no issue, no pull, fetch,
  merge or rebase. The merge is read out of local history.
- **NEVER commit to `migration-base` directly.**
- **One row, one commit**, so that any one can be reverted alone.

## How each row of "What the Go needs" runs

The five phases of [`TEMPLATE.md`](TEMPLATE.md), read for one row rather than
one package:

- **A — write the test.** At the exact site, failing for the right reason before
  any implementation moves. "The exact site" is the function the Java diff
  touched, not the package.
- **B — port the change.** What Apache wrote, in Go, with a comment naming the
  issue as the Java carries it.
- **C — run and fix.** `go build ./...`, `gofmt -l .`, `go vet ./...`, the
  package's tests.
- **D — adversarial review.** Re-run each new test with its fix reverted. A test
  that still passes is not a test.
- **E — user feedback.**

Assertion values are copied verbatim from the Java. Never recomputed.
