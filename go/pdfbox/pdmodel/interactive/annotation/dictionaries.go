package annotation

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
)

// PDAppearanceStream is the appearance of an annotation in one state.
//
// Port of PDAppearanceStream, which extends PDFormXObject and adds nothing.
type PDAppearanceStream struct {
	form.PDFormXObject
}

// NewPDAppearanceStreamOf wraps an existing stream.
func NewPDAppearanceStreamOf(stream *cos.Stream) *PDAppearanceStream {
	return &PDAppearanceStream{PDFormXObject: *form.NewPDFormXObjectOfStream(stream)}
}

// NewPDAppearanceStream creates a new empty appearance in the given document.
func NewPDAppearanceStream(document common.COSDocumentLike) *PDAppearanceStream {
	return &PDAppearanceStream{PDFormXObject: *form.NewPDFormXObject(document)}
}

// PDAppearanceEntry is one entry of an appearance dictionary: either a single
// stream, or a sub-dictionary of streams keyed by appearance state.
//
// Port of PDAppearanceEntry.
type PDAppearanceEntry struct {
	entry cos.Base
}

var _ common.COSObjectable = (*PDAppearanceEntry)(nil)

// NewPDAppearanceEntryOf creates an entry over the given object.
//
// Java's parameter is a COSDictionary, and a COSStream is one; the port takes
// the base, because isStream is the test the class is built around.
func NewPDAppearanceEntryOf(entry cos.Base) *PDAppearanceEntry {
	return &PDAppearanceEntry{entry: entry}
}

// COSObject returns the entry.
func (e *PDAppearanceEntry) COSObject() cos.Base { return e.entry }

// IsSubDictionary reports whether this entry holds a state to stream map.
func (e *PDAppearanceEntry) IsSubDictionary() bool { return !e.IsStream() }

// IsStream reports whether this entry is a single appearance stream.
func (e *PDAppearanceEntry) IsStream() bool {
	_, ok := e.entry.(*cos.Stream)
	return ok
}

// AppearanceStream returns the single appearance stream.
//
// Java throws IllegalStateException where the entry is a sub-dictionary, which
// is unchecked, so the port panics.
func (e *PDAppearanceEntry) AppearanceStream() *PDAppearanceStream {
	if !e.IsStream() {
		panic("This entry is not an appearance stream")
	}
	return NewPDAppearanceStreamOf(e.entry.(*cos.Stream))
}

// SubDictionary returns the state to stream map.
//
// Java throws IllegalStateException where the entry is a single stream.
func (e *PDAppearanceEntry) SubDictionary() *common.COSDictionaryMap[*PDAppearanceStream] {
	if !e.IsSubDictionary() {
		panic("This entry is not an appearance subdictionary")
	}
	dict := e.entry.(*cos.Dictionary)
	m := map[string]*PDAppearanceStream{}
	for _, name := range dict.KeySet() {
		stream, _ := dict.GetDictionaryObject(name).(*cos.Stream)
		// the file from PDFBOX-1599 contains /null as its entry, so we skip non-stream entries
		if stream != nil {
			m[name.Name()] = NewPDAppearanceStreamOf(stream)
		}
	}
	return common.NewCOSDictionaryMap(m, dict)
}

// PDAppearanceDictionary is the /AP entry of an annotation.
//
// Port of PDAppearanceDictionary.
type PDAppearanceDictionary struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDAppearanceDictionary)(nil)

// NewPDAppearanceDictionary creates a dictionary with the required /N entry.
func NewPDAppearanceDictionary() *PDAppearanceDictionary {
	dictionary := cos.NewDictionary()
	// the N entry is required.
	dictionary.SetItem(cos.N, cos.NewDictionary())
	return &PDAppearanceDictionary{dictionary: dictionary}
}

// NewPDAppearanceDictionaryOf creates one over the given dictionary.
func NewPDAppearanceDictionaryOf(dictionary *cos.Dictionary) *PDAppearanceDictionary {
	return &PDAppearanceDictionary{dictionary: dictionary}
}

// COSObject returns the dictionary.
func (d *PDAppearanceDictionary) COSObject() cos.Base { return d.dictionary }

// NormalAppearance returns the /N entry, or nil.
func (d *PDAppearanceDictionary) NormalAppearance() *PDAppearanceEntry {
	return d.entryOf(cos.N, nil)
}

