package xmptype

// The seventeen structured types.
//
// Port of ColorantType, DimensionsType, FontType, LayerType, ThumbnailType,
// ResourceEventType, ResourceRefType, VersionType, JobType, OECFType,
// CFAPatternType, DeviceSettingsType, FlashType, PDFASchemaType, PDFAFieldType,
// PDFAPropertyType and PDFATypeType. Java gives each a file; they are a
// constructor, a table of field names and a run of two-line accessors, so the
// port keeps them together the way it keeps the operator processors together.
//
// Each Java class carries a @StructuredType annotation and a run of
// @PropertyType-annotated String constants, which TypeMapping reads by
// reflection. Here each declares the same two things as values and registers
// them against its Types constant from the init at the end of this file.

import (
	"time"

	"github.com/shinguakira/pdfbox-go/go/internal/javafmt"
)

// stringOf reads the named field as a string, and answers the empty string
// where it is absent -- which is Java's null.
//
// Java writes this out per accessor, as either getPropertyValueAsString or a
// cast of getProperty/getFirstEquivalentProperty to a TextType. The three come
// to the same answer for a field the type declares, since the description
// decides what was stored.
func stringOf(s *StructuredType, fieldName string) string {
	return s.PropertyValueAsString(fieldName)
}

// ColorantType is a colour in a swatch.
type ColorantType struct{ StructuredType }

// The fields ColorantType declares.
const (
	ColorantA          = "A"
	ColorantB          = "B"
	ColorantL          = "L"
	ColorantBlack      = "black"
	ColorantCyan       = "cyan"
	ColorantMagenta    = "magenta"
	ColorantYellow     = "yellow"
	ColorantBlue       = "blue"
	ColorantGreen      = "green"
	ColorantRed        = "red"
	ColorantMode       = "mode"
	ColorantSwatchName = "swatchName"
	ColorantTypeField  = "type"
)

// colorantInfo is the @StructuredType annotation of ColorantType.
var colorantInfo = StructuredTypeInfo{
	PreferedPrefix: "xmpG",
	Namespace:      "http://ns.adobe.com/xap/1.0/g/",
}

// colorantProperties is the @PropertyType fields of ColorantType.
var colorantProperties = describe(map[string]PropertyType{
	ColorantA:          NewPropertyType(Integer),
	ColorantB:          NewPropertyType(Integer),
	ColorantL:          NewPropertyType(Real),
	ColorantBlack:      NewPropertyType(Real),
	ColorantCyan:       NewPropertyType(Real),
	ColorantMagenta:    NewPropertyType(Real),
	ColorantYellow:     NewPropertyType(Real),
	ColorantBlue:       NewPropertyType(Integer),
	ColorantGreen:      NewPropertyType(Integer),
	ColorantRed:        NewPropertyType(Integer),
	ColorantMode:       NewPropertyTypeCard(Choice, Simple),
	ColorantSwatchName: NewPropertyType(Text),
	ColorantTypeField:  NewPropertyTypeCard(Choice, Simple),
}, ColorantA, ColorantB, ColorantL, ColorantBlack, ColorantCyan, ColorantMagenta,
	ColorantYellow, ColorantBlue, ColorantGreen, ColorantRed, ColorantMode,
	ColorantSwatchName, ColorantTypeField)

// NewColorantType returns a colorant.
func NewColorantType(metadata MetadataLike) *ColorantType {
	c := &ColorantType{}
	_ = c.InitStructuredType(metadata, colorantInfo, "", "", "")
	return c
}

// TypeName returns the name of the Java class this stands for.
func (c *ColorantType) TypeName() string { return "ColorantType" }

// DimensionsType is a width and a height in some unit.
type DimensionsType struct{ StructuredType }

// The fields DimensionsType declares.
const (
	DimensionsH    = "h"
	DimensionsW    = "w"
	DimensionsUnit = "unit"
)

