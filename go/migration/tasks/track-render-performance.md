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
- [ ] **Resampling that grows with the source image.** Not started; see Open.

Result, all three together: every one of the 2,256 pages is identical to the
build before, and the whole list renders in 67 seconds against 298.

| File | PDFBox | Go, before | Go, after |
| --- | --- | --- | --- |
| `AndroidPdfViewer`'s `sample.pdf`, 84 pages | 5.6 s | 12.0 s | 7.7 s |
| `pdfjs/issue8078.pdf`, 1 page of 222,868 strokes | 6.9 s | 59.3 s | 6.4 s |

Each is one run of one file, start to finish, on an otherwise idle machine:
`corpus -renderpages` for the Go version, `JavaCorpus ... render` for PDFBox,
whose time includes starting the JVM.

## Open

**The image path.** On `sample.pdf` drawing images is now 73% of the time, and
`x/image/draw`'s CatmullRom 49%: it widens its kernel by the shrink factor, so an
image drawn small costs its whole source. PDFBox does something else below half
size: `drawBufferedImage` takes `getScaledInstance(w, h, SCALE_SMOOTH)` -- the
JDK's `AreaAveragingScaleFilter` -- and draws that with bicubic. The Go
`PageDrawer` keeps `imageDownscalingOptimizationThreshold` and never reads it.
Porting it changes what is drawn, towards PDFBox, and means porting the chain
behind `getScaledInstance`, not only the filter: how `OffScreenImageSource`
sends each image type, `ColorModel.getRGB`, and the conversion back to an image
`drawImage` can use. The second is not neutral: PDFBox makes a one-bit
DeviceGray image -- a scan -- as `TYPE_BYTE_GRAY`, whose `getRGB` converts a
linear grey to sRGB, 128 to 188, and the filter averages what that answers.
That is a decision to take before the work, and it is waiting on one.
