package tools

// The commands this port does not build, and what each waits for.
//
// B10. Java's `tools` module has 26 main files. This package ports 23 of them,
// the dispatcher among them, and the three below are missing.
//
// The count was "18 of them plus the dispatcher" until track/imageio,
// track/multipdf and track/raster were planned and the classes were counted
// against the list below: it was 17 and 9 then, and it is 23 and 3 now.
//
// track/raster took `PDFToImage`, which is the whole of what a Backend was
// wanted for: load, render each page, write it out. It did not take `PrintPDF`,
// and the row below now says what that one is really waiting for.
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
// **One waits for a printing system, and it is not the raster.**
// `rendering/raster` draws a page into pixels now, and `PDFPrintable` and
// `PDFPageable` -- everything PDFBox computes about where a page lands on a
// sheet -- were ported by slice 9 and finished by `track/raster`. What
// `PrintPDF` needs on top of that is `java.awt.print.PrinterJob` and
// `javax.print`: enumerating the printers on the machine, choosing one,
// reading the trays and media sizes it offers, showing the print dialog, and
// handing it a job. Go's standard library has none of it, and no pure-Go
// library does either -- printing is per-platform spooler API. Rendering the
// page and writing a file is what `render` does; sending it to a printer is
// the part that is missing.
//
// **Two wait for `multipdf`.** That is not the raster: `PDFMergerUtility`,
// `LayerUtility` and `Overlay` were deferred by slice 7 to slice 8, slice 8
// never took them, and the coverage survey counted them as ported because their
// names appear in a comment saying they are absent. `track/multipdf` claims
// them now.
var NotBuiltCommands = []NotBuilt{
	{Java: "PrintPDF", Name: "print",
		Waiting: "a printing system: Go has no PrinterJob, and no pure-Go library has one"},
}
