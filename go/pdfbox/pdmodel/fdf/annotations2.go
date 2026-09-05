package fdf

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/w3c/dom"
)

// FDFAnnotationFreeText is a free text annotation of an FDF document.
//
// Port of FDFAnnotationFreeText.
type FDFAnnotationFreeText struct{ FDFAnnotationBase }

// NewFDFAnnotationFreeText returns an empty free text annotation.
func NewFDFAnnotationFreeText() *FDFAnnotationFreeText {
	a := &FDFAnnotationFreeText{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeFreeText)
	return a
}

// NewFDFAnnotationFreeTextOf returns the free text annotation the given
// dictionary holds.
func NewFDFAnnotationFreeTextOf(dictionary *cos.Dictionary) *FDFAnnotationFreeText {
	a := &FDFAnnotationFreeText{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationFreeTextOfXML returns the free text annotation the given XFDF
// element describes.
func NewFDFAnnotationFreeTextOfXML(element *dom.Element) (*FDFAnnotationFreeText, error) {
	a := &FDFAnnotationFreeText{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeFreeText)
	a.SetJustification(element.GetAttribute("justification"))
	a.SetDefaultAppearance(firstElementText(element, "defaultappearance"))
	a.SetDefaultStyle(firstElementText(element, "defaultstyle"))
	a.initCallout(element)
	rotation := element.GetAttribute("rotation")
	if rotation != "" {
		rotationValue, err := strconv.Atoi(rotation)
		if err != nil {
			panic(fmt.Sprintf("For input string: %q", rotation))
		}
		a.SetRotation(rotationValue)
	}
	if err := a.initFringe(element); err != nil {
		return nil, err
	}
	lineEndingStyle := element.GetAttribute("head")
	if lineEndingStyle != "" {
		a.SetLineEndingStyle(lineEndingStyle)
	}
	return a, nil
}

// firstElementText returns the text of the first child element with the given
// name, which is the string value of the XPath step Java evaluates, and the
// empty string where there is no such element.
func firstElementText(element *dom.Element, tagName string) string {
	if first := dom.FirstElementByTagName(element, tagName); first != nil {
		return dom.TextContent(first)
	}
	return ""
}

// initCallout reads the callout attribute and sets it. Java declares it private.
func (a *FDFAnnotationFreeText) initCallout(element *dom.Element) {
	callout := element.GetAttribute("callout")
	if callout != "" {
		calloutValues := strings.Split(callout, ",")
		a.SetCallout(parseFloats(calloutValues))
	}
}

// SetCallout sets the /CL of the annotation.
func (a *FDFAnnotationFreeText) SetCallout(callout []float32) {
	a.annot.SetItem(cos.CL, cos.ArrayOfFloats(callout))
}

// Callout returns the /CL of the annotation, or nil where it has none.
func (a *FDFAnnotationFreeText) Callout() []float32 {
	array := a.annot.GetCOSArray(cos.CL)
	if array != nil {
		return array.ToFloatArray()
	}
	return nil
}

// SetJustification sets the /Q of the annotation from the XFDF justification
// name.
func (a *FDFAnnotationFreeText) SetJustification(justification string) {
	quadding := 0
	if justification == "centered" {
		quadding = 1
	} else if justification == "right" {
		quadding = 2
	}
	a.annot.SetInt(cos.Q, quadding)
}

// Justification returns the /Q of the annotation, written out as a number.
func (a *FDFAnnotationFreeText) Justification() string {
	return strconv.Itoa(a.annot.GetIntDefault(cos.Q, 0))
}

// SetRotation sets the /Rotate of the annotation.
func (a *FDFAnnotationFreeText) SetRotation(rotation int) { a.annot.SetInt(cos.Rotate, rotation) }

// Rotation returns the /Rotate of the annotation, or the empty string where it
// has none.
//
// Java reads the entry with getString, which answers null for the integer
// setRotation wrote; see migration/JAVA-BUGS.md.
func (a *FDFAnnotationFreeText) Rotation() string { return a.annot.GetString(cos.Rotate, "") }

// SetDefaultAppearance sets the /DA of the annotation.
func (a *FDFAnnotationFreeText) SetDefaultAppearance(appearance string) {
	a.annot.SetString(cos.DA, appearance)
}

// DefaultAppearance returns the /DA of the annotation, or the empty string where
// it has none.
func (a *FDFAnnotationFreeText) DefaultAppearance() string { return a.annot.GetString(cos.DA, "") }

// SetDefaultStyle sets the /DS of the annotation.
func (a *FDFAnnotationFreeText) SetDefaultStyle(style string) { a.annot.SetString(cos.DS, style) }

// DefaultStyle returns the /DS of the annotation, or the empty string where it
// has none.
func (a *FDFAnnotationFreeText) DefaultStyle() string { return a.annot.GetString(cos.DS, "") }

// SetFringe sets the /RD of the annotation.
func (a *FDFAnnotationFreeText) SetFringe(fringe *common.PDRectangle) { a.setFringe(fringe) }

// Fringe returns the /RD of the annotation, or nil where it has none.
func (a *FDFAnnotationFreeText) Fringe() *common.PDRectangle { return a.fringe() }

// SetLineEndingStyle sets the /LE of the annotation.
func (a *FDFAnnotationFreeText) SetLineEndingStyle(style string) { a.annot.SetName(cos.LE, style) }

// LineEndingStyle returns the /LE of the annotation, or the empty string where
// it has none.
func (a *FDFAnnotationFreeText) LineEndingStyle() string {
	return a.annot.GetNameAsString(cos.LE, "")
}

// FDFAnnotationLine is a line annotation of an FDF document.
//
// Port of FDFAnnotationLine.
type FDFAnnotationLine struct{ FDFAnnotationBase }

// NewFDFAnnotationLine returns an empty line annotation.
func NewFDFAnnotationLine() *FDFAnnotationLine {
	a := &FDFAnnotationLine{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeLine)
	return a
}

// NewFDFAnnotationLineOf returns the line annotation the given dictionary holds.
func NewFDFAnnotationLineOf(dictionary *cos.Dictionary) *FDFAnnotationLine {
	a := &FDFAnnotationLine{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationLineOfXML returns the line annotation the given XFDF element
// describes.
func NewFDFAnnotationLineOfXML(element *dom.Element) (*FDFAnnotationLine, error) {
	a := &FDFAnnotationLine{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeLine)

	startCoords := element.GetAttribute("start")
	if startCoords == "" {
		return nil, errors.New("Error: missing attribute 'start'")
	}
	endCoords := element.GetAttribute("end")
	if endCoords == "" {
		return nil, errors.New("Error: missing attribute 'end'")
	}
	line := startCoords + "," + endCoords
	values, err := parseRectangleAttributes(line, "Error: wrong amount of line coordinates")
	if err != nil {
		return nil, err
	}
	a.SetLine(values)

	leaderLine := element.GetAttribute("leaderLength")
	if leaderLine != "" {
		a.SetLeaderLength(parseFloat(leaderLine))
	}
	leaderLineExtension := element.GetAttribute("leaderExtend")
	if leaderLineExtension != "" {
		a.SetLeaderExtend(parseFloat(leaderLineExtension))
	}
	leaderLineOffset := element.GetAttribute("leaderOffset")
	if leaderLineOffset != "" {
		a.SetLeaderOffset(parseFloat(leaderLineOffset))
	}
	startStyle := element.GetAttribute("head")
	if startStyle != "" {
		a.SetStartPointEndingStyle(startStyle)
	}
	endStyle := element.GetAttribute("tail")
	if endStyle != "" {
		a.SetEndPointEndingStyle(endStyle)
	}
	a.interiorColorOfAttribute(element)

	caption := element.GetAttribute("caption")
	if caption == "yes" {
		a.SetCaption(true)
		captionH := element.GetAttribute("caption-offset-h")
		if captionH != "" {
			a.SetCaptionHorizontalOffset(parseFloat(captionH))
		}
		captionV := element.GetAttribute("caption-offset-v")
		if captionV != "" {
			a.SetCaptionVerticalOffset(parseFloat(captionV))
		}
		captionStyle := element.GetAttribute("caption-style")
		if captionStyle != "" {
			a.SetCaptionStyle(captionStyle)
		}
	}
	return a, nil
}

// SetLine sets the /L of the annotation.
func (a *FDFAnnotationLine) SetLine(line []float32) {
	a.annot.SetItem(cos.L, cos.ArrayOfFloats(line))
}

// Line returns the /L of the annotation, or nil where it has none.
func (a *FDFAnnotationLine) Line() []float32 {
	array := a.annot.GetCOSArray(cos.L)
	if array != nil {
		return array.ToFloatArray()
	}
	return nil
}

// SetStartPointEndingStyle sets the first entry of the /LE of the annotation.
func (a *FDFAnnotationLine) SetStartPointEndingStyle(style string) {
	a.setStartPointEndingStyle(style)
}

// StartPointEndingStyle returns the first entry of the /LE of the annotation.
func (a *FDFAnnotationLine) StartPointEndingStyle() string { return a.startPointEndingStyle() }

// SetEndPointEndingStyle sets the second entry of the /LE of the annotation.
func (a *FDFAnnotationLine) SetEndPointEndingStyle(style string) {
	a.setEndPointEndingStyle(style)
}

// EndPointEndingStyle returns the second entry of the /LE of the annotation.
func (a *FDFAnnotationLine) EndPointEndingStyle() string { return a.endPointEndingStyle() }

// SetInteriorColor sets the /IC of the annotation.
func (a *FDFAnnotationLine) SetInteriorColor(color *awt.Color) { a.setInteriorColor(color) }

// InteriorColor returns the /IC of the annotation, or nil where it has none.
func (a *FDFAnnotationLine) InteriorColor() *awt.Color { return a.interiorColor() }

// SetCaption sets the /Cap of the annotation.
func (a *FDFAnnotationLine) SetCaption(cap bool) { a.annot.SetBoolean(cos.Cap, cap) }

// Caption returns the /Cap of the annotation.
func (a *FDFAnnotationLine) Caption() bool { return a.annot.GetBoolean(cos.Cap, false) }

// LeaderLength returns the /LL of the annotation, which is -1 where it has none.
func (a *FDFAnnotationLine) LeaderLength() float32 { return a.annot.GetFloat(cos.LL, -1) }

// SetLeaderLength sets the /LL of the annotation.
func (a *FDFAnnotationLine) SetLeaderLength(leaderLength float32) {
	a.annot.SetFloat(cos.LL, leaderLength)
}

// LeaderExtend returns the /LLE of the annotation, which is -1 where it has
// none.
func (a *FDFAnnotationLine) LeaderExtend() float32 { return a.annot.GetFloat(cos.LLE, -1) }

// SetLeaderExtend sets the /LLE of the annotation.
func (a *FDFAnnotationLine) SetLeaderExtend(leaderExtend float32) {
	a.annot.SetFloat(cos.LLE, leaderExtend)
}

// LeaderOffset returns the /LLO of the annotation, which is -1 where it has
// none.
func (a *FDFAnnotationLine) LeaderOffset() float32 { return a.annot.GetFloat(cos.LLO, -1) }

// SetLeaderOffset sets the /LLO of the annotation.
func (a *FDFAnnotationLine) SetLeaderOffset(leaderOffset float32) {
	a.annot.SetFloat(cos.LLO, leaderOffset)
}

// CaptionStyle returns the /CP of the annotation, or the empty string where it
// has none.
func (a *FDFAnnotationLine) CaptionStyle() string { return a.annot.GetString(cos.CP, "") }

// SetCaptionStyle sets the /CP of the annotation.
func (a *FDFAnnotationLine) SetCaptionStyle(captionStyle string) {
	a.annot.SetString(cos.CP, captionStyle)
}

// SetCaptionHorizontalOffset sets the first entry of the /CO of the annotation.
func (a *FDFAnnotationLine) SetCaptionHorizontalOffset(offset float32) {
	array := a.annot.GetCOSArray(cos.CO)
	if array == nil {
		array = cos.ArrayOfFloats([]float32{offset, 0})
		a.annot.SetItem(cos.CO, array)
	} else {
		array.Set(0, cos.NewFloat(offset))
	}
}

// CaptionHorizontalOffset returns the first entry of the /CO of the annotation,
// which is zero where it has none.
func (a *FDFAnnotationLine) CaptionHorizontalOffset() float32 {
	array := a.annot.GetCOSArray(cos.CO)
	if array != nil {
		return array.ToFloatArray()[0]
	}
	return 0
}

// SetCaptionVerticalOffset sets the second entry of the /CO of the annotation.
func (a *FDFAnnotationLine) SetCaptionVerticalOffset(offset float32) {
	array := a.annot.GetCOSArray(cos.CO)
	if array == nil {
		a.annot.SetItem(cos.CO, cos.ArrayOfFloats([]float32{0, offset}))
	} else {
		array.Set(1, cos.NewFloat(offset))
	}
}

// CaptionVerticalOffset returns the second entry of the /CO of the annotation,
// which is zero where it has none.
func (a *FDFAnnotationLine) CaptionVerticalOffset() float32 {
	array := a.annot.GetCOSArray(cos.CO)
	if array != nil {
		return array.ToFloatArray()[1]
	}
	return 0
}

// FDFAnnotationStamp is a stamp annotation of an FDF document.
//
// Port of FDFAnnotationStamp.
type FDFAnnotationStamp struct{ FDFAnnotationBase }

// NewFDFAnnotationStamp returns an empty stamp annotation.
func NewFDFAnnotationStamp() *FDFAnnotationStamp {
	a := &FDFAnnotationStamp{}
	a.initFDFAnnotation()
	a.annot.SetName(cos.Subtype, SubTypeStamp)
	return a
}

// NewFDFAnnotationStampOf returns the stamp annotation the given dictionary
// holds.
func NewFDFAnnotationStampOf(dictionary *cos.Dictionary) *FDFAnnotationStamp {
	a := &FDFAnnotationStamp{}
	a.initFDFAnnotationOf(dictionary)
	return a
}

// NewFDFAnnotationStampOfXML returns the stamp annotation the given XFDF element
// describes.
func NewFDFAnnotationStampOfXML(element *dom.Element) (*FDFAnnotationStamp, error) {
	a := &FDFAnnotationStamp{}
	if err := a.initFDFAnnotationOfXML(element); err != nil {
		return nil, err
	}
	a.annot.SetName(cos.Subtype, SubTypeStamp)

	// PDFBOX-4437: Initialize the Stamp appearance from the XFDF
	// https://www.immagic.com/eLibrary/ARCHIVES/TECH/ADOBE/A070914X.pdf
	// appearance is only defined for stamps

	// Set the Appearance to the annotation
	slog.Debug("fdf: get the DOM Document for the stamp appearance")
	base64EncodedAppearance := firstElementText(element, "appearance")
	decodedAppearanceXML, err := util.DecodeBase64(base64EncodedAppearance)
	if err != nil {
		slog.Error("fdf: bad base64 encoded appearance ignored", slog.Any("err", err))
		return a, nil
	}
	if base64EncodedAppearance != "" {
		slog.Debug("fdf: decoded XML", slog.String("xml", string(decodedAppearanceXML)))
		stampAppearance, err := util.XMLParse(strings.NewReader(string(decodedAppearanceXML)))
		if err != nil {
			return nil, err
		}
		appearanceEl := stampAppearance.DocumentElement()
		// Is the root node have tag as DICT, error otherwise
		if !strings.EqualFold(appearanceEl.NodeName(), "dict") {
			return nil, fmt.Errorf(
				"Error while reading stamp document, root should be 'dict' and not '%s'",
				appearanceEl.NodeName())
		}
		slog.Debug("fdf: generate and set the appearance dictionary to the stamp annotation")
		appearance, err := parseStampAnnotationAppearanceXML(appearanceEl)
		if err != nil {
			return nil, err
		}
		a.annot.SetItem(cos.AP, appearance)
	}
	return a, nil
}

// parseStampAnnotationAppearanceXML builds the /AP dictionary of the stamp out
// of the appearance XML. Java declares it private.
func parseStampAnnotationAppearanceXML(appearanceXML *dom.Element) (*cos.Dictionary, error) {
	dictionary := cos.NewDictionary()
	// the N entry is required.
	dictionary.SetItem(cos.N, cos.NewStream(nil))
	slog.Debug("fdf: build dictionary for Appearance based on the appearanceXML")

	nodeList := appearanceXML.ChildNodes()
	parentAttrKey := appearanceXML.GetAttribute("KEY")
	slog.Debug("fdf: appearance root",
		slog.String("tag", appearanceXML.TagName()),
		slog.String("name", appearanceXML.NodeName()),
		slog.String("key", parentAttrKey),
		slog.Int("children", nodeList.Length()))

	// Currently only handles Appearance dictionary (AP key on the root)
	if appearanceXML.GetAttribute("KEY") != "AP" {
		slog.Warn("fdf: not handling element",
			slog.String("parent", parentAttrKey),
			slog.String("tag", appearanceXML.TagName()),
			slog.String("key", appearanceXML.GetAttribute("KEY")))
		return dictionary, nil
	}
	for i := 0; i < nodeList.Length(); i++ {
		node := nodeList.Item(i)
		child, isElement := node.(*dom.Element)
		if !isElement {
			continue
		}
		childTagName := child.TagName()
		if strings.EqualFold(childTagName, "STREAM") {
			slog.Debug("fdf: process item in the dictionary after processing",
				slog.String("parent", parentAttrKey),
				slog.String("key", child.GetAttribute("KEY")),
				slog.String("tag", childTagName))
			stream, err := parseStreamElement(child)
			if err != nil {
				return nil, err
			}
			dictionary.SetItem(cos.GetPDFName(child.GetAttribute("KEY")), stream)
			slog.Debug("fdf: set",
				slog.String("parent", parentAttrKey),
				slog.String("key", child.GetAttribute("KEY")))
		} else {
			slog.Warn("fdf: not handling element",
				slog.String("parent", parentAttrKey), slog.String("tag", childTagName))
		}
	}
	return dictionary, nil
}

// parseStreamElement builds a stream out of a STREAM element. Java declares it
// private.
func parseStreamElement(streamEl *dom.Element) (*cos.Stream, error) {
	slog.Debug("fdf: parse stream", slog.String("key", streamEl.GetAttribute("KEY")))
	stream := cos.NewStream(nil)
	nodeList := streamEl.ChildNodes()
	parentAttrKey := streamEl.GetAttribute("KEY")
	for i := 0; i < nodeList.Length(); i++ {
		node := nodeList.Item(i)
		child, isElement := node.(*dom.Element)
		if !isElement {
			continue
		}
		childAttrKey := child.GetAttribute("KEY")
		childAttrVal := child.GetAttribute("VAL")
		childTagName := child.TagName()
		slog.Debug("fdf: reading child",
			slog.String("parent", parentAttrKey),
			slog.String("tag", childTagName),
			slog.String("key", childAttrKey))
		switch strings.ToUpper(childTagName) {
		case "INT":
			if childAttrKey != "Length" {
				value, err := strconv.Atoi(childAttrVal)
				if err != nil {
					panic(fmt.Sprintf("For input string: %q", childAttrVal))
				}
				stream.SetInt(cos.GetPDFName(childAttrKey), value)
			}
		case "FIXED":
			stream.SetFloat(cos.GetPDFName(childAttrKey), parseFloat(childAttrVal))
		case "NAME":
			stream.SetName(cos.GetPDFName(childAttrKey), childAttrVal)
		case "BOOL":
			stream.SetBoolean(cos.GetPDFName(childAttrKey), parseBoolean(childAttrVal))
		case "ARRAY":
			array, err := parseArrayElement(child)
			if err != nil {
				return nil, err
			}
			stream.SetItem(cos.GetPDFName(childAttrKey), array)
		case "DICT":
			dict, err := parseDictElement(child)
			if err != nil {
				return nil, err
			}
			stream.SetItem(cos.GetPDFName(childAttrKey), dict)
		case "STREAM":
			inner, err := parseStreamElement(child)
			if err != nil {
				return nil, err
			}
			stream.SetItem(cos.GetPDFName(childAttrKey), inner)
		case "DATA":
			childEncodingAttr := child.GetAttribute("ENCODING")
			slog.Debug("fdf: handling DATA",
				slog.String("parent", parentAttrKey),
				slog.String("encoding", childEncodingAttr))
			switch childEncodingAttr {
			case "HEX":
				os, err := stream.CreateRawWriter()
				if err != nil {
					return nil, err
				}
				if _, err := os.Write(util.DecodeHex(dom.TextContent(child))); err != nil {
					os.Close()
					return nil, err
				}
				if err := os.Close(); err != nil {
					return nil, err
				}
			case "ASCII":
				os, err := stream.CreateWriter()
				if err != nil {
					return nil, err
				}
				// Java asks the document for its encoding and writes the text in
				// it; the port reads and writes UTF-8 throughout, which is what
				// the parser answers here, so the text goes out as it stands.
				if _, err := os.Write([]byte(dom.TextContent(child))); err != nil {
					os.Close()
					return nil, err
				}
				if err := os.Close(); err != nil {
					return nil, err
				}
			default:
				slog.Warn("fdf: not handling element DATA encoding",
					slog.String("parent", parentAttrKey),
					slog.String("encoding", childEncodingAttr))
			}
		default:
			slog.Warn("fdf: not handling child element",
				slog.String("parent", parentAttrKey), slog.String("tag", childTagName))
		}
	}
	return stream, nil
}

// parseArrayElement builds an array out of an ARRAY element. Java declares it
// private.
func parseArrayElement(arrayEl *dom.Element) (*cos.Array, error) {
	slog.Debug("fdf: parse array", slog.String("key", arrayEl.GetAttribute("KEY")))
	array := cos.NewArray()
	nodeList := arrayEl.ChildNodes()
	parentAttrKey := arrayEl.GetAttribute("KEY")
	if parentAttrKey == "BBox" && nodeList.Length() < 4 {
		return nil, fmt.Errorf("BBox does not have enough coordinates, only has: %d",
			nodeList.Length())
	} else if parentAttrKey == "Matrix" && nodeList.Length() < 6 {
		return nil, fmt.Errorf("Matrix does not have enough coordinates, only has: %d",
			nodeList.Length())
	}
	for i := 0; i < nodeList.Length(); i++ {
		node := nodeList.Item(i)
		child, isElement := node.(*dom.Element)
		if !isElement {
			continue
		}
		childAttrKey := child.GetAttribute("KEY")
		childAttrVal := child.GetAttribute("VAL")
		childTagName := child.TagName()
		slog.Debug("fdf: reading child",
			slog.String("parent", parentAttrKey),
			slog.String("tag", childTagName),
			slog.String("key", childAttrKey))
		switch strings.ToUpper(childTagName) {
		case "INT", "FIXED":
			number, err := cos.GetNumber(childAttrVal)
			if err != nil {
				return nil, err
			}
			array.Add(number)
		case "NAME":
			array.Add(cos.GetPDFName(childAttrVal))
		case "BOOL":
			array.Add(cos.GetBoolean(parseBoolean(childAttrVal)))
		case "DICT":
			dict, err := parseDictElement(child)
			if err != nil {
				return nil, err
			}
			array.Add(dict)
		case "STREAM":
			stream, err := parseStreamElement(child)
			if err != nil {
				return nil, err
			}
			array.Add(stream)
		case "ARRAY":
			inner, err := parseArrayElement(child)
			if err != nil {
				return nil, err
			}
			array.Add(inner)
		default:
			slog.Warn("fdf: not handling child element",
				slog.String("parent", parentAttrKey), slog.String("tag", childTagName))
		}
	}
	return array, nil
}

// parseDictElement builds a dictionary out of a DICT element. Java declares it
// private.
//
// Java switches on the tag name as it stands here, rather than on its upper
// case as the stream and array branches do.
func parseDictElement(dictEl *dom.Element) (*cos.Dictionary, error) {
	slog.Debug("fdf: parse dictionary", slog.String("key", dictEl.GetAttribute("KEY")))
	dict := cos.NewDictionary()
	nodeList := dictEl.ChildNodes()
	parentAttrKey := dictEl.GetAttribute("KEY")
	for i := 0; i < nodeList.Length(); i++ {
		node := nodeList.Item(i)
		child, isElement := node.(*dom.Element)
		if !isElement {
			continue
		}
		childAttrKey := child.GetAttribute("KEY")
		childAttrVal := child.GetAttribute("VAL")
		childTagName := child.TagName()
		switch childTagName {
		case "DICT":
			inner, err := parseDictElement(child)
			if err != nil {
				return nil, err
			}
			dict.SetItem(cos.GetPDFName(childAttrKey), inner)
		case "STREAM":
			stream, err := parseStreamElement(child)
			if err != nil {
				return nil, err
			}
			dict.SetItem(cos.GetPDFName(childAttrKey), stream)
		case "NAME":
			dict.SetName(cos.GetPDFName(childAttrKey), childAttrVal)
		case "INT":
			value, err := strconv.Atoi(childAttrVal)
			if err != nil {
				panic(fmt.Sprintf("For input string: %q", childAttrVal))
			}
			dict.SetInt(cos.GetPDFName(childAttrKey), value)
		case "FIXED":
			dict.SetFloat(cos.GetPDFName(childAttrKey), parseFloat(childAttrVal))
		case "BOOL":
			dict.SetBoolean(cos.GetPDFName(childAttrKey), parseBoolean(childAttrVal))
		case "ARRAY":
			array, err := parseArrayElement(child)
			if err != nil {
				return nil, err
			}
			dict.SetItem(cos.GetPDFName(childAttrKey), array)
		default:
			slog.Warn("fdf: NOT handling child element",
				slog.String("parent", parentAttrKey), slog.String("tag", childTagName))
		}
	}
	return dict, nil
}

// parseBoolean is Boolean.parseBoolean, which is true only for "true" in any
// case and false for everything else, the empty string included.
func parseBoolean(value string) bool { return strings.EqualFold(value, "true") }

// init fills the two tables the FDF reading dispatches through, which Java
// writes as a switch inside FDFDictionary(Element) and a chain of else-ifs
// inside FDFAnnotation.create.
func init() {
	annotationFromXML["text"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationTextOfXML(e)
	}
	annotationFromXML["caret"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationCaretOfXML(e)
	}
	annotationFromXML["freetext"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationFreeTextOfXML(e)
	}
	annotationFromXML["fileattachment"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationFileAttachmentOfXML(e)
	}
	annotationFromXML["highlight"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationHighlightOfXML(e)
	}
	annotationFromXML["ink"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationInkOfXML(e)
	}
	annotationFromXML["line"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationLineOfXML(e)
	}
	annotationFromXML["link"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationLinkOfXML(e)
	}
	annotationFromXML["circle"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationCircleOfXML(e)
	}
	annotationFromXML["square"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationSquareOfXML(e)
	}
	annotationFromXML["polygon"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationPolygonOfXML(e)
	}
	annotationFromXML["polyline"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationPolylineOfXML(e)
	}
	annotationFromXML["sound"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationSoundOfXML(e)
	}
	annotationFromXML["squiggly"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationSquigglyOfXML(e)
	}
	annotationFromXML["stamp"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationStampOfXML(e)
	}
	annotationFromXML["strikeout"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationStrikeOutOfXML(e)
	}
	annotationFromXML["underline"] = func(e *dom.Element) (FDFAnnotation, error) {
		return NewFDFAnnotationUnderlineOfXML(e)
	}

	annotationFromDictionary[SubTypeText] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationTextOf(d)
	}
	annotationFromDictionary[SubTypeCaret] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationCaretOf(d)
	}
	annotationFromDictionary[SubTypeFreeText] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationFreeTextOf(d)
	}
	annotationFromDictionary[SubTypeFileAttachment] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationFileAttachmentOf(d)
	}
	annotationFromDictionary[SubTypeHighlight] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationHighlightOf(d)
	}
	annotationFromDictionary[SubTypeInk] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationInkOf(d)
	}
	annotationFromDictionary[SubTypeLine] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationLineOf(d)
	}
	annotationFromDictionary[SubTypeLink] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationLinkOf(d)
	}
	annotationFromDictionary[SubTypeCircle] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationCircleOf(d)
	}
	annotationFromDictionary[SubTypeSquare] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationSquareOf(d)
	}
	annotationFromDictionary[SubTypePolygon] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationPolygonOf(d)
	}
	annotationFromDictionary[SubTypePolyline] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationPolylineOf(d)
	}
	annotationFromDictionary[SubTypeSound] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationSoundOf(d)
	}
	annotationFromDictionary[SubTypeSquiggly] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationSquigglyOf(d)
	}
	annotationFromDictionary[SubTypeStamp] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationStampOf(d)
	}
	annotationFromDictionary[SubTypeStrikeOut] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationStrikeOutOf(d)
	}
	annotationFromDictionary[SubTypeUnderline] = func(d *cos.Dictionary) FDFAnnotation {
		return NewFDFAnnotationUnderlineOf(d)
	}
}
