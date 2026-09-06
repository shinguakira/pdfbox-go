// Package fdf models the Forms Data Format: the import and export half of
// AcroForms.
//
// Port of org.apache.pdfbox.pdmodel.fdf.
package fdf

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/filespecification"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/action"
)

// escapeXML10 returns the string with everything XML gives meaning to escaped,
// everything above ASCII written as a numeric reference, and everything XML 1.0
// forbids replaced with U+FFFD.
//
// Port of the package-private FDFUtils.escapeXML10. Java walks code points; Go
// ranges over a string by rune, which is the same walk.
func escapeXML10(input string) string {
	escapedXML := &strings.Builder{}
	invalidCount := 0
	for _, cp := range input {
		if !isValidXML10Char(cp) {
			invalidCount++
			escapedXML.WriteRune('�')
			continue
		}
		switch cp {
		case '<':
			escapedXML.WriteString("&lt;")
		case '>':
			escapedXML.WriteString("&gt;")
		case '"':
			escapedXML.WriteString("&quot;")
		case '&':
			escapedXML.WriteString("&amp;")
		case '\'':
			escapedXML.WriteString("&apos;")
		default:
			if cp > 0x7e {
				escapedXML.WriteString("&#")
				escapedXML.WriteString(strconv.Itoa(int(cp)))
				escapedXML.WriteByte(';')
			} else {
				escapedXML.WriteRune(cp)
			}
		}
	}
	if invalidCount > 0 {
		slog.Info("fdf: replaced character(s) invalid in XML 1.0 with U+FFFD",
			slog.Int("count", invalidCount))
	}
	return escapedXML.String()
}

// isValidXML10Char reports whether XML 1.0 allows the given code point. Java
// declares it private static.
func isValidXML10Char(cp rune) bool {
	return cp == 0x9 || cp == 0xA || cp == 0xD ||
		(cp >= 0x20 && cp <= 0xD7FF) ||
		(cp >= 0xE000 && cp <= 0xFFFD) ||
		(cp >= 0x10000 && cp <= 0x10FFFF)
}

// FDFPageInfo is the page information of an FDF page.
//
// Port of FDFPageInfo, which has no accessor of its own.
type FDFPageInfo struct {
	pageInfo *cos.Dictionary
}

var _ common.COSObjectable = (*FDFPageInfo)(nil)

// NewFDFPageInfo returns an empty page information dictionary.
func NewFDFPageInfo() *FDFPageInfo {
	return &FDFPageInfo{pageInfo: cos.NewDictionary()}
}

// NewFDFPageInfoOf returns the page information the given dictionary holds.
func NewFDFPageInfoOf(p *cos.Dictionary) *FDFPageInfo { return &FDFPageInfo{pageInfo: p} }

// COSObject returns the dictionary.
func (i *FDFPageInfo) COSObject() cos.Base { return i.pageInfo }

// Dictionary returns the dictionary, typed.
func (i *FDFPageInfo) Dictionary() *cos.Dictionary { return i.pageInfo }

// FDFNamedPageReference names a page of another document.
//
// Port of FDFNamedPageReference.
type FDFNamedPageReference struct {
	ref *cos.Dictionary
}

var _ common.COSObjectable = (*FDFNamedPageReference)(nil)

// NewFDFNamedPageReference returns an empty page reference.
func NewFDFNamedPageReference() *FDFNamedPageReference {
	return &FDFNamedPageReference{ref: cos.NewDictionary()}
}

// NewFDFNamedPageReferenceOf returns the page reference the given dictionary
// holds.
func NewFDFNamedPageReferenceOf(r *cos.Dictionary) *FDFNamedPageReference {
	return &FDFNamedPageReference{ref: r}
}

// COSObject returns the dictionary.
func (r *FDFNamedPageReference) COSObject() cos.Base { return r.ref }

// Dictionary returns the dictionary, typed.
func (r *FDFNamedPageReference) Dictionary() *cos.Dictionary { return r.ref }

// Name returns the name of the referenced page, or the empty string where there
// is none.
func (r *FDFNamedPageReference) Name() string { return r.ref.GetString(cos.NameKey, "") }

// SetName sets the name of the referenced page.
func (r *FDFNamedPageReference) SetName(name string) { r.ref.SetString(cos.NameKey, name) }

// FileSpecification returns the file the referenced page is in, or nil where
// there is none.
func (r *FDFNamedPageReference) FileSpecification() (filespecification.PDFileSpecification, error) {
	return filespecification.CreateFS(r.ref.GetDictionaryObject(cos.F))
}

// SetFileSpecification sets the file the referenced page is in.
func (r *FDFNamedPageReference) SetFileSpecification(fs filespecification.PDFileSpecification) {
	r.ref.SetItem(cos.F, common.COSObjectOrNil(fs))
}