// entryOf reads one appearance entry, falling back to the given entry where it
// is absent. Java's getRolloverAppearance and getDownAppearance fall back to
// the normal appearance; getNormalAppearance falls back to null.
func (d *PDAppearanceDictionary) entryOf(key *cos.Name, fallback *PDAppearanceEntry) *PDAppearanceEntry {
	entry := d.dictionary.GetDictionaryObject(key)
	if _, isDictionary := asDictionary(entry); isDictionary {
		return NewPDAppearanceEntryOf(entry)
	}
	return fallback
}

// SetNormalAppearance sets the /N entry.
func (d *PDAppearanceDictionary) SetNormalAppearance(entry *PDAppearanceEntry) {
	d.setEntry(cos.N, entry)
}

// SetNormalAppearanceStream sets the /N entry to a single stream.
func (d *PDAppearanceDictionary) SetNormalAppearanceStream(ap *PDAppearanceStream) {
	d.setStream(cos.N, ap)
}

// RolloverAppearance returns the /R entry, or the normal appearance.
func (d *PDAppearanceDictionary) RolloverAppearance() *PDAppearanceEntry {
	return d.entryOf(cos.R, d.NormalAppearance())
}

// SetRolloverAppearance sets the /R entry.
func (d *PDAppearanceDictionary) SetRolloverAppearance(entry *PDAppearanceEntry) {
	d.setEntry(cos.R, entry)
}

// SetRolloverAppearanceStream sets the /R entry to a single stream.
func (d *PDAppearanceDictionary) SetRolloverAppearanceStream(ap *PDAppearanceStream) {
	d.setStream(cos.R, ap)
}

// DownAppearance returns the /D entry, or the normal appearance.
func (d *PDAppearanceDictionary) DownAppearance() *PDAppearanceEntry {
	return d.entryOf(cos.D, d.NormalAppearance())
}

// SetDownAppearance sets the /D entry.
func (d *PDAppearanceDictionary) SetDownAppearance(entry *PDAppearanceEntry) {
	d.setEntry(cos.D, entry)
}

// SetDownAppearanceStream sets the /D entry to a single stream.
func (d *PDAppearanceDictionary) SetDownAppearanceStream(ap *PDAppearanceStream) {
	d.setStream(cos.D, ap)
}

func (d *PDAppearanceDictionary) setEntry(key *cos.Name, entry *PDAppearanceEntry) {
	if entry == nil {
		d.dictionary.SetItem(key, nil)
		return
	}
	d.dictionary.SetItem(key, entry.COSObject())
}

func (d *PDAppearanceDictionary) setStream(key *cos.Name, ap *PDAppearanceStream) {
	if ap == nil {
		d.dictionary.SetItem(key, nil)
		return
	}
	d.dictionary.SetItem(key, ap.COSObject())
}

// The border styles, PDF 32000-1:2008 Table 166.
const (
	// BorderStyleSolid is a solid border.
	BorderStyleSolid = "S"
	// BorderStyleDashed is a dashed border.
	BorderStyleDashed = "D"
	// BorderStyleBeveled is a bevelled border.
	BorderStyleBeveled = "B"
	// BorderStyleInset is an inset border.
	BorderStyleInset = "I"
	// BorderStyleUnderline is an underline.
	BorderStyleUnderline = "U"
)

// PDBorderStyleDictionary is the /BS entry of an annotation.
//
// Port of PDBorderStyleDictionary.
type PDBorderStyleDictionary struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDBorderStyleDictionary)(nil)

// NewPDBorderStyleDictionary creates an empty border style.
func NewPDBorderStyleDictionary() *PDBorderStyleDictionary {
	return &PDBorderStyleDictionary{dictionary: cos.NewDictionary()}
}

// NewPDBorderStyleDictionaryOf creates one over the given dictionary.
func NewPDBorderStyleDictionaryOf(dict *cos.Dictionary) *PDBorderStyleDictionary {
	return &PDBorderStyleDictionary{dictionary: dict}
}

// COSObject returns the dictionary.
func (d *PDBorderStyleDictionary) COSObject() cos.Base { return d.dictionary }

// SetWidth sets the /W border width.
func (d *PDBorderStyleDictionary) SetWidth(w float32) {
	// PDFBOX-3929 workaround
	if w == float32(int(w)) {
		d.dictionary.SetInt(cos.W, int(w))
	} else {
		d.dictionary.SetFloat(cos.W, w)
	}
}

// Width returns the /W border width, which defaults to 1.
func (d *PDBorderStyleDictionary) Width() float32 {
	if _, isName := d.dictionary.GetDictionaryObject(cos.W).(*cos.Name); isName {
		// replicate Adobe behavior although it contradicts the specification
		// https://github.com/mozilla/pdf.js/issues/10385
		return 0
	}
	return d.dictionary.GetFloat(cos.W, 1)
}