var dimensionsInfo = StructuredTypeInfo{
	PreferedPrefix: "stDim",
	Namespace:      "http://ns.adobe.com/xap/1.0/sType/Dimensions#",
}

var dimensionsProperties = describe(map[string]PropertyType{
	DimensionsH:    NewPropertyType(Real),
	DimensionsW:    NewPropertyType(Real),
	DimensionsUnit: NewPropertyType(Text),
}, DimensionsH, DimensionsW, DimensionsUnit)

// NewDimensionsType returns a dimensions value.
func NewDimensionsType(metadata MetadataLike) *DimensionsType {
	d := &DimensionsType{}
	_ = d.InitStructuredType(metadata, dimensionsInfo, "", "", "")
	return d
}

// H returns the height, the second result being false where there is none.
func (d *DimensionsType) H() (float32, bool) { return d.realOf(DimensionsH) }

// W returns the width, the second result being false where there is none.
func (d *DimensionsType) W() (float32, bool) { return d.realOf(DimensionsW) }

// realOf is Java's `getProperty(x) instanceof RealType` arm.
func (d *DimensionsType) realOf(fieldName string) (float32, bool) {
	if real, isReal := d.Property(fieldName).(*RealType); isReal {
		return real.RealValue(), true
	}
	return 0, false
}

// Unit returns the unit the two are measured in.
func (d *DimensionsType) Unit() string { return stringOf(&d.StructuredType, DimensionsUnit) }

// TypeName returns the name of the Java class this stands for.
func (d *DimensionsType) TypeName() string { return "DimensionsType" }

// String returns the Java toString form.
//
// Java concatenates the two Floats, which write themselves as "4.0" and a
// missing one as "null"; the unit is a String, and a missing one is "null" as
// well.
func (d *DimensionsType) String() string {
	return "DimensionsType{" + dimensionOf(d.W()) + " x " + dimensionOf(d.H()) + " " +
		nullable(d.Property(DimensionsUnit), d.Unit()) + "}"
}

// dimensionOf writes a Float the way Java's string concatenation does.
func dimensionOf(value float32, held bool) string {
	if !held {
		return "null"
	}
	return javafmt.Float32(value)
}

// nullable writes a String the way Java's string concatenation does, the
// property saying whether there is one at all.
func nullable(property AbstractField, value string) string {
	if property == nil {
		return "null"
	}
	return value
}

// FontType is a font a document used.
type FontType struct{ StructuredType }

// The fields FontType declares.
const (
	FontChildFontFiles = "childFontFiles"
	FontComposite      = "composite"
	FontFontFace       = "fontFace"
	FontFontFamily     = "fontFamily"
	FontFontFileName   = "fontFileName"
	FontFontName       = "fontName"
	FontFontType       = "fontType"
	FontVersionString  = "versionString"
)

var fontInfo = StructuredTypeInfo{
	PreferedPrefix: "stFnt",
	Namespace:      "http://ns.adobe.com/xap/1.0/sType/Font#",
}

var fontProperties = describe(map[string]PropertyType{
	FontChildFontFiles: NewPropertyTypeCard(Text, Seq),
	FontComposite:      NewPropertyType(Boolean),
	FontFontFace:       NewPropertyType(Text),
	FontFontFamily:     NewPropertyType(Text),
	FontFontFileName:   NewPropertyType(Text),
	FontFontName:       NewPropertyType(Text),
	FontFontType:       NewPropertyTypeCard(Choice, Simple),
	FontVersionString:  NewPropertyType(Text),
}, FontChildFontFiles, FontComposite, FontFontFace, FontFontFamily, FontFontFileName,
	FontFontName, FontFontType, FontVersionString)

// NewFontType returns a font value.
func NewFontType(metadata MetadataLike) *FontType {
	f := &FontType{}
	_ = f.InitStructuredType(metadata, fontInfo, "", "", "")
	return f
}