// FDFOptionElement is one entry of the /Opt array of a choice field: the value
// and the default appearance string that goes with it.
//
// Port of FDFOptionElement.
type FDFOptionElement struct {
	option *cos.Array
}

var _ common.COSObjectable = (*FDFOptionElement)(nil)

// NewFDFOptionElement returns an option with two empty strings in it.
func NewFDFOptionElement() *FDFOptionElement {
	option := cos.NewArray()
	option.Add(cos.NewStringObj(""))
	option.Add(cos.NewStringObj(""))
	return &FDFOptionElement{option: option}
}

// NewFDFOptionElementOf returns the option the given array holds.
func NewFDFOptionElementOf(o *cos.Array) *FDFOptionElement { return &FDFOptionElement{option: o} }

// COSObject returns the array.
func (e *FDFOptionElement) COSObject() cos.Base { return e.option }

// COSArray returns the array, typed.
func (e *FDFOptionElement) COSArray() *cos.Array { return e.option }

// Option returns the value of the option, or the empty string where there is
// none.
func (e *FDFOptionElement) Option() string {
	if e.option != nil && !e.option.IsEmpty() {
		if base, isString := e.option.GetObject(0).(*cos.StringObj); isString {
			return base.Value()
		}
	}
	return ""
}

// SetOption sets the value of the option.
func (e *FDFOptionElement) SetOption(opt string) {
	e.option.GrowToSize(1)
	e.option.Set(0, cos.NewStringObj(opt))
}

// DefaultAppearanceString returns the /DA of the option, or the empty string
// where there is none.
func (e *FDFOptionElement) DefaultAppearanceString() string {
	if e.option != nil && e.option.Size() > 1 {
		if base, isString := e.option.GetObject(1).(*cos.StringObj); isString {
			return base.Value()
		}
	}
	return ""
}

// SetDefaultAppearanceString sets the /DA of the option.
func (e *FDFOptionElement) SetDefaultAppearanceString(da string) {
	e.option.GrowToSize(2)
	e.option.Set(1, cos.NewStringObj(da))
}

// FDFTemplate is a template page an FDF page is built from.
//
// Port of FDFTemplate.
type FDFTemplate struct {
	template *cos.Dictionary
}

var _ common.COSObjectable = (*FDFTemplate)(nil)

// NewFDFTemplate returns an empty template.
func NewFDFTemplate() *FDFTemplate { return &FDFTemplate{template: cos.NewDictionary()} }

// NewFDFTemplateOf returns the template the given dictionary holds.
func NewFDFTemplateOf(t *cos.Dictionary) *FDFTemplate { return &FDFTemplate{template: t} }

// COSObject returns the dictionary.
func (t *FDFTemplate) COSObject() cos.Base { return t.template }

// Dictionary returns the dictionary, typed.
func (t *FDFTemplate) Dictionary() *cos.Dictionary { return t.template }

// TemplateReference returns the page the template names, or nil where it names
// none.
func (t *FDFTemplate) TemplateReference() *FDFNamedPageReference {
	dict := t.template.GetCOSDictionary(cos.TRef)
	if dict != nil {
		return NewFDFNamedPageReferenceOf(dict)
	}
	return nil
}

// SetTemplateReference sets the page the template names.
func (t *FDFTemplate) SetTemplateReference(tRef *FDFNamedPageReference) {
	t.template.SetItem(cos.TRef, common.COSObjectOrNil(tRef))
}

// Fields returns the fields of the template, or nil where it has none.
//
// The list is backed by the fields array, so adding to it or deleting from it
// changes the document too.
func (t *FDFTemplate) Fields() *common.COSArrayList[*FDFField] {
	array := t.template.GetCOSArray(cos.Fields)
	if array != nil {
		fields := make([]*FDFField, 0, array.Size())
		for i := 0; i < array.Size(); i++ {
			fields = append(fields, NewFDFFieldOf(array.GetObject(i).(*cos.Dictionary)))
		}
		return common.NewCOSArrayListOf(fields, array)
	}
	return nil
}

// SetFields sets the fields of the template.
func (t *FDFTemplate) SetFields(fields []*FDFField) {
	t.template.SetItem(cos.Fields, common.NewCOSArrayOfObjectables(fields))
}

// ShouldRename reports whether the fields are renamed when they clash.
func (t *FDFTemplate) ShouldRename() bool { return t.template.GetBoolean(cos.Rename, false) }

// SetRename sets whether the fields are renamed when they clash.
func (t *FDFTemplate) SetRename(value bool) { t.template.SetBoolean(cos.Rename, value) }

