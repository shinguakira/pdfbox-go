// Package processor holds the individual steps a document fixup is built from.
//
// Port of org.apache.pdfbox.pdmodel.fixup.processor.
package processor

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// PDDocumentProcessor is one step of a document fixup.
//
// Port of the interface PDDocumentProcessor.
type PDDocumentProcessor interface {
	// Process performs the step.
	Process()
}

// AbstractProcessor holds the document a processor works on.
//
// Port of the abstract class AbstractProcessor. Java declares the field
// protected; the port keeps it unexported and the concrete processors embed
// this.
type AbstractProcessor struct {
	document *pdmodel.PDDocument
}

// initAbstractProcessor is the protected AbstractProcessor(PDDocument)
// constructor.
func (p *AbstractProcessor) initAbstractProcessor(document *pdmodel.PDDocument) {
	p.document = document
}

// AcroFormDefaultsProcessor verifies and ensures the default resources of the
// form:
//
//   - a default appearance string is defined
//   - default resources are defined
//   - Helvetica as /Helv and Zapf Dingbats as /ZaDb are included. ZaDb is
//     required for most check boxes and radio buttons
//
// Port of AcroFormDefaultsProcessor.
type AcroFormDefaultsProcessor struct {
	AbstractProcessor
}

var _ PDDocumentProcessor = (*AcroFormDefaultsProcessor)(nil)

// NewAcroFormDefaultsProcessor returns the processor for the given document.
func NewAcroFormDefaultsProcessor(document *pdmodel.PDDocument) *AcroFormDefaultsProcessor {
	p := &AcroFormDefaultsProcessor{}
	p.initAbstractProcessor(document)
	return p
}

// Process gives the form its default appearance string and resources.
func (p *AcroFormDefaultsProcessor) Process() {
	// Get the AcroForm in it's current state.
	//
	// Also note: getAcroForm() applies a default fixup which this processor
	// is part of. So keep the null parameter otherwise this will end
	// in an endless recursive call
	acroForm := form.AcroFormOfCatalogFixup(p.document.DocumentCatalog(), nil)
	if acroForm != nil {
		verifyOrCreateDefaults(acroForm)
	}
}

// verifyOrCreateDefaults checks that there are default entries for the required
// properties, and creates entries like those of Adobe Reader and Adobe Acrobat
// where they are missing. Java declares it private.
func verifyOrCreateDefaults(acroForm *form.PDAcroForm) {
	const adobeDefaultAppearanceString = "/Helv 0 Tf 0 g "

	// DA entry is required
	if acroForm.DefaultAppearance() == "" {
		acroForm.SetDefaultAppearance(adobeDefaultAppearanceString)
		acroForm.Dictionary().SetNeedToBeUpdated(true)
	}

	// DR entry is required
	defaultResources := acroForm.DefaultResources()
	if defaultResources == nil {
		defaultResources = pdmodel.NewPDResources()
		acroForm.SetDefaultResources(defaultResources)
		acroForm.Dictionary().SetNeedToBeUpdated(true)
	}

	// PDFBOX-3732: Adobe Acrobat uses Helvetica as a default font and
	// stores that under the name '/Helv' in the resources dictionary
	// Zapf Dingbats is included per default for check boxes and
	// radio buttons as /ZaDb.
	// PDFBOX-4393: the two fonts are added by Adobe when signing
	// and this breaks a previous signature. (Might be an Adobe bug)
	fontDict := defaultResources.Dictionary().GetCOSDictionary(cos.Font)
	if fontDict == nil {
		fontDict = cos.NewDictionary()
		defaultResources.Dictionary().SetItem(cos.Font, fontDict)
	}
	if !fontDict.ContainsKey(cos.Helv) {
		helvetica, err := font.NewPDType1FontStandard14(font.Helvetica)
		if err != nil {
			// Java's PDType1Font(FontName) constructor declares no exception:
			// the metrics of a standard 14 font are on the class path, and it
			// throws IllegalArgumentException where they are not.
			panic(err)
		}
		defaultResources.PutFont(cos.Helv, helvetica)
		defaultResources.Dictionary().SetNeedToBeUpdated(true)
		fontDict.SetNeedToBeUpdated(true)
	}
	if !fontDict.ContainsKey(cos.ZaDb) {
		zapfDingbats, err := font.NewPDType1FontStandard14(font.ZapfDingbatsFontName)
		if err != nil {
			panic(err)
		}
		defaultResources.PutFont(cos.ZaDb, zapfDingbats)
		defaultResources.Dictionary().SetNeedToBeUpdated(true)
		fontDict.SetNeedToBeUpdated(true)
	}
}

// AcroFormGenerateAppearancesProcessor draws the appearance of every field of
// the form.
//
// Port of AcroFormGenerateAppearancesProcessor.
type AcroFormGenerateAppearancesProcessor struct {
	AbstractProcessor
}

var _ PDDocumentProcessor = (*AcroFormGenerateAppearancesProcessor)(nil)

// NewAcroFormGenerateAppearancesProcessor returns the processor for the given
// document.
func NewAcroFormGenerateAppearancesProcessor(
	document *pdmodel.PDDocument) *AcroFormGenerateAppearancesProcessor {
	p := &AcroFormGenerateAppearancesProcessor{}
	p.initAbstractProcessor(document)
	return p
}

// Process refreshes the appearances of the form and clears /NeedAppearances.
func (p *AcroFormGenerateAppearancesProcessor) Process() {
	// Get the AcroForm in it's current state.
	//
	// Also note: getAcroForm() applies a default fixup which this processor
	// is part of. So keep the null parameter otherwise this will end
	// in an endless recursive call
	acroForm := form.AcroFormOfCatalogFixup(p.document.DocumentCatalog(), nil)

	if acroForm != nil {
		slog.Debug("processor: trying to generate appearance streams for fields " +
			"as NeedAppearances is true()")
		// Java catches IOException and IllegalArgumentException here. The
		// second is unchecked and the port raises it as a panic, so the
		// recover is what catches it.
		if err := refreshAppearancesCatching(acroForm); err != nil {
			slog.Debug("processor: couldn't generate appearance stream for some fields " +
				"- check output")
			slog.Debug(err.Error())
			return
		}
		acroForm.SetNeedAppearances(false)
	}
}

// refreshAppearancesCatching runs refreshAppearances and turns the
// IllegalArgumentException the port raises as a panic back into an error, which
// is what the catch of Java does with it.
func refreshAppearancesCatching(acroForm *form.PDAcroForm) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = illegalArgumentError{recovered}
		}
	}()
	return acroForm.RefreshAppearances()
}

// illegalArgumentError carries a recovered panic as an error.
type illegalArgumentError struct{ recovered any }

// Error returns the message of the panic, which is what getMessage answers on
// the IllegalArgumentException Java catches.
func (e illegalArgumentError) Error() string {
	if err, isError := e.recovered.(error); isError {
		return err.Error()
	}
	if message, isString := e.recovered.(string); isString {
		return message
	}
	return "IllegalArgumentException"
}