// TypeName returns the name of the Java class this stands for.
func (f *FontType) TypeName() string { return "FontType" }

// LayerType is a Photoshop layer.
type LayerType struct{ StructuredType }

// The fields LayerType declares.
const (
	LayerLayerName = "LayerName"
	LayerLayerText = "LayerText"
)

var layerInfo = StructuredTypeInfo{
	PreferedPrefix: "photoshop",
	Namespace:      "http://ns.adobe.com/photoshop/1.0/",
}

var layerProperties = describe(map[string]PropertyType{
	LayerLayerName: NewPropertyTypeCard(Text, Simple),
	LayerLayerText: NewPropertyTypeCard(Text, Simple),
}, LayerLayerName, LayerLayerText)

// NewLayerType returns a layer value.
func NewLayerType(metadata MetadataLike) *LayerType {
	l := &LayerType{}
	_ = l.InitStructuredType(metadata, layerInfo, "", "", "")
	l.SetAttribute(NewAttribute(RDFNamespace, "parseType", "Resource"))
	return l
}

// LayerName returns the name of the layer.
func (l *LayerType) LayerName() string {
	if text, isText := l.FirstEquivalentProperty(LayerLayerName, "TextType").(*TextType); isText {
		return text.StringValue()
	}
	return ""
}

// SetLayerName sets the name of the layer.
func (l *LayerType) SetLayerName(image string) error {
	text, err := l.CreateTextType(LayerLayerName, image)
	if err != nil {
		return err
	}
	l.AddProperty(text)
	return nil
}

// LayerText returns the text of the layer.
func (l *LayerType) LayerText() string {
	if text, isText := l.FirstEquivalentProperty(LayerLayerText, "TextType").(*TextType); isText {
		return text.StringValue()
	}
	return ""
}

// SetLayerText sets the text of the layer.
func (l *LayerType) SetLayerText(image string) error {
	text, err := l.CreateTextType(LayerLayerText, image)
	if err != nil {
		return err
	}
	l.AddProperty(text)
	return nil
}

// TypeName returns the name of the Java class this stands for.
func (l *LayerType) TypeName() string { return "LayerType" }

// ThumbnailType is a thumbnail image.
type ThumbnailType struct{ StructuredType }

// The fields ThumbnailType declares.
const (
	ThumbnailFormat = "format"
	ThumbnailHeight = "height"
	ThumbnailWidth  = "width"
	ThumbnailImage  = "image"
)

var thumbnailInfo = StructuredTypeInfo{
	PreferedPrefix: "xmpGImg",
	Namespace:      "http://ns.adobe.com/xap/1.0/g/img/",
}

var thumbnailProperties = describe(map[string]PropertyType{
	ThumbnailFormat: NewPropertyTypeCard(Choice, Simple),
	ThumbnailHeight: NewPropertyTypeCard(Integer, Simple),
	ThumbnailWidth:  NewPropertyTypeCard(Integer, Simple),
	ThumbnailImage:  NewPropertyTypeCard(Text, Simple),
}, ThumbnailFormat, ThumbnailHeight, ThumbnailWidth, ThumbnailImage)

// NewThumbnailType returns a thumbnail value.
func NewThumbnailType(metadata MetadataLike) *ThumbnailType {
	t := &ThumbnailType{}
	_ = t.InitStructuredType(metadata, thumbnailInfo, "", "", "")
	t.SetAttribute(NewAttribute(RDFNamespace, "parseType", "Resource"))
	return t
}

// Height returns the height in pixels, the second result being false where
// there is none.
func (t *ThumbnailType) Height() (int, bool) { return t.integerOf(ThumbnailHeight) }

// SetHeight sets the height in pixels.
func (t *ThumbnailType) SetHeight(height int) error {
	return t.AddSimpleProperty(thumbnailProperties, ThumbnailHeight, height)
}

