package processor

import (
	"log/slog"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// AcroFormOrphanWidgetsProcessor generates field entries from the widget
// annotations of the pages, where the /Fields entry of the form is empty.
//
// Port of AcroFormOrphanWidgetsProcessor.
type AcroFormOrphanWidgetsProcessor struct {
	AbstractProcessor
}

var _ PDDocumentProcessor = (*AcroFormOrphanWidgetsProcessor)(nil)

// NewAcroFormOrphanWidgetsProcessor returns the processor for the given
// document.
func NewAcroFormOrphanWidgetsProcessor(
	document *pdmodel.PDDocument) *AcroFormOrphanWidgetsProcessor {
	p := &AcroFormOrphanWidgetsProcessor{}
	p.initAbstractProcessor(document)
	return p
}

// Process rebuilds the fields of the form from the widgets of the pages.
func (p *AcroFormOrphanWidgetsProcessor) Process() {
	// Get the AcroForm in it's current state.
	//
	// Also note: getAcroForm() applies a default fixup which this processor
	// is part of. So keep the null parameter otherwise this will end
	// in an endless recursive call
	acroForm := form.AcroFormOfCatalogFixup(p.document.DocumentCatalog(), nil)

	if acroForm != nil {
		p.resolveFieldsFromWidgets(acroForm)
	}
}

// resolveFieldsFromWidgets builds the field list of the form out of the widgets
// of every page. Java declares it private.
func (p *AcroFormOrphanWidgetsProcessor) resolveFieldsFromWidgets(acroForm *form.PDAcroForm) {
	slog.Debug("processor: rebuilding fields from widgets")

	resources := acroForm.DefaultResources()
	if resources == nil {
		// failsafe. Currently resources is never null because defaultfixup is called first.
		slog.Debug("processor: AcroForm default resources is null")
		return
	}

	fields := []form.PDField{}
	nonTerminalFieldsMap := map[string]form.PDField{}
	for page := range p.document.Pages().All {
		p.handleAnnotations(acroForm, resources, &fields, page.Annotations().ToSlice(),
			nonTerminalFieldsMap)
	}

	acroForm.SetFields(fields)

	for field := range acroForm.FieldTree().All() {
		if variableText := form.AsVariableText(field); variableText != nil {
			p.ensureFontResources(resources, variableText)
		}
	}
}

// handleAnnotations turns each widget of the given list into a field. Java
// declares it private.
func (p *AcroFormOrphanWidgetsProcessor) handleAnnotations(acroForm *form.PDAcroForm,
	acroFormResources *pdmodel.PDResources, fields *[]form.PDField,
	annotations []annotation.PDAnnotation, nonTerminalFieldsMap map[string]form.PDField) {
	for _, annot := range annotations {
		if _, isWidget := annot.(*annotation.PDAnnotationWidget); !isWidget {
			continue
		}
		addFontFromWidget(acroFormResources, annot)

		parent := annot.AnnotationDictionary().GetCOSDictionary(cos.Parent)
		if parent != nil {
			resolvedField := resolveNonRootField(acroForm, parent, nonTerminalFieldsMap)
			if resolvedField != nil {
				*fields = append(*fields, resolvedField)
			}
		} else {
			field := form.CreateField(acroForm, annot.AnnotationDictionary(), nil)
			if field != nil {
				*fields = append(*fields, field)
			}
		}
	}
}

// addFontFromWidget adds font resources from the widget to the AcroForm to make
// sure embedded fonts are being used and not added by ensureFontResources
// potentially using a fallback font. Java declares it private.
func addFontFromWidget(acroFormResources *pdmodel.PDResources, annot annotation.PDAnnotation) {
	normalAppearanceStream := annot.NormalAppearanceStream()
	if normalAppearanceStream == nil {
		return
	}
	widgetResources, _ := normalAppearanceStream.Resources().(*pdmodel.PDResources)
	if widgetResources == nil {
		return
	}
	for _, fontName := range widgetResources.FontNames() {
		if strings.HasPrefix(fontName.Name(), "+") {
			slog.Debug("processor: font resource for widget was a subsetted font - ignored",
				slog.String("font", fontName.Name()))
			continue
		}
		existing, err := acroFormResources.GetFont(fontName)
		if err != nil {
			slog.Debug("processor: unable to add font to AcroForm for font name",
				slog.String("font", fontName.Name()))
			continue
		}
		if existing != nil {
			continue
		}
		widgetFont, err := widgetResources.GetFont(fontName)
		if err != nil {
			slog.Debug("processor: unable to add font to AcroForm for font name",
				slog.String("font", fontName.Name()))
			continue
		}
		acroFormResources.PutFont(fontName, widgetFont)
		slog.Debug("processor: added font resource to AcroForm from widget for font name",
			slog.String("font", fontName.Name()))
	}
}

// resolveNonRootField walks up from a widget that has a /Parent entry to the
// root node and builds the field from there. Java declares it private.
func resolveNonRootField(acroForm *form.PDAcroForm, parent *cos.Dictionary,
	nonTerminalFieldsMap map[string]form.PDField) form.PDField {
	visited := map[*cos.Dictionary]bool{}
	for parent.ContainsKey(cos.Parent) {
		if visited[parent] {
			slog.Warn("processor: field ignored", slog.String("parent", parent.String()))
			return nil // Cycle detected
		}
		visited[parent] = true
		parent = parent.GetCOSDictionary(cos.Parent)
		if parent == nil {
			return nil
		}
	}

	if nonTerminalFieldsMap[parent.GetString(cos.T, "")] == nil {
		field := form.CreateField(acroForm, parent, nil)
		if field != nil {
			nonTerminalFieldsMap[field.FullyQualifiedName()] = field
		}
		return field
	}

	// this should not happen, likely broken PDF
	return nil
}

// ensureFontResources looks up the font used in the default appearance and, if
// this is not available, tries to find a suitable font and use that. This may
// not be the original font but a similar font replacement. Java declares it
// private.
//
// TODO: implement a font lookup similar as discussed in PDFBOX-2661 so that
// already existing font resources might be accepatble. In such case this must be
// implemented in PDDefaultAppearanceString too!
//
// The replacement itself is not ported: Java embeds the font it found with
// PDType0Font.load, and the font embedders are not ported yet. The lookup and
// the logging around it are here, so that the shape of the fixup is right when
// they land. See migration/STATUS.md.
func (p *AcroFormOrphanWidgetsProcessor) ensureFontResources(
	defaultResources *pdmodel.PDResources, field *form.PDVariableText) {
	daString := field.DefaultAppearance()
	if !strings.HasPrefix(daString, "/") || len(daString) <= 1 {
		return
	}
	// Java writes daString.substring(1, daString.indexOf(.)), which throws
	// StringIndexOutOfBoundsException where the string holds no space; the slice
	// below panics there, which is the same unchecked failure.
	fontName := cos.GetPDFName(daString[1:strings.Index(daString, " ")])
	existing, err := defaultResources.GetFont(fontName)
	if err != nil {
		slog.Debug("processor: unable to handle font resources for field",
			slog.String("field", field.FullyQualifiedName()),
			slog.String("err", err.Error()))
		return
	}
	if existing != nil {
		return
	}
	slog.Debug("processor: trying to add missing font resource for field",
		slog.String("field", field.FullyQualifiedName()))
	mapper := font.FontMappersInstance()
	fontMapping := mapper.GetTrueTypeFont(fontName.Name(), nil)
	if fontMapping == nil {
		slog.Debug("processor: no suitable font found for field for font name",
			slog.String("field", field.FullyQualifiedName()),
			slog.String("font", fontName.Name()))
		return
	}
	slog.Debug("processor: looked up font",
		slog.String("for", fontName.Name()),
		slog.String("found", fontMappingName(fontMapping)))
	// Java embeds it here with PDType0Font.load(document, fontMapping.getFont(),
	// false) and puts it in the default resources; the embedders are not ported.
	slog.Debug("processor: the replacement font is not embedded, " +
		"because the font embedders are not ported yet")
}

// fontMappingName returns the name of the mapped font, which is what Java logs.
func fontMappingName(fontMapping *font.FontMapping[*ttf.TrueTypeFont]) string {
	name, err := fontMapping.Font().Name()
	if err != nil {
		return ""
	}
	return name
}
