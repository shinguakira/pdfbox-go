// Package measurement holds the measure and viewport dictionaries, which say
// how the coordinates of a page map to real-world units.
//
// Port of org.apache.pdfbox.pdmodel.interactive.measurement.
package measurement

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// TypeMeasure is the /Type of a measure dictionary.
//
// Port of PDMeasureDictionary.TYPE.
const TypeMeasure = "Measure"

// SubTypeRectlinear is the /Subtype of a rectilinear measure dictionary, and
// the default.
//
// Port of PDRectlinearMeasureDictionary.SUBTYPE.
const SubTypeRectlinear = "RL"

// PDMeasureDictionary is the measure dictionary of a viewport.
//
// Port of PDMeasureDictionary.
type PDMeasureDictionary struct {
	measureDictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDMeasureDictionary)(nil)

// newPDMeasureDictionary is the protected PDMeasureDictionary() constructor.
func newPDMeasureDictionary() *PDMeasureDictionary {
	d := &PDMeasureDictionary{measureDictionary: cos.NewDictionary()}
	d.measureDictionary.SetName(cos.Type, TypeMeasure)
	return d
}

// NewPDMeasureDictionaryOf creates one over the given dictionary.
func NewPDMeasureDictionaryOf(dictionary *cos.Dictionary) *PDMeasureDictionary {
	return &PDMeasureDictionary{measureDictionary: dictionary}
}

// COSObject returns the dictionary.
func (d *PDMeasureDictionary) COSObject() cos.Base { return d.measureDictionary }

// Dictionary returns the dictionary, typed.
func (d *PDMeasureDictionary) Dictionary() *cos.Dictionary { return d.measureDictionary }

// Type returns the /Type, which is always "Measure".
func (d *PDMeasureDictionary) Type() string { return TypeMeasure }

// Subtype returns the /Subtype, which defaults to rectilinear.
func (d *PDMeasureDictionary) Subtype() string {
	return d.measureDictionary.GetNameAsString(cos.Subtype, SubTypeRectlinear)
}

// SetSubtype sets the /Subtype. Java declares it protected.
func (d *PDMeasureDictionary) SetSubtype(subtype string) {
	d.measureDictionary.SetName(cos.Subtype, subtype)
}

// The /O label positions of a number format dictionary.
const (
	// LabelSuffixToValue is LABEL_SUFFIX_TO_VALUE, and the default.
	LabelSuffixToValue = "S"
	// LabelPrefixToValue is LABEL_PREFIX_TO_VALUE.
	LabelPrefixToValue = "P"
)

// The /F fractional displays of a number format dictionary.
const (
	// FractionalDisplayDecimal is FRACTIONAL_DISPLAY_DECIMAL, and the default.
	FractionalDisplayDecimal = "D"
	// FractionalDisplayFraction is FRACTIONAL_DISPLAY_FRACTION.
	FractionalDisplayFraction = "F"
	// FractionalDisplayRound is FRACTIONAL_DISPLAY_ROUND.
	FractionalDisplayRound = "R"
	// FractionalDisplayTruncate is FRACTIONAL_DISPLAY_TRUNCATE.
	FractionalDisplayTruncate = "T"
)

// TypeNumberFormat is the /Type of a number format dictionary.
const TypeNumberFormat = "NumberFormat"

// The keys of a number format dictionary. Java writes them as bare strings,
// which COSDictionary interns through COSName.getPDFName, so these are the same
// names it ends up with.
var (
	keyU  = cos.GetPDFName("U")
	keyC  = cos.GetPDFName("C")
	keyF  = cos.GetPDFName("F")
	keyD  = cos.GetPDFName("D")
	keyFD = cos.GetPDFName("FD")
	keyRT = cos.GetPDFName("RT")
	keyRD = cos.GetPDFName("RD")
	keyPS = cos.GetPDFName("PS")
	keySS = cos.GetPDFName("SS")
	keyO  = cos.GetPDFName("O")
)