// FDFPage is one page of an FDF document.
//
// Port of FDFPage.
type FDFPage struct {
	page *cos.Dictionary
}

var _ common.COSObjectable = (*FDFPage)(nil)

// NewFDFPage returns an empty page.
func NewFDFPage() *FDFPage { return &FDFPage{page: cos.NewDictionary()} }

// NewFDFPageOf returns the page the given dictionary holds.
func NewFDFPageOf(p *cos.Dictionary) *FDFPage { return &FDFPage{page: p} }

// COSObject returns the dictionary.
func (p *FDFPage) COSObject() cos.Base { return p.page }

// Dictionary returns the dictionary, typed.
func (p *FDFPage) Dictionary() *cos.Dictionary { return p.page }

// Templates returns the templates of the page, or nil where it has none.
//
// The list is backed by the templates array, so adding to it or deleting from
// it changes the document too.
func (p *FDFPage) Templates() *common.COSArrayList[*FDFTemplate] {
	array := p.page.GetCOSArray(cos.Templates)
	if array != nil {
		objects := make([]*FDFTemplate, 0, array.Size())
		for i := 0; i < array.Size(); i++ {
			objects = append(objects, NewFDFTemplateOf(array.GetObject(i).(*cos.Dictionary)))
		}
		return common.NewCOSArrayListOf(objects, array)
	}
	return nil
}

// SetTemplates sets the templates of the page.
func (p *FDFPage) SetTemplates(templates []*FDFTemplate) {
	p.page.SetItem(cos.Templates, common.NewCOSArrayOfObjectables(templates))
}

// PageInfo returns the page information, or nil where there is none.
func (p *FDFPage) PageInfo() *FDFPageInfo {
	var retval *FDFPageInfo
	dict := p.page.GetCOSDictionary(cos.Info)
	if dict != nil {
		retval = NewFDFPageInfoOf(dict)
	}
	return retval
}

// SetPageInfo sets the page information.
func (p *FDFPage) SetPageInfo(info *FDFPageInfo) {
	p.page.SetItem(cos.Info, common.COSObjectOrNil(info))
}

// FDFJavaScript is the JavaScript an FDF document runs on import.
//
// Port of FDFJavaScript.
type FDFJavaScript struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*FDFJavaScript)(nil)

// NewFDFJavaScript returns an empty JavaScript dictionary.
func NewFDFJavaScript() *FDFJavaScript { return &FDFJavaScript{dictionary: cos.NewDictionary()} }

// NewFDFJavaScriptOf returns the JavaScript the given dictionary holds.
func NewFDFJavaScriptOf(javaScript *cos.Dictionary) *FDFJavaScript {
	return &FDFJavaScript{dictionary: javaScript}
}

// COSObject returns the dictionary.
func (j *FDFJavaScript) COSObject() cos.Base { return j.dictionary }

// Dictionary returns the dictionary, typed.
func (j *FDFJavaScript) Dictionary() *cos.Dictionary { return j.dictionary }

// Before returns the script to run before the fields are imported, or the empty
// string where there is none, which is the null Java answers.
func (j *FDFJavaScript) Before() string { return scriptOf(j.dictionary, cos.Before) }

// SetBefore sets the script to run before the fields are imported.
func (j *FDFJavaScript) SetBefore(before string) {
	j.dictionary.SetItem(cos.Before, cos.NewStringObj(before))
}

// After returns the script to run after the fields are imported, or the empty
// string where there is none.
func (j *FDFJavaScript) After() string { return scriptOf(j.dictionary, cos.After) }

// SetAfter sets the script to run after the fields are imported.
func (j *FDFJavaScript) SetAfter(after string) {
	j.dictionary.SetItem(cos.After, cos.NewStringObj(after))
}

// scriptOf reads a script that may be written as a string or as a stream, which
// is the body getBefore and getAfter share.
func scriptOf(dictionary *cos.Dictionary, key *cos.Name) string {
	switch base := dictionary.GetDictionaryObject(key).(type) {
	case *cos.StringObj:
		return base.Value()
	case *cos.Stream:
		return base.ToTextString()
	}
	return ""
}

// Doc returns the document level scripts, in the order the array holds them, or
// nil where there are none.
//
// Java answers a LinkedHashMap, which keeps that order; a Go map does not, so
// the port answers the names and the actions as two slices that stay in step.
type FDFJavaScriptDoc struct {
	// Names are the names of the scripts, in the order the array holds them.
	Names []string

	// Actions are the scripts, one for each name.
	Actions []*action.PDActionJavaScript
}

