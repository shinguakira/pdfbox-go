package fdf

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/shinguakira/pdfbox-go/go/awt"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/w3c/dom"
)

// The bits of the /F flags of an FDF annotation. Java declares them private.
const (
	flagInvisible      = 1
	flagHidden         = 1 << 1
	flagPrinted        = 1 << 2
	flagNoZoom         = 1 << 3
	flagNoRotate       = 1 << 4
	flagNoView         = 1 << 5
	flagReadOnly       = 1 << 6
	flagLocked         = 1 << 7
	flagToggleNoView   = 1 << 8
	flagLockedContents = 1 << 9
)

// FDFAnnotation is an annotation of an FDF document.
//
// Java's FDFAnnotation is an abstract class; the port splits it into this
// interface for the contract and FDFAnnotationBase below for the state.
type FDFAnnotation interface {
	common.COSObjectable

	// AnnotationDictionary returns the annotation dictionary, which
	// getCOSObject narrows to in Java.
	AnnotationDictionary() *cos.Dictionary
}

// FDFAnnotationBase holds the state every FDF annotation shares.
//
// Port of the fields and the concrete methods of the abstract FDFAnnotation.
// Java declares the annot field protected; the port keeps it unexported and the
// concrete annotations embed this.
type FDFAnnotationBase struct {
	annot *cos.Dictionary
}

var _ FDFAnnotation = (*FDFAnnotationBase)(nil)

// initFDFAnnotation is the protected FDFAnnotation() constructor.
func (a *FDFAnnotationBase) initFDFAnnotation() {
	a.annot = cos.NewDictionary()
	a.annot.SetItem(cos.Type, cos.Annot)
}

// initFDFAnnotationOf is the protected FDFAnnotation(COSDictionary)
// constructor.
func (a *FDFAnnotationBase) initFDFAnnotationOf(dictionary *cos.Dictionary) {
	a.annot = dictionary
}

// initFDFAnnotationOfXML is the protected FDFAnnotation(Element) constructor.
func (a *FDFAnnotationBase) initFDFAnnotationOfXML(element *dom.Element) error {
	a.initFDFAnnotation()

	page := element.GetAttribute("page")
	if page == "" {
		return errors.New("Error: missing required attribute 'page'")
	}
	pageNumber, err := strconv.Atoi(page)
	if err != nil {
		// Java's Integer.parseInt throws NumberFormatException, which is
		// unchecked, so the port panics.
		panic(fmt.Sprintf("For input string: %q", page))
	}
	a.SetPage(pageNumber)

	color := element.GetAttribute("color")
	if len(color) == 7 && color[0] == '#' {
		colorValue, err := strconv.ParseInt(color[1:7], 16, 64)
		if err != nil {
			panic(fmt.Sprintf("For input string: %q", color[1:7]))
		}
		c := awt.NewColorOfRGB(int(colorValue))
		a.SetColor(&c)
	}
	a.SetDate(element.GetAttribute("date"))

	flags := element.GetAttribute("flags")
	for _, flagToken := range strings.Split(flags, ",") {
		switch flagToken {
		case "invisible":
			a.SetInvisible(true)
		case "hidden":
			a.SetHidden(true)
		case "print":
			a.SetPrinted(true)
		case "nozoom":
			a.SetNoZoom(true)
		case "norotate":
			a.SetNoRotate(true)
		case "noview":
			a.SetNoView(true)
		case "readonly":
			a.SetReadOnly(true)
		case "locked":
			a.SetLocked(true)
		case "togglenoview":
			a.SetToggleNoView(true)
		}
	}
	a.SetName(element.GetAttribute("name"))

	rect := element.GetAttribute("rect")
	values, err := parseRectangleAttributes(rect,
		"Error: wrong amount of numbers in attribute 'rect'")
	if err != nil {
		return err
	}
	a.SetRectangle(common.NewPDRectangleOfCOSArray(cos.ArrayOfFloats(values)))
	a.SetTitle(element.GetAttribute("title"))

	creationDate, hasCreationDate := util.ToCalendar(element.GetAttribute("creationdate"))
	if hasCreationDate {
		a.SetCreationDate(creationDate)
	}

	opac := element.GetAttribute("opacity")
	if opac != "" {
		a.SetOpacity(parseFloat(opac))
	}
	a.SetSubject(element.GetAttribute("subject"))

	intent := element.GetAttribute("intent")
	if intent == "" {
		// not conforming to spec, but qoppa produces it and Adobe accepts it
		intent = element.GetAttribute("IT")
	}
	if intent != "" {
		a.SetIntent(intent)
	}

	// Java evaluates the XPath "contents[1]" for its string value, which is the
	// text of the first contents child and the empty string where there is
	// none; see dom.FirstElementByTagName.
	contents := dom.FirstElementByTagName(element, "contents")
	if contents != nil {
		a.SetContents(dom.TextContent(contents))
	} else {
		a.SetContents("")
	}

	richContents := dom.FirstElementByTagName(element, "contents-richtext")
	if richContents != nil {
		a.SetRichContents(richContentsToString(richContents, true))
		a.SetContents(strings.TrimSpace(dom.TextContent(richContents)))
	}

	borderStyle := annotation.NewPDBorderStyleDictionary()
	width := element.GetAttribute("width")
	if width != "" {
		borderStyle.SetWidth(parseFloat(width))
	}
	if borderStyle.Width() > 0 {
		style := element.GetAttribute("style")
		if style != "" {
			switch style {
			case "dash":
				borderStyle.SetStyle(annotation.BorderStyleDashed)
			case "bevelled":
				borderStyle.SetStyle(annotation.BorderStyleBeveled)
			case "inset":
				borderStyle.SetStyle(annotation.BorderStyleInset)
			case "underline":
				borderStyle.SetStyle(annotation.BorderStyleSolid)
			case "cloudy":
				borderStyle.SetStyle(annotation.BorderStyleSolid)
				borderEffect := annotation.NewPDBorderEffectDictionary()
				borderEffect.SetStyle(annotation.BorderEffectStyleCloudy)
				intensity := element.GetAttribute("intensity")
				if intensity != "" {
					borderEffect.SetIntensity(parseFloat(element.GetAttribute("intensity")))
				}
				a.SetBorderEffect(borderEffect)
			default:
				borderStyle.SetStyle(annotation.BorderStyleSolid)
			}
		}
		dashes := element.GetAttribute("dashes")
		if dashes != "" {
			dashesValues := strings.Split(dashes, ",")
			dashPattern := cos.NewArray()
			for _, dashesValue := range dashesValues {
				number, err := cos.GetNumber(dashesValue)
				if err != nil {
					return err
				}
				dashPattern.Add(number)
			}
			borderStyle.SetDashStyle(dashPattern)
		}
	}
	a.SetBorderStyle(borderStyle)
	return nil
}

