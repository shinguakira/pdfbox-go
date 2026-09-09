package schema

// The Photoshop, media management, TIFF and EXIF schemas.
//
// Port of PhotoshopSchema, XMPMediaManagementSchema, TiffSchema and ExifSchema.
// The last two are tables of field names with no accessors of their own beyond
// the four TiffSchema adds for its two language alternatives.

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// PhotoshopSchema is the photoshop namespace.
type PhotoshopSchema struct {
	XMPSchema

	seqLayer *xmptype.ArrayProperty
}

// The fields PhotoshopSchema declares.
const (
	PhotoshopAncestorID             = "AncestorID"
	PhotoshopAuthorsPosition        = "AuthorsPosition"
	PhotoshopCaptionWriter          = "CaptionWriter"
	PhotoshopCategory               = "Category"
	PhotoshopCity                   = "City"
	PhotoshopColorMode              = "ColorMode"
	PhotoshopCountry                = "Country"
	PhotoshopCredit                 = "Credit"
	PhotoshopDateCreated            = "DateCreated"
	PhotoshopDocumentAncestors      = "DocumentAncestors"
	PhotoshopHeadline               = "Headline"
	PhotoshopHistory                = "History"
	PhotoshopICCProfile             = "ICCProfile"
	PhotoshopInstructions           = "Instructions"
	PhotoshopSource                 = "Source"
	PhotoshopState                  = "State"
	PhotoshopSupplementalCategories = "SupplementalCategories"
	PhotoshopTextLayers             = "TextLayers"
	PhotoshopTransmissionReference  = "TransmissionReference"
	PhotoshopUrgency                = "Urgency"
)

var photoshopInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: "photoshop",
	Namespace:      PhotoshopNamespace,
}

var photoshopProperties = describe(map[string]xmptype.PropertyType{
	PhotoshopAncestorID:             card(xmptype.URI, xmptype.Simple),
	PhotoshopAuthorsPosition:        card(xmptype.Text, xmptype.Simple),
	PhotoshopCaptionWriter:          card(xmptype.ProperName, xmptype.Simple),
	PhotoshopCategory:               card(xmptype.Text, xmptype.Simple),
	PhotoshopCity:                   card(xmptype.Text, xmptype.Simple),
	PhotoshopColorMode:              card(xmptype.Integer, xmptype.Simple),
	PhotoshopCountry:                card(xmptype.Text, xmptype.Simple),
	PhotoshopCredit:                 card(xmptype.Text, xmptype.Simple),
	PhotoshopDateCreated:            card(xmptype.Date, xmptype.Simple),
	PhotoshopDocumentAncestors:      card(xmptype.Text, xmptype.Bag),
	PhotoshopHeadline:               card(xmptype.Text, xmptype.Simple),
	PhotoshopHistory:                card(xmptype.Text, xmptype.Simple),
	PhotoshopICCProfile:             card(xmptype.Text, xmptype.Simple),
	PhotoshopInstructions:           card(xmptype.Text, xmptype.Simple),
	PhotoshopSource:                 card(xmptype.Text, xmptype.Simple),
	PhotoshopState:                  card(xmptype.Text, xmptype.Simple),
	PhotoshopSupplementalCategories: card(xmptype.Text, xmptype.Simple),
	PhotoshopTextLayers:             card(xmptype.Layer, xmptype.Seq),
	PhotoshopTransmissionReference:  card(xmptype.Text, xmptype.Simple),
	PhotoshopUrgency:                card(xmptype.Integer, xmptype.Simple),
}, PhotoshopAncestorID, PhotoshopAuthorsPosition, PhotoshopCaptionWriter, PhotoshopCategory,
	PhotoshopCity, PhotoshopColorMode, PhotoshopCountry, PhotoshopCredit, PhotoshopDateCreated,
	PhotoshopDocumentAncestors, PhotoshopHeadline, PhotoshopHistory, PhotoshopICCProfile,
	PhotoshopInstructions, PhotoshopSource, PhotoshopState, PhotoshopSupplementalCategories,
	PhotoshopTextLayers, PhotoshopTransmissionReference, PhotoshopUrgency)

// NewPhotoshopSchema returns the photoshop schema.
func NewPhotoshopSchema(metadata xmptype.MetadataLike) (*PhotoshopSchema, error) {
	return NewPhotoshopSchemaPrefixed(metadata, "")
}

