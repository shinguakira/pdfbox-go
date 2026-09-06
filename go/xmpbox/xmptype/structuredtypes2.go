package xmptype

// The rest of the structured types: the resource reference, the four exif ones
// and the four PDF/A extension schema ones.
//
// Port of ResourceRefType, OECFType, CFAPatternType, DeviceSettingsType,
// FlashType, PDFASchemaType, PDFAFieldType, PDFAPropertyType and PDFATypeType,
// for the reason structuredtypes.go gives.

import "time"

// ResourceRefType is a reference to another resource.
type ResourceRefType struct{ StructuredType }

// The fields ResourceRefType declares.
const (
	ResourceRefDocumentID      = "documentID"
	ResourceRefFilePath        = "filePath"
	ResourceRefInstanceID      = "instanceID"
	ResourceRefLastModifyDate  = "lastModifyDate"
	ResourceRefManageTo        = "manageTo"
	ResourceRefManageUI        = "manageUI"
	ResourceRefManager         = "manager"
	ResourceRefManagerVariant  = "managerVariant"
	ResourceRefPartMapping     = "partMapping"
	ResourceRefRenditionParams = "renditionParams"
	ResourceRefVersionID       = "versionID"
	ResourceRefMaskMarkers     = "maskMarkers"
	ResourceRefRenditionClass  = "renditionClass"
	ResourceRefFromPart        = "fromPart"
	ResourceRefToPart          = "toPart"

	// ResourceRefAlternatePaths is the one field Java declares without a
	// @PropertyType annotation, so it is not in the description below.
	ResourceRefAlternatePaths = "alternatePaths"
)

var resourceRefInfo = StructuredTypeInfo{
	PreferedPrefix: "stRef",
	Namespace:      "http://ns.adobe.com/xap/1.0/sType/ResourceRef#",
}

var resourceRefProperties = describe(map[string]PropertyType{
	ResourceRefDocumentID:      NewPropertyTypeCard(URI, Simple),
	ResourceRefFilePath:        NewPropertyTypeCard(URI, Simple),
	ResourceRefInstanceID:      NewPropertyTypeCard(URI, Simple),
	ResourceRefLastModifyDate:  NewPropertyTypeCard(Date, Simple),
	ResourceRefManageTo:        NewPropertyTypeCard(URI, Simple),
	ResourceRefManageUI:        NewPropertyTypeCard(URI, Simple),
	ResourceRefManager:         NewPropertyTypeCard(AgentName, Simple),
	ResourceRefManagerVariant:  NewPropertyTypeCard(Text, Simple),
	ResourceRefPartMapping:     NewPropertyTypeCard(Text, Simple),
	ResourceRefRenditionParams: NewPropertyTypeCard(Text, Simple),
	ResourceRefVersionID:       NewPropertyTypeCard(Text, Simple),
	ResourceRefMaskMarkers:     NewPropertyTypeCard(Choice, Simple),
	ResourceRefRenditionClass:  NewPropertyTypeCard(RenditionClass, Simple),
	ResourceRefFromPart:        NewPropertyTypeCard(Part, Simple),
	ResourceRefToPart:          NewPropertyTypeCard(Part, Simple),
}, ResourceRefDocumentID, ResourceRefFilePath, ResourceRefInstanceID,
	ResourceRefLastModifyDate, ResourceRefManageTo, ResourceRefManageUI,
	ResourceRefManager, ResourceRefManagerVariant, ResourceRefPartMapping,
	ResourceRefRenditionParams, ResourceRefVersionID, ResourceRefMaskMarkers,
	ResourceRefRenditionClass, ResourceRefFromPart, ResourceRefToPart)

// NewResourceRefType returns a resource reference.
func NewResourceRefType(metadata MetadataLike) *ResourceRefType {
	r := &ResourceRefType{}
	_ = r.InitStructuredType(metadata, resourceRefInfo, "", "", "")
	r.AddNamespace(r.Namespace(), r.PreferedPrefix())
	return r
}