// PDNumberFormatDictionary says how one measurement is written out.
//
// Port of PDNumberFormatDictionary.
type PDNumberFormatDictionary struct {
	numberFormatDictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDNumberFormatDictionary)(nil)

// NewPDNumberFormatDictionary creates a new number format dictionary.
func NewPDNumberFormatDictionary() *PDNumberFormatDictionary {
	d := &PDNumberFormatDictionary{numberFormatDictionary: cos.NewDictionary()}
	d.numberFormatDictionary.SetName(cos.Type, TypeNumberFormat)
	return d
}

// NewPDNumberFormatDictionaryOf creates one over the given dictionary.
func NewPDNumberFormatDictionaryOf(dictionary *cos.Dictionary) *PDNumberFormatDictionary {
	return &PDNumberFormatDictionary{numberFormatDictionary: dictionary}
}

// COSObject returns the dictionary.
func (d *PDNumberFormatDictionary) COSObject() cos.Base { return d.numberFormatDictionary }

// Dictionary returns the dictionary, typed.
func (d *PDNumberFormatDictionary) Dictionary() *cos.Dictionary { return d.numberFormatDictionary }

// Type returns the /Type, which is always "NumberFormat".
func (d *PDNumberFormatDictionary) Type() string { return TypeNumberFormat }

// Units returns the /U units.
func (d *PDNumberFormatDictionary) Units() string {
	return d.numberFormatDictionary.GetString(keyU, "")
}

// SetUnits sets the /U units.
func (d *PDNumberFormatDictionary) SetUnits(units string) {
	d.numberFormatDictionary.SetString(keyU, units)
}

// ConversionFactor returns the /C conversion factor.
func (d *PDNumberFormatDictionary) ConversionFactor() float32 {
	return d.numberFormatDictionary.GetFloat(keyC, -1)
}

// SetConversionFactor sets the /C conversion factor.
func (d *PDNumberFormatDictionary) SetConversionFactor(conversionFactor float32) {
	d.numberFormatDictionary.SetFloat(keyC, conversionFactor)
}

// FractionalDisplay returns the /F fractional display, which defaults to
// decimal.
func (d *PDNumberFormatDictionary) FractionalDisplay() string {
	return d.numberFormatDictionary.GetString(keyF, FractionalDisplayDecimal)
}

// SetFractionalDisplay sets the /F fractional display, which must be D, F, R or
// T.
//
// Java throws IllegalArgumentException otherwise, which is unchecked, so the
// port panics. The empty string is Java's null, which it accepts.
func (d *PDNumberFormatDictionary) SetFractionalDisplay(fractionalDisplay string) {
	switch fractionalDisplay {
	case "", FractionalDisplayDecimal, FractionalDisplayFraction,
		FractionalDisplayRound, FractionalDisplayTruncate:
		d.numberFormatDictionary.SetString(keyF, fractionalDisplay)
	default:
		panic("Value must be \"D\", \"F\", \"R\", or \"T\", (or null).")
	}
}

// Denominator returns the /D denominator.
func (d *PDNumberFormatDictionary) Denominator() int {
	return d.numberFormatDictionary.GetInt(keyD)
}

// SetDenominator sets the /D denominator.
func (d *PDNumberFormatDictionary) SetDenominator(denominator int) {
	d.numberFormatDictionary.SetInt(keyD, denominator)
}

// IsFD reports the /FD flag.
func (d *PDNumberFormatDictionary) IsFD() bool {
	return d.numberFormatDictionary.GetBoolean(keyFD, false)
}

// SetFD sets the /FD flag.
func (d *PDNumberFormatDictionary) SetFD(fd bool) {
	d.numberFormatDictionary.SetBoolean(keyFD, fd)
}

// ThousandsSeparator returns the /RT separator, which defaults to a comma.
func (d *PDNumberFormatDictionary) ThousandsSeparator() string {
	return d.numberFormatDictionary.GetString(keyRT, ",")
}