// NewPhotoshopSchemaPrefixed returns the photoshop schema with the given
// prefix.
func NewPhotoshopSchemaPrefixed(metadata xmptype.MetadataLike,
	ownPrefix string) (*PhotoshopSchema, error) {
	s := &PhotoshopSchema{}
	s.setDescription(photoshopProperties, "PhotoshopSchema")
	if err := s.InitSchema(metadata, photoshopInfo, "", ownPrefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newPhotoshopSchemaAs is the factory's constructor.
func newPhotoshopSchemaAs(metadata xmptype.MetadataLike, prefix string) (Schema, error) {
	return NewPhotoshopSchemaPrefixed(metadata, prefix)
}

// addSimple is Java's `addProperty(instanciateSimple(name, value))`, which every
// setter here goes through.
func (s *PhotoshopSchema) addSimple(name string, value any) error {
	property, err := s.InstanciateSimple(name, value)
	if err != nil {
		return err
	}
	s.AddProperty(property)
	return nil
}

// AncestorIDProperty returns the ancestor identifier property, or nil.
func (s *PhotoshopSchema) AncestorIDProperty() *xmptype.URIValueType {
	return PropertyAs[*xmptype.URIValueType](&s.XMPSchema, PhotoshopAncestorID)
}

// AncestorID returns the ancestor identifier.
func (s *PhotoshopSchema) AncestorID() string {
	return TextValueOf(&s.XMPSchema, PhotoshopAncestorID)
}

// SetAncestorID sets the ancestor identifier.
func (s *PhotoshopSchema) SetAncestorID(text string) error {
	return s.addSimple(PhotoshopAncestorID, text)
}

// SetAncestorIDProperty sets the ancestor identifier property outright.
func (s *PhotoshopSchema) SetAncestorIDProperty(text *xmptype.URIValueType) { s.AddProperty(text) }

// AuthorsPositionProperty returns the author's position property, or nil.
func (s *PhotoshopSchema) AuthorsPositionProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopAuthorsPosition)
}

// AuthorsPosition returns the author's position.
func (s *PhotoshopSchema) AuthorsPosition() string {
	return TextValueOf(&s.XMPSchema, PhotoshopAuthorsPosition)
}

// SetAuthorsPosition sets the author's position.
func (s *PhotoshopSchema) SetAuthorsPosition(text string) error {
	return s.addSimple(PhotoshopAuthorsPosition, text)
}

// SetAuthorsPositionProperty sets the author's position property outright.
func (s *PhotoshopSchema) SetAuthorsPositionProperty(text *xmptype.TextType) { s.AddProperty(text) }

// CaptionWriterProperty returns the caption writer property, or nil.
func (s *PhotoshopSchema) CaptionWriterProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopCaptionWriter)
}

// CaptionWriter returns the caption writer.
func (s *PhotoshopSchema) CaptionWriter() string {
	return TextValueOf(&s.XMPSchema, PhotoshopCaptionWriter)
}

// SetCaptionWriter sets the caption writer.
func (s *PhotoshopSchema) SetCaptionWriter(text string) error {
	return s.addSimple(PhotoshopCaptionWriter, text)
}

// SetCaptionWriterProperty sets the caption writer property outright.
//
// Java's getter answers a TextType and its setter takes a ProperNameType, which
// is what the field is declared as.
func (s *PhotoshopSchema) SetCaptionWriterProperty(text *xmptype.ProperNameType) {
	s.AddProperty(text)
}

// CategoryProperty returns the category property, or nil.
func (s *PhotoshopSchema) CategoryProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopCategory)
}

// Category returns the category.
func (s *PhotoshopSchema) Category() string { return TextValueOf(&s.XMPSchema, PhotoshopCategory) }

// SetCategory sets the category.
func (s *PhotoshopSchema) SetCategory(text string) error {
	return s.addSimple(PhotoshopCategory, text)
}

// SetCategoryProperty sets the category property outright.
func (s *PhotoshopSchema) SetCategoryProperty(text *xmptype.TextType) { s.AddProperty(text) }

// CityProperty returns the city property, or nil.
func (s *PhotoshopSchema) CityProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopCity)
}

// City returns the city.
func (s *PhotoshopSchema) City() string { return TextValueOf(&s.XMPSchema, PhotoshopCity) }

// SetCity sets the city.
func (s *PhotoshopSchema) SetCity(text string) error { return s.addSimple(PhotoshopCity, text) }

// SetCityProperty sets the city property outright.
func (s *PhotoshopSchema) SetCityProperty(text *xmptype.TextType) { s.AddProperty(text) }