// parseFloat is Float.parseFloat, whose NumberFormatException is unchecked, so
// the port panics.
func parseFloat(value string) float32 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 32)
	if err != nil {
		panic(fmt.Sprintf("For input string: %q", value))
	}
	return float32(parsed)
}

// parseRectangleAttributes reads four numbers out of a comma separated
// attribute. Java declares it final and package-private.
func parseRectangleAttributes(rect string, errorMessage string) ([]float32, error) {
	rectValues := strings.Split(rect, ",")
	if len(rectValues) != 4 {
		return nil, errors.New(errorMessage)
	}
	values := make([]float32, 4)
	values[0] = parseFloat(rectValues[0])
	values[1] = parseFloat(rectValues[1])
	values[2] = parseFloat(rectValues[2])
	values[3] = parseFloat(rectValues[3])
	return values, nil
}

// parseFloats reads a number out of every string. Java declares it final and
// package-private.
func parseFloats(srcValues []string) []float32 {
	values := make([]float32, len(srcValues))
	for i, srcValue := range srcValues {
		values[i] = parseFloat(srcValue)
	}
	return values
}

// createRectangleFromAttributes reads a rectangle out of a comma separated
// attribute. Java declares it final and package-private.
func createRectangleFromAttributes(rect string, errorMessage string) (*common.PDRectangle, error) {
	rectValues := strings.Split(rect, ",")
	if len(rectValues) != 4 {
		return nil, errors.New(errorMessage)
	}
	rectangle := common.NewPDRectangle()
	rectangle.SetLowerLeftX(parseFloat(rectValues[0]))
	rectangle.SetLowerLeftY(parseFloat(rectValues[1]))
	rectangle.SetUpperRightX(parseFloat(rectValues[2]))
	rectangle.SetUpperRightY(parseFloat(rectValues[3]))
	return rectangle, nil
}

// annotationFromDictionary builds the annotation a /Subtype names.
//
// Java writes the chain of else-ifs inline in create; the port lifts it to a
// table so that the annotation types can live in their own file. The entries
// are exactly the branches of that chain.
var annotationFromDictionary = map[string]func(dictionary *cos.Dictionary) FDFAnnotation{}

