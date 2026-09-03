# PDFBox module structure

"PDFBox" is twelve Maven modules, not one. This is what each does and how they
depend on each other, verified from the `pom.xml` files on `trunk` rather than
from documentation.

Per-package counts live in [`inventory.tsv`](inventory.tsv); this file is about
the modules above them.

## Library modules

| Module | Maven artifact | Role | Main files | Main lines |
| --- | --- | --- | ---: | ---: |
| `io` | `pdfbox-io` | Random-access byte plumbing — reading, writing, buffering, view slicing. **New as its own module in 3.0** | 18 | 3,967 |
| `fontbox` | `fontbox` | Font file parsing: TrueType, CFF/Type2, Type1, AFM, CMap, GSUB. Usable standalone | 143 | 28,638 |
| `pdfbox` | `pdfbox` | The PDF library proper — object model, parser, writer, filters, document model, content streams, text extraction, rendering | 623 | 136,475 |
| `xmpbox` | `xmpbox` | XMP metadata (Adobe's RDF/XML format) | 74 | 12,271 |
| `pdfbox-layout-awt` | `pdfbox-layout-awt` | Glyph layout backend over `java.awt` — bidi, ligatures, kerning | 2 | 436 |
| `pdfbox-layout-fop` | `pdfbox-layout-fop` | The same glyph layout, backed by Apache FOP | 3 | 541 |

## Application modules — out of scope for the port

| Module | What it is |
| --- | --- |
| `tools` | Command line utilities (26 files). Uses picocli |
| `debugger` | Swing GUI for inspecting PDFs (93 files) |
| `examples` | Standalone usage examples (105 files), documentation for the Java API |
| `benchmark` | JMH benchmarks |
| `app`, `debugger-app` | **Packaging only** — zero Java files, they assemble standalone jars |
| `parent` | Shared Maven configuration |

## Dependency graph

```
io                          no dependencies — the foundation
└── fontbox                 needs io
    └── pdfbox              needs io + fontbox (+ jbig2-imageio)
        ├── pdfbox-layout-awt
        ├── pdfbox-layout-fop
        ├── debugger ── tools
        └── examples

xmpbox                      standalone — nothing in the build depends on it
```

Two things worth knowing from this graph:

**`io` first is not a preference.** It is the only module with no dependencies,
so it is the only place a port can start.

**`xmpbox` is independent.** `pdfbox` does not depend on it — `PDMetadata`
hands back a raw stream and `xmpbox` parses it separately. So xmpbox can be
ported at any time, by anyone, in parallel with everything else. It is the one
part of this project with no ordering constraint at all.

## The layout modules are the pattern to copy

`pdfbox-layout-awt` and `pdfbox-layout-fop` implement the same thing —
`org.apache.pdfbox.glyphlayout` — against two different backends, and the caller
picks one. Their tests are named `GlyphLayoutBidiTest`,
`GlyphLayoutLigaturesAndKerningTest`, `GlyphLayoutDin91379Test`: complex script
shaping, which is deeply platform-dependent.

This matters beyond the 5 files involved. PDFBox already solves
"platform-dependent concern behind a swappable backend" in its own tree, and
that is exactly the shape proposed for rendering in slice 9 of
[`../PLAN.md`](../PLAN.md). When that slice needs a precedent for the interface
boundary, this is the in-repo one — no need to reach for PdfBox-Android.

## Consequence for the port

The module graph is a **constraint**, not a schedule. It says what cannot be
skipped: no fontbox without io, no pdfbox without fontbox at link time.

It is a poor unit of *work*, though. `pdfbox` alone is 623 files and 136k lines
— 73% of the whole project in a single bucket. "Port the pdfbox module" is not a
milestone anyone can act on or measure. And porting module-at-a-time would mean
finishing all 143 files of fontbox before a single PDF could be opened, since
opening a PDF needs no fonts at all.

See [`../PLAN.md`](../PLAN.md) for how the work is actually ordered.