// ColorModeProperty returns the colour mode property, or nil.
func (s *PhotoshopSchema) ColorModeProperty() *xmptype.IntegerType {
	return PropertyAs[*xmptype.IntegerType](&s.XMPSchema, PhotoshopColorMode)
}

// ColorMode returns the colour mode, the second result being false where there
// is none.
func (s *PhotoshopSchema) ColorMode() (int, bool) {
	return integerValueOf(s.ColorModeProperty())
}

// SetColorMode sets the colour mode from its string form, which is what Java's
// only setter takes for an integer field.
func (s *PhotoshopSchema) SetColorMode(text string) error {
	return s.addSimple(PhotoshopColorMode, text)
}

// SetColorModeProperty sets the colour mode property outright.
func (s *PhotoshopSchema) SetColorModeProperty(text *xmptype.IntegerType) { s.AddProperty(text) }

// CountryProperty returns the country property, or nil.
func (s *PhotoshopSchema) CountryProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopCountry)
}

// Country returns the country.
func (s *PhotoshopSchema) Country() string { return TextValueOf(&s.XMPSchema, PhotoshopCountry) }

// SetCountry sets the country.
func (s *PhotoshopSchema) SetCountry(text string) error {
	return s.addSimple(PhotoshopCountry, text)
}

// SetCountryProperty sets the country property outright.
func (s *PhotoshopSchema) SetCountryProperty(text *xmptype.TextType) { s.AddProperty(text) }

// CreditProperty returns the credit property, or nil.
func (s *PhotoshopSchema) CreditProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopCredit)
}

// Credit returns the credit.
func (s *PhotoshopSchema) Credit() string { return TextValueOf(&s.XMPSchema, PhotoshopCredit) }

// SetCredit sets the credit.
func (s *PhotoshopSchema) SetCredit(text string) error { return s.addSimple(PhotoshopCredit, text) }

// SetCreditProperty sets the credit property outright.
func (s *PhotoshopSchema) SetCreditProperty(text *xmptype.TextType) { s.AddProperty(text) }

// DateCreatedProperty returns the creation date property, or nil.
func (s *PhotoshopSchema) DateCreatedProperty() *xmptype.DateType {
	return PropertyAs[*xmptype.DateType](&s.XMPSchema, PhotoshopDateCreated)
}

// DateCreated returns the creation date as the string it is written as, which
// is what Java's getter answers for this one date field.
func (s *PhotoshopSchema) DateCreated() string {
	dt := s.DateCreatedProperty()
	if dt == nil {
		return ""
	}
	return dt.StringValue()
}

// SetDateCreated sets the creation date from its string form.
func (s *PhotoshopSchema) SetDateCreated(text string) error {
	return s.addSimple(PhotoshopDateCreated, text)
}

// SetDateCreatedProperty sets the creation date property outright.
func (s *PhotoshopSchema) SetDateCreatedProperty(text *xmptype.DateType) { s.AddProperty(text) }

// AddDocumentAncestors adds a document ancestor.
func (s *PhotoshopSchema) AddDocumentAncestors(text string) error {
	return s.AddQualifiedBagValue(PhotoshopDocumentAncestors, text)
}

// DocumentAncestorsProperty returns the document ancestors array, or nil.
func (s *PhotoshopSchema) DocumentAncestorsProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, PhotoshopDocumentAncestors)
}

// DocumentAncestors returns the document ancestors.
func (s *PhotoshopSchema) DocumentAncestors() []string {
	return s.UnqualifiedBagValueList(PhotoshopDocumentAncestors)
}

// HeadlineProperty returns the headline property, or nil.
func (s *PhotoshopSchema) HeadlineProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopHeadline)
}

// Headline returns the headline.
func (s *PhotoshopSchema) Headline() string { return TextValueOf(&s.XMPSchema, PhotoshopHeadline) }

// SetHeadline sets the headline.
func (s *PhotoshopSchema) SetHeadline(text string) error {
	return s.addSimple(PhotoshopHeadline, text)
}

// SetHeadlineProperty sets the headline property outright.
func (s *PhotoshopSchema) SetHeadlineProperty(text *xmptype.TextType) { s.AddProperty(text) }

// HistoryProperty returns the history property, or nil.
func (s *PhotoshopSchema) HistoryProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopHistory)
}

// History returns the history.
func (s *PhotoshopSchema) History() string { return TextValueOf(&s.XMPSchema, PhotoshopHistory) }

// SetHistory sets the history.
func (s *PhotoshopSchema) SetHistory(text string) error {
	return s.addSimple(PhotoshopHistory, text)
}

