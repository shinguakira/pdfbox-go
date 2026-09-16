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
| `track/test-backfill` | [`track-test-backfill.md`](track-test-backfill.md) | **merged** |
| `track/font-embedding` | [`track-font-embedding.md`](track-font-embedding.md) | **merged** |
| `track/tools` | [`track-tools.md`](track-tools.md) | **merged** — 17 of 26, the 9 left are these last three tracks |
| `track/pdfbox-layout` | [`track-pdfbox-layout.md`](track-pdfbox-layout.md) | **merged** |
| `track/stale-deferrals` | [`track-stale-deferrals.md`](track-stale-deferrals.md) | **merged** |
| `track/imageio` | [`track-imageio.md`](track-imageio.md) | **merged** |
| `track/multipdf` | [`track-multipdf.md`](track-multipdf.md) | **merged** — also finished `Splitter` |
| `track/raster` | [`track-raster.md`](track-raster.md) | **merged** — the last of the port |
| `track/java-bug-fixes` | [`track-java-bug-fixes.md`](track-java-bug-fixes.md) | **merged** — not a port; it fixed in the Go what the port carried from the Java |
| `track/upstream-sync` | [`track-upstream-sync.md`](track-upstream-sync.md) | **merged** — what the Apache merge of 2026-09-07 changed, measured against the frozen snapshot |
| `track/performance` | — | **carried**, not merged: its two commits rode into `track/testdata-sources` when that branch reopened. `../PERFORMANCE-PLAN.md` is its record |
| `track/testdata-sources` | [`track-testdata-sources.md`](track-testdata-sources.md) | open — reopened 2026-09-15; where the test data stands, and the open tasks |

**The port is merged**, every slice and every track of it, bar the two
`track/performance` commits that ride in the open branch. What is open is
`track/testdata-sources`, which ports nothing: it collects other projects' test
PDFs, scores the port against PDFBox over them, and fixes what that finds.

## The order they were taken in

[`../BRANCHING.md`](../BRANCHING.md) is where the order lives, with the
dependency graph and the argument for each edge — the last four tracks in "The
last four tracks", the earlier ones in the branch table above it. Two things
about this list rather than about the branches:

- **A branch gets a row here when it gets a row there.** `PLAN.md` is not
  edited to add one; it counts the work, not the branches.
- **The contents of the last four came from an audit, not from the 891-class
  survey**, which missed `multipdf` outright. `../STATUS.md` carries that audit
  and the commands to re-run it. Its important half is not finding unported
  classes: it is checking whether the reason each recorded deferral gives is
  **still true**.

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
| `pdfbox/multipdf` | 6 | slice 7 (3), `track/multipdf` (the other 3, and `PDFCloneUtilityTest`) |
| `pdfbox/pdfparser` | 12 | slice 1 (6), slice 3 conditionally, slice 8 for `FDFParser` |
| `pdfbox/pdfparser/xref` | 6 | slice 1 — done |
| `pdfbox/pdfwriter` | 3 | slice 7 |
| `pdfbox/pdfwriter/compress` | 4 | slice 7 |
| `pdfbox/pdmodel` | 29 | slice 2 (4), slice 3 conditionally, slice 7, `track/test-backfill` (the `ResourceCacheFactory` trio), `track/raster` (`PDPatternContentStream`), `track/stale-deferrals` (`TestPDDocument`) |
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
| `pdmodel/graphics/blend` | 2 | slice 2 (1), `track/raster` (`BlendComposite`) |
| `pdmodel/graphics/color` | 23 | slice 2 (3), slice 9 (rest) |
| `pdmodel/graphics/form` | 3 | slice 9 |
| `pdmodel/graphics/image` | 9 | slice 6 |
| `pdmodel/graphics/optionalcontent` | 3 | slice 8 |
| `pdmodel/graphics/pattern` | 3 | slice 9 |
| `pdmodel/graphics/shading` | 37 | slice 9 (18), `track/raster` (the 19 contexts and paints) |
| `pdmodel/graphics/state` | 6 | slice 2 (4), slice 9 (2) |
| `pdmodel/interactive/*` | 144 | slice 8 |
| `pdfbox/printing` | 4 | slice 9 |
| `pdfbox/rendering` | 10 | slice 9 (6), `track/raster` (the 4 that make pixels) |
| `pdfbox/text` | 6 | slice 3 |
| `pdfbox/tools`, `tools/imageio` | 26 | `track/tools` (17), `track/imageio` (5), `track/multipdf` (2), `track/raster` (2) |
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

A to E — write the test, port the implementation, run and fix, adversarial
review, user feedback — are set out in [`TEMPLATE.md`](TEMPLATE.md), "How each
unit of work runs", and every branch file carries that copy. **A to D run
without stopping**; E1 is the only stop, and finishing a task, a phase, a
package or a commit is not one.