// SetThousandsSeparator sets the /RT separator.
func (d *PDNumberFormatDictionary) SetThousandsSeparator(thousandsSeparator string) {
	d.numberFormatDictionary.SetString(keyRT, thousandsSeparator)
}

// DecimalSeparator returns the /RD separator, which defaults to a full stop.
func (d *PDNumberFormatDictionary) DecimalSeparator() string {
	return d.numberFormatDictionary.GetString(keyRD, ".")
}

// SetDecimalSeparator sets the /RD separator.
func (d *PDNumberFormatDictionary) SetDecimalSeparator(decimalSeparator string) {
	d.numberFormatDictionary.SetString(keyRD, decimalSeparator)
}

// LabelPrefixString returns the /PS prefix, which defaults to a space.
func (d *PDNumberFormatDictionary) LabelPrefixString() string {
	return d.numberFormatDictionary.GetString(keyPS, " ")
}

// SetLabelPrefixString sets the /PS prefix.
func (d *PDNumberFormatDictionary) SetLabelPrefixString(labelPrefixString string) {
	d.numberFormatDictionary.SetString(keyPS, labelPrefixString)
}

// LabelSuffixString returns the /SS suffix, which defaults to a space.
func (d *PDNumberFormatDictionary) LabelSuffixString() string {
	return d.numberFormatDictionary.GetString(keySS, " ")
}

// SetLabelSuffixString sets the /SS suffix.
func (d *PDNumberFormatDictionary) SetLabelSuffixString(labelSuffixString string) {
	d.numberFormatDictionary.SetString(keySS, labelSuffixString)
}

// LabelPositionToValue returns the /O label position, which defaults to suffix.
func (d *PDNumberFormatDictionary) LabelPositionToValue() string {
	return d.numberFormatDictionary.GetString(keyO, LabelSuffixToValue)
}

// SetLabelPositionToValue sets the /O label position, which must be S or P.
//
// Java throws IllegalArgumentException otherwise; the empty string is its null,
// which it accepts.
func (d *PDNumberFormatDictionary) SetLabelPositionToValue(labelPositionToValue string) {
	switch labelPositionToValue {
	case "", LabelPrefixToValue, LabelSuffixToValue:
		d.numberFormatDictionary.SetString(keyO, labelPositionToValue)
	default:
		panic("Value must be \"S\", or \"P\" (or null).")
	}
}

// PDRectlinearMeasureDictionary measures a page whose coordinates are a linear
// scaling of the real world.
//
// Port of PDRectlinearMeasureDictionary.
type PDRectlinearMeasureDictionary struct {
	PDMeasureDictionary
}

var _ common.COSObjectable = (*PDRectlinearMeasureDictionary)(nil)

// NewPDRectlinearMeasureDictionary creates a new rectilinear measure.
func NewPDRectlinearMeasureDictionary() *PDRectlinearMeasureDictionary {
	d := &PDRectlinearMeasureDictionary{PDMeasureDictionary: *newPDMeasureDictionary()}
	d.SetSubtype(SubTypeRectlinear)
	return d
}

// NewPDRectlinearMeasureDictionaryOf creates one over the given dictionary.
func NewPDRectlinearMeasureDictionaryOf(dictionary *cos.Dictionary) *PDRectlinearMeasureDictionary {
	return &PDRectlinearMeasureDictionary{
		PDMeasureDictionary: *NewPDMeasureDictionaryOf(dictionary),
	}
}

// ScaleRatio returns the /R scale ratio.
func (d *PDRectlinearMeasureDictionary) ScaleRatio() string {
	return d.Dictionary().GetString(cos.R, "")
}

// SetScaleRatio sets the /R scale ratio.
func (d *PDRectlinearMeasureDictionary) SetScaleRatio(scaleRatio string) {
	d.Dictionary().SetString(cos.R, scaleRatio)
}

