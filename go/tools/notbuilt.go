package tools

// The commands this port does not build, and what each waits for.
//
// B10. Java's `tools` module has 26 main files. This package ports 22 of them,
// the dispatcher among them, and the four below are missing.
//
// The count was "18 of them plus the dispatcher" until track/imageio,
// track/multipdf and track/raster were planned and the classes were counted
// against the list below: it was 17 and 9 then, and it is 22 and 4 now.
//
// track/imageio took five of the nine: the four `tools/imageio` classes and
// `ExtractImages`, which never imported `rendering` -- it walks the content
// stream and writes what it finds, and the port already decoded an embedded
// image to pixels. It was on this list for the imageio that writes them out.
//
// `Encrypt` was on this list a third time, for -certFile, and is not any more:
// track/stale-deferrals found that PublicKeySecurityHandler reported its
// encrypting half unported for a reason -- the writer of slice 7 -- that had
// stopped being true, and wrote the CMS encoder over the RC2 cipher and ASN.1
// structures the decrypting side already had.

// NotBuilt is one command that is not here.
type NotBuilt struct {
	// Java is the class, and Name the subcommand it would answer to, where it
	// has one.
	Java string
	Name string

	// Waiting is what has to exist first.
	Waiting string
}

// NotBuiltCommands is the list, so that the dispatcher's help and STATUS.md
// cannot drift apart from each other.
//
// **Two wait for a raster.** `rendering.Backend` is the interface slice 9
// defined for drawing a page into pixels, and nothing implements it. What Go
// draws with is a design decision outside this branch, and it is named rather
// than taken in passing.
//
// **Two wait for `multipdf`.** That is not the raster: `PDFMergerUtility`,
// `LayerUtility` and `Overlay` were deferred by slice 7 to slice 8, slice 8
// never took them, and the coverage survey counted them as ported because their
// names appear in a comment saying they are absent. `track/multipdf` claims
// them now.
var NotBuiltCommands = []NotBuilt{
	{Java: "PDFToImage", Name: "render", Waiting: "a rendering.Backend implementation"},
	{Java: "PrintPDF", Name: "print", Waiting: "a rendering.Backend implementation"},
}