// SetHistoryProperty sets the history property outright.
func (s *PhotoshopSchema) SetHistoryProperty(text *xmptype.TextType) { s.AddProperty(text) }

// ICCProfileProperty returns the ICC profile property, or nil.
func (s *PhotoshopSchema) ICCProfileProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopICCProfile)
}

// ICCProfile returns the ICC profile.
func (s *PhotoshopSchema) ICCProfile() string {
	return TextValueOf(&s.XMPSchema, PhotoshopICCProfile)
}

// SetICCProfile sets the ICC profile.
func (s *PhotoshopSchema) SetICCProfile(text string) error {
	return s.addSimple(PhotoshopICCProfile, text)
}

// SetICCProfileProperty sets the ICC profile property outright.
func (s *PhotoshopSchema) SetICCProfileProperty(text *xmptype.TextType) { s.AddProperty(text) }

// InstructionsProperty returns the instructions property, or nil.
func (s *PhotoshopSchema) InstructionsProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopInstructions)
}

// Instructions returns the instructions.
func (s *PhotoshopSchema) Instructions() string {
	return TextValueOf(&s.XMPSchema, PhotoshopInstructions)
}

// SetInstructions sets the instructions.
func (s *PhotoshopSchema) SetInstructions(text string) error {
	return s.addSimple(PhotoshopInstructions, text)
}

// SetInstructionsProperty sets the instructions property outright.
func (s *PhotoshopSchema) SetInstructionsProperty(text *xmptype.TextType) { s.AddProperty(text) }

// SourceProperty returns the source property, or nil.
func (s *PhotoshopSchema) SourceProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopSource)
}

// Source returns the source.
func (s *PhotoshopSchema) Source() string { return TextValueOf(&s.XMPSchema, PhotoshopSource) }

// SetSource sets the source.
func (s *PhotoshopSchema) SetSource(text string) error { return s.addSimple(PhotoshopSource, text) }

// SetSourceProperty sets the source property outright.
func (s *PhotoshopSchema) SetSourceProperty(text *xmptype.TextType) { s.AddProperty(text) }

// StateProperty returns the state property, or nil.
func (s *PhotoshopSchema) StateProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopState)
}

// State returns the state.
func (s *PhotoshopSchema) State() string { return TextValueOf(&s.XMPSchema, PhotoshopState) }

// SetState sets the state.
func (s *PhotoshopSchema) SetState(text string) error { return s.addSimple(PhotoshopState, text) }

// SetStateProperty sets the state property outright.
func (s *PhotoshopSchema) SetStateProperty(text *xmptype.TextType) { s.AddProperty(text) }

// SupplementalCategoriesProperty returns the supplemental categories property,
// or nil.
func (s *PhotoshopSchema) SupplementalCategoriesProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopSupplementalCategories)
}

// SupplementalCategories returns the supplemental categories.
func (s *PhotoshopSchema) SupplementalCategories() string {
	return TextValueOf(&s.XMPSchema, PhotoshopSupplementalCategories)
}

// SetSupplementalCategories sets the supplemental categories.
func (s *PhotoshopSchema) SetSupplementalCategories(text string) error {
	return s.addSimple(PhotoshopSupplementalCategories, text)
}

// SetSupplementalCategoriesProperty sets the supplemental categories property
// outright.
func (s *PhotoshopSchema) SetSupplementalCategoriesProperty(text *xmptype.TextType) {
	s.AddProperty(text)
}

// AddTextLayers appends a text layer with the given name and text.
func (s *PhotoshopSchema) AddTextLayers(layerName, layerText string) error {
	if s.seqLayer == nil {
		s.seqLayer = s.CreateArrayProperty(PhotoshopTextLayers, xmptype.Seq)
		s.AddProperty(s.seqLayer)
	}
	layer := xmptype.NewLayerType(s.Metadata())
	if err := layer.SetLayerName(layerName); err != nil {
		return err
	}
	if err := layer.SetLayerText(layerText); err != nil {
		return err
	}
	s.seqLayer.Container().AddProperty(layer)
	return nil
}

// TextLayers returns the text layers, and nil where there are none.
func (s *PhotoshopSchema) TextLayers() ([]*xmptype.LayerType, error) {
	tmp, err := s.UnqualifiedArrayList(PhotoshopTextLayers)
	if err != nil || tmp == nil {
		return nil, err
	}
	layers := make([]*xmptype.LayerType, 0, len(tmp))
	for _, abstractField := range tmp {
		layer, isLayer := abstractField.(*xmptype.LayerType)
		if !isLayer {
			return nil, fmt.Errorf("%w: Layer expected and %s found.",
				xmptype.ErrBadFieldValue, abstractField.TypeName())
		}
		layers = append(layers, layer)
	}
	return layers, nil
}