// SetStyle sets the /S border style.
func (d *PDBorderStyleDictionary) SetStyle(s string) { d.dictionary.SetName(cos.S, s) }

// Style returns the /S border style, which defaults to solid.
func (d *PDBorderStyleDictionary) Style() string {
	return d.dictionary.GetNameAsString(cos.S, BorderStyleSolid)
}

// SetDashStyle sets the /D dash array.
func (d *PDBorderStyleDictionary) SetDashStyle(dashArray *cos.Array) {
	d.dictionary.SetItem(cos.D, dashArray)
}

// DashStyle returns the /D dash pattern, writing the default [3] into the
// dictionary where it has none, which is what Java does.
func (d *PDBorderStyleDictionary) DashStyle() *graphics.PDLineDashPattern {
	dash := d.dictionary.GetCOSArray(cos.D)
	if dash == nil {
		dash = cos.NewArray()
		dash.Add(cos.GetInteger(3))
		d.dictionary.SetItem(cos.D, dash)
	}
	return graphics.NewPDLineDashPatternOf(dash, 0)
}

// The border effect styles, PDF 32000-1:2008 Table 167.
const (
	// BorderEffectStyleSolid is no effect.
	BorderEffectStyleSolid = "S"
	// BorderEffectStyleCloudy is the cloudy effect.
	BorderEffectStyleCloudy = "C"
)

// PDBorderEffectDictionary is the /BE entry of an annotation.
//
// Port of PDBorderEffectDictionary.
type PDBorderEffectDictionary struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDBorderEffectDictionary)(nil)

// NewPDBorderEffectDictionary creates an empty border effect.
func NewPDBorderEffectDictionary() *PDBorderEffectDictionary {
	return &PDBorderEffectDictionary{dictionary: cos.NewDictionary()}
}

// NewPDBorderEffectDictionaryOf creates one over the given dictionary.
func NewPDBorderEffectDictionaryOf(dict *cos.Dictionary) *PDBorderEffectDictionary {
	return &PDBorderEffectDictionary{dictionary: dict}
}

// COSObject returns the dictionary.
func (d *PDBorderEffectDictionary) COSObject() cos.Base { return d.dictionary }

// SetIntensity sets the /I intensity.
func (d *PDBorderEffectDictionary) SetIntensity(i float32) {
	d.dictionary.SetFloat(cos.I, i)
}

// Intensity returns the /I intensity, which defaults to 0.
func (d *PDBorderEffectDictionary) Intensity() float32 {
	return d.dictionary.GetFloat(cos.I, 0)
}

// SetStyle sets the /S effect style.
func (d *PDBorderEffectDictionary) SetStyle(s string) { d.dictionary.SetName(cos.S, s) }

// Style returns the /S effect style, which defaults to solid.
func (d *PDBorderEffectDictionary) Style() string {
	return d.dictionary.GetNameAsString(cos.S, BorderEffectStyleSolid)
}

// PDExternalDataDictionary is the /ExData entry of a markup annotation.
//
// Port of PDExternalDataDictionary.
type PDExternalDataDictionary struct {
	dataDictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDExternalDataDictionary)(nil)

// NewPDExternalDataDictionary creates an empty external data dictionary.
func NewPDExternalDataDictionary() *PDExternalDataDictionary {
	dataDictionary := cos.NewDictionary()
	dataDictionary.SetName(cos.Type, "ExData")
	return &PDExternalDataDictionary{dataDictionary: dataDictionary}
}

// NewPDExternalDataDictionaryOf creates one over the given dictionary.
func NewPDExternalDataDictionaryOf(dictionary *cos.Dictionary) *PDExternalDataDictionary {
	return &PDExternalDataDictionary{dataDictionary: dictionary}
}

// COSObject returns the dictionary.
func (d *PDExternalDataDictionary) COSObject() cos.Base { return d.dataDictionary }

// Type returns the /Type, which defaults to "ExData".
func (d *PDExternalDataDictionary) Type() string {
	return d.dataDictionary.GetNameAsString(cos.Type, "ExData")
}

// Subtype returns the /Subtype.
func (d *PDExternalDataDictionary) Subtype() string {
	return d.dataDictionary.GetNameAsString(cos.Subtype, "")
}

// SetSubtype sets the /Subtype.
func (d *PDExternalDataDictionary) SetSubtype(subtype string) {
	d.dataDictionary.SetName(cos.Subtype, subtype)
}