// textOfType reads the named field, which must be stored as the given type, as
// a string.
//
// Java writes each of these as a cast of getFirstEquivalentProperty to the
// declared type and then to TextType, which is where the type name comes from.
func (r *ResourceRefType) textOfType(fieldName, typeName string) string {
	if text, isText := r.FirstEquivalentProperty(fieldName, typeName).(AbstractSimpleProperty); isText {
		return text.StringValue()
	}
	return ""
}

// DocumentID returns the document the reference points at.
func (r *ResourceRefType) DocumentID() string {
	return r.textOfType(ResourceRefDocumentID, "URIType")
}

// SetDocumentID sets the document the reference points at.
func (r *ResourceRefType) SetDocumentID(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefDocumentID, value)
}

// FilePath returns the path of the referenced file.
func (r *ResourceRefType) FilePath() string {
	return r.textOfType(ResourceRefFilePath, "URIType")
}

// SetFilePath sets the path of the referenced file.
func (r *ResourceRefType) SetFilePath(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefFilePath, value)
}

// InstanceID returns the instance the reference points at.
func (r *ResourceRefType) InstanceID() string {
	return r.textOfType(ResourceRefInstanceID, "URIType")
}

// SetInstanceID sets the instance the reference points at.
func (r *ResourceRefType) SetInstanceID(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefInstanceID, value)
}

// LastModifyDate returns when the referenced resource last changed, the second
// result being false where there is no date.
func (r *ResourceRefType) LastModifyDate() (time.Time, bool) {
	return r.DatePropertyAsCalendar(ResourceRefLastModifyDate)
}

// SetLastModifyDate sets when the referenced resource last changed.
func (r *ResourceRefType) SetLastModifyDate(value time.Time) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefLastModifyDate, value)
}

// ManageUI returns the URI of the management interface.
func (r *ResourceRefType) ManageUI() string {
	return r.textOfType(ResourceRefManageUI, "URIType")
}

// SetManageUI sets the URI of the management interface.
func (r *ResourceRefType) SetManageUI(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefManageUI, value)
}

// ManageTo returns the URI the resource is managed to.
func (r *ResourceRefType) ManageTo() string {
	return r.textOfType(ResourceRefManageTo, "URIType")
}

// SetManageTo sets the URI the resource is managed to.
func (r *ResourceRefType) SetManageTo(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefManageTo, value)
}

// Manager returns the software managing the resource.
func (r *ResourceRefType) Manager() string {
	return r.textOfType(ResourceRefManager, "AgentNameType")
}

// SetManager sets the software managing the resource.
func (r *ResourceRefType) SetManager(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefManager, value)
}

// ManagerVariant returns which variant of the manager it is.
func (r *ResourceRefType) ManagerVariant() string {
	return r.textOfType(ResourceRefManagerVariant, "TextType")
}

// SetManagerVariant sets which variant of the manager it is.
func (r *ResourceRefType) SetManagerVariant(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefManagerVariant, value)
}

// PartMapping returns how the parts map onto each other.
func (r *ResourceRefType) PartMapping() string {
	return r.textOfType(ResourceRefPartMapping, "TextType")
}

// SetPartMapping sets how the parts map onto each other.
func (r *ResourceRefType) SetPartMapping(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefPartMapping, value)
}

// RenditionParams returns the parameters of the rendition.
func (r *ResourceRefType) RenditionParams() string {
	return r.textOfType(ResourceRefRenditionParams, "TextType")
}

// SetRenditionParams sets the parameters of the rendition.
func (r *ResourceRefType) SetRenditionParams(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefRenditionParams, value)
}

// VersionID returns the version referred to.
func (r *ResourceRefType) VersionID() string {
	return r.textOfType(ResourceRefVersionID, "TextType")
}

// SetVersionID sets the version referred to.
func (r *ResourceRefType) SetVersionID(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefVersionID, value)
}

