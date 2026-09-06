# Branch task lists

One file per branch. Each carries the rules, the branch's scope, and the five
phases the work runs through.

These are **not** migration-wide documents. The migration-wide ones are
[`../PLAN.md`](../PLAN.md), [`../BRANCHING.md`](../BRANCHING.md),
[`../STATUS.md`](../STATUS.md) and [`../JAVA-BUGS.md`](../JAVA-BUGS.md).

[`TEMPLATE.md`](TEMPLATE.md) is the template. Copy it for a new branch. Do not
change it to suit one branch — change the copy.

| Branch | File | State |
| --- | --- | --- |
| `slice/0-*` | — | **merged**, predates these files |
| `slice/1-open-document` | — | **merged**, predates these files |
| `slice/2-content-streams` | — | **merged**, predates these files |
| `slice/3-text-simple-fonts` | [`slice-3-text-simple-fonts.md`](slice-3-text-simple-fonts.md) | **merged** |
| `slice/4-text-cid-cff` | [`slice-4-text-cid-cff.md`](slice-4-text-cid-cff.md) | **merged** |
| `slice/5-encryption` | [`slice-5-encryption.md`](slice-5-encryption.md) | **merged** |
| `slice/6-filters-images` | [`slice-6-filters-images.md`](slice-6-filters-images.md) | **merged** |
| `slice/7-write-merge` | [`slice-7-write-merge.md`](slice-7-write-merge.md) | **merged** |
| `slice/8-forms-annotations` | [`slice-8-forms-annotations.md`](slice-8-forms-annotations.md) | **merged** |
| `slice/9-rendering` | [`slice-9-rendering.md`](slice-9-rendering.md) | **merged** — minus the raster half, behind `rendering.Backend` |
| `track/xmpbox` | [`track-xmpbox.md`](track-xmpbox.md) | **merged** |
| `track/scratchfile` | [`track-scratchfile.md`](track-scratchfile.md) | **merged** |
| `track/test-backfill` | [`track-test-backfill.md`](track-test-backfill.md) | open — depends on nothing |
| `track/font-embedding` | [`track-font-embedding.md`](track-font-embedding.md) | open — needs 4 and 7, both merged |
| `track/tools` | [`track-tools.md`](track-tools.md) | open — needs every slice, all merged |
| `track/pdfbox-layout` | [`track-pdfbox-layout.md`](track-pdfbox-layout.md) | open — **take last**, see its A0 |

Every slice in `PLAN.md` is merged. The four open branches are tracks: work
`PLAN.md` counts in scope that no slice claimed.

## Order for the four open tracks

**`track/test-backfill` first.** It is the only one that can find defects in
work already merged; the other three add surface area on top of a base whose
test coverage has a known hole. Sixteen Java test classes, 107 cases, sitting in
packages a merged slice calls done and mentioned nowhere in
[`../STATUS.md`](../STATUS.md) — not deferred with a reason, missed.

Then `track/font-embedding`, which closes a capability gap: nothing in the port
can write a PDF with an embedded font today, and the method that would is a
panic. Then `track/tools`, whose 22 commands finally all have their libraries.
Then `track/pdfbox-layout`, last, because its A0 is a substitution decision
entangled with `rendering.Backend`.

## The two gaps, and how they were closed

This file used to carry two rows marked **gap** — work `PLAN.md` counted in
scope with no branch anywhere. Both were closed by the user, in the survey that
also produced `track/test-backfill` and `track/font-embedding`:

- **`pdfbox-layout-*`** now has `track/pdfbox-layout`, and a row in
  [`../BRANCHING.md`](../BRANCHING.md).
- **`tools`** now has `track/tools`. Its file was `tools-unassigned.md`, whose
  whole purpose was to make the gap visible rather than close it; it is renamed
  and rewritten, and the argument it made — that `tools` is not one unit of work
  — is kept, because it is still why this is a track and not a slice.

`PLAN.md` is **not** changed by any of this. It counted both in scope already;
what was missing was a branch, and branches live in
[`../BRANCHING.md`](../BRANCHING.md).

## Coverage

Every Java package in scope, and the branch that claims it. Regenerate the left
column with:

```
find fontbox/src/main pdfbox/src/main io/src/main xmpbox/src/main \
     tools/src/main pdfbox-layout-*/src/main -name '*.java' \
  | sed 's|/[^/]*\.java$||' | sed 's|^[a-z-]*/src/main/java/||' \
  | sort | uniq -c | sort -k2
```