// CreateFDFAnnotation returns the annotation the given dictionary describes,
// and nil for a nil dictionary or an unknown subtype.
//
// Port of the static FDFAnnotation.create.
func CreateFDFAnnotation(fdfDic *cos.Dictionary) (FDFAnnotation, error) {
	if fdfDic == nil {
		return nil, nil
	}
	fdfDicName := fdfDic.GetNameAsString(cos.Subtype, "")
	factory := annotationFromDictionary[fdfDicName]
	if factory == nil {
		slog.Warn("fdf: unknown or unsupported annotation type",
			slog.String("type", fdfDicName))
		return nil, nil
	}
	return factory(fdfDic), nil
}

// COSObject returns the dictionary.
func (a *FDFAnnotationBase) COSObject() cos.Base { return a.annot }

// AnnotationDictionary returns the dictionary, typed.
func (a *FDFAnnotationBase) AnnotationDictionary() *cos.Dictionary { return a.annot }

// Page returns the 0-based page the annotation is on, and reports whether the
// dictionary carries one -- Java answers a nullable Integer.
//
// Java casts the entry to COSNumber without a check; the port panics where it
// is not one, which is the same unchecked failure.
func (a *FDFAnnotationBase) Page() (int, bool) { return intEntryOf(a.annot, cos.Page) }

// SetPage sets the 0-based page the annotation is on.
func (a *FDFAnnotationBase) SetPage(page int) { a.annot.SetInt(cos.Page, page) }

// Color returns the /C colour of the annotation, or nil where it has none.
func (a *FDFAnnotationBase) Color() *awt.Color { return a.ColorOf(cos.C) }

// ColorOf returns the colour under the given key, or nil where there is none.
// Java declares it final and package-private.
func (a *FDFAnnotationBase) ColorOf(colorName *cos.Name) *awt.Color {
	var retval *awt.Color
	array := a.annot.GetCOSArray(colorName)
	if array != nil {
		rgb := array.ToFloatArray()
		if len(rgb) >= 3 {
			c := awt.NewColor(rgb[0], rgb[1], rgb[2])
			retval = &c
		}
	}
	return retval
}

// SetColor sets the /C colour of the annotation, and removes it for a nil one.
func (a *FDFAnnotationBase) SetColor(c *awt.Color) {
	var color *cos.Array
	if c != nil {
		r, g, b := c.RGBColorComponents()
		color = cos.ArrayOfFloats([]float32{r, g, b})
	}
	a.annot.SetItem(cos.C, color)
}

// Date returns the /M date of the annotation, or the empty string where it has
// none, which is the null Java answers.
func (a *FDFAnnotationBase) Date() string { return a.annot.GetString(cos.M, "") }

// SetDate sets the /M date of the annotation.
func (a *FDFAnnotationBase) SetDate(date string) { a.annot.SetString(cos.M, date) }

// IsInvisible reports the invisible flag.
func (a *FDFAnnotationBase) IsInvisible() bool { return a.annot.GetFlag(cos.F, flagInvisible) }

// SetInvisible sets the invisible flag.
func (a *FDFAnnotationBase) SetInvisible(invisible bool) {
	a.annot.SetFlag(cos.F, flagInvisible, invisible)
}

// IsHidden reports the hidden flag.
func (a *FDFAnnotationBase) IsHidden() bool { return a.annot.GetFlag(cos.F, flagHidden) }

// SetHidden sets the hidden flag.
func (a *FDFAnnotationBase) SetHidden(hidden bool) { a.annot.SetFlag(cos.F, flagHidden, hidden) }

// IsPrinted reports the printed flag.
func (a *FDFAnnotationBase) IsPrinted() bool { return a.annot.GetFlag(cos.F, flagPrinted) }

// SetPrinted sets the printed flag.
func (a *FDFAnnotationBase) SetPrinted(printed bool) {
	a.annot.SetFlag(cos.F, flagPrinted, printed)
}

// IsNoZoom reports the no zoom flag.
func (a *FDFAnnotationBase) IsNoZoom() bool { return a.annot.GetFlag(cos.F, flagNoZoom) }

// SetNoZoom sets the no zoom flag.
func (a *FDFAnnotationBase) SetNoZoom(noZoom bool) { a.annot.SetFlag(cos.F, flagNoZoom, noZoom) }