// MaskMarkers returns which markers are masked.
func (r *ResourceRefType) MaskMarkers() string {
	return r.textOfType(ResourceRefMaskMarkers, "ChoiceType")
}

// SetMaskMarkers sets which markers are masked.
func (r *ResourceRefType) SetMaskMarkers(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefMaskMarkers, value)
}

// RenditionClass returns the class of the rendition.
func (r *ResourceRefType) RenditionClass() string {
	return r.textOfType(ResourceRefRenditionClass, "RenditionClassType")
}

// SetRenditionClass sets the class of the rendition.
func (r *ResourceRefType) SetRenditionClass(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefRenditionClass, value)
}

// FromPart returns the part referred from.
func (r *ResourceRefType) FromPart() string {
	return r.textOfType(ResourceRefFromPart, "PartType")
}

// SetFromPart sets the part referred from.
func (r *ResourceRefType) SetFromPart(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefFromPart, value)
}

// ToPart returns the part referred to.
func (r *ResourceRefType) ToPart() string {
	return r.textOfType(ResourceRefToPart, "PartType")
}

// SetToPart sets the part referred to.
func (r *ResourceRefType) SetToPart(value string) error {
	return r.AddSimpleProperty(resourceRefProperties, ResourceRefToPart, value)
}

// AddAlternatePath appends a path the resource may also be found at.
func (r *ResourceRefType) AddAlternatePath(value string) error {
	seq, isSeq := r.FirstEquivalentProperty(
		ResourceRefAlternatePaths, "ArrayProperty").(*ArrayProperty)
	if !isSeq {
		seq = r.Metadata().TypeMapping().CreateArrayProperty(
			"", r.PreferedPrefix(), ResourceRefAlternatePaths, Seq)
		r.AddProperty(seq)
	}
	tm := r.Metadata().TypeMapping()
	tt, err := tm.InstanciateSimpleProperty("", "rdf", "li", value, Text)
	if err != nil {
		return err
	}
	seq.AddProperty(tt)
	return nil
}

// AlternatePathsProperty returns the array of alternate paths, or nil.
func (r *ResourceRefType) AlternatePathsProperty() *ArrayProperty {
	seq, isSeq := r.FirstEquivalentProperty(
		ResourceRefAlternatePaths, "ArrayProperty").(*ArrayProperty)
	if !isSeq {
		return nil
	}
	return seq
}

// AlternatePaths returns the paths the resource may also be found at, and nil
// where there are none.
func (r *ResourceRefType) AlternatePaths() []string {
	if seq := r.AlternatePathsProperty(); seq != nil {
		return seq.ElementsAsString()
	}
	return nil
}

// TypeName returns the name of the Java class this stands for.
func (r *ResourceRefType) TypeName() string { return "ResourceRefType" }

// exifNamespace is the namespace the four exif structured types share, which is
// also the namespace of ExifSchema.
const exifNamespace = "http://ns.adobe.com/exif/1.0/"

// OECFType is an opto-electronic conversion function.
type OECFType struct{ StructuredType }

// The fields OECFType declares.
const (
	OECFColumns = "Columns"
	OECFNames   = "Names"
	OECFRows    = "Rows"
	OECFValues  = "Values"
)

var oecfInfo = StructuredTypeInfo{PreferedPrefix: "exif", Namespace: exifNamespace}

var oecfProperties = describe(map[string]PropertyType{
	OECFColumns: NewPropertyType(Integer),
	OECFNames:   NewPropertyTypeCard(Text, Seq),
	OECFRows:    NewPropertyType(Integer),
	OECFValues:  NewPropertyTypeCard(Real, Seq),
}, OECFColumns, OECFNames, OECFRows, OECFValues)

// NewOECFType returns an OECF value.
func NewOECFType(metadata MetadataLike) *OECFType {
	o := &OECFType{}
	_ = o.InitStructuredType(metadata, oecfInfo, "", "", "")
	return o
}