// TransmissionReferenceProperty returns the transmission reference property, or
// nil.
func (s *PhotoshopSchema) TransmissionReferenceProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PhotoshopTransmissionReference)
}

// TransmissionReference returns the transmission reference.
func (s *PhotoshopSchema) TransmissionReference() string {
	return TextValueOf(&s.XMPSchema, PhotoshopTransmissionReference)
}

// SetTransmissionReference sets the transmission reference.
func (s *PhotoshopSchema) SetTransmissionReference(text string) error {
	return s.addSimple(PhotoshopTransmissionReference, text)
}

// SetTransmissionReferenceProperty sets the transmission reference property
// outright.
func (s *PhotoshopSchema) SetTransmissionReferenceProperty(text *xmptype.TextType) {
	s.AddProperty(text)
}

// UrgencyProperty returns the urgency property, or nil.
func (s *PhotoshopSchema) UrgencyProperty() *xmptype.IntegerType {
	return PropertyAs[*xmptype.IntegerType](&s.XMPSchema, PhotoshopUrgency)
}

// Urgency returns the urgency, the second result being false where there is
// none.
func (s *PhotoshopSchema) Urgency() (int, bool) { return integerValueOf(s.UrgencyProperty()) }

// SetUrgencyString sets the urgency from its string form.
//
// Java overloads setUrgency on String and Integer; Go has no overloading.
func (s *PhotoshopSchema) SetUrgencyString(value string) error {
	return s.addSimple(PhotoshopUrgency, value)
}

// SetUrgency sets the urgency.
func (s *PhotoshopSchema) SetUrgency(value int) error {
	return s.addSimple(PhotoshopUrgency, value)
}

// SetUrgencyProperty sets the urgency property outright.
func (s *PhotoshopSchema) SetUrgencyProperty(text *xmptype.IntegerType) { s.AddProperty(text) }

// integerValueOf is Java's `it == null ? null : it.getValue()`.
func integerValueOf(integer *xmptype.IntegerType) (int, bool) {
	if integer == nil {
		return 0, false
	}
	return integer.IntegerValue(), true
}

// XMPMediaManagementSchema is the xmpMM namespace: where a document came from
// and what has been done to it.
type XMPMediaManagementSchema struct{ XMPSchema }

// The fields XMPMediaManagementSchema declares.
const (
	MMLastURL            = "LastURL"
	MMRenditionOf        = "RenditionOf"
	MMSaveID             = "SaveID"
	MMDerivedFrom        = "DerivedFrom"
	MMDocumentID         = "DocumentID"
	MMManager            = "Manager"
	MMManageTo           = "ManageTo"
	MMManageUI           = "ManageUI"
	MMManagerVariant     = "ManagerVariant"
	MMInstanceID         = "InstanceID"
	MMManagedFrom        = "ManagedFrom"
	MMOriginalDocumentID = "OriginalDocumentID"
	MMRenditionClass     = "RenditionClass"
	MMRenditionParams    = "RenditionParams"
	MMVersionID          = "VersionID"
	MMVersions           = "Versions"
	MMHistory            = "History"
	MMIngredients        = "Ingredients"
)

var mediaManagementInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: "xmpMM",
	Namespace:      MediaManagementNamespace,
}

var mediaManagementProperties = describe(map[string]xmptype.PropertyType{
	MMLastURL:            card(xmptype.URL, xmptype.Simple),
	MMRenditionOf:        card(xmptype.ResourceRef, xmptype.Simple),
	MMSaveID:             card(xmptype.Integer, xmptype.Simple),
	MMDerivedFrom:        card(xmptype.ResourceRef, xmptype.Simple),
	MMDocumentID:         card(xmptype.URI, xmptype.Simple),
	MMManager:            card(xmptype.AgentName, xmptype.Simple),
	MMManageTo:           card(xmptype.URI, xmptype.Simple),
	MMManageUI:           card(xmptype.URI, xmptype.Simple),
	MMManagerVariant:     card(xmptype.Text, xmptype.Simple),
	MMInstanceID:         card(xmptype.URI, xmptype.Simple),
	MMManagedFrom:        card(xmptype.ResourceRef, xmptype.Simple),
	MMOriginalDocumentID: card(xmptype.Text, xmptype.Simple),
	MMRenditionClass:     card(xmptype.RenditionClass, xmptype.Simple),
	MMRenditionParams:    card(xmptype.Text, xmptype.Simple),
	MMVersionID:          card(xmptype.Text, xmptype.Simple),
	MMVersions:           card(xmptype.Version, xmptype.Seq),
	MMHistory:            card(xmptype.ResourceEvent, xmptype.Seq),
	MMIngredients:        card(xmptype.Text, xmptype.Bag),
}, MMLastURL, MMRenditionOf, MMSaveID, MMDerivedFrom, MMDocumentID, MMManager, MMManageTo,
	MMManageUI, MMManagerVariant, MMInstanceID, MMManagedFrom, MMOriginalDocumentID,
	MMRenditionClass, MMRenditionParams, MMVersionID, MMVersions, MMHistory, MMIngredients)