// Doc returns the document level scripts, or nil where there are none.
func (j *FDFJavaScript) Doc() *FDFJavaScriptDoc {
	array := j.dictionary.GetCOSArray(cos.Doc)
	if array == nil {
		return nil
	}
	doc := &FDFJavaScriptDoc{}
	for i := 0; i+1 < array.Size(); i += 2 {
		name := array.GetName(i, "")
		if name != "" {
			if base, isDictionary := array.GetObject(i + 1).(*cos.Dictionary); isDictionary {
				act := action.CreateAction(base)
				if javaScript, isJavaScript := act.(*action.PDActionJavaScript); isJavaScript {
					doc.Names = append(doc.Names, name)
					doc.Actions = append(doc.Actions, javaScript)
				}
			}
		}
	}
	return doc
}

// SetDoc sets the document level scripts.
func (j *FDFJavaScript) SetDoc(doc *FDFJavaScriptDoc) {
	array := cos.NewArray()
	for i, key := range doc.Names {
		array.Add(cos.NewStringObj(key))
		array.Add(doc.Actions[i].COSObject())
	}
	j.dictionary.SetItem(cos.Doc, array)
}

// The scale options of an icon fit.
//
// Port of the SCALE_OPTION_ constants of FDFIconFit.
const (
	ScaleOptionAlways                = "A"
	ScaleOptionOnlyWhenIconIsBigger  = "B"
	ScaleOptionOnlyWhenIconIsSmaller = "S"
	ScaleOptionNever                 = "N"
)

// The scale types of an icon fit.
//
// Port of the SCALE_TYPE_ constants of FDFIconFit.
const (
	ScaleTypeAnamorphic   = "A"
	ScaleTypeProportional = "P"
)

// FDFIconFit says how the icon of a push button is fitted into its annotation.
//
// Port of FDFIconFit.
type FDFIconFit struct {
	fit *cos.Dictionary
}

var _ common.COSObjectable = (*FDFIconFit)(nil)

// NewFDFIconFit returns an empty icon fit.
func NewFDFIconFit() *FDFIconFit { return &FDFIconFit{fit: cos.NewDictionary()} }

// NewFDFIconFitOf returns the icon fit the given dictionary holds.
func NewFDFIconFitOf(f *cos.Dictionary) *FDFIconFit { return &FDFIconFit{fit: f} }

// COSObject returns the dictionary.
func (f *FDFIconFit) COSObject() cos.Base { return f.fit }

// Dictionary returns the dictionary, typed.
func (f *FDFIconFit) Dictionary() *cos.Dictionary { return f.fit }

// ScaleOption returns when the icon is scaled, which is ScaleOptionAlways where
// the dictionary says nothing.
func (f *FDFIconFit) ScaleOption() string {
	retval := f.fit.GetNameAsString(cos.SW, "")
	if retval == "" {
		retval = ScaleOptionAlways
	}
	return retval
}

// SetScaleOption sets when the icon is scaled.
func (f *FDFIconFit) SetScaleOption(option string) { f.fit.SetName(cos.SW, option) }

// ScaleType returns how the icon is scaled, which is ScaleTypeProportional
// where the dictionary says nothing.
func (f *FDFIconFit) ScaleType() string {
	retval := f.fit.GetNameAsString(cos.S, "")
	if retval == "" {
		retval = ScaleTypeProportional
	}
	return retval
}

// SetScaleType sets how the icon is scaled.
func (f *FDFIconFit) SetScaleType(scale string) { f.fit.SetName(cos.S, scale) }

// FractionalSpaceToAllocate returns how the leftover space is divided, and
// writes the default of half and half into the dictionary where it says
// nothing.
func (f *FDFIconFit) FractionalSpaceToAllocate() *common.PDRange {
	var retval *common.PDRange
	array := f.fit.GetCOSArray(cos.A)
	if array == nil {
		retval = common.NewPDRange()
		retval.SetMin(.5)
		retval.SetMax(.5)
		f.SetFractionalSpaceToAllocate(retval)
	} else {
		retval = common.NewPDRangeOf(array)
	}
	return retval
}

// SetFractionalSpaceToAllocate sets how the leftover space is divided.
func (f *FDFIconFit) SetFractionalSpaceToAllocate(space *common.PDRange) {
	f.fit.SetItem(cos.A, common.COSObjectOrNil(space))
}

// ShouldScaleToFitAnnotation reports whether the icon is scaled to fill the
// annotation rather than to fit inside it.
func (f *FDFIconFit) ShouldScaleToFitAnnotation() bool {
	return f.fit.GetBoolean(cos.FB, false)
}

// SetScaleToFitAnnotation sets whether the icon is scaled to fill the
// annotation.
func (f *FDFIconFit) SetScaleToFitAnnotation(value bool) { f.fit.SetBoolean(cos.FB, value) }
