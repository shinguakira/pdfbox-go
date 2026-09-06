package fdf

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/filespecification"
	"github.com/shinguakira/pdfbox-go/go/w3c/dom"
)

// FDFDictionary is the /FDF entry of the catalogue of an FDF document: the
// fields, the annotations and the pages it carries.
//
// Port of FDFDictionary.
type FDFDictionary struct {
	fdf *cos.Dictionary
}

var _ common.COSObjectable = (*FDFDictionary)(nil)

// NewFDFDictionary returns an empty FDF dictionary.
func NewFDFDictionary() *FDFDictionary { return &FDFDictionary{fdf: cos.NewDictionary()} }

// NewFDFDictionaryOf returns the FDF dictionary the given dictionary holds.
func NewFDFDictionaryOf(fdfDictionary *cos.Dictionary) *FDFDictionary {
	return &FDFDictionary{fdf: fdfDictionary}
}

// NewFDFDictionaryOfXML returns the FDF dictionary the given XFDF element
// describes.
//
// Java declares no exception here and logs everything it could not read, so the
// port returns no error either.
func NewFDFDictionaryOfXML(fdfXML *dom.Element) *FDFDictionary {
	d := NewFDFDictionary()
	nodeList := fdfXML.ChildNodes()
	for i := 0; i < nodeList.Length(); i++ {
		node := nodeList.Item(i)
		child, isElement := node.(*dom.Element)
		if !isElement {
			continue
		}
		switch child.TagName() {
		case "f":
			fs := filespecification.NewPDSimpleFileSpecification()
			fs.SetFile(child.GetAttribute("href"))
			d.SetFile(fs)
		case "ids":
			ids := cos.NewArray()
			original := child.GetAttribute("original")
			modified := child.GetAttribute("modified")
			if parsed, err := cos.ParseHexString(original); err != nil {
				slog.Warn("fdf: error parsing ID entry for attribute 'original'. ID entry ignored",
					slog.String("original", original), slog.Any("err", err))
			} else {
				ids.Add(parsed)
			}
			if parsed, err := cos.ParseHexString(modified); err != nil {
				slog.Warn("fdf: error parsing ID entry for attribute 'modified'. ID entry ignored",
					slog.String("modified", modified), slog.Any("err", err))
			} else {
				ids.Add(parsed)
			}
			d.SetID(ids)
		case "fields":
			fields := child.ChildNodes()
			fieldList := []*FDFField{}
			for f := 0; f < fields.Length(); f++ {
				currentNode := fields.Item(f)
				element, isElement := currentNode.(*dom.Element)
				if !isElement || element.TagName() != "field" {
					continue
				}
				field, err := NewFDFFieldOfXML(element)
				if err != nil {
					slog.Warn("fdf: error parsing field entry. Field ignored",
						slog.String("entry", currentNode.NodeValue()), slog.Any("err", err))
					continue
				}
				fieldList = append(fieldList, field)
			}
			d.SetFields(fieldList)
		case "annots":
			annots := child.ChildNodes()
			annotList := []FDFAnnotation{}
			for j := 0; j < annots.Length(); j++ {
				annotNode := annots.Item(j)
				annot, isElement := annotNode.(*dom.Element)
				if !isElement {
					continue
				}
				// the node name defines the annotation type
				annotationName := annot.NodeName()
				factory := annotationFromXML[annotationName]
				if factory == nil {
					slog.Warn("fdf: unknown or unsupported annotation type",
						slog.String("type", annotationName))
					continue
				}
				parsed, err := factory(annot)
				if err != nil {
					slog.Warn("fdf: error parsing annotation information. Annotation ignored",
						slog.String("annotation", annot.NodeValue()), slog.Any("err", err))
					continue
				}
				annotList = append(annotList, parsed)
			}
			d.SetAnnotations(annotList)
		}
	}
	return d
}

// annotationFromXML builds the annotation an XFDF element names.
//
// Java writes the switch inline in the constructor above; the port lifts it to
// a table so that the annotation types can live in their own file. The entries
// are exactly the cases of that switch, and an unknown name has none, which is
// its default branch.
var annotationFromXML = map[string]func(element *dom.Element) (FDFAnnotation, error){}

