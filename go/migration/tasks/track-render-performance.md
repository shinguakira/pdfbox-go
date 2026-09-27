# Implementation Plan

Track — rendering speed. Issue #40: the Go renderer was 2 to 10 times slower
than PDFBox, which is why the render comparison ran over every sixteenth file.

**Branch: `track/render-performance`** — from and back to `migration-base`.

## Rules — do not break these

- **NEVER change the Java.** It is the reference.
- **Nothing that is drawn changes**, except where a change brings the Go version
  closer to PDFBox, and then only test-first with PDFBox's output as the
  expected value.
- **Every change is held to the renders before it**: the same list of files,
  rendered by the build before the change and the build after, every page's
  digest compared.

## How a change is checked

`cmd/corpus -renderpages` over every nineteenth file of
`go/testdata/oracle/list.txt` -- 998 files, 2,256 pages, 33 of which fail the
same way on both sides -- once with the build before the change and once with
the build after, `-workers 4 -timeout 0`, and the `exact` column of the two
tables joined page by page. The rendering tests pin pixel counts against
Java2D and PDFBox and must not move.

## Scope

The three causes the profiles named in #40:

- [x] **A coverage buffer the size of the page for every fill and stroke.**
  `coverageOf` and `outlineRecorder.rasterize` now make a mask over what the
  shape reaches, and borrow a rasteriser that keeps its row index between
  shapes. The shape is rasterised where it is: freetype splits a cubic by
  `a-3(b+c)+d`, which is not translation invariant, so moving a shape to its
  mask's corner changed 646 of the 2,256 pages; and it finds a pixel with a
  truncating division, so a shape between -1 and 0 still draws on row or column
  0, which a first version of the bound missed on 2 pages.
  `TestACoverageMaskHoldsWhatTheWholeSurfaceWould`.
- [x] **Two heap allocations a composited pixel.** `blendInto`'s arrays escaped
  because the nonseparable branch handed out slices of them;
  `blendNonSeparable` hands out the backend's own scratch.
- [x] **The arithmetic of an opaque pixel in Normal.** `blendInto` replaces the
  pixel, which is what the arithmetic comes to for every byte;
  `TestAnOpaqueNormalCompositeIsTheSource` runs all 16.7 million cases. Normal
  also no longer calls its channel function to answer the source.
- [x] **Resampling that grows with the source image.** `x/image/draw`'s
  CatmullRom widens its kernel by the shrink factor, so an image drawn small
  cost its whole source, and it drew a different picture from PDFBox's. Images
  are now drawn as Java2D draws them: `transformhelper.go` is `DrawImage`'s
  copy-or-transform choice and `TransformHelper.c`'s bicubic, fixed-point walk
  and edge rules included, and `areaaverage.go` is PDFBox's shrink below half
  size -- `getScaledInstance(SCALE_SMOOTH)`, which is `AreaAveragingScaleFilter`
  fed as `OffScreenImageSource` feeds it, a `TYPE_BYTE_GRAY` image through
  `getRGB`'s linear to sRGB table. `PageDrawer` now tells the backend the
  threshold and `KEY_RENDERING`. Eight Java2D image cases in `java2d_test.go`,
  six probed shrinks in `areaaverage_test.go`, and `downscale.pdf` against
  PDFBox's render of it.

Results. The first three changed nothing that is drawn: every one of the 2,256
pages is identical to the build before them. The fourth changes what images
look like, towards PDFBox: on `downscale.pdf` 76 pixels are one level out of
PDFBox's, where 5752 were out by as much as 146, and over the 998 files 179
pages are PDFBox's to the last bit, where 174 were, with no page in a worse
bucket.

| File | PDFBox | Go, before | Go, after |
| --- | ---: | ---: | ---: |
| the 998 files, 4 workers | -- | 298 s | 42 s |
| `AndroidPdfViewer`'s `sample.pdf`, 84 pages | 5.6 s | 12.0 s | 5.2 s |
| `pdfjs/issue8078.pdf`, 1 page of 222,868 strokes | 6.9 s | 59.3 s | 5.6 s |
| `pdfjs/ecma262.pdf`, 258 pages | 9.3 s | 85 s | 9.7 s |
| `itextsharp`'s `readCompressedPdfTest1.pdf`, 6 pages | 4.8 s | 203 s | 7.1 s |
| `itextsharp`'s `cmp_copyLargeFile.pdf`, 958 pages | 26.7 s | 379 s | 49.5 s |

Each is one run of one file, start to finish, on an otherwise idle machine:
`corpus -renderpages` for the Go version, `JavaCorpus ... render` for PDFBox,
whose time includes starting the JVM. The "before" of the last three is the
run `TESTDATA.md` records, after the compositor fix of 2026-09-18.

**One port defect came out of it.** Java2D's fixed-point walk made visible a
transform the old sampling blurred: `geom.Path2D.Bounds2D` took a
single-precision path's width in double where `Path2D.Float` subtracts in
float, and `processAnnotation` divides by that width, so an annotation's
appearance was drawn through a transform a last bit off. `PDAcroFormFlattenTest`
had pinned it: `Signed-Document-1.pdf` differed in 2 pixels after flattening,
256 once images were drawn as Java2D draws them, and 0 with the bounds fixed.
`TestAFloatPathsBoundsAreFloat`.

## Open

- **Compositing a partly transparent image.** The sampling is Java2D's, but the
  result goes onto the page through the backend's compositing, in floats, where
  Java2D's SrcOver MaskBlit is MUL8 bytes: `imageAlpha` is 30 pixels 2 levels
  out, `downscale.pdf`'s masked image 76 pixels 1 level.
- ~~**TexturePaintContext's walk of a texture.**~~ Ported, with where Java2D
  asks for each rectangle; see STATUS.md, "TexturePaintContext's walk".
  `softmaskimage.pdf` went from 3884 pixels out, 433 far out, to 2983 and 0.
- ~~**Java2D's non-antialiased fill.**~~ Ported: the two native fillers, which
  of them the stroke state picks, and the point each samples; see STATUS.md,
  "Java2D's fill with antialiasing off". The scaled tiling fixture fell from
  1250 pixels out, 600 far out, to 1008 and 370, and what is left of it is
  `JAVA-BUGS.md` 85 alone.
- **A stroke drawn with antialiasing off.** Java2D hands a thin one to
  `doDrawPath`, a line algorithm of its own; the port fills the outline. Only
  a page rendered with antialiasing off throughout reaches it, since PDFBox
  turns it off for fills alone.
- **The clip as a Region.** Java2D's clip is a `Region` built by
  `ShapeSpanIterator`, curves flattened within a pixel and pixels held whole;
  the port's is antialiased coverage of the exact curve. Along a curved clip
  it moves where a texture's rectangles begin, and it is how every clipped
  pixel differs.
- **Nearest neighbour.** An image scaled up with `/Interpolate false` is still
  sampled by `x/image/draw`; Java2D's is ScaledBlit, which this branch did not
  port.
- **The two flatten differences left**, `PDFBOX-4955.pdf` and
  `PDFBOX-5225.pdf`, may have a cause like `Signed-Document-1.pdf`'s.