// Width returns the width in pixels, the second result being false where there
// is none.
func (t *ThumbnailType) Width() (int, bool) { return t.integerOf(ThumbnailWidth) }

// SetWidth sets the width in pixels.
func (t *ThumbnailType) SetWidth(width int) error {
	return t.AddSimpleProperty(thumbnailProperties, ThumbnailWidth, width)
}

// integerOf is Java's cast of getFirstEquivalentProperty to an IntegerType.
func (t *ThumbnailType) integerOf(fieldName string) (int, bool) {
	if integer, isInteger := t.FirstEquivalentProperty(fieldName, "IntegerType").(*IntegerType); isInteger {
		return integer.IntegerValue(), true
	}
	return 0, false
}

// Image returns the image, base 64 encoded.
func (t *ThumbnailType) Image() string {
	if text, isText := t.FirstEquivalentProperty(ThumbnailImage, "TextType").(*TextType); isText {
		return text.StringValue()
	}
	return ""
}

// SetImage sets the image.
func (t *ThumbnailType) SetImage(image string) error {
	return t.AddSimpleProperty(thumbnailProperties, ThumbnailImage, image)
}

// Format returns the image format.
func (t *ThumbnailType) Format() string {
	if choice, isChoice := t.FirstEquivalentProperty(ThumbnailFormat, "ChoiceType").(*ChoiceType); isChoice {
		return choice.StringValue()
	}
	return ""
}

// SetFormat sets the image format.
func (t *ThumbnailType) SetFormat(format string) error {
	return t.AddSimpleProperty(thumbnailProperties, ThumbnailFormat, format)
}

// TypeName returns the name of the Java class this stands for.
func (t *ThumbnailType) TypeName() string { return "ThumbnailType" }

// ResourceEventType is something that happened to a resource.
type ResourceEventType struct{ StructuredType }

// The fields ResourceEventType declares.
const (
	ResourceEventAction        = "action"
	ResourceEventChanged       = "changed"
	ResourceEventInstanceID    = "instanceID"
	ResourceEventParameters    = "parameters"
	ResourceEventSoftwareAgent = "softwareAgent"
	ResourceEventWhen          = "when"
)

var resourceEventInfo = StructuredTypeInfo{
	PreferedPrefix: "stEvt",
	Namespace:      "http://ns.adobe.com/xap/1.0/sType/ResourceEvent#",
}

var resourceEventProperties = describe(map[string]PropertyType{
	ResourceEventAction:        NewPropertyTypeCard(Choice, Simple),
	ResourceEventChanged:       NewPropertyTypeCard(Text, Simple),
	ResourceEventInstanceID:    NewPropertyTypeCard(GUID, Simple),
	ResourceEventParameters:    NewPropertyTypeCard(Text, Simple),
	ResourceEventSoftwareAgent: NewPropertyTypeCard(AgentName, Simple),
	ResourceEventWhen:          NewPropertyTypeCard(Date, Simple),
}, ResourceEventAction, ResourceEventChanged, ResourceEventInstanceID,
	ResourceEventParameters, ResourceEventSoftwareAgent, ResourceEventWhen)

// NewResourceEventType returns a resource event.
func NewResourceEventType(metadata MetadataLike) *ResourceEventType {
	r := &ResourceEventType{}
	_ = r.InitStructuredType(metadata, resourceEventInfo, "", "", "")
	r.AddNamespace(r.Namespace(), r.PreferedPrefix())
	return r
}

// InstanceID returns the instance the event happened to.
func (r *ResourceEventType) InstanceID() string {
	return stringOf(&r.StructuredType, ResourceEventInstanceID)
}

// SetInstanceID sets the instance the event happened to.
func (r *ResourceEventType) SetInstanceID(value string) error {
	return r.AddSimpleProperty(resourceEventProperties, ResourceEventInstanceID, value)
}