// IsNoRotate reports the no rotate flag.
func (a *FDFAnnotationBase) IsNoRotate() bool { return a.annot.GetFlag(cos.F, flagNoRotate) }

// SetNoRotate sets the no rotate flag.
func (a *FDFAnnotationBase) SetNoRotate(noRotate bool) {
	a.annot.SetFlag(cos.F, flagNoRotate, noRotate)
}

// IsNoView reports the no view flag.
func (a *FDFAnnotationBase) IsNoView() bool { return a.annot.GetFlag(cos.F, flagNoView) }

// SetNoView sets the no view flag.
func (a *FDFAnnotationBase) SetNoView(noView bool) { a.annot.SetFlag(cos.F, flagNoView, noView) }

// IsReadOnly reports the read only flag.
func (a *FDFAnnotationBase) IsReadOnly() bool { return a.annot.GetFlag(cos.F, flagReadOnly) }

// SetReadOnly sets the read only flag.
func (a *FDFAnnotationBase) SetReadOnly(readOnly bool) {
	a.annot.SetFlag(cos.F, flagReadOnly, readOnly)
}

// IsLocked reports the locked flag.
func (a *FDFAnnotationBase) IsLocked() bool { return a.annot.GetFlag(cos.F, flagLocked) }

// SetLocked sets the locked flag.
func (a *FDFAnnotationBase) SetLocked(locked bool) { a.annot.SetFlag(cos.F, flagLocked, locked) }

// IsToggleNoView reports the toggle no view flag.
func (a *FDFAnnotationBase) IsToggleNoView() bool {
	return a.annot.GetFlag(cos.F, flagToggleNoView)
}

// SetToggleNoView sets the toggle no view flag.
func (a *FDFAnnotationBase) SetToggleNoView(toggleNoView bool) {
	a.annot.SetFlag(cos.F, flagToggleNoView, toggleNoView)
}

// IsLockedContents reports the locked contents flag.
func (a *FDFAnnotationBase) IsLockedContents() bool {
	return a.annot.GetFlag(cos.F, flagLockedContents)
}

// SetLockedContents sets the locked contents flag.
func (a *FDFAnnotationBase) SetLockedContents(lockedContents bool) {
	a.annot.SetFlag(cos.F, flagLockedContents, lockedContents)
}

// SetName sets the /NM of the annotation.
func (a *FDFAnnotationBase) SetName(name string) { a.annot.SetString(cos.NM, name) }

// Name returns the /NM of the annotation, or the empty string where it has
// none.
func (a *FDFAnnotationBase) Name() string { return a.annot.GetString(cos.NM, "") }

// SetRectangle sets the /Rect of the annotation.
func (a *FDFAnnotationBase) SetRectangle(rectangle *common.PDRectangle) {
	a.annot.SetItem(cos.Rect, common.COSObjectOrNil(rectangle))
}

// Rectangle returns the /Rect of the annotation, or nil where it has none.
func (a *FDFAnnotationBase) Rectangle() *common.PDRectangle {
	rectArray := a.annot.GetCOSArray(cos.Rect)
	if rectArray != nil {
		return common.NewPDRectangleOfCOSArray(rectArray)
	}
	return nil
}

// SetContents sets the /Contents of the annotation.
func (a *FDFAnnotationBase) SetContents(contents string) {
	a.annot.SetString(cos.Contents, contents)
}

// Contents returns the /Contents of the annotation, or the empty string where
// it has none.
func (a *FDFAnnotationBase) Contents() string { return a.annot.GetString(cos.Contents, "") }

// SetTitle sets the /T of the annotation.
func (a *FDFAnnotationBase) SetTitle(title string) { a.annot.SetString(cos.T, title) }

// Title returns the /T of the annotation, or the empty string where it has
// none.
func (a *FDFAnnotationBase) Title() string { return a.annot.GetString(cos.T, "") }

// CreationDate returns the /CreationDate of the annotation, and reports whether
// it has one.
func (a *FDFAnnotationBase) CreationDate() (time.Time, bool) {
	return util.DictionaryDate(a.annot, cos.CreationDate)
}

// SetCreationDate sets the /CreationDate of the annotation.
func (a *FDFAnnotationBase) SetCreationDate(date time.Time) {
	util.SetDictionaryDate(a.annot, cos.CreationDate, date)
}

// SetOpacity sets the /CA of the annotation.
func (a *FDFAnnotationBase) SetOpacity(opacity float32) { a.annot.SetFloat(cos.CA, opacity) }