// NewXMPMediaManagementSchema returns the xmpMM schema.
func NewXMPMediaManagementSchema(
	metadata xmptype.MetadataLike) (*XMPMediaManagementSchema, error) {
	return NewXMPMediaManagementSchemaPrefixed(metadata, "")
}

// NewXMPMediaManagementSchemaPrefixed returns the xmpMM schema with the given
// prefix.
func NewXMPMediaManagementSchemaPrefixed(metadata xmptype.MetadataLike,
	ownPrefix string) (*XMPMediaManagementSchema, error) {
	s := &XMPMediaManagementSchema{}
	s.setDescription(mediaManagementProperties, "XMPMediaManagementSchema")
	if err := s.InitSchema(metadata, mediaManagementInfo, "", ownPrefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newXMPMediaManagementSchemaAs is the factory's constructor.
func newXMPMediaManagementSchemaAs(metadata xmptype.MetadataLike,
	prefix string) (Schema, error) {
	return NewXMPMediaManagementSchemaPrefixed(metadata, prefix)
}

// addSimple is Java's `addProperty(instanciateSimple(name, value))`.
func (s *XMPMediaManagementSchema) addSimple(name string, value any) error {
	property, err := s.InstanciateSimple(name, value)
	if err != nil {
		return err
	}
	s.AddProperty(property)
	return nil
}

// SetDerivedFromProperty sets what the document was derived from.
func (s *XMPMediaManagementSchema) SetDerivedFromProperty(tt *xmptype.ResourceRefType) {
	s.AddProperty(tt)
}

// DerivedFromProperty returns what the document was derived from, or nil.
func (s *XMPMediaManagementSchema) DerivedFromProperty() *xmptype.ResourceRefType {
	return PropertyAs[*xmptype.ResourceRefType](&s.XMPSchema, MMDerivedFrom)
}

// SetDocumentID sets the document identifier.
func (s *XMPMediaManagementSchema) SetDocumentID(url string) error {
	return s.addSimple(MMDocumentID, url)
}

// SetDocumentIDProperty sets the document identifier property outright.
func (s *XMPMediaManagementSchema) SetDocumentIDProperty(tt *xmptype.URIValueType) {
	s.AddProperty(tt)
}

// DocumentIDProperty returns the document identifier property, or nil.
func (s *XMPMediaManagementSchema) DocumentIDProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, MMDocumentID)
}

// DocumentID returns the document identifier.
func (s *XMPMediaManagementSchema) DocumentID() string {
	return TextValueOf(&s.XMPSchema, MMDocumentID)
}

// SetLastURL sets the last URL.
func (s *XMPMediaManagementSchema) SetLastURL(url string) error {
	return s.addSimple(MMLastURL, url)
}

// SetLastURLProperty sets the last URL property outright.
func (s *XMPMediaManagementSchema) SetLastURLProperty(tt *xmptype.URLValueType) {
	s.AddProperty(tt)
}

// LastURLProperty returns the last URL property, or nil.
func (s *XMPMediaManagementSchema) LastURLProperty() *xmptype.URLValueType {
	return PropertyAs[*xmptype.URLValueType](&s.XMPSchema, MMLastURL)
}

// LastURL returns the last URL.
func (s *XMPMediaManagementSchema) LastURL() string { return TextValueOf(&s.XMPSchema, MMLastURL) }

// SetSaveID sets the save identifier.
func (s *XMPMediaManagementSchema) SetSaveID(url int) error {
	return s.addSimple(MMSaveID, url)
}

// SetSaveIDProperty sets the save identifier property outright.
func (s *XMPMediaManagementSchema) SetSaveIDProperty(tt *xmptype.IntegerType) { s.AddProperty(tt) }