// SoftwareAgent returns the software that did it.
func (r *ResourceEventType) SoftwareAgent() string {
	return stringOf(&r.StructuredType, ResourceEventSoftwareAgent)
}

// SetSoftwareAgent sets the software that did it.
func (r *ResourceEventType) SetSoftwareAgent(value string) error {
	return r.AddSimpleProperty(resourceEventProperties, ResourceEventSoftwareAgent, value)
}

// When returns when it happened, the second result being false where there is
// no date.
func (r *ResourceEventType) When() (time.Time, bool) {
	return r.DatePropertyAsCalendar(ResourceEventWhen)
}

// SetWhen sets when it happened.
func (r *ResourceEventType) SetWhen(value time.Time) error {
	return r.AddSimpleProperty(resourceEventProperties, ResourceEventWhen, value)
}

// Action returns what was done.
func (r *ResourceEventType) Action() string {
	return stringOf(&r.StructuredType, ResourceEventAction)
}

// SetAction sets what was done.
func (r *ResourceEventType) SetAction(value string) error {
	return r.AddSimpleProperty(resourceEventProperties, ResourceEventAction, value)
}

// Changed returns which parts changed.
func (r *ResourceEventType) Changed() string {
	return stringOf(&r.StructuredType, ResourceEventChanged)
}

// SetChanged sets which parts changed.
func (r *ResourceEventType) SetChanged(value string) error {
	return r.AddSimpleProperty(resourceEventProperties, ResourceEventChanged, value)
}

// Parameters returns the parameters of the action.
func (r *ResourceEventType) Parameters() string {
	return stringOf(&r.StructuredType, ResourceEventParameters)
}

// SetParameters sets the parameters of the action.
func (r *ResourceEventType) SetParameters(value string) error {
	return r.AddSimpleProperty(resourceEventProperties, ResourceEventParameters, value)
}

// TypeName returns the name of the Java class this stands for.
func (r *ResourceEventType) TypeName() string { return "ResourceEventType" }

// VersionType is one version of a document.
type VersionType struct{ StructuredType }

// The fields VersionType declares.
const (
	VersionComments   = "comments"
	VersionEvent      = "event"
	VersionModifier   = "modifier"
	VersionModifyDate = "modifyDate"
	VersionVersion    = "version"
)

var versionInfo = StructuredTypeInfo{
	PreferedPrefix: "stVer",
	Namespace:      "http://ns.adobe.com/xap/1.0/sType/Version#",
}

var versionProperties = describe(map[string]PropertyType{
	VersionComments:   NewPropertyTypeCard(Text, Simple),
	VersionEvent:      NewPropertyTypeCard(ResourceEvent, Simple),
	VersionModifier:   NewPropertyTypeCard(ProperName, Simple),
	VersionModifyDate: NewPropertyTypeCard(Date, Simple),
	VersionVersion:    NewPropertyTypeCard(Text, Simple),
}, VersionComments, VersionEvent, VersionModifier, VersionModifyDate, VersionVersion)

// NewVersionType returns a version value.
func NewVersionType(metadata MetadataLike) *VersionType {
	v := &VersionType{}
	_ = v.InitStructuredType(metadata, versionInfo, "", "", "")
	v.AddNamespace(v.Namespace(), v.PreferedPrefix())
	return v
}

// Comments returns the comments on the version.
func (v *VersionType) Comments() string { return stringOf(&v.StructuredType, VersionComments) }

// SetComments sets the comments on the version.
func (v *VersionType) SetComments(value string) error {
	return v.AddSimpleProperty(versionProperties, VersionComments, value)
}

// Event returns the event that made the version, or nil.
func (v *VersionType) Event() *ResourceEventType {
	event, isEvent := v.FirstEquivalentProperty(VersionEvent, "ResourceEventType").(*ResourceEventType)
	if !isEvent {
		return nil
	}
	return event
}

// SetEvent sets the event that made the version.
func (v *VersionType) SetEvent(value *ResourceEventType) { v.AddProperty(value) }