// WriteXML writes this dictionary out as XFDF.
func (d *FDFDictionary) WriteXML(output io.Writer) error {
	w := &xmlWriter{out: output}
	fs, err := d.File()
	if err != nil {
		return err
	}
	if fs != nil && fs.File() != "" {
		w.write("<f href=\"" + escapeXML10(fs.File()) + "\" />\n")
	}
	ids := d.ID()
	if ids != nil {
		original, isOriginalString := ids.GetObject(0).(*cos.StringObj)
		modified, isModifiedString := ids.GetObject(1).(*cos.StringObj)
		if !isOriginalString || !isModifiedString {
			// Java casts both without a check, so a /ID entry that is not a
			// string raises ClassCastException; the port panics.
			panic(fmt.Sprintf("fdf: %T cannot be cast to COSString",
				ids.GetObject(0)))
		}
		w.write("<ids original=\"" + original.ToHexString() + "\" ")
		w.write("modified=\"" + modified.ToHexString() + "\" />\n")
	}
	if w.err != nil {
		return w.err
	}
	fields := d.Fields()
	if fields != nil && !fields.IsEmpty() {
		w.write("<fields>\n")
		if w.err != nil {
			return w.err
		}
		for _, field := range fields.ToSlice() {
			if err := field.WriteXML(output); err != nil {
				return err
			}
		}
		w.write("</fields>\n")
	}
	return w.err
}

// COSObject returns the dictionary.
func (d *FDFDictionary) COSObject() cos.Base { return d.fdf }

// Dictionary returns the dictionary, typed.
func (d *FDFDictionary) Dictionary() *cos.Dictionary { return d.fdf }

// File returns the source file this data came from, or nil where there is none.
func (d *FDFDictionary) File() (filespecification.PDFileSpecification, error) {
	return filespecification.CreateFS(d.fdf.GetDictionaryObject(cos.F))
}

// SetFile sets the source file this data came from.
func (d *FDFDictionary) SetFile(fs filespecification.PDFileSpecification) {
	d.fdf.SetItem(cos.F, common.COSObjectOrNil(fs))
}

// ID returns the /ID of the source file, or nil where there is none.
func (d *FDFDictionary) ID() *cos.Array { return d.fdf.GetCOSArray(cos.ID) }

// SetID sets the /ID of the source file.
func (d *FDFDictionary) SetID(id *cos.Array) { d.fdf.SetItem(cos.ID, id) }

// Fields returns the fields, or nil where there are none.
//
// The list is backed by the fields array, so adding to it or deleting from it
// changes the document too.
func (d *FDFDictionary) Fields() *common.COSArrayList[*FDFField] {
	var retval *common.COSArrayList[*FDFField]
	fieldArray := d.fdf.GetCOSArray(cos.Fields)
	if fieldArray != nil {
		fields := make([]*FDFField, 0, fieldArray.Size())
		for i := 0; i < fieldArray.Size(); i++ {
			fields = append(fields, NewFDFFieldOf(fieldArray.GetObject(i).(*cos.Dictionary)))
		}
		retval = common.NewCOSArrayListOf(fields, fieldArray)
	}
	return retval
}

// SetFields sets the fields.
func (d *FDFDictionary) SetFields(fields []*FDFField) {
	d.fdf.SetItem(cos.Fields, common.NewCOSArrayOfObjectables(fields))
}

// Status returns the status message the reader shows, or the empty string where
// there is none.
func (d *FDFDictionary) Status() string { return d.fdf.GetString(cos.Status, "") }

// SetStatus sets the status message the reader shows.
func (d *FDFDictionary) SetStatus(status string) { d.fdf.SetString(cos.Status, status) }

// Pages returns the pages, or nil where there are none.
//
// The list is backed by the pages array, so adding to it or deleting from it
// changes the document too.
//
// Java reads each entry with COSArray.get rather than getObject, so an entry
// written as an indirect reference raises ClassCastException; the port carries
// that, and panics. See migration/JAVA-BUGS.md.
func (d *FDFDictionary) Pages() *common.COSArrayList[*FDFPage] {
	var retval *common.COSArrayList[*FDFPage]
	pageArray := d.fdf.GetCOSArray(cos.Pages)
	if pageArray != nil {
		pages := make([]*FDFPage, 0, pageArray.Size())
		for i := 0; i < pageArray.Size(); i++ {
			dictionary, isDictionary := pageArray.Get(i).(*cos.Dictionary)
			if !isDictionary {
				panic(fmt.Sprintf("fdf: %T cannot be cast to COSDictionary", pageArray.Get(i)))
			}
			pages = append(pages, NewFDFPageOf(dictionary))
		}
		retval = common.NewCOSArrayListOf(pages, pageArray)
	}
	return retval
}