// SaveIDProperty returns the save identifier property, or nil.
func (s *XMPMediaManagementSchema) SaveIDProperty() *xmptype.IntegerType {
	return PropertyAs[*xmptype.IntegerType](&s.XMPSchema, MMSaveID)
}

// SaveID returns the save identifier, the second result being false where there
// is none.
func (s *XMPMediaManagementSchema) SaveID() (int, bool) {
	return integerValueOf(s.SaveIDProperty())
}

// SetManager sets the manager.
func (s *XMPMediaManagementSchema) SetManager(value string) error {
	return s.addSimple(MMManager, value)
}

// SetManagerProperty sets the manager property outright.
func (s *XMPMediaManagementSchema) SetManagerProperty(tt *xmptype.AgentNameType) {
	s.AddProperty(tt)
}

// ManagerProperty returns the manager property, or nil.
func (s *XMPMediaManagementSchema) ManagerProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, MMManager)
}

// Manager returns the manager.
func (s *XMPMediaManagementSchema) Manager() string { return TextValueOf(&s.XMPSchema, MMManager) }

// SetManageTo sets where the document is managed to.
func (s *XMPMediaManagementSchema) SetManageTo(value string) error {
	return s.addSimple(MMManageTo, value)
}

// SetManageToProperty sets the manage-to property outright.
func (s *XMPMediaManagementSchema) SetManageToProperty(tt *xmptype.URIValueType) {
	s.AddProperty(tt)
}

// ManageToProperty returns the manage-to property, or nil.
func (s *XMPMediaManagementSchema) ManageToProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, MMManageTo)
}

// ManageTo returns where the document is managed to.
func (s *XMPMediaManagementSchema) ManageTo() string {
	return TextValueOf(&s.XMPSchema, MMManageTo)
}

// SetManageUI sets the management interface.
func (s *XMPMediaManagementSchema) SetManageUI(value string) error {
	return s.addSimple(MMManageUI, value)
}

// SetManageUIProperty sets the management interface property outright.
func (s *XMPMediaManagementSchema) SetManageUIProperty(tt *xmptype.URIValueType) {
	s.AddProperty(tt)
}

// ManageUIProperty returns the management interface property, or nil.
func (s *XMPMediaManagementSchema) ManageUIProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, MMManageUI)
}

// ManageUI returns the management interface.
func (s *XMPMediaManagementSchema) ManageUI() string {
	return TextValueOf(&s.XMPSchema, MMManageUI)
}

// SetManagerVariant sets the manager variant.
func (s *XMPMediaManagementSchema) SetManagerVariant(value string) error {
	return s.addSimple(MMManagerVariant, value)
}

// SetManagerVariantProperty sets the manager variant property outright.
func (s *XMPMediaManagementSchema) SetManagerVariantProperty(tt *xmptype.TextType) {
	s.AddProperty(tt)
}

// ManagerVariantProperty returns the manager variant property, or nil.
func (s *XMPMediaManagementSchema) ManagerVariantProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, MMManagerVariant)
}

// ManagerVariant returns the manager variant.
func (s *XMPMediaManagementSchema) ManagerVariant() string {
	return TextValueOf(&s.XMPSchema, MMManagerVariant)
}

// SetInstanceID sets the instance identifier.
func (s *XMPMediaManagementSchema) SetInstanceID(value string) error {
	return s.addSimple(MMInstanceID, value)
}

// SetInstanceIDProperty sets the instance identifier property outright.
func (s *XMPMediaManagementSchema) SetInstanceIDProperty(tt *xmptype.URIValueType) {
	s.AddProperty(tt)
}

// InstanceIDProperty returns the instance identifier property, or nil.
func (s *XMPMediaManagementSchema) InstanceIDProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, MMInstanceID)
}

// InstanceID returns the instance identifier.
func (s *XMPMediaManagementSchema) InstanceID() string {
	return TextValueOf(&s.XMPSchema, MMInstanceID)
}

// SetManagedFromProperty sets what the document is managed from.
func (s *XMPMediaManagementSchema) SetManagedFromProperty(managedFrom *xmptype.ResourceRefType) {
	s.AddProperty(managedFrom)
}

// ManagedFromProperty returns what the document is managed from, or nil.
func (s *XMPMediaManagementSchema) ManagedFromProperty() *xmptype.ResourceRefType {
	return PropertyAs[*xmptype.ResourceRefType](&s.XMPSchema, MMManagedFrom)
}

// SetOriginalDocumentID sets the original document identifier.
func (s *XMPMediaManagementSchema) SetOriginalDocumentID(url string) error {
	return s.addSimple(MMOriginalDocumentID, url)
}