// TypeName returns the name of the Java class this stands for.
func (o *OECFType) TypeName() string { return "OECFType" }

// CFAPatternType is a colour filter array pattern.
type CFAPatternType struct{ StructuredType }

// The fields CFAPatternType declares.
const (
	CFAPatternColumns = "Columns"
	CFAPatternRows    = "Rows"
	CFAPatternValues  = "Values"
)

var cfaPatternInfo = StructuredTypeInfo{PreferedPrefix: "exif", Namespace: exifNamespace}

var cfaPatternProperties = describe(map[string]PropertyType{
	CFAPatternColumns: NewPropertyType(Integer),
	CFAPatternRows:    NewPropertyType(Integer),
	CFAPatternValues:  NewPropertyTypeCard(Integer, Seq),
}, CFAPatternColumns, CFAPatternRows, CFAPatternValues)

// NewCFAPatternType returns a colour filter array pattern.
func NewCFAPatternType(metadata MetadataLike) *CFAPatternType {
	c := &CFAPatternType{}
	_ = c.InitStructuredType(metadata, cfaPatternInfo, "", "", "")
	return c
}

// TypeName returns the name of the Java class this stands for.
func (c *CFAPatternType) TypeName() string { return "CFAPatternType" }

// DeviceSettingsType is a camera's settings.
type DeviceSettingsType struct{ StructuredType }

// The fields DeviceSettingsType declares.
const (
	DeviceSettingsColumns  = "Columns"
	DeviceSettingsRows     = "Rows"
	DeviceSettingsSettings = "Settings"
)

var deviceSettingsInfo = StructuredTypeInfo{PreferedPrefix: "exif", Namespace: exifNamespace}

var deviceSettingsProperties = describe(map[string]PropertyType{
	DeviceSettingsColumns:  NewPropertyType(Integer),
	DeviceSettingsRows:     NewPropertyType(Integer),
	DeviceSettingsSettings: NewPropertyTypeCard(Text, Seq),
}, DeviceSettingsColumns, DeviceSettingsRows, DeviceSettingsSettings)

// NewDeviceSettingsType returns a device settings value.
func NewDeviceSettingsType(metadata MetadataLike) *DeviceSettingsType {
	d := &DeviceSettingsType{}
	_ = d.InitStructuredType(metadata, deviceSettingsInfo, "", "", "")
	return d
}

// TypeName returns the name of the Java class this stands for.
func (d *DeviceSettingsType) TypeName() string { return "DeviceSettingsType" }

// FlashType is what the flash did.
type FlashType struct{ StructuredType }

// The fields FlashType declares.
const (
	FlashFired       = "Fired"
	FlashFunction    = "Function"
	FlashRedEyeMode  = "RedEyeMode"
	FlashMode        = "Mode"
	FlashReturnField = "Return"
)

var flashInfo = StructuredTypeInfo{PreferedPrefix: "exif", Namespace: exifNamespace}

var flashProperties = describe(map[string]PropertyType{
	FlashFired:       NewPropertyType(Boolean),
	FlashFunction:    NewPropertyType(Boolean),
	FlashRedEyeMode:  NewPropertyType(Boolean),
	FlashMode:        NewPropertyType(Integer),
	FlashReturnField: NewPropertyType(Integer),
}, FlashFired, FlashFunction, FlashRedEyeMode, FlashMode, FlashReturnField)

// NewFlashType returns a flash value.
func NewFlashType(metadata MetadataLike) *FlashType {
	f := &FlashType{}
	_ = f.InitStructuredType(metadata, flashInfo, "", "", "")
	return f
}

// TypeName returns the name of the Java class this stands for.
func (f *FlashType) TypeName() string { return "FlashType" }

// PDFASchemaType is one schema a PDF/A extension schema describes.
type PDFASchemaType struct{ StructuredType }