// Opacity returns the /CA of the annotation, which is 1 where it has none.
func (a *FDFAnnotationBase) Opacity() float32 { return a.annot.GetFloat(cos.CA, 1) }

// SetSubject sets the /Subj of the annotation.
func (a *FDFAnnotationBase) SetSubject(subject string) { a.annot.SetString(cos.Subj, subject) }

// Subject returns the /Subj of the annotation, or the empty string where it has
// none.
func (a *FDFAnnotationBase) Subject() string { return a.annot.GetString(cos.Subj, "") }

// SetIntent sets the /IT of the annotation.
func (a *FDFAnnotationBase) SetIntent(intent string) { a.annot.SetName(cos.IT, intent) }

// Intent returns the /IT of the annotation, or the empty string where it has
// none.
func (a *FDFAnnotationBase) Intent() string { return a.annot.GetNameAsString(cos.IT, "") }

// RichContents returns the /RC of the annotation, or the empty string where it
// has none.
func (a *FDFAnnotationBase) RichContents() string {
	return stringOrStream(a.annot.GetDictionaryObject(cos.RC))
}

// SetRichContents sets the /RC of the annotation.
func (a *FDFAnnotationBase) SetRichContents(rc string) {
	a.annot.SetItem(cos.RC, cos.NewStringObj(rc))
}

// SetBorderStyle sets the /BS of the annotation.
func (a *FDFAnnotationBase) SetBorderStyle(bs *annotation.PDBorderStyleDictionary) {
	a.annot.SetItem(cos.BS, common.COSObjectOrNil(bs))
}

// BorderStyle returns the /BS of the annotation, or nil where it has none.
func (a *FDFAnnotationBase) BorderStyle() *annotation.PDBorderStyleDictionary {
	bs := a.annot.GetCOSDictionary(cos.BS)
	if bs != nil {
		return annotation.NewPDBorderStyleDictionaryOf(bs)
	}
	return nil
}

// SetBorderEffect sets the /BE of the annotation.
func (a *FDFAnnotationBase) SetBorderEffect(be *annotation.PDBorderEffectDictionary) {
	a.annot.SetItem(cos.BE, common.COSObjectOrNil(be))
}

// BorderEffect returns the /BE of the annotation, or nil where it has none.
func (a *FDFAnnotationBase) BorderEffect() *annotation.PDBorderEffectDictionary {
	be := a.annot.GetCOSDictionary(cos.BE)
	if be != nil {
		return annotation.NewPDBorderEffectDictionaryOf(be)
	}
	return nil
}

// stringOrStream reads an entry that may be written as a string or as a stream,
// and answers the empty string for anything else.
//
// Port of the protected final getStringOrStream.
func stringOrStream(base cos.Base) string {
	switch typed := base.(type) {
	case nil:
		return ""
	case *cos.StringObj:
		return typed.Value()
	case *cos.Stream:
		return typed.ToTextString()
	}
	return ""
}

// richContentsToString renders the given node back out as XML, which is how the
// /RC of an annotation is built out of its contents-richtext element. Java
// declares it private.
func richContentsToString(node dom.Node, root bool) string {
	sb := &strings.Builder{}

	nodelist := node.ChildNodes()
	for i := 0; i < nodelist.Length(); i++ {
		child := nodelist.Item(i)
		switch typed := child.(type) {
		case *dom.Element:
			sb.WriteString(richContentsToString(child, false))
		case *dom.Text:
			if typed.IsCDATASection() {
				sb.WriteString("<![CDATA[")
				sb.WriteString(typed.Data())
				sb.WriteString("]]>")
				continue
			}
			cdata := typed.Data()
			cdata = strings.ReplaceAll(cdata, "&", "&amp;")
			cdata = strings.ReplaceAll(cdata, "<", "&lt;")
			sb.WriteString(cdata)
		}
	}
	if root {
		return sb.String()
	}

	attributes := node.Attributes()
	builder := &strings.Builder{}
	for i := 0; i < attributes.Length(); i++ {
		attribute := attributes.Item(i)
		attributeNodeValue := strings.ReplaceAll(attribute.NodeValue(), "\"", "&quot;")
		fmt.Fprintf(builder, " %s=\"%s\"", attribute.NodeName(), attributeNodeValue)
	}
	return fmt.Sprintf("<%s%s>%s</%s>", node.NodeName(), builder, sb, node.NodeName())
}