// SetOriginalDocumentIDProperty sets the original document identifier property
// outright.
func (s *XMPMediaManagementSchema) SetOriginalDocumentIDProperty(tt *xmptype.TextType) {
	s.AddProperty(tt)
}

// OriginalDocumentIDProperty returns the original document identifier property,
// or nil.
func (s *XMPMediaManagementSchema) OriginalDocumentIDProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, MMOriginalDocumentID)
}

// OriginalDocumentID returns the original document identifier.
func (s *XMPMediaManagementSchema) OriginalDocumentID() string {
	return TextValueOf(&s.XMPSchema, MMOriginalDocumentID)
}

// SetRenditionClass sets the rendition class.
func (s *XMPMediaManagementSchema) SetRenditionClass(value string) error {
	return s.addSimple(MMRenditionClass, value)
}

// SetRenditionClassProperty sets the rendition class property outright.
func (s *XMPMediaManagementSchema) SetRenditionClassProperty(tt *xmptype.RenditionClassType) {
	s.AddProperty(tt)
}

// RenditionClassProperty returns the rendition class property, or nil.
func (s *XMPMediaManagementSchema) RenditionClassProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, MMRenditionClass)
}

// RenditionClass returns the rendition class.
func (s *XMPMediaManagementSchema) RenditionClass() string {
	return TextValueOf(&s.XMPSchema, MMRenditionClass)
}

// SetRenditionParams sets the rendition parameters.
func (s *XMPMediaManagementSchema) SetRenditionParams(url string) error {
	return s.addSimple(MMRenditionParams, url)
}

// SetRenditionParamsProperty sets the rendition parameters property outright.
func (s *XMPMediaManagementSchema) SetRenditionParamsProperty(tt *xmptype.TextType) {
	s.AddProperty(tt)
}

// RenditionParamsProperty returns the rendition parameters property, or nil.
func (s *XMPMediaManagementSchema) RenditionParamsProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, MMRenditionParams)
}

// RenditionParams returns the rendition parameters.
func (s *XMPMediaManagementSchema) RenditionParams() string {
	return TextValueOf(&s.XMPSchema, MMRenditionParams)
}

// SetVersionID sets the version identifier.
func (s *XMPMediaManagementSchema) SetVersionID(value string) error {
	return s.addSimple(MMVersionID, value)
}

// SetVersionIDProperty sets the version identifier property outright.
func (s *XMPMediaManagementSchema) SetVersionIDProperty(tt *xmptype.TextType) {
	s.AddProperty(tt)
}

// VersionIDProperty returns the version identifier property, or nil.
func (s *XMPMediaManagementSchema) VersionIDProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, MMVersionID)
}

// VersionID returns the version identifier.
func (s *XMPMediaManagementSchema) VersionID() string {
	return TextValueOf(&s.XMPSchema, MMVersionID)
}

// AddVersions adds a version.
//
// The field is declared as a Seq of Version, and Java adds to it with
// addQualifiedBagValue, which makes a Bag of Text; the array is a sequence
// here, as the neighbouring AddHistory writes for its own Seq field. The value
// is still text rather than the declared VersionType: that is a structured
// type this method has no parameter for, and giving it one is a port task
// rather than a fix. See migration/JAVA-BUGS.md 55.
func (s *XMPMediaManagementSchema) AddVersions(value string) error {
	return s.AddUnqualifiedSequenceValue(MMVersions, value)
}

// VersionsProperty returns the versions array, or nil.
func (s *XMPMediaManagementSchema) VersionsProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, MMVersions)
}

// Versions returns the versions.
func (s *XMPMediaManagementSchema) Versions() []string {
	return s.UnqualifiedBagValueList(MMVersions)
}

// AddHistory appends a history entry.
func (s *XMPMediaManagementSchema) AddHistory(history string) error {
	return s.AddUnqualifiedSequenceValue(MMHistory, history)
}

// HistoryProperty returns the history array, or nil.
func (s *XMPMediaManagementSchema) HistoryProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, MMHistory)
}

// AddIngredients adds an ingredient.
func (s *XMPMediaManagementSchema) AddIngredients(ingredients string) error {
	return s.AddQualifiedBagValue(MMIngredients, ingredients)
}

// IngredientsProperty returns the ingredients array, or nil.
func (s *XMPMediaManagementSchema) IngredientsProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, MMIngredients)
}

// Ingredients returns the ingredients.
func (s *XMPMediaManagementSchema) Ingredients() []string {
	return s.UnqualifiedBagValueList(MMIngredients)
}