// SetPages sets the pages.
func (d *FDFDictionary) SetPages(pages []*FDFPage) {
	d.fdf.SetItem(cos.Pages, common.NewCOSArrayOfObjectables(pages))
}

// Encoding returns the encoding the strings are written in, which is
// PDFDocEncoding where the dictionary says nothing.
func (d *FDFDictionary) Encoding() string {
	encoding := d.fdf.GetNameAsString(cos.Encoding, "")
	if encoding == "" {
		encoding = "PDFDocEncoding"
	}
	return encoding
}

// SetEncoding sets the encoding the strings are written in.
func (d *FDFDictionary) SetEncoding(encoding string) { d.fdf.SetName(cos.Encoding, encoding) }

// Annotations returns the annotations, or nil where there are none.
//
// The list is backed by the annotations array, so adding to it or deleting from
// it changes the document too.
func (d *FDFDictionary) Annotations() (*common.COSArrayList[FDFAnnotation], error) {
	var retval *common.COSArrayList[FDFAnnotation]
	annotArray := d.fdf.GetCOSArray(cos.Annots)
	if annotArray != nil {
		annots := make([]FDFAnnotation, 0, annotArray.Size())
		for i := 0; i < annotArray.Size(); i++ {
			annot, err := CreateFDFAnnotation(annotArray.GetObject(i).(*cos.Dictionary))
			if err != nil {
				return nil, err
			}
			annots = append(annots, annot)
		}
		retval = common.NewCOSArrayListOf(annots, annotArray)
	}
	return retval, nil
}

// SetAnnotations sets the annotations.
func (d *FDFDictionary) SetAnnotations(annots []FDFAnnotation) {
	d.fdf.SetItem(cos.Annots, common.NewCOSArrayOfObjectables(annots))
}

// Differences returns the stream of the incremental changes, or nil where there
// is none.
func (d *FDFDictionary) Differences() *cos.Stream {
	return d.fdf.GetCOSStream(cos.Differences)
}

// SetDifferences sets the stream of the incremental changes.
func (d *FDFDictionary) SetDifferences(diff *cos.Stream) {
	d.fdf.SetItem(cos.Differences, diff)
}

// Target returns the name of the frame the data is submitted into, or the empty
// string where there is none.
func (d *FDFDictionary) Target() string { return d.fdf.GetString(cos.Target, "") }

// SetTarget sets the name of the frame the data is submitted into.
func (d *FDFDictionary) SetTarget(target string) { d.fdf.SetString(cos.Target, target) }

// EmbeddedFDFs returns the embedded FDF files, or nil where there are none.
//
// The list is backed by the array, so adding to it or deleting from it changes
// the document too.
func (d *FDFDictionary) EmbeddedFDFs() (
	*common.COSArrayList[filespecification.PDFileSpecification], error) {
	var retval *common.COSArrayList[filespecification.PDFileSpecification]
	embeddedArray := d.fdf.GetCOSArray(cos.EmbeddedFDFs)
	if embeddedArray != nil {
		embedded := make([]filespecification.PDFileSpecification, 0, embeddedArray.Size())
		for i := 0; i < embeddedArray.Size(); i++ {
			fs, err := filespecification.CreateFS(embeddedArray.Get(i))
			if err != nil {
				return nil, err
			}
			embedded = append(embedded, fs)
		}
		retval = common.NewCOSArrayListOf(embedded, embeddedArray)
	}
	return retval, nil
}

// SetEmbeddedFDFs sets the embedded FDF files.
func (d *FDFDictionary) SetEmbeddedFDFs(embedded []filespecification.PDFileSpecification) {
	d.fdf.SetItem(cos.EmbeddedFDFs, common.NewCOSArrayOfObjectables(embedded))
}

// JavaScript returns the JavaScript to run on import, or nil where there is
// none.
func (d *FDFDictionary) JavaScript() *FDFJavaScript {
	dic := d.fdf.GetCOSDictionary(cos.JavaScript)
	if dic != nil {
		return NewFDFJavaScriptOf(dic)
	}
	return nil
}

// SetJavaScript sets the JavaScript to run on import.
func (d *FDFDictionary) SetJavaScript(js *FDFJavaScript) {
	d.fdf.SetItem(cos.JavaScript, common.COSObjectOrNil(js))
}
