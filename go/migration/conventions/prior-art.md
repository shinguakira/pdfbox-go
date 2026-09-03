# Prior art: how PDFBox has been ported before

Two attempts to make PDFBox available outside the JVM are worth knowing about.
Neither is a Go port — as far as this research found, there is none.

Their value here is uneven, and worth saying up front: **PdfPig and the 3.0
migration guide bear directly on decisions in this port. The IKVM story mostly
does not.** It is recorded because it is the standing evidence for one question
that will keep being asked — why port at all instead of wrapping the Java? — not
because its technique is one we could have chosen.

## PDFBox .NET, via IKVM — evidence about wrapping, not about translating

The Apache project itself shipped a .NET build path. It was not a port: IKVM.NET
recompiled the Java bytecode into .NET IL and supplied a JVM and Java class
library implementation on top of the CLR.

**This has no Go analogue and never could.** IKVM was possible because the JVM
and the CLR are both managed bytecode VMs with near-identical object models —
class loading, GC, reflection, the lot. Go compiles to native code and has none
of that surface. There is no Go IKVM, and "we might be tempted to transpile Java
to Go" is not a real fork in the road; nothing usable exists to be tempted by.
So do not read this section as a caution against mechanical translation. That
option was never on the table.

How it died, in the order it happened:

- **The build never joined the main build.** `ant build.NET` only; the
  [Apache guide](https://svn.apache.org/repos/asf/pdfbox/site/publish/userguide/dot_net.html?p=1400000)
  states plainly that Maven does not support it. A build path outside the
  project's real build is a build path nobody runs.
- **It pinned a specific tool version.** The guide requires IKVM 0.42 exactly —
  "our build script is version-specific" — and says it will not work with 0.38
  or earlier.
- **Binary coupling propagated.** Rebuild a dependency and you had to rebuild
  PDFBox against it, because the produced DLLs were only valid against the exact
  DLLs they were built from.
- **The foundation went away.** IKVM's original author discontinued it in 2015.
  The [`Pdfbox-IKVM` NuGet package](https://www.nuget.org/packages/Pdfbox-IKVM)
  was last published in 2017.

**What actually transfers.** Note what killed it: not the translation technique,
which worked fine for years. It was the packaging around it — a second build
system, a pinned toolchain version, binary coupling, and an abandoned bridge
layer. Those are properties of *shipping someone else's runtime alongside your
own*, and they apply to any wrap-instead-of-port approach regardless of how the
wrapping is done.

That question does have live Go-shaped answers, and someone will eventually
raise one of them rather than face 186k lines:

- cgo into a real JVM over JNI
- GraalVM `native-image` compiling PDFBox to a shared library behind a C ABI,
  called via cgo
- Java compiled to WASM and run in an in-process Go runtime such as wazero

None of these are transpilation, and none should be dismissed by analogy — they
are legitimate engineering options with different trade-offs, and this research
did not establish whether anyone has made one work for PDFBox specifically. What
the IKVM history supplies is the failure mode to check each of them against:
does it need a build nobody runs, does it pin a toolchain version, does it
couple binaries, and who maintains the bridge in five years? The .NET attempt
answered those badly on all four counts and did not survive them.

The reason this port is hand-written Go is narrower than "wrapping is wrong":
a cgo or WASM binding cannot be read, debugged, or fixed by a Go developer
holding a broken PDF, and every one of the four questions above lands on us
rather than upstream.

## PdfPig, a hand-written C# port — the route that worked

[PdfPig](https://github.com/UglyToad/PdfPig) states it "started as an effort to
port PDFBox to C#". It is Apache 2.0, actively maintained, and has over 21
million NuGet downloads. That is the outcome the IKVM route did not get, and the
approach was the opposite one: humans reading Java and writing C#.

Its structural decisions matter to us because it faced the same fork in the road.

### How it actually started: a literal transliteration, deleted 10 weeks later

There is no porting document — not in the README, not in the author's writing,
not in the 12-page wiki. The git history is the documentation, and it is worth
more than a document would have been.

The first commit, 2017-11-09, is 149 files and 22,449 lines, and its README says
in full: *"Convert the [PdfBox](https://github.com/apache/pdfbox) code to C#."*
What landed was a near-literal transliteration of PDFBox's COS layer —
`Cos/CosBase.cs`, `CosArray.cs`, `CosDictionary.cs`, `CosName.cs`, `CosInt.cs`,
`CosFloat.cs`, `CosStream.cs`, `ICosVisitor.cs`, and a `COSDocument.cs` that
still carried the Java capitalisation.

Then, over 157 commits:

| Date | What happened |
| --- | --- |
| 2017-11-09 | Literal COS transliteration imported |
| 2017-11-12 | Day 3: a **parallel** token model started — *"seems like the approach should be valid"* |
| Nov–Dec 2017 | More PDFBox parsers ported (cmap, AFM, encodings, fonts), but routed through the new tokens |
| 2017-12-22 | *"heavy duty refactoring to inject dependencies rather than god object"* |
| Jan 2018 | Piecewise cutover: *"move cosboolean to pdfboolean"*, *"migrating cross reference parsing to token scanner"*, *"remove cos object key completely"* |
| 2018-01-21 | *"remove all old cos objects"* — the transliterated layer deleted entirely |

Four things this tells us that no prose document would have:

1. **The literal port was scaffolding, not the destination.** It bought a
   working parser quickly and was thrown away once something better existed.
2. **The replacement began on day 3, not after the port was finished.** The two
   models coexisted for ten weeks while subsystems moved across one at a time.
3. **The algorithms were kept; the object model was not.** Commits say *"start
   porting the afm parser from pdf box"* and *"port most encoding classes from
   pdfbox"* — those came across. `COSBase` and its visitor did not.
4. **Every step carried tests.** *"with tests"* appears in commit after commit,
   during a period when the thing under test was being replaced underneath.

### What it kept from PDFBox, and what it dropped

### It abandoned PDFBox's package layout entirely

PdfPig organises by functional domain rather than mirroring the Java tree:
`AcroForms`, `Actions`, `Annotations`, `Content`, `CrossReference`, `Encryption`,
`Filters`, `Functions`, `Geometry`, `Graphics`, `IO`, `Images`, `Logging`,
`Outline`, `Parser`, `PdfFonts`, `Polyfills`, `Resources`, `Tokenization`,
`Util`, `Writer`, `XObjects`.

Three things stand out:

- **The `cos` / `pdmodel` split is gone.** PDFBox separates the low-level object
  model from the typed document model; PdfPig does not.
- **FontBox is not a separate module.** It is folded in as `PdfFonts`.
- **There is no rendering namespace.** They did not port PDFBox's Java2D
  rendering at all.

### Where this port disagrees, and why

**We mirror the Java package tree; PdfPig did not.** This is a real trade, and
PdfPig's choice is the more ergonomic one for the target language. We are taking
the other side deliberately:

PDFBox is not a finished artifact. Per `AGENTS.md`, `3.0` and `2.0` are both
actively maintained, security fixes land in both, and `trunk` is where new work
goes. A port that mirrors the upstream layout can diff a Go file against the
Java file it came from, and can absorb an upstream fix by finding the one place
it belongs. A port that has reorganised cannot do either without a translation
step in someone's head. For a library whose value is two decades of absorbed
PDF-format breakage, staying diffable against upstream is worth more than a
tidier tree.

The cost is real and should be expected: parts of the mirrored layout will read
as Java-shaped in Go. `pdmodel/interchange/logicalstructure` is not a package
path anyone would choose from scratch. We accept that.

**There is a precedent on each side of this.** PdfPig reorganised and is
thriving. PdfBox-Android kept the package layout as a flat namespace rename and
can therefore rebase onto upstream releases — and is still pinned to 2.0.27
while upstream ships 3.0.7. So the evidence says mirroring buys the *ability* to
track upstream, not the act of doing it; someone still has to do the rebase.

**And the git history sharpens it further: PdfPig's divergence was not a day-one
decision.** They mirrored first and diverged over ten weeks, once they had
something working to diverge from. That is a materially different claim than
"they chose a different layout", and it is the one the evidence supports.

This leaves an open question for phase 1, which should be decided deliberately
rather than by inertia:

- **Mirror COS and keep it.** Diffable against upstream forever. But no
  successful PDFBox port has done this — PdfPig deleted its COS layer, and
  PdfBox-Android only keeps its layout by staying a major version behind.
- **Mirror COS as scaffolding, expect to replace it.** What PdfPig actually did.
  Gets a parser working fast, at the cost of writing the object model twice.
- **Design the Go object model up front.** Skips the rewrite, but gives up the
  bootstrap and commits to a design before anything parses a real PDF.

`COSBase` is where this bites hardest: it is an abstract class with a visitor
over a mutable, reference-identity object graph, and Go has neither inheritance
nor a natural visitor. The awkwardness PdfPig felt in C# will be worse in Go.

The current plan assumes option 1. That assumption is now known to be the one
without precedent, and should be revisited before phase 1 starts rather than
discovered ten weeks in.

*Caveat:* this research did not establish whether PdfPig still tracks upstream
PDFBox or has permanently diverged — their documentation does not say, and the
question was not answerable from public sources.

### Independent confirmation on rendering

PdfPig skipping Java2D rendering is the second data point saying that phase 6 of
[`../PLAN.md`](../PLAN.md) is the genuinely hard part, and that a useful PDF
library can exist without it. Text extraction, parsing and document manipulation
carried a project to 21 million downloads with no renderer. If phase 6 needs to
be cut or deferred indefinitely, there is precedent that the result is still
worth having.

## PdfBox-Android — the closest precedent to phase 6

[PdfBox-Android](https://github.com/TomRoush/PdfBox-Android) is a fork of PDFBox
made to run on Android, where `java.awt` does not exist. That is the same wall
phase 6 of [`../PLAN.md`](../PLAN.md) runs into, and this project is the only
one found that actually got past it.

**How they split the problem.** Not by reimplementing Java2D, and not by
dropping rendering. They split it in two:

- **Geometry was vendored.** `AffineTransform` lives in the tree as
  `com.tom_roush.harmony.awt.geom.AffineTransform` — taken from Apache Harmony,
  the clean-room Java SE implementation. It is pure math with no rasteriser
  behind it, so it ports as ordinary code.
- **Rasterisation was delegated to the platform.** Files under
  `com.tom_roush.pdfbox.rendering` — `PageDrawer`, `TTFGlyph2D` — import
  `android.graphics.Path` alongside that vendored `AffineTransform`. Paths,
  canvas and bitmaps come from Android; PDFBox's drawing logic sits on top.

**This is independent confirmation of option 3 in phase 6.** The plan already
proposed porting the geometry and putting the raster backend behind an
interface. PdfBox-Android did exactly that split against a real 2D API and
shipped it. The lesson is that the boundary is a real seam in PDFBox's design,
not a hopeful one — `AffineTransform` and the path-construction math separate
cleanly from the pixels.

For Go this maps to: port the geometry into `pdfbox/util` as plain Go, define a
narrow raster interface at the `PageDrawer` boundary, and let a backend supply
paths and compositing. Harmony's `AffineTransform` is Apache 2.0 and is a
reasonable reference for the geometry, the same as it was for them.

**The other lesson is about drift.** PdfBox-Android is "currently based on
PDFBox v2.0.27" while upstream ships 3.0.7 and develops on `trunk`. It kept the
package structure — a flat rename of `org.apache.pdfbox.*` to
`com.tom_roush.pdfbox.*` — which is what makes rebasing onto a new upstream
release possible at all. It is still a major version behind. Mirroring makes
tracking upstream feasible; it does not make it free.

Note also that `android-awt`, the general-purpose "replace java.awt for Android"
library built from Harmony and Commons Imaging, was archived in July 2024. The
narrow, vendored-what-you-need approach outlived the general one.

## PDFBox's own 2.0 to 3.0 migration guide

The [3.0 migration guide](https://pdfbox.apache.org/3.0/migration.html) is not
about porting to another language, but it tells us which parts of the API the
project itself considers unsettled — which is exactly what a port should avoid
freezing prematurely.

Points that bear directly on our plan:

- **`pdfbox-io` is new in 3.0.** The IO classes were extracted into their own
  Maven module, moving from scratch files to `java.nio` with memory-mapped and
  buffered file access. Phase 0 therefore ported the *newest* architecture in the
  codebase, not legacy — and `RandomAccessRead` / `RandomAccessStreamCache` are
  deliberate recent design, which is why they were worth following closely.
- **The scratch-file machinery deferred in phase 0 is current, not legacy.**
  `MemoryUsageSetting` and the memory-mapped reader belong to this same redesign.
- **Loading moved to a `Loader` class.** All load methods were removed from
  `PDDocument`. This maps cleanly onto Go: package-level `pdfbox.Open(path)`
  rather than constructors, and we should adopt the 3.0 shape rather than the
  2.0 one.
- **Standard 14 fonts moved from static instances to a `Standard14Fonts.FontName`
  enum.** Port the enum form.
- **Unstable areas named by the project:** reader/writer infrastructure, font
  instantiation, colour operation signatures, the CLI, and incremental parsing.

**Rule this adds:** port against `trunk` / 3.0 API shapes, and never port a
member that 3.0 deprecated or removed. Check the migration guide before porting
any class it names.

## Sources

- [PDFBox .NET build guide (Apache SVN)](https://svn.apache.org/repos/asf/pdfbox/site/publish/userguide/dot_net.html?p=1400000)
- [`Pdfbox-IKVM` on NuGet](https://www.nuget.org/packages/Pdfbox-IKVM)
- [IKVM (Wikipedia)](https://en.wikipedia.org/wiki/IKVM)
- [PDFBox in .NET, Square PDF](http://www.squarepdf.net/pdfbox-in-net)
- [PdfPig](https://github.com/UglyToad/PdfPig) and [its documentation](https://uglytoad.github.io/PdfPig/)
- [PDFBox 3.0 migration guide](https://pdfbox.apache.org/3.0/migration.html)