// The fields PDFASchemaType declares.
const (
	PDFASchemaSchema       = "schema"
	PDFASchemaNamespaceURI = "namespaceURI"
	PDFASchemaPrefix       = "prefix"
	PDFASchemaProperty     = "property"
	PDFASchemaValueType    = "valueType"
)

var pdfaSchemaInfo = StructuredTypeInfo{
	PreferedPrefix: "pdfaSchema",
	Namespace:      "http://www.aiim.org/pdfa/ns/schema#",
}

var pdfaSchemaProperties = describe(map[string]PropertyType{
	PDFASchemaSchema:       NewPropertyTypeCard(Text, Simple),
	PDFASchemaNamespaceURI: NewPropertyTypeCard(URI, Simple),
	PDFASchemaPrefix:       NewPropertyTypeCard(Text, Simple),
	PDFASchemaProperty:     NewPropertyTypeCard(PDFAProperty, Seq),
	PDFASchemaValueType:    NewPropertyTypeCard(PDFAType, Seq),
}, PDFASchemaSchema, PDFASchemaNamespaceURI, PDFASchemaPrefix, PDFASchemaProperty,
	PDFASchemaValueType)

// NewPDFASchemaType returns a PDF/A schema description.
func NewPDFASchemaType(metadata MetadataLike) *PDFASchemaType {
	p := &PDFASchemaType{}
	_ = p.InitStructuredType(metadata, pdfaSchemaInfo, "", "", "")
	return p
}

// NamespaceURI returns the namespace the described schema uses.
func (p *PDFASchemaType) NamespaceURI() string {
	return stringOf(&p.StructuredType, PDFASchemaNamespaceURI)
}

// PrefixValue returns the prefix the described schema prefers.
func (p *PDFASchemaType) PrefixValue() string {
	return stringOf(&p.StructuredType, PDFASchemaPrefix)
}

// PropertyArray returns the properties the described schema declares.
//
// Java names it getProperty, which collides with the getProperty every complex
// property has; the port spells this one out.
func (p *PDFASchemaType) PropertyArray() *ArrayProperty {
	return p.ArrayPropertyOf(PDFASchemaProperty)
}

// ValueType returns the types the described schema declares.
func (p *PDFASchemaType) ValueType() *ArrayProperty {
	return p.ArrayPropertyOf(PDFASchemaValueType)
}

// TypeName returns the name of the Java class this stands for.
func (p *PDFASchemaType) TypeName() string { return "PDFASchemaType" }

// PDFAFieldType is one field of a type a PDF/A extension schema defines.
type PDFAFieldType struct{ StructuredType }

// The fields PDFAFieldType declares.
const (
	PDFAFieldName        = "name"
	PDFAFieldValueType   = "valueType"
	PDFAFieldDescription = "description"
)

var pdfaFieldInfo = StructuredTypeInfo{
	PreferedPrefix: "pdfaField",
	Namespace:      "http://www.aiim.org/pdfa/ns/field#",
}

var pdfaFieldProperties = describe(map[string]PropertyType{
	PDFAFieldName:        NewPropertyTypeCard(Text, Simple),
	PDFAFieldValueType:   NewPropertyTypeCard(Choice, Simple),
	PDFAFieldDescription: NewPropertyTypeCard(Text, Simple),
}, PDFAFieldName, PDFAFieldValueType, PDFAFieldDescription)

// NewPDFAFieldType returns a PDF/A field description.
func NewPDFAFieldType(metadata MetadataLike) *PDFAFieldType {
	p := &PDFAFieldType{}
	_ = p.InitStructuredType(metadata, pdfaFieldInfo, "", "", "")
	return p
}

// Name returns the name of the field.
func (p *PDFAFieldType) Name() string { return stringOf(&p.StructuredType, PDFAFieldName) }

// ValueType returns the type of the field.
func (p *PDFAFieldType) ValueType() string {
	return stringOf(&p.StructuredType, PDFAFieldValueType)
}

// Description returns what the field is for.
func (p *PDFAFieldType) Description() string {
	return stringOf(&p.StructuredType, PDFAFieldDescription)
}