// numberFormatsOf reads an array of number format dictionaries, or nil.
//
// Java casts each element to COSDictionary without a check and does not
// dereference an indirect one, using get rather than getObject; the port does
// the same, so a file that stores them indirectly makes it throw where Java
// throws.
func numberFormatsOf(dict *cos.Dictionary, key *cos.Name) []*PDNumberFormatDictionary {
	array := dict.GetCOSArray(key)
	if array == nil {
		return nil
	}
	retval := make([]*PDNumberFormatDictionary, array.Size())
	for i := 0; i < array.Size(); i++ {
		retval[i] = NewPDNumberFormatDictionaryOf(array.Get(i).(*cos.Dictionary))
	}
	return retval
}

// setNumberFormats writes an array of number format dictionaries.
func setNumberFormats(dict *cos.Dictionary, key *cos.Name, formats []*PDNumberFormatDictionary) {
	array := cos.NewArray()
	for _, format := range formats {
		array.Add(format.COSObject())
	}
	dict.SetItem(key, array)
}

// ChangeXs returns the /X formats, which measure a change in x.
func (d *PDRectlinearMeasureDictionary) ChangeXs() []*PDNumberFormatDictionary {
	return numberFormatsOf(d.Dictionary(), cos.X)
}

// SetChangeXs sets the /X formats.
func (d *PDRectlinearMeasureDictionary) SetChangeXs(changeXs []*PDNumberFormatDictionary) {
	setNumberFormats(d.Dictionary(), cos.X, changeXs)
}

// ChangeYs returns the /Y formats, which measure a change in y.
func (d *PDRectlinearMeasureDictionary) ChangeYs() []*PDNumberFormatDictionary {
	return numberFormatsOf(d.Dictionary(), cos.Y)
}

// SetChangeYs sets the /Y formats.
func (d *PDRectlinearMeasureDictionary) SetChangeYs(changeYs []*PDNumberFormatDictionary) {
	setNumberFormats(d.Dictionary(), cos.Y, changeYs)
}

// Distances returns the /D formats, which measure a distance.
func (d *PDRectlinearMeasureDictionary) Distances() []*PDNumberFormatDictionary {
	return numberFormatsOf(d.Dictionary(), cos.D)
}

// SetDistances sets the /D formats.
func (d *PDRectlinearMeasureDictionary) SetDistances(distances []*PDNumberFormatDictionary) {
	setNumberFormats(d.Dictionary(), cos.D, distances)
}

// Areas returns the /A formats, which measure an area.
func (d *PDRectlinearMeasureDictionary) Areas() []*PDNumberFormatDictionary {
	return numberFormatsOf(d.Dictionary(), cos.A)
}

// SetAreas sets the /A formats.
func (d *PDRectlinearMeasureDictionary) SetAreas(areas []*PDNumberFormatDictionary) {
	setNumberFormats(d.Dictionary(), cos.A, areas)
}

// Angles returns the /T formats, which measure an angle.
func (d *PDRectlinearMeasureDictionary) Angles() []*PDNumberFormatDictionary {
	return numberFormatsOf(d.Dictionary(), cos.T)
}

// SetAngles sets the /T formats.
func (d *PDRectlinearMeasureDictionary) SetAngles(angles []*PDNumberFormatDictionary) {
	setNumberFormats(d.Dictionary(), cos.T, angles)
}

// LineSloaps returns the /S formats, which measure the slope of a line.
//
// The Java method is spelled getLineSloaps.
func (d *PDRectlinearMeasureDictionary) LineSloaps() []*PDNumberFormatDictionary {
	return numberFormatsOf(d.Dictionary(), cos.S)
}

// SetLineSloaps sets the /S formats.
func (d *PDRectlinearMeasureDictionary) SetLineSloaps(lineSloaps []*PDNumberFormatDictionary) {
	setNumberFormats(d.Dictionary(), cos.S, lineSloaps)
}

