# What everyone else does about the raster

Surveyed 2026-09-09 (JST), for `track/raster`'s A0 — the decision of what draws
pixels. `PLAN.md` cites PdfPig as the precedent for shipping a PDF library with
no renderer at all, and this is the check of that citation: what PdfPig
actually did, and what the alternatives look like outside Java.

**The short version: Java is the outlier.** `java.awt.Graphics2D` ships in the
JDK, so PDFBox draws a page without writing or binding a rasteriser. No other
ecosystem in this survey has that. Everywhere else the answer is *bind a C or
C++ library, or write it yourself* — which is exactly the choice A0 had to
make, and it is not a Go problem.

---

## PdfPig (.NET)

**The core library renders nothing.** `UglyToad.PdfPig` reads, writes and
extracts; there is no page-to-image API in it.

**Rendering is a separate package, and it binds Skia.**
[`PdfPig.Rendering.Skia`](https://github.com/BobLd/PdfPig.Rendering.Skia) is by
BobLd, a PdfPig maintainer, and its own description is "Cross-platform library
to render pdf documents as images with `PdfPig` using `SkiaSharp`". SkiaSharp
is a managed binding over Skia, Google's C++ 2D library — the same library
Chrome and Android draw with. It is not a managed rasteriser.

Two things about it are worth recording:

- The author calls it **"very early stage"** and says **"not everything is
  supported"** in the discussion where it was announced.
- Its **expected images are platform specific**: the README says Windows, Linux
  and macOS do not produce identical output, so the visual regression tests
  keep a set per operating system. A renderer built on a native library
  inherits that library's platform differences, and cross-platform pixel
  identity is not available even to it.

So the precedent `PLAN.md` cites is real — PdfPig shipped for years with no
renderer and became a standard choice anyway — and it is now a precedent for
something else as well: **when PdfPig did get a renderer, it got one by binding
C++.**

## .NET has no built-in Graphics2D either, and lost the one it had

`System.Drawing.Common` is GDI+, and outside Windows it was implemented by
`libgdiplus`, a native re-implementation of the Windows API.

- **.NET 6** made `System.Drawing.Common`
  [Windows-only](https://learn.microsoft.com/en-us/dotnet/core/compatibility/core-libraries/6.0/system-drawing-common-windows-only),
  with an `EnableUnixSupport` switch to opt back in.
- **.NET 7**
  [removed the switch](https://learn.microsoft.com/en-us/dotnet/core/compatibility/core-libraries/7.0/system-drawing).

Microsoft's stated reason is that `libgdiplus` was *"unmaintainable and
incompatible with .NET's quality standards"*, and non-Windows bugs would not be
fixed. The migration guidance names four alternatives: **SkiaSharp** (native),
**ImageSharp** (tiered licence — commercial for some uses), **Aspose.Drawing**
(commercial), and **Microsoft.Maui.Graphics** (native).

**There is no free, pure-managed, cross-platform 2D drawing API in .NET.** A
.NET PDF renderer either binds a native library or buys a licence.

## Go

The same shape, measured for A0 and written up in
[`STATUS.md`](STATUS.md) under "Track `raster` — A0, what draws":

| | fill | stroke: width, cap, join, dash | arbitrary clip | PDF's 16 blend modes | groups, soft masks |
| --- | :-: | :-: | :-: | :-: | :-: |
| Cairo, via cgo | yes | yes | yes | **yes** | **yes** |
| `srwiley/rasterx` | yes | **yes** | rectangle only | no | no |
| `fogleman/gg` | yes | **no miter join** | yes, via mask | no | no |
| `x/image/vector` | yes | no | no | no | no |

Cairo covers all of it because Cairo was built to implement the PDF and SVG
imaging model: its `OPERATOR_MULTIPLY` through `OPERATOR_HSL_LUMINOSITY` *are*
PDF's sixteen blend modes, and `cairo_push_group` is the transparency group.
That is the same bargain SkiaSharp offers .NET, and it is barred here for the
same reason cgo is barred everywhere in this port.

**No pure-Go library has PDF's blend modes, transparency groups or soft masks**,
and none is likely to: they are not something a general 2D library does. That
part is written by hand whatever else is chosen.

## What this port does, and why it is not a worse position

`track/raster` builds on `srwiley/rasterx` — one dependency,
`golang.org/x/image` — for the fills and the stroke model, and writes the
compositor, the clip and the transparency groups over the `blend.BlendMode`
the port already has. About 600 lines with no Java to port from.

Set against the survey, that is the ordinary cost of the job rather than a
penalty for choosing Go:

- **PDFBox** gets it free, from the JDK. Nobody else does.
- **PdfPig** pays for it with a native dependency, an early-stage renderer and
  per-platform expected images.
- **This port** pays for it with ~600 lines of Go and keeps a pure-Go build,
  cross-compilation, and pixel output that does not depend on which machine
  ran it.

The last of those is worth more here than it looks: `checkRenderIdent` and the
deferred pixel comparisons render *both* sides through this port's own
backend, so they are exact. A renderer whose output varies by operating system
could not make that comparison at all — which is the position
`PdfPig.Rendering.Skia` documents itself as being in.

## Sources

- [PdfPig.Rendering.Skia](https://github.com/BobLd/PdfPig.Rendering.Skia) —
  README: "Cross-platform library to render pdf documents as images with
  `PdfPig` using `SkiaSharp`"; platform-specific expected images.
- [PdfPig discussion #783](https://github.com/UglyToad/PdfPig/discussions/783) —
  the package's announcement by a PdfPig maintainer; "very early stage", "not
  everything is supported".
- [Breaking change: System.Drawing.Common only supported on Windows (.NET 6)](https://learn.microsoft.com/en-us/dotnet/core/compatibility/core-libraries/6.0/system-drawing-common-windows-only)
- [Breaking change: System.Drawing.Common config switch removed (.NET 7)](https://learn.microsoft.com/en-us/dotnet/core/compatibility/core-libraries/7.0/system-drawing)

The Go half of the table was measured rather than read: the four libraries were
fetched into a scratch module and their sources read. `go/pdfbox/rendering/
raster/rasterx_test.go` is what came out of it and pins the behaviour relied on
here.