| Java package | Files | Claimed by |
| --- | ---: | --- |
| `org/apache/fontbox` | 2 | slice 3 |
| `fontbox/afm` | 8 | slice 3 |
| `fontbox/cff` | 26 | slice 4 |
| `fontbox/cmap` | 5 | slice 4 |
| `fontbox/encoding` | 4 | slice 3 |
| `fontbox/pfb` | 1 | slice 4 |
| `fontbox/ttf` | 44 | slice 3 (~15), slice 4 (rest) |
| `fontbox/ttf/gsub` | 13 | slice 4 |
| `fontbox/ttf/model` | 5 | slice 4 |
| `fontbox/ttf/table/common` | 12 | slice 4 |
| `fontbox/ttf/table/gsub` | 9 | slice 4 |
| `fontbox/type1` | 6 | slice 4 |
| `fontbox/util` | 1 | slice 2 — done |
| `fontbox/util/autodetect` | 7 | slice 4 |
| `org/apache/pdfbox` — `Loader` | 1 | slice 3, conditionally — see its Blocked |
| `pdfbox/contentstream` | 3 | slice 2 — done; slice 9 for the graphics engine |
| `contentstream/operator` | 5 | slice 2 — done |
| `contentstream/operator/color` | 13 | slice 9 |
| `contentstream/operator/graphics` | 23 | slice 9 |
| `contentstream/operator/markedcontent` | 6 | slice 2 — done bar `DrawObject` |
| `contentstream/operator/state` | 13 | slice 2 — done bar `gs` |
| `contentstream/operator/text` | 16 | slice 2 (11), slice 3 (5) |
| `pdfbox/cos` | 24 | slice 1, slice 7 for the update-state files — **done** |
| `pdfbox/filter` | 23 | slice 1 (4), slice 6 (rest) — **done** |
| `pdfbox/glyphlayout/*` | 7 | `track/pdfbox-layout` |
| `pdfbox/io` | 18 | slice 0 (13), `track/scratchfile` (5) — **done** |
| `pdfbox/multipdf` | 6 | slice 7 |
| `pdfbox/pdfparser` | 12 | slice 1 (6), slice 3 conditionally, slice 8 for `FDFParser` |
| `pdfbox/pdfparser/xref` | 6 | slice 1 — done |
| `pdfbox/pdfwriter` | 3 | slice 7 |
| `pdfbox/pdfwriter/compress` | 4 | slice 7 |
| `pdfbox/pdmodel` | 29 | slice 2 (4), slice 3 conditionally, slice 7 |
| `pdmodel/common` | 16 | slice 2 (5), slice 8 (rest) |
| `pdmodel/common/filespecification` | 4 | slice 8 |
| `pdmodel/common/function` | 6 | slice 9 |
| `pdmodel/common/function/type4` | 11 | slice 9 |
| `pdmodel/documentinterchange/*` | 24 | slice 8 |
| `pdmodel/encryption` | 19 | slice 5 |
| `pdmodel/fdf` | 31 | slice 8 |
| `pdmodel/fixup`, `fixup/processor` | 8 | slice 8 |
| `pdmodel/font` | 39 | slice 3 (~12), slice 4 (rest), `track/font-embedding` (the 5 embedders) |
| `pdmodel/font/encoding` | 12 | slice 3 |
| `pdmodel/graphics` | 4 | slice 2 (1), slice 6 (2), slice 9 (`PDFontSetting`) |
| `pdmodel/graphics/blend` | 2 | slice 2 (1), slice 9 (1) |
| `pdmodel/graphics/color` | 23 | slice 2 (3), slice 9 (rest) |
| `pdmodel/graphics/form` | 3 | slice 9 |
| `pdmodel/graphics/image` | 9 | slice 6 |
| `pdmodel/graphics/optionalcontent` | 3 | slice 8 |
| `pdmodel/graphics/pattern` | 3 | slice 9 |
| `pdmodel/graphics/shading` | 37 | slice 9 |
| `pdmodel/graphics/state` | 6 | slice 2 (4), slice 9 (2) |
| `pdmodel/interactive/*` | 144 | slice 8 |
| `pdfbox/printing` | 4 | slice 9 |
| `pdfbox/rendering` | 10 | slice 9 |
| `pdfbox/text` | 6 | slice 3 |
| `pdfbox/tools`, `tools/imageio` | 26 | `track/tools` |
| `pdfbox/util` | 9 | slice 2 (2), slice 3 (2), slice 6 (1), slice 7 (2), slice 8 (1), `tools` (1) — see below |
| `pdfbox/util/filetypedetector` | 3 | slice 6 |
| `xmpbox/*` | 74 | `track/xmpbox` |

`pdfbox/util` is nine unrelated helpers with no single home, so each goes to the
branch that first needs it. Found by grepping for each import:

| Helper | Used by | Branch | Java test |
| --- | --- | --- | --- |
| `Matrix`, `Vector` | the graphics state | slice 2 — **done** | `MatrixTest` |
| `IterativeMergeSort` | `PDFTextStripper`, when its comparator is not transitive | slice 3 | `TestSort` |
| `DateConverter` | `COSDictionary` dates, `FDFAnnotation` | slice 3 with `PDDocumentInformation` if the loader lands there, otherwise slice 8 | `TestDateUtil` |
| `Hex` | `COSName`, `COSString`, `ASCIIHexFilter`, `COSWriter`, `ToUnicodeWriter`, `FDFAnnotationStamp` | slice 6, with `ASCIIHexFilter` | `TestHexUtil` |
| `NumberFormatUtil` | `PDAbstractContentStream` | slice 7 | `TestNumberFormatUtil` |
| `StringUtil` | `PDAbstractContentStream` | slice 7 | `StringUtilTest` |
| `XMLUtil` | `Loader`, `FDFField`, `FDFAnnotationStamp` | slice 8, with `fdf` | — |
| `Version` | `tools` only | `track/tools` | — |

`Hex` is a special case: `cos.ParseHexString` already exists in the port, so
slice 1 folded part of it away. Check what is left of the Java class before
porting it whole.

`COSDictionary`'s date accessors are the other half of `DateConverter`.
`STATUS.md` records them as the "minus dates" in the slice 1 `cos` row; they
land with whichever branch takes `DateConverter`.

## The five phases

**A — write the test.** Port the Java test to Go. Assertion values are copied
from the Java, never read off the Go. The implementation does not exist yet.

**B — port the implementation.** Write the Go from the Java source, line for
line. Do not look at what makes the test pass; look at what the Java does.

**C — run and fix.** `gofmt`, `go vet`, `go test`. A failure is a defect in the
port, not in the test.

**D — adversarial review.** 敵対的レビュー. Green tests prove the port passes
the tests, not that the migration is faithful.

**E — user feedback.** Stop. Wait. Judge each item before acting, and where it
needs fixing, write a strict failing test first.

**A to D run without stopping.** E1 is the only stop; finishing a task, a
phase, a package or a commit is not one.