// TypeName returns the name of the Java class this stands for.
func (p *PDFAFieldType) TypeName() string { return "PDFAFieldType" }

// PDFAPropertyType is one property a PDF/A extension schema declares.
type PDFAPropertyType struct{ StructuredType }

// The fields PDFAPropertyType declares.
const (
	PDFAPropertyName        = "name"
	PDFAPropertyValueType   = "valueType"
	PDFAPropertyCategory    = "category"
	PDFAPropertyDescription = "description"
)

var pdfaPropertyInfo = StructuredTypeInfo{
	PreferedPrefix: "pdfaProperty",
	Namespace:      "http://www.aiim.org/pdfa/ns/property#",
}

var pdfaPropertyProperties = describe(map[string]PropertyType{
	PDFAPropertyName:        NewPropertyTypeCard(Text, Simple),
	PDFAPropertyValueType:   NewPropertyTypeCard(Choice, Simple),
	PDFAPropertyCategory:    NewPropertyTypeCard(Choice, Simple),
	PDFAPropertyDescription: NewPropertyTypeCard(Text, Simple),
}, PDFAPropertyName, PDFAPropertyValueType, PDFAPropertyCategory, PDFAPropertyDescription)

// NewPDFAPropertyType returns a PDF/A property description.
func NewPDFAPropertyType(metadata MetadataLike) *PDFAPropertyType {
	p := &PDFAPropertyType{}
	_ = p.InitStructuredType(metadata, pdfaPropertyInfo, "", "", "")
	return p
}

// Name returns the name of the property.
func (p *PDFAPropertyType) Name() string { return stringOf(&p.StructuredType, PDFAPropertyName) }

// ValueType returns the type of the property.
func (p *PDFAPropertyType) ValueType() string {
	return stringOf(&p.StructuredType, PDFAPropertyValueType)
}

// Description returns what the property is for.
func (p *PDFAPropertyType) Description() string {
	return stringOf(&p.StructuredType, PDFAPropertyDescription)
}

// Category returns whether the property is internal or external.
func (p *PDFAPropertyType) Category() string {
	return stringOf(&p.StructuredType, PDFAPropertyCategory)
}

// TypeName returns the name of the Java class this stands for.
func (p *PDFAPropertyType) TypeName() string { return "PDFAPropertyType" }

// PDFATypeType is one value type a PDF/A extension schema defines.
type PDFATypeType struct{ StructuredType }

// The fields PDFATypeType declares.
const (
	PDFATypeType_       = "type"
	PDFATypeNSURI       = "namespaceURI"
	PDFATypePrefix      = "prefix"
	PDFATypeDescription = "description"
	PDFATypeField       = "field"
)

var pdfaTypeInfo = StructuredTypeInfo{
	PreferedPrefix: "pdfaType",
	Namespace:      "http://www.aiim.org/pdfa/ns/type#",
}

var pdfaTypeProperties = describe(map[string]PropertyType{
	PDFATypeType_:       NewPropertyTypeCard(Text, Simple),
	PDFATypeNSURI:       NewPropertyTypeCard(URI, Simple),
	PDFATypePrefix:      NewPropertyTypeCard(Text, Simple),
	PDFATypeDescription: NewPropertyTypeCard(Text, Simple),
	PDFATypeField:       NewPropertyTypeCard(PDFAField, Seq),
}, PDFATypeType_, PDFATypeNSURI, PDFATypePrefix, PDFATypeDescription, PDFATypeField)

// NewPDFATypeType returns a PDF/A value type description.
func NewPDFATypeType(metadata MetadataLike) *PDFATypeType {
	p := &PDFATypeType{}
	_ = p.InitStructuredType(metadata, pdfaTypeInfo, "", "", "")
	return p
}

// NamespaceURI returns the namespace of the described type.
func (p *PDFATypeType) NamespaceURI() string { return stringOf(&p.StructuredType, PDFATypeNSURI) }