// CoordSystemOrigin returns the /O origin, or nil.
func (d *PDRectlinearMeasureDictionary) CoordSystemOrigin() []float32 {
	if o := d.Dictionary().GetCOSArray(cos.O); o != nil {
		return o.ToFloatArray()
	}
	return nil
}

// SetCoordSystemOrigin sets the /O origin.
func (d *PDRectlinearMeasureDictionary) SetCoordSystemOrigin(coordSystemOrigin []float32) {
	d.Dictionary().SetItem(cos.O, cos.ArrayOfFloats(coordSystemOrigin))
}

// CYX returns the /CYX factor, the ratio of y to x units.
func (d *PDRectlinearMeasureDictionary) CYX() float32 {
	return d.Dictionary().GetFloat(cos.CYX, -1)
}

// SetCYX sets the /CYX factor.
func (d *PDRectlinearMeasureDictionary) SetCYX(cyx float32) {
	d.Dictionary().SetFloat(cos.CYX, cyx)
}

// TypeViewport is the /Type of a viewport dictionary.
const TypeViewport = "Viewport"

// PDViewportDictionary is a region of a page with its own measure.
//
// Port of PDViewportDictionary.
type PDViewportDictionary struct {
	viewportDictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDViewportDictionary)(nil)

// NewPDViewportDictionary creates an empty viewport.
//
// Java's no-argument constructor does not write the /Type, unlike the measure
// and number format ones.
func NewPDViewportDictionary() *PDViewportDictionary {
	return &PDViewportDictionary{viewportDictionary: cos.NewDictionary()}
}

// NewPDViewportDictionaryOf creates one over the given dictionary.
func NewPDViewportDictionaryOf(dictionary *cos.Dictionary) *PDViewportDictionary {
	return &PDViewportDictionary{viewportDictionary: dictionary}
}

// COSObject returns the dictionary.
func (d *PDViewportDictionary) COSObject() cos.Base { return d.viewportDictionary }

// Dictionary returns the dictionary, typed.
func (d *PDViewportDictionary) Dictionary() *cos.Dictionary { return d.viewportDictionary }

// Type returns the /Type, which is always "Viewport".
func (d *PDViewportDictionary) Type() string { return TypeViewport }

// BBox returns the /BBox of the viewport, or nil.
func (d *PDViewportDictionary) BBox() *common.PDRectangle {
	if bbox := d.viewportDictionary.GetCOSArray(cos.BBox); bbox != nil {
		return common.NewPDRectangleOfCOSArray(bbox)
	}
	return nil
}

// SetBBox sets the /BBox of the viewport.
func (d *PDViewportDictionary) SetBBox(rectangle *common.PDRectangle) {
	if rectangle == nil {
		d.viewportDictionary.SetItem(cos.BBox, nil)
		return
	}
	d.viewportDictionary.SetItem(cos.BBox, rectangle.COSObject())
}

// Name returns the /Name of the viewport.
func (d *PDViewportDictionary) Name() string {
	return d.viewportDictionary.GetNameAsString(cos.NameKey, "")
}

// SetName sets the /Name of the viewport.
func (d *PDViewportDictionary) SetName(name string) {
	d.viewportDictionary.SetName(cos.NameKey, name)
}

// Measure returns the /Measure dictionary, or nil.
func (d *PDViewportDictionary) Measure() *PDMeasureDictionary {
	if base := d.viewportDictionary.GetCOSDictionary(cos.Measure); base != nil {
		return NewPDMeasureDictionaryOf(base)
	}
	return nil
}

// SetMeasure sets the /Measure dictionary.
func (d *PDViewportDictionary) SetMeasure(measure *PDMeasureDictionary) {
	if measure == nil {
		d.viewportDictionary.SetItem(cos.Measure, nil)
		return
	}
	d.viewportDictionary.SetItem(cos.Measure, measure.COSObject())
}
