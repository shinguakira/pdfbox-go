// Package fixup holds the repairs a document has applied to it before it is
// read.
//
// Port of org.apache.pdfbox.pdmodel.fixup.
//
// Java reaches this package from PDDocumentCatalog.getAcroForm, which builds an
// AcroFormDefaultFixup. The port cannot: this package names PDAcroForm, so
// interactive/form, where getAcroForm lives, cannot import it back, and takes
// the constructor from the init below instead. Nothing else can import this
// package either, because everything in the graph below it is what the fixups
// repair. A program that wants the default fixup therefore has to link this
// package in itself:
//
//	import _ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fixup"
//
// Without it, form.AcroFormOfCatalog reads the form with no fixup applied,
// which is what getAcroForm(null) of Java does. See migration/STATUS.md.
package fixup

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fixup/processor"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// PDDocumentFixup is a repair applied to a document.
//
// Port of the interface PDDocumentFixup.
type PDDocumentFixup interface {
	// Apply performs the repair.
	Apply()
}

// AbstractFixup holds the document a fixup repairs.
//
// Port of the abstract class AbstractFixup. Java declares the field protected;
// the port keeps it unexported and the concrete fixups embed this.
type AbstractFixup struct {
	document *pdmodel.PDDocument
}

// initAbstractFixup is the protected AbstractFixup(PDDocument) constructor.
func (f *AbstractFixup) initAbstractFixup(document *pdmodel.PDDocument) {
	f.document = document
}

// AcroFormDefaultFixup is the repair PDDocumentCatalog.getAcroForm applies when
// it is not given another one.
//
// Port of AcroFormDefaultFixup.
type AcroFormDefaultFixup struct {
	AbstractFixup
}

var _ PDDocumentFixup = (*AcroFormDefaultFixup)(nil)

// NewAcroFormDefaultFixup returns the default fixup for the given document.
func NewAcroFormDefaultFixup(document *pdmodel.PDDocument) *AcroFormDefaultFixup {
	f := &AcroFormDefaultFixup{}
	f.initAbstractFixup(document)
	return f
}

// Apply ensures the form has its defaults, and builds the appearances of the
// widgets where there are none.
func (f *AcroFormDefaultFixup) Apply() {
	processor.NewAcroFormDefaultsProcessor(f.document).Process()

	// Get the AcroForm in it's current state.
	//
	// Also note: getAcroForm() applies a default fixup which this processor
	// is part of. So keep the null parameter otherwise this will end
	// in an endless recursive call
	acroForm := form.AcroFormOfCatalogFixup(f.document.DocumentCatalog(), nil)

	// PDFBOX-4985
	// build the visual appearance as there is none for the widgets
	if acroForm != nil && acroForm.NeedAppearances() {
		if len(acroForm.Fields()) == 0 {
			processor.NewAcroFormOrphanWidgetsProcessor(f.document).Process()
		}

		// PDFBOX-4985
		// build the visual appearance as there is none for the widgets
		processor.NewAcroFormGenerateAppearancesProcessor(f.document).Process()
	}
}

// init gives interactive/form the constructor of the default fixup, which its
// AcroFormOfCatalog applies. That package cannot import this one, because this
// one names PDAcroForm.
func init() {
	form.NewAcroFormDefaultFixup = func(document *pdmodel.PDDocument) form.DocumentFixup {
		return NewAcroFormDefaultFixup(document)
	}
}