// Type returns the name of the described type.
func (p *PDFATypeType) Type() string { return stringOf(&p.StructuredType, PDFATypeType_) }

// PrefixValue returns the prefix the described type prefers.
func (p *PDFATypeType) PrefixValue() string { return stringOf(&p.StructuredType, PDFATypePrefix) }

// Description returns what the type is for.
func (p *PDFATypeType) Description() string {
	return stringOf(&p.StructuredType, PDFATypeDescription)
}

// Fields returns the fields the described type has.
func (p *PDFATypeType) Fields() *ArrayProperty { return p.ArrayPropertyOf(PDFATypeField) }

// TypeName returns the name of the Java class this stands for.
func (p *PDFATypeType) TypeName() string { return "PDFATypeType" }

// init records each structured type against the Types constant that names it,
// which is the Class each Java constant carries, together with the
// @StructuredType annotation and the @PropertyType fields Java reads by
// reflection.
func init() {
	RegisterStructuredType(Colorant, colorantInfo, colorantProperties,
		func(m MetadataLike) AbstractStructuredType { return NewColorantType(m) })
	RegisterStructuredType(Dimensions, dimensionsInfo, dimensionsProperties,
		func(m MetadataLike) AbstractStructuredType { return NewDimensionsType(m) })
	RegisterStructuredType(Font, fontInfo, fontProperties,
		func(m MetadataLike) AbstractStructuredType { return NewFontType(m) })
	RegisterStructuredType(Layer, layerInfo, layerProperties,
		func(m MetadataLike) AbstractStructuredType { return NewLayerType(m) })
	RegisterStructuredType(Thumbnail, thumbnailInfo, thumbnailProperties,
		func(m MetadataLike) AbstractStructuredType { return NewThumbnailType(m) })
	RegisterStructuredType(ResourceEvent, resourceEventInfo, resourceEventProperties,
		func(m MetadataLike) AbstractStructuredType { return NewResourceEventType(m) })
	RegisterStructuredType(ResourceRef, resourceRefInfo, resourceRefProperties,
		func(m MetadataLike) AbstractStructuredType { return NewResourceRefType(m) })
	RegisterStructuredType(Version, versionInfo, versionProperties,
		func(m MetadataLike) AbstractStructuredType { return NewVersionType(m) })
	RegisterStructuredType(Job, jobInfo, jobProperties,
		func(m MetadataLike) AbstractStructuredType { return NewJobType(m) })
	RegisterStructuredType(OECF, oecfInfo, oecfProperties,
		func(m MetadataLike) AbstractStructuredType { return NewOECFType(m) })
	RegisterStructuredType(CFAPattern, cfaPatternInfo, cfaPatternProperties,
		func(m MetadataLike) AbstractStructuredType { return NewCFAPatternType(m) })
	RegisterStructuredType(DeviceSettings, deviceSettingsInfo, deviceSettingsProperties,
		func(m MetadataLike) AbstractStructuredType { return NewDeviceSettingsType(m) })
	RegisterStructuredType(Flash, flashInfo, flashProperties,
		func(m MetadataLike) AbstractStructuredType { return NewFlashType(m) })
	RegisterStructuredType(PDFASchema, pdfaSchemaInfo, pdfaSchemaProperties,
		func(m MetadataLike) AbstractStructuredType { return NewPDFASchemaType(m) })
	RegisterStructuredType(PDFAField, pdfaFieldInfo, pdfaFieldProperties,
		func(m MetadataLike) AbstractStructuredType { return NewPDFAFieldType(m) })
	RegisterStructuredType(PDFAProperty, pdfaPropertyInfo, pdfaPropertyProperties,
		func(m MetadataLike) AbstractStructuredType { return NewPDFAPropertyType(m) })
	RegisterStructuredType(PDFAType, pdfaTypeInfo, pdfaTypeProperties,
		func(m MetadataLike) AbstractStructuredType { return NewPDFATypeType(m) })
}
