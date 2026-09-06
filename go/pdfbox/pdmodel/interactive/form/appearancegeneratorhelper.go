package form

import (
	"bytes"
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// The marked content operators the generated appearance is bracketed with.
// Java holds them in static fields.
var (
	bmcOperator = operator.Get("BMC")
	emcOperator = operator.Get("EMC")
)

// linePattern matches the line breaks a single line field has its value
// stripped of. Java holds the same expression in a static Pattern.
var linePattern = regexp.MustCompile(
	`\x{000D}\x{000A}|[\x{000A}\x{000B}\x{000C}\x{000D}\x{0085}\x{2028}\x{2029}]`)

// highlightColor is the colour Adobe draws the highlight box of a selected list
// box entry in, whatever an existing appearance stream says.
var highlightColor = [3]float32{153 / 255.0, 193 / 255.0, 215 / 255.0}

const (
	// fontScale is the scaling factor from font units to PDF units.
	fontScale = 1000

	// appearanceDefaultFontSize is the font size multiline text is drawn at by
	// default. Java names it DEFAULT_FONT_SIZE, which
	// PDDefaultAppearanceString already has in this package.
	appearanceDefaultFontSize = 12

	// minimumFontSize is the smallest size auto sizing multiline text picks.
	minimumFontSize = 4

	// defaultPadding is the padding Acrobat applies to the bbox of a field.
	defaultPadding = 0.5
)

// appearanceGeneratorHelper draws the appearance of a variable text field.
//
// Port of AppearanceGeneratorHelper, which Java declares package-private.
type appearanceGeneratorHelper struct {
	field *PDVariableText

	defaultAppearance *pdDefaultAppearanceString
	value             string
}

// newAppearanceGeneratorHelper returns a helper over the given field.
//
// Port of the package-private AppearanceGeneratorHelper(PDVariableText)
// constructor.
func newAppearanceGeneratorHelper(field *PDVariableText) (*appearanceGeneratorHelper, error) {
	h := &appearanceGeneratorHelper{field: field}
	if err := h.validateAndEnsureAcroFormResources(); err != nil {
		return nil, err
	}

	defaultAppearance, err := field.defaultAppearanceString()
	if err != nil {
		return nil, fmt.Errorf("Could not process default appearance string '%s' for field '%s': %w",
			field.DefaultAppearance(), field.FullyQualifiedName(), err)
	}
	h.defaultAppearance = defaultAppearance
	return h, nil
}

// validateAndEnsureAcroFormResources lifts the font resources a widget carries
// up to the form, which is what Adobe Reader and Acrobat do. Java declares it
// private.
func (h *appearanceGeneratorHelper) validateAndEnsureAcroFormResources() error {
	// add font resources which might be available at the field
	// level but are not at the AcroForm level to the AcroForm
	// to match Adobe Reader/Acrobat behavior
	acroFormResources := h.field.AcroForm().DefaultResources()
	if acroFormResources == nil {
		return nil
	}

	for _, widget := range h.field.Widgets() {
		stream := widget.NormalAppearanceStream()
		if stream == nil {
			continue
		}
		widgetResources, _ := stream.Resources().(*pdmodel.PDResources)
		if widgetResources == nil {
			continue
		}
		widgetFontDict := widgetResources.Dictionary().GetCOSDictionary(cos.Font)
		acroFormFontDict := acroFormResources.Dictionary().GetCOSDictionary(cos.Font)
		for _, fontResourceName := range widgetResources.FontNames() {
			existing, err := acroFormResources.GetFont(fontResourceName)
			if err != nil {
				slog.Warn("form: unable to match field level font with AcroForm font",
					slog.Any("err", err))
				continue
			}
			if existing == nil {
				slog.Debug("form: adding font resource from widget to AcroForm",
					slog.String("font", fontResourceName.Name()))
				// use the COS-object to preserve a possible indirect object reference
				acroFormFontDict.SetItem(fontResourceName, widgetFontDict.GetItem(fontResourceName))
			}
		}
	}
	return nil
}

// setAppearanceValue draws the given value into the appearance streams of every
// widget of the field.
//
// Java throws IllegalArgumentException where the string holds a character the
// field font has no glyph for, which is unchecked, so the port panics from
// deeper down.
func (h *appearanceGeneratorHelper) setAppearanceValue(apValue string) error {
	value, err := h.formattedValue(apValue)
	if err != nil {
		return err
	}
	h.value = value

	// Treat multiline field values in single lines as single lime values.
	// This is in line with how Adobe Reader behaves when entering text
	// interactively but NOT how it behaves when the field value has been
	// set programmatically and Reader is forced to generate the appearance
	// using PDAcroForm.setNeedAppearances
	// see PDFBOX-3911
	if textField, isTextField := h.field.self.(*PDTextField); isTextField && !textField.IsMultiline() {
		h.value = linePattern.ReplaceAllString(h.value, " ")
	}

	for _, widget := range h.field.Widgets() {
		if widget.AnnotationDictionary().ContainsKey(cos.GetPDFName("PMD")) {
			slog.Warn("form: widget of field is a PaperMetaData widget, "+
				"no appearance stream created",
				slog.String("field", h.field.FullyQualifiedName()))
			continue
		}

		// some fields have the /Da at the widget level if the
		// widgets differ in layout.
		acroFormAppearance := h.defaultAppearance

		if widget.AnnotationDictionary().GetDictionaryObject(cos.DA) != nil {
			widgetAppearance, err := h.widgetDefaultAppearanceString(widget)
			if err != nil {
				return err
			}
			h.defaultAppearance = widgetAppearance
		}

		if err := h.setWidgetAppearanceValue(widget); err != nil {
			return err
		}

		// restore the field level appearance
		h.defaultAppearance = acroFormAppearance
	}
	return nil
}

// setWidgetAppearanceValue is the body of the widget loop of setAppearanceValue,
// which Go pulls out so that the restore of the field level appearance happens
// on every path, as the loop of Java does.
func (h *appearanceGeneratorHelper) setWidgetAppearanceValue(
	widget *annotation.PDAnnotationWidget) error {
	rect := widget.Rectangle()
	if rect == nil {
		widget.AnnotationDictionary().RemoveItem(cos.AP)
		slog.Warn("form: widget of field has no rectangle, no appearance stream created",
			slog.String("field", h.field.FullyQualifiedName()))
		return nil
	}

	appearanceDict := widget.Appearance()
	if appearanceDict == nil {
		appearanceDict = annotation.NewPDAppearanceDictionary()
		widget.SetAppearance(appearanceDict)
	}

	appearance := appearanceDict.NormalAppearance()
	// TODO support appearances other than "normal"

	appearanceCharacteristics := widget.AppearanceCharacteristics()
	widgetRotation := resolveRotation(appearanceCharacteristics)
	newBBox := computeBBox(widget, widgetRotation)
	var appearanceStream *annotation.PDAppearanceStream
	// We're using the existing appearance if possible (since 2013 or even earlier)
	// However, except for the file from PDFBOX-2586 we could ignore it
	if isValidAppearanceStream(appearance, newBBox) {
		appearanceStream = appearance.AppearanceStream()
	} else {
		appearanceStream = h.prepareNormalAppearanceStream(newBBox, widgetRotation)
		appearanceDict.SetNormalAppearanceStream(appearanceStream)
	}

	// Adobe Acrobat always recreates the complete appearance stream if there is
	// an appearance characteristics entry (the widget dictionaries MK entry). In
	// addition if there is no content yet also create the appearance stream from
	// the entries.
	if appearanceCharacteristics != nil || appearanceStream.ContentStream().Length() == 0 {
		if err := h.initializeAppearanceContent(
			widget, appearanceCharacteristics, appearanceStream); err != nil {
			return err
		}
	}

	return h.setAppearanceContent(widget, appearanceStream)
}

// formattedValue runs the format action of the field over the value, where the
// form has a scripting handler. Java declares it private.
func (h *appearanceGeneratorHelper) formattedValue(apValue string) (string, error) {
	// format the field value for the appearance if there is scripting support and the field
	// has a format event
	actions := h.field.Actions()
	if actions == nil {
		return apValue, nil
	}
	actionF := actions.F()
	if actionF != nil {
		scriptingHandler := h.field.AcroForm().ScriptingHandler()
		if scriptingHandler != nil {
			javaScript, _ := actionF.(*action.PDActionJavaScript)
			return scriptingHandler.Format(javaScript, apValue), nil
		}
		slog.Info("form: field contains a formatting action but no ScriptingHandler " +
			"has been supplied - formatted value might be incorrect")
	}
	return apValue, nil
}

// isValidAppearanceStream reports whether the existing appearance can be drawn
// into again. Java declares it private static.
func isValidAppearanceStream(appearance *annotation.PDAppearanceEntry,
	newBBox *common.PDRectangle) bool {
	if appearance == nil {
		return false
	}
	if !appearance.IsStream() {
		return false
	}
	bbox := appearance.AppearanceStream().BBox()
	if bbox == nil {
		return false
	}
	if math.Abs(float64(newBBox.Width()-bbox.Width())) > 1 ||
		math.Abs(float64(newBBox.Height()-bbox.Height())) > 1 {
		// PDFBOX-6223: don't like it if bbox and rectangle are of very different sizes
		return false
	}
	return math.Abs(float64(bbox.Width())) > 0 && math.Abs(float64(bbox.Height())) > 0
}

// prepareNormalAppearanceStream returns an empty appearance of the given size.
// Java declares it private.
func (h *appearanceGeneratorHelper) prepareNormalAppearanceStream(bbox *common.PDRectangle,
	widgetRotation int) *annotation.PDAppearanceStream {
	appearanceStream := annotation.NewPDAppearanceStream(h.field.AcroForm().Document().Document())

	appearanceStream.SetBBox(bbox)
	at := calculateMatrix(bbox, widgetRotation)
	if !at.IsIdentity() {
		appearanceStream.SetMatrix(util.NewMatrixFromAffineTransform(at))
	}
	appearanceStream.SetFormType(1)
	appearanceStream.SetResources(pdmodel.NewPDResources())
	return appearanceStream
}

// computeBBox returns the size of the appearance of a widget turned by the
// given rotation. Java declares it private static.
func computeBBox(widget *annotation.PDAnnotationWidget, widgetRotation int) *common.PDRectangle {
	rect := widget.Rectangle()
	matrix := util.RotateInstance(toRadians(float64(widgetRotation)), 0, 0)
	point2D := matrix.TransformPoint(rect.Width(), rect.Height())
	return common.NewPDRectangleOfSize(
		float32(math.Abs(point2D.X())), float32(math.Abs(point2D.Y())))
}

// toRadians is java.lang.Math.toRadians.
func toRadians(angdeg float64) float64 { return angdeg / 180.0 * math.Pi }

// widgetDefaultAppearanceString reads the /DA a widget carries of its own. Java
// declares it private.
func (h *appearanceGeneratorHelper) widgetDefaultAppearanceString(
	widget *annotation.PDAnnotationWidget) (*pdDefaultAppearanceString, error) {
	da, _ := widget.AnnotationDictionary().GetDictionaryObject(cos.DA).(*cos.StringObj)
	dr := h.field.AcroForm().DefaultResources()
	return newPDDefaultAppearanceString(da, dr)
}

// resolveRotation returns the /R of the appearance characteristics, which is
// zero where there are none. Java declares it private static.
func resolveRotation(characteristicsDictionary *annotation.PDAppearanceCharacteristicsDictionary) int {
	if characteristicsDictionary != nil {
		// 0 is the default value if the R key doesn't exist
		return characteristicsDictionary.Rotation()
	}
	return 0
}

// initializeAppearanceContent draws the background and border of the widget
// into the appearance. Java declares it private.
func (h *appearanceGeneratorHelper) initializeAppearanceContent(
	widget *annotation.PDAnnotationWidget,
	appearanceCharacteristics *annotation.PDAppearanceCharacteristicsDictionary,
	appearanceStream *annotation.PDAppearanceStream) (err error) {
	output := &bytes.Buffer{}
	contents := pdmodel.NewPDAppearanceContentStreamOf(appearanceStream, nopWriteCloser{output})
	defer func() {
		if closeErr := contents.Close(); err == nil {
			err = closeErr
		}
	}()

	// TODO: support more entries like patterns, etc.
	if appearanceCharacteristics != nil {
		backgroundColour := appearanceCharacteristics.Background()
		if backgroundColour != nil {
			if err := contents.SetNonStrokingColor(backgroundColour); err != nil {
				return err
			}
			bbox := resolveBoundingBox(widget, appearanceStream)
			if err := contents.AddRect(bbox.LowerLeftX(), bbox.LowerLeftY(),
				bbox.Width(), bbox.Height()); err != nil {
				return err
			}
			if err := contents.Fill(); err != nil {
				return err
			}
		}

		lineWidth := float32(0)
		borderColour := appearanceCharacteristics.BorderColour()
		if borderColour != nil {
			if err := contents.SetStrokingColor(borderColour); err != nil {
				return err
			}
			lineWidth = 1
		}
		borderStyle := widget.BorderStyle()
		if borderStyle != nil && borderStyle.Width() > 0 {
			lineWidth = borderStyle.Width()
		}

		if lineWidth > 0 && borderColour != nil {
			if lineWidth != 1 {
				if err := contents.SetLineWidth(lineWidth); err != nil {
					return err
				}
			}
			bbox := resolveBoundingBox(widget, appearanceStream)
			clipRect := applyPadding(bbox, maxFloat32(defaultPadding, lineWidth/2))
			if err := contents.AddRect(clipRect.LowerLeftX(), clipRect.LowerLeftY(),
				clipRect.Width(), clipRect.Height()); err != nil {
				return err
			}
			if err := contents.CloseAndStroke(); err != nil {
				return err
			}
		}

		// draw the dividers for a comb field
		if borderColour != nil && h.shallComb() {
			maxLen := h.field.self.(*PDTextField).MaxLen()
			bbox := resolveBoundingBox(widget, appearanceStream)
			clipRect := applyPadding(bbox, maxFloat32(defaultPadding, lineWidth/2))
			lowerLeft := clipRect.LowerLeftX()
			height := clipRect.Height()

			combWidth := bbox.Width() / float32(maxLen)

			for i := 0; i < maxLen-1; i++ {
				if err := contents.MoveTo(combWidth+combWidth*float32(i), height); err != nil {
					return err
				}
				if err := contents.LineTo(combWidth+combWidth*float32(i), lowerLeft); err != nil {
					return err
				}
			}
			if err := contents.CloseAndStroke(); err != nil {
				return err
			}
		}
	}

	// Java writes the buffer out inside the try-with-resources, before the
	// content stream is closed; the stream writes straight through to the
	// buffer, so the bytes are all there by now.
	return writeToStream(output, appearanceStream)
}

// setAppearanceContent replaces the generated part of the appearance, which is
// what sits between /Tx BMC and the matching EMC. Java declares it private.
func (h *appearanceGeneratorHelper) setAppearanceContent(
	widget *annotation.PDAnnotationWidget,
	appearanceStream *annotation.PDAppearanceStream) error {
	// first copy any needed resources from the document's DR dictionary into
	// the stream's Resources dictionary
	if err := h.defaultAppearance.copyNeededResourcesTo(appearanceStream); err != nil {
		return err
	}

	// then replace the existing contents of the appearance stream from /Tx BMC
	// to the matching EMC
	output := &bytes.Buffer{}
	writer := pdfwriter.NewContentStreamWriter(output)

	tokens, err := parseAppearanceTokens(appearanceStream)
	if err != nil {
		return err
	}
	bmcIndex := indexOfToken(tokens, bmcOperator)
	if bmcIndex == -1 {
		// append to existing stream
		if err := writer.WriteTokens(tokens); err != nil {
			return err
		}
		if err := writer.WriteTokensVarargs(cos.Tx, bmcOperator); err != nil {
			return err
		}
	} else {
		// prepend content before BMC
		if err := writer.WriteTokens(tokens[:bmcIndex+1]); err != nil {
			return err
		}
	}

	// insert field contents
	if err := h.insertGeneratedAppearance(widget, appearanceStream, output); err != nil {
		return err
	}

	emcIndex := indexOfToken(tokens, emcOperator)
	if emcIndex == -1 {
		// append EMC
		if err := writer.WriteTokensVarargs(emcOperator); err != nil {
			return err
		}
	} else {
		// append contents after EMC
		if err := writer.WriteTokens(tokens[emcIndex:]); err != nil {
			return err
		}
	}
	return writeToStream(output, appearanceStream)
}

// parseAppearanceTokens reads the tokens of the appearance stream, which is
// what new PDFStreamParser(appearanceStream).parse() does.
//
// Java reaches the content through getContentsForStreamParsing, whose default
// on PDContentStream answers getContents; PDFormXObject does not override it.
func parseAppearanceTokens(appearanceStream *annotation.PDAppearanceStream) ([]any, error) {
	content, err := appearanceStream.ContentsForRandomAccess()
	if err != nil {
		return nil, err
	}
	parser, err := pdfparser.NewStreamTokenParserSource(content)
	if err != nil {
		return nil, err
	}
	return parser.Parse()
}

// indexOfToken returns where the given operator sits in the token list, which
// is what List.indexOf does over the shared operators Operator.getOperator
// hands out.
func indexOfToken(tokens []any, want *operator.Operator) int {
	for i, token := range tokens {
		if op, isOperator := token.(*operator.Operator); isOperator && op == want {
			return i
		}
	}
	return -1
}

// insertGeneratedAppearance draws the text of the field, and the clipping path
// around it. Java declares it private.
func (h *appearanceGeneratorHelper) insertGeneratedAppearance(
	widget *annotation.PDAnnotationWidget,
	appearanceStream *annotation.PDAppearanceStream, output *bytes.Buffer) (err error) {
	contents := pdmodel.NewPDAppearanceContentStreamOf(appearanceStream, nopWriteCloser{output})
	defer func() {
		if closeErr := contents.Close(); err == nil {
			err = closeErr
		}
	}()

	if glyphLayoutProcessor := h.field.AcroForm().GlyphLayoutProcessor(); glyphLayoutProcessor != nil {
		contents.SetGlyphLayoutProcessor(glyphLayoutProcessor)
	}
	bbox := resolveBoundingBox(widget, appearanceStream)

	// Acrobat calculates the left and right padding dependent on the offset of the border edge
	// This calculation works for forms having been generated by Acrobat.
	// The minimum distance is always 1f even if there is no rectangle being drawn around.
	borderWidth := float32(0)
	if widget.BorderStyle() != nil {
		borderWidth = widget.BorderStyle().Width()
	}
	padding := maxFloat32(1, borderWidth)
	clipRect := applyPadding(bbox, padding)
	clipRectLowerLeftY := clipRect.LowerLeftY()
	clipRectHeight := clipRect.Height()

	contentRect := applyPadding(clipRect, padding)

	if err := contents.SaveGraphicsState(); err != nil {
		return err
	}

	// Acrobat always adds a clipping path
	if err := contents.AddRect(clipRect.LowerLeftX(), clipRectLowerLeftY,
		clipRect.Width(), clipRectHeight); err != nil {
		return err
	}
	if err := contents.Clip(); err != nil {
		return err
	}

	// get the font
	fieldFont := h.defaultAppearance.Font()
	if fieldFont == nil {
		panic("font is null, check whether /DA entry is incomplete or incorrect")
	}
	if strings.Contains(fieldFont.Name(), "+") {
		slog.Warn("form: font of field contains subsetted font",
			slog.String("da font", h.defaultAppearance.FontName().Name()),
			slog.String("field", h.field.FullyQualifiedName()),
			slog.String("font", fieldFont.Name()))
		slog.Warn("form: this may bring trouble with PDField.setValue(), " +
			"PDAcroForm.flatten() or PDAcroForm.refreshAppearances()")
		slog.Warn("form: you should replace this font with a non-subsetted font:")
		slog.Warn("form: PDFont font = PDType0Font.load(doc, new FileInputStream(fontfile), false);")
		slog.Warn(fmt.Sprintf("form: acroForm.getDefaultResources().put(COSName.getPDFName(\"%s\", font);",
			h.defaultAppearance.FontName().Name()))
	}
	// calculate the fontSize (because 0 = autosize)
	fontSize := h.defaultAppearance.FontSize()

	if fontSize == 0 {
		fontSize, err = h.calculateFontSize(fieldFont, contentRect)
		if err != nil {
			return err
		}
	}

	// for a listbox generate the highlight rectangle for the selected
	// options
	if _, isListBox := h.field.self.(*PDListBox); isListBox {
		if err := h.insertGeneratedListboxSelectionHighlight(
			contents, appearanceStream, fieldFont, fontSize); err != nil {
			return err
		}
	}

	// start the text output
	if err := contents.BeginText(); err != nil {
		return err
	}

	// write font and color from the /DA string, with the calculated font size
	if err := h.defaultAppearance.writeTo(contents, fontSize); err != nil {
		return err
	}

	// calculate the y-position of the baseline
	var y float32

	// calculate font metrics at font size
	fontScaleY := fontSize / fontScale
	fontBoundingBox, err := fieldFont.BoundingBox()
	if err != nil {
		return err
	}
	fontBoundingBoxAtSize := fontBoundingBox.Height() * fontScaleY

	var fontCapAtSize float32
	var fontDescentAtSize float32

	if fieldFont.FontDescriptor() != nil {
		fontCapAtSize = fieldFont.FontDescriptor().CapHeight() * fontScaleY
		fontDescentAtSize = fieldFont.FontDescriptor().Descent() * fontScaleY
	} else {
		fontCapHeight, err := resolveCapHeight(fieldFont)
		if err != nil {
			return err
		}
		fontDescent, err := resolveDescent(fieldFont)
		if err != nil {
			return err
		}
		slog.Debug("form: missing font descriptor - resolved Cap/Descent",
			slog.Float64("cap", float64(fontCapHeight)),
			slog.Float64("descent", float64(fontDescent)))
		fontCapAtSize = fontCapHeight * fontScaleY
		fontDescentAtSize = fontDescent * fontScaleY
	}

	if textField, isTextField := h.field.self.(*PDTextField); isTextField && textField.IsMultiline() {
		y = contentRect.UpperRightY() - fontBoundingBoxAtSize
	} else {
		// Adobe shows the text 'shifted up' in case the caps don't fit into the clipping area
		if fontCapAtSize > clipRectHeight {
			y = clipRectLowerLeftY + -fontDescentAtSize
		} else {
			// calculate the position based on the content rectangle
			y = clipRectLowerLeftY + (clipRectHeight-fontCapAtSize)/2

			// check to ensure that ascents and descents fit
			if y-clipRectLowerLeftY < -fontDescentAtSize {
				contentRectLowerLeftY := contentRect.LowerLeftY()
				fontDescentBased := -fontDescentAtSize + contentRectLowerLeftY
				fontCapBased := contentRect.Height() - contentRectLowerLeftY - fontCapAtSize

				y = minFloat32(fontDescentBased, maxFloat32(y, fontCapBased))
			}
		}
	}

	// show the text
	x := contentRect.LowerLeftX()

	// special handling for comb boxes as these are like table cells with individual
	// chars
	switch {
	case h.shallComb():
		if err := h.insertGeneratedCombAppearance(
			contents, appearanceStream, fieldFont, fontSize); err != nil {
			return err
		}
	case h.isListBox():
		if err := h.insertGeneratedListboxAppearance(
			contents, appearanceStream, contentRect, fieldFont, fontSize); err != nil {
			return err
		}
	default:
		textContent := interactive.NewPlainText(h.value)
		appearanceStyle := interactive.NewAppearanceStyle()
		appearanceStyle.SetFont(fieldFont)
		appearanceStyle.SetFontSize(fontSize)

		// Adobe Acrobat uses the font's bounding box for the leading between the lines
		appearanceStyle.SetLeading(fontBoundingBox.Height() * fontScaleY)

		formatter := interactive.NewPlainTextFormatterBuilder(contents).
			Style(appearanceStyle).
			Text(textContent).
			Width(contentRect.Width()).
			WrapLines(h.isMultiLine()).
			InitialOffset(x, y).
			TextAlignValue(h.textAlign(widget)).
			Build()
		if err := formatter.Format(); err != nil {
			return err
		}
	}

	if err := contents.EndText(); err != nil {
		return err
	}
	return contents.RestoreGraphicsState()
}

// textAlign returns the quadding of the widget, falling back to that of the
// field. Java declares it private.
//
// PDFBox handles a widget with a joined in field dictionary and without an
// individual name as a widget only. As a result -- as a widget can't have a
// quadding /Q entry we need to do a low level access to the dictionary and
// otherwise get the quadding from the field.
func (h *appearanceGeneratorHelper) textAlign(widget *annotation.PDAnnotationWidget) int {
	// Use quadding value from joined field/widget if set, else use from field.
	return widget.AnnotationDictionary().GetIntDefault(cos.Q, h.field.Q())
}

// calculateMatrix returns the transform that turns the appearance by the given
// rotation. Java declares it private.
func calculateMatrix(bbox *common.PDRectangle, rotation int) *geom.AffineTransform {
	if rotation == 0 {
		return geom.NewIdentityTransform()
	}
	var tx, ty float32
	switch rotation {
	case 90:
		tx = bbox.UpperRightY()
	case 180:
		tx = bbox.UpperRightY()
		ty = bbox.UpperRightX()
	case 270:
		ty = bbox.UpperRightX()
	}
	matrix := util.RotateInstance(toRadians(float64(rotation)), tx, ty)
	return matrix.CreateAffineTransform()
}

// isMultiLine reports whether the field holds more than one line. Java declares
// it private.
func (h *appearanceGeneratorHelper) isMultiLine() bool {
	textField, isTextField := h.field.self.(*PDTextField)
	return isTextField && textField.IsMultiline()
}

// isListBox reports whether the field is a list box, which Java writes inline
// as an instanceof.
func (h *appearanceGeneratorHelper) isListBox() bool {
	_, isListBox := h.field.self.(*PDListBox)
	return isListBox
}

// shallComb reports whether the appearance is divided into equal cells. Java
// declares it private.
//
// May be set only if the MaxLen entry is present in the text field dictionary
// and if the Multiline, Password, and FileSelect flags are clear. If set, the
// field shall be automatically divided into as many equally spaced positions,
// or combs, as the value of MaxLen, and the text is laid out into those combs.
func (h *appearanceGeneratorHelper) shallComb() bool {
	textField, isTextField := h.field.self.(*PDTextField)
	return isTextField &&
		textField.IsComb() &&
		textField.MaxLen() != -1 &&
		!textField.IsMultiline() &&
		!textField.IsPassword() &&
		!textField.IsFileSelect()
}

// insertGeneratedCombAppearance draws the value one cell at a time. Java
// declares it private.
//
// Java takes each comb cell with value.substring(i, i+1), one UTF-16 code
// unit, and counts the cells with value.length(); the port takes one rune and
// counts runes. The two agree for every character in the basic plane and
// differ for one outside it, where Java splits the surrogate pair across two
// cells and the port keeps it in one.
//
// That difference is not reachable. Both sides measure each cell through
// PDFont.getStringWidth before drawing it -- Java asks the font for the lone
// surrogate U+D83D, the port for the whole U+1F600 -- and no font this port can
// build has either, so both refuse the value instead of laying it out.
// TestCombFieldRefusesASupplementaryCharacter pins that, and
// migration/STATUS.md records it.
func (h *appearanceGeneratorHelper) insertGeneratedCombAppearance(
	contents *pdmodel.PDAppearanceContentStream,
	appearanceStream *annotation.PDAppearanceStream,
	fieldFont font.PDFont, fontSize float32) error {
	if h.value == "" {
		return nil
	}
	runes := []rune(h.value)
	maxLen := h.field.self.(*PDTextField).MaxLen()
	quadding := h.field.Q()
	numChars := min(len(runes), maxLen)

	bBox := appearanceStream.BBox()
	combWidth := bBox.Width() / float32(maxLen)
	ascentAtFontSize := fieldFont.FontDescriptor().Ascent() / fontScale * fontSize

	baselineOffset := bBox.LowerLeftY() + (bBox.Height()-ascentAtFontSize)/2

	prevCharWidth := float32(0)

	// set initial offset based on width of first char.
	firstStringWidth, err := fieldFont.StringWidth(string(runes[0]))
	if err != nil {
		return err
	}
	firstCharWidth := firstStringWidth / fontScale * fontSize
	initialOffset := (combWidth - firstCharWidth) / 2

	// add to initial offset if right aligned or centered
	if quadding == 2 {
		initialOffset = initialOffset + float32(maxLen-numChars)*combWidth
	} else if quadding == 1 {
		initialOffset = initialOffset + float32(floorDiv(maxLen-numChars, 2))*combWidth
	}

	xOffset := initialOffset

	for i := 0; i < numChars; i++ {
		combString := string(runes[i])
		stringWidth, err := fieldFont.StringWidth(combString)
		if err != nil {
			return err
		}
		currCharWidth := stringWidth / fontScale * fontSize / 2

		xOffset = xOffset + prevCharWidth/2 - currCharWidth/2

		if i == 0 {
			err = contents.NewLineAtOffset(initialOffset, baselineOffset)
		} else {
			err = contents.NewLineAtOffset(xOffset, baselineOffset)
		}
		if err != nil {
			return err
		}
		if err := contents.ShowText(combString); err != nil {
			return err
		}

		baselineOffset = 0
		prevCharWidth = currCharWidth
		xOffset = combWidth
	}
	return nil
}

// insertGeneratedListboxSelectionHighlight draws the highlight box behind every
// selected entry. Java declares it private.
func (h *appearanceGeneratorHelper) insertGeneratedListboxSelectionHighlight(
	contents *pdmodel.PDAppearanceContentStream,
	appearanceStream *annotation.PDAppearanceStream,
	fieldFont font.PDFont, fontSize float32) error {
	listBox := h.field.self.(*PDListBox)
	indexEntries := listBox.SelectedOptionsIndex()
	values := listBox.Value()
	options := listBox.OptionsExportValues()

	if len(values) != 0 && len(options) != 0 && len(indexEntries) == 0 {
		// create indexEntries from options
		indexEntries = make([]int, 0, len(values))
		for _, v := range values {
			indexEntries = append(indexEntries, indexOfString(options, v))
		}
	}

	// The first entry which shall be presented might be adjusted by the optional TI key
	// If this entry is present, the first entry to be displayed is the keys value,
	// otherwise display starts with the first entry in Opt.
	topIndex := listBox.TopIndex()

	fontBoundingBox, err := fieldFont.BoundingBox()
	if err != nil {
		return err
	}
	highlightBoxHeight := fontBoundingBox.Height() * fontSize / fontScale

	// the padding area
	paddingEdge := applyPadding(appearanceStream.BBox(), 1)

	for _, selectedIndex := range indexEntries {
		if err := contents.SetNonStrokingColorRGB(
			highlightColor[0], highlightColor[1], highlightColor[2]); err != nil {
			return err
		}

		if err := contents.AddRect(paddingEdge.LowerLeftX(),
			paddingEdge.UpperRightY()-highlightBoxHeight*float32(selectedIndex-topIndex+1)+2,
			paddingEdge.Width(),
			highlightBoxHeight); err != nil {
			return err
		}
		if err := contents.Fill(); err != nil {
			return err
		}
	}
	return contents.SetNonStrokingColorGray(0)
}

// insertGeneratedListboxAppearance draws the options of a list box. Java
// declares it private.
func (h *appearanceGeneratorHelper) insertGeneratedListboxAppearance(
	contents *pdmodel.PDAppearanceContentStream,
	appearanceStream *annotation.PDAppearanceStream,
	contentRect *common.PDRectangle, fieldFont font.PDFont, fontSize float32) error {
	if err := contents.SetNonStrokingColorGray(0); err != nil {
		return err
	}

	q := h.field.Q()

	if q == QuaddingCentered || q == QuaddingRight {
		fieldWidth := appearanceStream.BBox().Width()
		width, err := fieldFont.StringWidth(h.value)
		if err != nil {
			return err
		}
		stringWidth := width / fontScale * fontSize
		adjustAmount := fieldWidth - stringWidth - 4

		if q == QuaddingCentered {
			adjustAmount = adjustAmount / 2.0
		}

		if err := contents.NewLineAtOffset(adjustAmount, 0); err != nil {
			return err
		}
	} else if q != QuaddingLeft {
		return fmt.Errorf("Error: Unknown justification value:%d", q)
	}

	options := h.field.self.(*PDListBox).OptionsDisplayValues()
	numOptions := len(options)

	yTextPos := contentRect.UpperRightY()

	topIndex := h.field.self.(*PDListBox).TopIndex()
	ascent := fieldFont.FontDescriptor().Ascent()
	fontBoundingBox, err := fieldFont.BoundingBox()
	if err != nil {
		return err
	}
	height := fontBoundingBox.Height()

	for i := topIndex; i < numOptions; i++ {
		if i == topIndex {
			yTextPos = yTextPos - ascent/fontScale*fontSize
		} else {
			yTextPos = yTextPos - height/fontScale*fontSize
			if err := contents.BeginText(); err != nil {
				return err
			}
		}

		if err := contents.NewLineAtOffset(contentRect.LowerLeftX(), yTextPos); err != nil {
			return err
		}
		if err := contents.ShowText(options[i]); err != nil {
			return err
		}

		if i != numOptions-1 {
			if err := contents.EndText(); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeToStream copies the buffer into the appearance stream. Java declares it
// private.
func writeToStream(baos *bytes.Buffer, appearanceStream *annotation.PDAppearanceStream) error {
	os, err := appearanceStream.Stream().CreateWriter()
	if err != nil {
		return err
	}
	if _, err := os.Write(baos.Bytes()); err != nil {
		os.Close()
		return err
	}
	return os.Close()
}

// calculateFontSize returns the size the value is drawn at, which is what the
// /DA string says unless that is zero, where the size is fitted. Java declares
// it private, and its own comment calls it "my not so great method for
// calculating the fontsize. It does not work superb, but it handles ok."
func (h *appearanceGeneratorHelper) calculateFontSize(fieldFont font.PDFont,
	contentRect *common.PDRectangle) (float32, error) {
	fontSize := h.defaultAppearance.FontSize()

	// zero is special, it means the text is auto-sized
	if fontSize != 0 {
		return fontSize, nil
	}
	if h.isMultiLine() {
		textContent := interactive.NewPlainText(h.value)
		if textContent.Paragraphs() != nil {
			width := contentRect.Width() - contentRect.LowerLeftX()
			fs := float32(minimumFontSize)
			for fs <= appearanceDefaultFontSize {
				// determine the number of lines needed for this font and contentRect
				numLines := 0
				for _, paragraph := range textContent.Paragraphs() {
					lines, err := paragraph.Lines(fieldFont, fs, width)
					if err != nil {
						return 0, err
					}
					numLines += len(lines)
				}
				// calculate the height required for this font size
				fontScaleY := fs / fontScale
				fontBoundingBox, err := fieldFont.BoundingBox()
				if err != nil {
					return 0, err
				}
				leading := fontBoundingBox.Height() * fontScaleY
				height := leading * float32(numLines)

				// if this font size didn't fit, use the prior size that did fit
				if height > contentRect.Height() {
					return maxFloat32(fs-1, minimumFontSize), nil
				}
				fs += 1.0
			}
			return minFloat32(fs, appearanceDefaultFontSize), nil
		}

		// Acrobat defaults to 12 for multiline text with size 0
		return appearanceDefaultFontSize, nil
	}

	fontMatrix := fieldFont.FontMatrix()
	yScalingFactor := fontScale * fontMatrix.ScaleY()
	xScalingFactor := fontScale * fontMatrix.ScaleX()

	// fit width
	stringWidth, err := fieldFont.StringWidth(h.value)
	if err != nil {
		return 0, err
	}
	width := stringWidth * fontMatrix.ScaleX()
	widthBasedFontSize := contentRect.Width() / width * xScalingFactor

	// fit height
	height := (fieldFont.FontDescriptor().CapHeight() +
		-fieldFont.FontDescriptor().Descent()) * fontMatrix.ScaleY()
	if height <= 0 {
		fontBoundingBox, err := fieldFont.BoundingBox()
		if err != nil {
			return 0, err
		}
		height = fontBoundingBox.Height() * fontMatrix.ScaleY()
	}

	heightBasedFontSize := contentRect.Height() / height * yScalingFactor
	if math.IsInf(float64(widthBasedFontSize), 0) {
		// PDFBOX-5763: avoids -Infinity if empty value and tiny rectangle
		return heightBasedFontSize, nil
	}

	return minFloat32(heightBasedFontSize, widthBasedFontSize), nil
}

// resolveCapHeight returns the cap height of the font, which this very basic
// implementation reads off the height of "H". Java declares it private.
func resolveCapHeight(fieldFont font.PDFont) (float32, error) {
	return resolveGlyphHeight(fieldFont, int('H'))
}

// resolveDescent returns the descent of the font, which this very basic
// implementation reads off the height of "y" less that of "a". Java declares it
// private.
func resolveDescent(fieldFont font.PDFont) (float32, error) {
	heightY, err := resolveGlyphHeight(fieldFont, int('y'))
	if err != nil {
		return 0, err
	}
	heightA, err := resolveGlyphHeight(fieldFont, int('a'))
	if err != nil {
		return 0, err
	}
	return heightY - heightA, nil
}

// resolveGlyphHeight calculates the real (except for type 3 fonts) individual
// glyph bounds. Java declares it private.
func resolveGlyphHeight(fieldFont font.PDFont, code int) (float32, error) {
	var path *geom.Path2D
	switch typedFont := fieldFont.(type) {
	case *font.PDType3Font:
		// It is difficult to calculate the real individual glyph bounds for type 3
		// fonts because these are not vector fonts, the content stream could contain
		// almost anything that is found in page content streams.
		charProc := typedFont.CharProc(code)
		if charProc != nil {
			fontBBox, err := typedFont.BoundingBox()
			if err != nil {
				return 0, err
			}
			glyphBBox, err := charProc.GlyphBBox()
			if err != nil {
				return 0, err
			}
			if glyphBBox != nil {
				// PDFBOX-3850: glyph bbox could be larger than the font bbox
				glyphBBox.SetLowerLeftX(maxFloat32(fontBBox.LowerLeftX(), glyphBBox.LowerLeftX()))
				glyphBBox.SetLowerLeftY(maxFloat32(fontBBox.LowerLeftY(), glyphBBox.LowerLeftY()))
				glyphBBox.SetUpperRightX(minFloat32(fontBBox.UpperRightX(), glyphBBox.UpperRightX()))
				glyphBBox.SetUpperRightY(minFloat32(fontBBox.UpperRightY(), glyphBBox.UpperRightY()))
				path = glyphBBox.ToGeneralPath()
			}
		}
	default:
		var err error
		if vectorFont, isVectorFont := fieldFont.(font.PDVectorFont); isVectorFont {
			path, err = vectorFont.GetPath(code)
		} else if simpleFont, isSimpleFont := fieldFont.(font.PDSimpleFont); isSimpleFont {
			// these two lines do not always work, e.g. for the TT fonts in file 032431.pdf
			// which is why PDVectorFont is tried first.
			name := simpleFont.Encoding().Name(code)
			path, err = simpleFont.GetPathByName(name)
		} else {
			// shouldn't happen, please open issue in JIRA
			slog.Warn("form: unknown font class", slog.String("class", fmt.Sprintf("%T", fieldFont)))
		}
		if err != nil {
			return 0, err
		}
	}
	if path == nil {
		return -1, nil
	}
	return float32(path.Bounds2D().Height), nil
}

// resolveBoundingBox returns the bounding box of the appearance, falling back to
// the rectangle of the widget where it has none. Java declares it private.
func resolveBoundingBox(fieldWidget *annotation.PDAnnotationWidget,
	appearanceStream *annotation.PDAppearanceStream) *common.PDRectangle {
	boundingBox := appearanceStream.BBox()
	if boundingBox == nil {
		boundingBox = fieldWidget.Rectangle().CreateRetranslatedRectangle()
	}
	return boundingBox
}

// applyPadding returns the box shrunk by the given padding on each side. Java
// declares it private.
func applyPadding(box *common.PDRectangle, padding float32) *common.PDRectangle {
	return common.NewPDRectangleOf(box.LowerLeftX()+padding,
		box.LowerLeftY()+padding,
		box.Width()-2*padding,
		box.Height()-2*padding)
}

// maxFloat32 is java.lang.Math.max over floats.
func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// minFloat32 is java.lang.Math.min over floats.
func minFloat32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// floorDiv is java.lang.Math.floorDiv.
func floorDiv(x, y int) int {
	q := x / y
	if x%y != 0 && (x < 0) != (y < 0) {
		q--
	}
	return q
}

// indexOfString is List.indexOf over a list of strings, which answers -1 where
// the value is not in it.
func indexOfString(list []string, want string) int {
	for i, value := range list {
		if value == want {
			return i
		}
	}
	return -1
}

// nopWriteCloser lets the appearance content stream write into a buffer, which
// is the ByteArrayOutputStream Java hands it. Closing the content stream must
// not close the buffer, because the buffer is written out afterwards.
type nopWriteCloser struct{ *bytes.Buffer }

// Close does nothing.
func (nopWriteCloser) Close() error { return nil }