// ModifyDate returns when the version was made, the second result being false
// where there is no date.
func (v *VersionType) ModifyDate() (time.Time, bool) {
	return v.DatePropertyAsCalendar(VersionModifyDate)
}

// SetModifyDate sets when the version was made.
func (v *VersionType) SetModifyDate(value time.Time) error {
	return v.AddSimpleProperty(versionProperties, VersionModifyDate, value)
}

// Version returns the version identifier.
func (v *VersionType) Version() string { return stringOf(&v.StructuredType, VersionVersion) }

// SetVersion sets the version identifier.
func (v *VersionType) SetVersion(value string) error {
	return v.AddSimpleProperty(versionProperties, VersionVersion, value)
}

// Modifier returns who made the version.
func (v *VersionType) Modifier() string { return stringOf(&v.StructuredType, VersionModifier) }

// SetModifier sets who made the version.
func (v *VersionType) SetModifier(value string) error {
	return v.AddSimpleProperty(versionProperties, VersionModifier, value)
}

// TypeName returns the name of the Java class this stands for.
func (v *VersionType) TypeName() string { return "VersionType" }

// JobType is a print job a document belongs to.
type JobType struct{ StructuredType }

// The fields JobType declares.
const (
	JobID   = "id"
	JobName = "name"
	JobURL  = "url"
)

var jobInfo = StructuredTypeInfo{
	PreferedPrefix: "stJob",
	Namespace:      "http://ns.adobe.com/xap/1.0/sType/Job#",
}

var jobProperties = describe(map[string]PropertyType{
	JobID:   NewPropertyTypeCard(Text, Simple),
	JobName: NewPropertyTypeCard(Text, Simple),
	JobURL:  NewPropertyTypeCard(URL, Simple),
}, JobID, JobName, JobURL)

// NewJobType returns a job value.
//
// Port of JobType(XMPMetadata), which calls the two-argument constructor with a
// null prefix.
func NewJobType(metadata MetadataLike) *JobType { return NewJobTypePrefixed(metadata, "") }

// NewJobTypePrefixed returns a job value written with the given prefix.
//
// Port of JobType(XMPMetadata, String), which passes a null namespace so that
// the @StructuredType annotation supplies it.
func NewJobTypePrefixed(metadata MetadataLike, fieldPrefix string) *JobType {
	j := &JobType{}
	_ = j.InitStructuredType(metadata, jobInfo, "", fieldPrefix, "")
	j.AddNamespace(j.Namespace(), j.Prefix())
	return j
}

// SetID sets the job identifier.
func (j *JobType) SetID(id string) error {
	return j.AddSimpleProperty(jobProperties, JobID, id)
}

// SetName sets the job name.
func (j *JobType) SetName(name string) error {
	return j.AddSimpleProperty(jobProperties, JobName, name)
}

// SetURL sets the job URL.
func (j *JobType) SetURL(url string) error {
	return j.AddSimpleProperty(jobProperties, JobURL, url)
}

// ID returns the job identifier.
func (j *JobType) ID() string { return stringOf(&j.StructuredType, JobID) }

// Name returns the job name.
func (j *JobType) Name() string { return stringOf(&j.StructuredType, JobName) }

// URL returns the job URL.
func (j *JobType) URL() string { return stringOf(&j.StructuredType, JobURL) }

// TypeName returns the name of the Java class this stands for.
func (j *JobType) TypeName() string { return "JobType" }

// describe builds a PropertiesDescription from the fields a type declares, in
// the order they are named.
//
// Java reads them off the class with getFields, whose order is unspecified; the
// port takes the declaration order, which is what a reader of the Java sees.
func describe(types map[string]PropertyType, order ...string) *PropertiesDescription {
	description := NewPropertiesDescription()
	for _, name := range order {
		description.AddNewProperty(name, types[name])
	}
	return description
}
