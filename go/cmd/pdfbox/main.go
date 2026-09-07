// Command pdfbox is the command line utilities of PDFBox.
//
// Port of the `main` of org.apache.pdfbox.tools.PDFBox, which is the main class
// of the runnable standalone PDFBox jar. Everything it dispatches to is in
// go/tools; this file is only the entry point, so that the commands stay
// testable as a library.
package main

import (
	"os"

	"github.com/shinguakira/pdfbox-go/go/tools"
)

func main() {
	// Java sets apple.awt.UIElement here to suppress the Dock icon on OS X,
	// which is a JVM property with no counterpart.
	os.Exit(tools.NewPDFBox(os.Stdout, os.Stderr).Run(os.Args[1:]))
}
