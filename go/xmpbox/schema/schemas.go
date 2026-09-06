package schema

// Seven of the twelve schemas: the two PDF/A ones, Adobe PDF, Dublin Core,
// rights management, page text and the job ticket.
//
// Port of PDFAExtensionSchema, PDFAIdentificationSchema, AdobePDFSchema,
// DublinCoreSchema, XMPRightsManagementSchema, XMPPageTextSchema and
// XMPBasicJobTicketSchema. Java gives each a file; each is a @StructuredType
// annotation, a table of @PropertyType field names and a run of accessors over
// XMPSchema, so the port groups them the way it groups the structured types.

import (
	"fmt"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// describe builds a PropertiesDescription from the fields a schema declares, in
// the order they are named, which is what Java reads off the class.
func describe(types map[string]xmptype.PropertyType,
	order ...string) *xmptype.PropertiesDescription {
	description := xmptype.NewPropertiesDescription()
	for _, name := range order {
		description.AddNewProperty(name, types[name])
	}
	return description
}

// simple and card are the two forms of the @PropertyType annotation.
func simple(t xmptype.Types) xmptype.PropertyType { return xmptype.NewPropertyType(t) }

func card(t xmptype.Types, c xmptype.Cardinality) xmptype.PropertyType {
	return xmptype.NewPropertyTypeCard(t, c)
}

// AdobePDFSchema is the pdf namespace: the keywords, the version and the
// producer.
type AdobePDFSchema struct{ XMPSchema }

// The fields AdobePDFSchema declares.
const (
	PDFKeywords   = "Keywords"
	PDFPDFVersion = "PDFVersion"
	PDFProducer   = "Producer"
)

var adobePDFInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: "pdf",
	Namespace:      AdobePDFNamespace,
}

var adobePDFProperties = describe(map[string]xmptype.PropertyType{
	PDFKeywords:   card(xmptype.Text, xmptype.Simple),
	PDFPDFVersion: card(xmptype.Text, xmptype.Simple),
	PDFProducer:   card(xmptype.Text, xmptype.Simple),
}, PDFKeywords, PDFPDFVersion, PDFProducer)

// NewAdobePDFSchema returns the pdf schema.
func NewAdobePDFSchema(metadata xmptype.MetadataLike) (*AdobePDFSchema, error) {
	return NewAdobePDFSchemaPrefixed(metadata, "")
}

// NewAdobePDFSchemaPrefixed returns the pdf schema with the given prefix.
func NewAdobePDFSchemaPrefixed(metadata xmptype.MetadataLike,
	ownPrefix string) (*AdobePDFSchema, error) {
	s := &AdobePDFSchema{}
	s.setDescription(adobePDFProperties, "AdobePDFSchema")
	if err := s.InitSchema(metadata, adobePDFInfo, "", ownPrefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newAdobePDFSchemaAs is the factory's constructor.
func newAdobePDFSchemaAs(metadata xmptype.MetadataLike, prefix string) (Schema, error) {
	return NewAdobePDFSchemaPrefixed(metadata, prefix)
}

// SetKeywords sets the keywords.
func (s *AdobePDFSchema) SetKeywords(value string) error {
	return SetTextValue(&s.XMPSchema, PDFKeywords, value)
}

// SetKeywordsProperty sets the keywords property outright.
func (s *AdobePDFSchema) SetKeywordsProperty(keywords *xmptype.TextType) {
	s.AddProperty(keywords)
}

// SetPDFVersion sets the PDF version.
func (s *AdobePDFSchema) SetPDFVersion(value string) error {
	return SetTextValue(&s.XMPSchema, PDFPDFVersion, value)
}

// SetPDFVersionProperty sets the version property outright.
func (s *AdobePDFSchema) SetPDFVersionProperty(version *xmptype.TextType) {
	s.AddProperty(version)
}

// SetProducer sets the producer.
func (s *AdobePDFSchema) SetProducer(value string) error {
	return SetTextValue(&s.XMPSchema, PDFProducer, value)
}

// SetProducerProperty sets the producer property outright.
func (s *AdobePDFSchema) SetProducerProperty(producer *xmptype.TextType) {
	s.AddProperty(producer)
}

// KeywordsProperty returns the keywords property, or nil.
func (s *AdobePDFSchema) KeywordsProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PDFKeywords)
}

// Keywords returns the keywords.
func (s *AdobePDFSchema) Keywords() string { return TextValueOf(&s.XMPSchema, PDFKeywords) }

// PDFVersionProperty returns the version property, or nil.
func (s *AdobePDFSchema) PDFVersionProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PDFPDFVersion)
}

// PDFVersion returns the PDF version.
func (s *AdobePDFSchema) PDFVersion() string { return TextValueOf(&s.XMPSchema, PDFPDFVersion) }

// ProducerProperty returns the producer property, or nil.
func (s *AdobePDFSchema) ProducerProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PDFProducer)
}

// Producer returns the producer.
func (s *AdobePDFSchema) Producer() string { return TextValueOf(&s.XMPSchema, PDFProducer) }

// DublinCoreSchema is the dc namespace: the title, the creators and the rest of
// the Dublin Core.
type DublinCoreSchema struct{ XMPSchema }

// The fields DublinCoreSchema declares.
const (
	DCContributor = "contributor"
	DCCoverage    = "coverage"
	DCCreator     = "creator"
	DCDate        = "date"
	DCDescription = "description"
	DCFormat      = "format"
	DCIdentifier  = "identifier"
	DCLanguage    = "language"
	DCPublisher   = "publisher"
	DCRelation    = "relation"
	DCRights      = "rights"
	DCSource      = "source"
	DCSubject     = "subject"
	DCTitle       = "title"
	DCType        = "type"
)

var dublinCoreInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: "dc",
	Namespace:      DublinCoreNamespace,
}

var dublinCoreProperties = describe(map[string]xmptype.PropertyType{
	DCContributor: card(xmptype.Text, xmptype.Bag),
	DCCoverage:    card(xmptype.Text, xmptype.Simple),
	DCCreator:     card(xmptype.Text, xmptype.Seq),
	DCDate:        card(xmptype.Date, xmptype.Seq),
	DCDescription: card(xmptype.LangAlt, xmptype.Simple),
	DCFormat:      card(xmptype.MIMEType, xmptype.Simple),
	DCIdentifier:  card(xmptype.Text, xmptype.Simple),
	DCLanguage:    card(xmptype.Text, xmptype.Bag),
	DCPublisher:   card(xmptype.Text, xmptype.Bag),
	DCRelation:    card(xmptype.Text, xmptype.Bag),
	DCRights:      card(xmptype.LangAlt, xmptype.Simple),
	DCSource:      card(xmptype.Text, xmptype.Simple),
	DCSubject:     card(xmptype.Text, xmptype.Bag),
	DCTitle:       card(xmptype.LangAlt, xmptype.Simple),
	DCType:        card(xmptype.Text, xmptype.Bag),
}, DCContributor, DCCoverage, DCCreator, DCDate, DCDescription, DCFormat, DCIdentifier,
	DCLanguage, DCPublisher, DCRelation, DCRights, DCSource, DCSubject, DCTitle, DCType)

// NewDublinCoreSchema returns the dc schema.
func NewDublinCoreSchema(metadata xmptype.MetadataLike) (*DublinCoreSchema, error) {
	return NewDublinCoreSchemaPrefixed(metadata, "")
}

// NewDublinCoreSchemaPrefixed returns the dc schema with the given prefix.
func NewDublinCoreSchemaPrefixed(metadata xmptype.MetadataLike,
	ownPrefix string) (*DublinCoreSchema, error) {
	s := &DublinCoreSchema{}
	s.setDescription(dublinCoreProperties, "DublinCoreSchema")
	if err := s.InitSchema(metadata, dublinCoreInfo, "", ownPrefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newDublinCoreSchemaAs is the factory's constructor.
func newDublinCoreSchemaAs(metadata xmptype.MetadataLike, prefix string) (Schema, error) {
	return NewDublinCoreSchemaPrefixed(metadata, prefix)
}

// AddContributor adds a contributor.
func (s *DublinCoreSchema) AddContributor(properName string) error {
	return s.AddQualifiedBagValue(DCContributor, properName)
}

// RemoveContributor removes a contributor.
func (s *DublinCoreSchema) RemoveContributor(properName string) {
	s.RemoveUnqualifiedBagValue(DCContributor, properName)
}

// SetCoverage sets the coverage.
func (s *DublinCoreSchema) SetCoverage(text string) error {
	return SetTextValue(&s.XMPSchema, DCCoverage, text)
}

// SetCoverageProperty sets the coverage property outright.
func (s *DublinCoreSchema) SetCoverageProperty(text *xmptype.TextType) { s.AddProperty(text) }

// AddCreator appends a creator.
func (s *DublinCoreSchema) AddCreator(properName string) error {
	return s.AddUnqualifiedSequenceValue(DCCreator, properName)
}

// RemoveCreator removes a creator.
func (s *DublinCoreSchema) RemoveCreator(name string) {
	s.RemoveUnqualifiedSequenceValue(DCCreator, name)
}

// AddDate appends a date.
func (s *DublinCoreSchema) AddDate(date time.Time) error {
	return s.AddUnqualifiedSequenceDateValue(DCDate, date)
}

// RemoveDate removes a date.
func (s *DublinCoreSchema) RemoveDate(date time.Time) {
	s.RemoveUnqualifiedSequenceDateValue(DCDate, date)
}

// AddDescription sets the description in the given language.
func (s *DublinCoreSchema) AddDescription(lang, value string) error {
	return s.SetUnqualifiedLanguagePropertyValue(DCDescription, lang, value)
}

// SetDescription sets the description in the default language.
func (s *DublinCoreSchema) SetDescription(value string) error {
	return s.AddDescription("", value)
}

// SetFormat sets the MIME type.
func (s *DublinCoreSchema) SetFormat(mimeType string) error {
	return SetTextValue(&s.XMPSchema, DCFormat, mimeType)
}

// SetIdentifier sets the identifier.
func (s *DublinCoreSchema) SetIdentifier(text string) error {
	return SetTextValue(&s.XMPSchema, DCIdentifier, text)
}

// SetIdentifierProperty sets the identifier property outright.
func (s *DublinCoreSchema) SetIdentifierProperty(text *xmptype.TextType) { s.AddProperty(text) }

// AddLanguage adds a language.
func (s *DublinCoreSchema) AddLanguage(locale string) error {
	return s.AddQualifiedBagValue(DCLanguage, locale)
}

// RemoveLanguage removes a language.
func (s *DublinCoreSchema) RemoveLanguage(locale string) {
	s.RemoveUnqualifiedBagValue(DCLanguage, locale)
}

// AddPublisher adds a publisher.
func (s *DublinCoreSchema) AddPublisher(properName string) error {
	return s.AddQualifiedBagValue(DCPublisher, properName)
}

// RemovePublisher removes a publisher.
func (s *DublinCoreSchema) RemovePublisher(name string) {
	s.RemoveUnqualifiedBagValue(DCPublisher, name)
}

// AddRelation adds a relation.
func (s *DublinCoreSchema) AddRelation(text string) error {
	return s.AddQualifiedBagValue(DCRelation, text)
}

// RemoveRelation removes a relation.
func (s *DublinCoreSchema) RemoveRelation(text string) {
	s.RemoveUnqualifiedBagValue(DCRelation, text)
}

// AddRights sets the rights in the given language.
func (s *DublinCoreSchema) AddRights(lang, value string) error {
	return s.SetUnqualifiedLanguagePropertyValue(DCRights, lang, value)
}

// SetSource sets the source.
func (s *DublinCoreSchema) SetSource(text string) error {
	return SetTextValue(&s.XMPSchema, DCSource, text)
}

// SetSourceProperty sets the source property outright.
func (s *DublinCoreSchema) SetSourceProperty(text *xmptype.TextType) { s.AddProperty(text) }

// SetFormatProperty sets the format property outright.
func (s *DublinCoreSchema) SetFormatProperty(text *xmptype.MIMEValueType) { s.AddProperty(text) }

// AddSubject adds a subject.
func (s *DublinCoreSchema) AddSubject(text string) error {
	return s.AddQualifiedBagValue(DCSubject, text)
}

// RemoveSubject removes a subject.
func (s *DublinCoreSchema) RemoveSubject(text string) {
	s.RemoveUnqualifiedBagValue(DCSubject, text)
}

// SetTitleOfLanguage sets the title in the given language.
func (s *DublinCoreSchema) SetTitleOfLanguage(lang, value string) error {
	return s.SetUnqualifiedLanguagePropertyValue(DCTitle, lang, value)
}

// SetTitle sets the title in the default language.
func (s *DublinCoreSchema) SetTitle(value string) error {
	return s.SetTitleOfLanguage("", value)
}

// AddTitle sets the title in the given language.
func (s *DublinCoreSchema) AddTitle(lang, value string) error {
	return s.SetTitleOfLanguage(lang, value)
}

// AddType adds a type.
func (s *DublinCoreSchema) AddType(t string) error { return s.AddQualifiedBagValue(DCType, t) }

// ContributorsProperty returns the contributors array, or nil.
func (s *DublinCoreSchema) ContributorsProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, DCContributor)
}

// Contributors returns the contributors.
func (s *DublinCoreSchema) Contributors() []string {
	return s.UnqualifiedBagValueList(DCContributor)
}

// CoverageProperty returns the coverage property, or nil.
func (s *DublinCoreSchema) CoverageProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, DCCoverage)
}

// Coverage returns the coverage.
func (s *DublinCoreSchema) Coverage() string { return TextValueOf(&s.XMPSchema, DCCoverage) }

// CreatorsProperty returns the creators array, or nil.
func (s *DublinCoreSchema) CreatorsProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, DCCreator)
}

// Creators returns the creators.
func (s *DublinCoreSchema) Creators() []string {
	return s.UnqualifiedSequenceValueList(DCCreator)
}

// DatesProperty returns the dates array, or nil.
func (s *DublinCoreSchema) DatesProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, DCDate)
}

// Dates returns the dates.
func (s *DublinCoreSchema) Dates() []time.Time {
	return s.UnqualifiedSequenceDateValueList(DCDate)
}

// DescriptionProperty returns the description array, or nil.
func (s *DublinCoreSchema) DescriptionProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, DCDescription)
}

// DescriptionLanguages returns the languages the description is written in.
func (s *DublinCoreSchema) DescriptionLanguages() ([]string, error) {
	return s.UnqualifiedLanguagePropertyLanguagesValue(DCDescription)
}

// DescriptionOfLanguage returns the description in the given language.
func (s *DublinCoreSchema) DescriptionOfLanguage(lang string) (string, error) {
	return s.UnqualifiedLanguagePropertyValue(DCDescription, lang)
}

// Description returns the description in the default language.
func (s *DublinCoreSchema) Description() (string, error) {
	return s.DescriptionOfLanguage("")
}

// FormatProperty returns the format property, or nil.
func (s *DublinCoreSchema) FormatProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, DCFormat)
}

// Format returns the MIME type.
func (s *DublinCoreSchema) Format() string { return TextValueOf(&s.XMPSchema, DCFormat) }

// IdentifierProperty returns the identifier property, or nil.
func (s *DublinCoreSchema) IdentifierProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, DCIdentifier)
}

// Identifier returns the identifier.
func (s *DublinCoreSchema) Identifier() string { return TextValueOf(&s.XMPSchema, DCIdentifier) }

// LanguagesProperty returns the languages array, or nil.
func (s *DublinCoreSchema) LanguagesProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, DCLanguage)
}

// Languages returns the languages.
func (s *DublinCoreSchema) Languages() []string { return s.UnqualifiedBagValueList(DCLanguage) }

// PublishersProperty returns the publishers array, or nil.
func (s *DublinCoreSchema) PublishersProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, DCPublisher)
}

// Publishers returns the publishers.
func (s *DublinCoreSchema) Publishers() []string { return s.UnqualifiedBagValueList(DCPublisher) }

// RelationsProperty returns the relations array, or nil.
func (s *DublinCoreSchema) RelationsProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, DCRelation)
}

// Relations returns the relations.
func (s *DublinCoreSchema) Relations() []string { return s.UnqualifiedBagValueList(DCRelation) }

// RightsProperty returns the rights array, or nil.
func (s *DublinCoreSchema) RightsProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, DCRights)
}

// RightsLanguages returns the languages the rights are written in.
func (s *DublinCoreSchema) RightsLanguages() ([]string, error) {
	return s.UnqualifiedLanguagePropertyLanguagesValue(DCRights)
}

// RightsOfLanguage returns the rights in the given language.
func (s *DublinCoreSchema) RightsOfLanguage(lang string) (string, error) {
	return s.UnqualifiedLanguagePropertyValue(DCRights, lang)
}

// Rights returns the rights in the default language.
func (s *DublinCoreSchema) Rights() (string, error) { return s.RightsOfLanguage("") }

// SourceProperty returns the source property, or nil.
func (s *DublinCoreSchema) SourceProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, DCSource)
}

// Source returns the source.
func (s *DublinCoreSchema) Source() string { return TextValueOf(&s.XMPSchema, DCSource) }

// SubjectsProperty returns the subjects array, or nil.
func (s *DublinCoreSchema) SubjectsProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, DCSubject)
}

// Subjects returns the subjects.
func (s *DublinCoreSchema) Subjects() []string { return s.UnqualifiedBagValueList(DCSubject) }

// TitleProperty returns the title array, or nil.
func (s *DublinCoreSchema) TitleProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, DCTitle)
}

// TitleLanguages returns the languages the title is written in.
func (s *DublinCoreSchema) TitleLanguages() ([]string, error) {
	return s.UnqualifiedLanguagePropertyLanguagesValue(DCTitle)
}

// TitleOfLanguage returns the title in the given language.
func (s *DublinCoreSchema) TitleOfLanguage(lang string) (string, error) {
	return s.UnqualifiedLanguagePropertyValue(DCTitle, lang)
}

// Title returns the title in the default language.
func (s *DublinCoreSchema) Title() (string, error) { return s.TitleOfLanguage("") }

// TypesProperty returns the types array, or nil.
func (s *DublinCoreSchema) TypesProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, DCType)
}

// Types returns the types.
func (s *DublinCoreSchema) Types() []string { return s.UnqualifiedBagValueList(DCType) }

// RemoveType removes a type.
func (s *DublinCoreSchema) RemoveType(t string) { s.RemoveUnqualifiedBagValue(DCType, t) }

// PDFAExtensionSchema is the pdfaExtension namespace: the schemas a PDF/A file
// declares that are not among the twelve.
type PDFAExtensionSchema struct{ XMPSchema }

// PDFAExtensionSchemas is the one field PDFAExtensionSchema declares.
const PDFAExtensionSchemas = "schemas"

var pdfaExtensionInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: PDFAExtensionPreferedPrefix,
	Namespace:      PDFAExtensionNamespace,
}

var pdfaExtensionProperties = describe(map[string]xmptype.PropertyType{
	PDFAExtensionSchemas: card(xmptype.PDFASchema, xmptype.Bag),
}, PDFAExtensionSchemas)

// NewPDFAExtensionSchema returns the pdfaExtension schema.
func NewPDFAExtensionSchema(metadata xmptype.MetadataLike) (*PDFAExtensionSchema, error) {
	return NewPDFAExtensionSchemaPrefixed(metadata, "")
}

// NewPDFAExtensionSchemaPrefixed returns the pdfaExtension schema with the
// given prefix.
func NewPDFAExtensionSchemaPrefixed(metadata xmptype.MetadataLike,
	prefix string) (*PDFAExtensionSchema, error) {
	s := &PDFAExtensionSchema{}
	s.setDescription(pdfaExtensionProperties, "PDFAExtensionSchema")
	if err := s.InitSchema(metadata, pdfaExtensionInfo, "", prefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newPDFAExtensionSchemaAs is the factory's constructor.
func newPDFAExtensionSchemaAs(metadata xmptype.MetadataLike, prefix string) (Schema, error) {
	return NewPDFAExtensionSchemaPrefixed(metadata, prefix)
}

// SchemasProperty returns the array of declared schemas, or nil.
func (s *PDFAExtensionSchema) SchemasProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, PDFAExtensionSchemas)
}

// validConformanceValues are the five conformance levels PDF/A allows.
//
// Port of PDFAIdentificationSchema.VALID_VALUES.
var validConformanceValues = map[string]bool{"A": true, "B": true, "U": true, "e": true, "f": true}

// PDFAIdentificationSchema is the pdfaid namespace: which part and conformance
// level of PDF/A a file claims.
type PDFAIdentificationSchema struct{ XMPSchema }

// The fields PDFAIdentificationSchema declares.
const (
	PDFAIDPart        = "part"
	PDFAIDAmd         = "amd"
	PDFAIDConformance = "conformance"

	// PDFAIDRev is PDFBOX-6088,
	// https://pdfa.org/future-proofing-xmp-identification-schema/
	PDFAIDRev = "rev"
)

var pdfaIdentificationInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: "pdfaid",
	Namespace:      PDFAIdentificationNamespace,
}

var pdfaIdentificationProperties = describe(map[string]xmptype.PropertyType{
	PDFAIDPart:        card(xmptype.Integer, xmptype.Simple),
	PDFAIDAmd:         card(xmptype.Text, xmptype.Simple),
	PDFAIDConformance: card(xmptype.Text, xmptype.Simple),
	PDFAIDRev:         card(xmptype.Integer, xmptype.Simple),
}, PDFAIDPart, PDFAIDAmd, PDFAIDConformance, PDFAIDRev)

// NewPDFAIdentificationSchema returns the pdfaid schema.
func NewPDFAIdentificationSchema(metadata xmptype.MetadataLike) (*PDFAIdentificationSchema, error) {
	return NewPDFAIdentificationSchemaPrefixed(metadata, "")
}

// NewPDFAIdentificationSchemaPrefixed returns the pdfaid schema with the given
// prefix.
func NewPDFAIdentificationSchemaPrefixed(metadata xmptype.MetadataLike,
	prefix string) (*PDFAIdentificationSchema, error) {
	s := &PDFAIdentificationSchema{}
	s.setDescription(pdfaIdentificationProperties, "PDFAIdentificationSchema")
	if err := s.InitSchema(metadata, pdfaIdentificationInfo, "", prefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newPDFAIdentificationSchemaAs is the factory's constructor.
func newPDFAIdentificationSchemaAs(metadata xmptype.MetadataLike, prefix string) (Schema, error) {
	return NewPDFAIdentificationSchemaPrefixed(metadata, prefix)
}

// SetPartValueWithString sets the part from its string form.
func (s *PDFAIdentificationSchema) SetPartValueWithString(value string) error {
	return s.addSimple(PDFAIDPart, value)
}

// SetPartValueWithInt sets the part.
func (s *PDFAIdentificationSchema) SetPartValueWithInt(value int) error {
	return s.addSimple(PDFAIDPart, value)
}

// SetPart sets the part.
func (s *PDFAIdentificationSchema) SetPart(value int) error {
	return s.SetPartValueWithInt(value)
}

// SetPartProperty sets the part property outright.
func (s *PDFAIdentificationSchema) SetPartProperty(part *xmptype.IntegerType) {
	s.AddProperty(part)
}

// addSimple is Java's `addProperty(instanciateSimple(name, value))`.
func (s *PDFAIdentificationSchema) addSimple(name string, value any) error {
	property, err := s.InstanciateSimple(name, value)
	if err != nil {
		return err
	}
	s.AddProperty(property)
	return nil
}

// SetAmd sets the amendment.
func (s *PDFAIdentificationSchema) SetAmd(value string) error {
	return SetTextValue(&s.XMPSchema, PDFAIDAmd, value)
}

// SetAmdProperty sets the amendment property outright.
func (s *PDFAIdentificationSchema) SetAmdProperty(amd *xmptype.TextType) { s.AddProperty(amd) }

// SetConformance sets the conformance level, which must be one of the five.
func (s *PDFAIdentificationSchema) SetConformance(value string) error {
	conf, err := s.CreateTextType(PDFAIDConformance, value)
	if err != nil {
		return err
	}
	return s.SetConformanceProperty(conf)
}

// SetConformanceProperty sets the conformance property, which must hold one of
// the five levels.
func (s *PDFAIdentificationSchema) SetConformanceProperty(conf *xmptype.TextType) error {
	value := conf.StringValue()
	if validConformanceValues[value] {
		s.AddProperty(conf)
		return nil
	}
	return fmt.Errorf(
		"%w: The value '%s' isn't a valid PDF/A conformance level (must be A, B, U, e or f)",
		xmptype.ErrBadFieldValue, value)
}

// Part returns the PDF/A part, the second result being false where there is
// none.
func (s *PDFAIdentificationSchema) Part() (int, bool) {
	tmp := s.PartProperty()
	if tmp == nil {
		return 0, false
	}
	return tmp.IntegerValue(), true
}

// PartProperty returns the part property, or nil.
func (s *PDFAIdentificationSchema) PartProperty() *xmptype.IntegerType {
	return PropertyAs[*xmptype.IntegerType](&s.XMPSchema, PDFAIDPart)
}

// Amendment returns the amendment, reading only the property.
func (s *PDFAIdentificationSchema) Amendment() string {
	return TextValueOf(&s.XMPSchema, PDFAIDAmd)
}

// AmdProperty returns the amendment property, or nil.
func (s *PDFAIdentificationSchema) AmdProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PDFAIDAmd)
}

// Amd returns the amendment, falling back to an attribute of that name where
// there is no property.
func (s *PDFAIdentificationSchema) Amd() string {
	return s.propertyOrAttribute(PDFAIDAmd, s.AmdProperty())
}

// ConformanceProperty returns the conformance property, or nil.
func (s *PDFAIdentificationSchema) ConformanceProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, PDFAIDConformance)
}

// Conformance returns the conformance level, falling back to an attribute of
// that name where there is no property.
func (s *PDFAIdentificationSchema) Conformance() string {
	return s.propertyOrAttribute(PDFAIDConformance, s.ConformanceProperty())
}

// propertyOrAttribute is the shape Amd and Conformance share: the property
// where there is one, and otherwise the attribute of the same name.
func (s *PDFAIdentificationSchema) propertyOrAttribute(name string,
	property *xmptype.TextType) string {
	if property != nil {
		return property.StringValue()
	}
	for _, attribute := range s.AllAttributes() {
		if attribute.Name() == name {
			return attribute.Value()
		}
	}
	return ""
}

// SetRevValueWithString sets the revision from its string form.
func (s *PDFAIdentificationSchema) SetRevValueWithString(value string) error {
	return s.addSimple(PDFAIDRev, value)
}

// SetRevValueWithInt sets the revision.
func (s *PDFAIdentificationSchema) SetRevValueWithInt(value int) error {
	return s.addSimple(PDFAIDRev, value)
}

// SetRev sets the revision.
func (s *PDFAIdentificationSchema) SetRev(value int) error { return s.SetRevValueWithInt(value) }

// SetRevProperty sets the revision property outright.
func (s *PDFAIdentificationSchema) SetRevProperty(rev *xmptype.IntegerType) { s.AddProperty(rev) }

// RevProperty returns the revision property, or nil.
func (s *PDFAIdentificationSchema) RevProperty() *xmptype.IntegerType {
	return PropertyAs[*xmptype.IntegerType](&s.XMPSchema, PDFAIDRev)
}

// Rev returns the revision, the second result being false where there is none.
func (s *PDFAIdentificationSchema) Rev() (int, bool) {
	tmp := s.RevProperty()
	if tmp == nil {
		return 0, false
	}
	return tmp.IntegerValue(), true
}

// XMPRightsManagementSchema is the xmpRights namespace: who owns a document and
// how it may be used.
type XMPRightsManagementSchema struct{ XMPSchema }

// The fields XMPRightsManagementSchema declares.
const (
	RightsCertificate  = "Certificate"
	RightsMarked       = "Marked"
	RightsOwner        = "Owner"
	RightsUsageTerms   = "UsageTerms"
	RightsWebStatement = "WebStatement"
)

var rightsManagementInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: "xmpRights",
	Namespace:      RightsManagementNamespace,
}

var rightsManagementProperties = describe(map[string]xmptype.PropertyType{
	RightsCertificate:  card(xmptype.URL, xmptype.Simple),
	RightsMarked:       card(xmptype.Boolean, xmptype.Simple),
	RightsOwner:        card(xmptype.ProperName, xmptype.Bag),
	RightsUsageTerms:   card(xmptype.LangAlt, xmptype.Simple),
	RightsWebStatement: card(xmptype.URL, xmptype.Simple),
}, RightsCertificate, RightsMarked, RightsOwner, RightsUsageTerms, RightsWebStatement)

// NewXMPRightsManagementSchema returns the xmpRights schema.
func NewXMPRightsManagementSchema(
	metadata xmptype.MetadataLike) (*XMPRightsManagementSchema, error) {
	return NewXMPRightsManagementSchemaPrefixed(metadata, "")
}

// NewXMPRightsManagementSchemaPrefixed returns the xmpRights schema with the
// given prefix.
func NewXMPRightsManagementSchemaPrefixed(metadata xmptype.MetadataLike,
	ownPrefix string) (*XMPRightsManagementSchema, error) {
	s := &XMPRightsManagementSchema{}
	s.setDescription(rightsManagementProperties, "XMPRightsManagementSchema")
	if err := s.InitSchema(metadata, rightsManagementInfo, "", ownPrefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newXMPRightsManagementSchemaAs is the factory's constructor.
func newXMPRightsManagementSchemaAs(metadata xmptype.MetadataLike,
	prefix string) (Schema, error) {
	return NewXMPRightsManagementSchemaPrefixed(metadata, prefix)
}

// AddOwner adds an owner.
func (s *XMPRightsManagementSchema) AddOwner(value string) error {
	return s.AddQualifiedBagValue(RightsOwner, value)
}

// RemoveOwner removes an owner.
func (s *XMPRightsManagementSchema) RemoveOwner(value string) {
	s.RemoveUnqualifiedBagValue(RightsOwner, value)
}

// OwnersProperty returns the owners array, or nil.
func (s *XMPRightsManagementSchema) OwnersProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, RightsOwner)
}

// Owners returns the owners.
func (s *XMPRightsManagementSchema) Owners() []string {
	return s.UnqualifiedBagValueList(RightsOwner)
}

// SetMarked says whether the document is marked as rights-managed.
func (s *XMPRightsManagementSchema) SetMarked(marked bool) error {
	value := xmptype.False
	if marked {
		value = xmptype.True
	}
	property, err := s.InstanciateSimple(RightsMarked, value)
	if err != nil {
		return err
	}
	boolean, isBoolean := property.(*xmptype.BooleanType)
	if !isBoolean {
		return fmt.Errorf("%w: %s is not a boolean property", xmptype.ErrBadFieldValue,
			RightsMarked)
	}
	s.SetMarkedProperty(boolean)
	return nil
}

// SetMarkedProperty sets the marked property outright.
func (s *XMPRightsManagementSchema) SetMarkedProperty(marked *xmptype.BooleanType) {
	s.AddProperty(marked)
}

// MarkedProperty returns the marked property, or nil.
func (s *XMPRightsManagementSchema) MarkedProperty() *xmptype.BooleanType {
	return PropertyAs[*xmptype.BooleanType](&s.XMPSchema, RightsMarked)
}

// Marked returns whether the document is marked, the second result being false
// where it does not say.
func (s *XMPRightsManagementSchema) Marked() (bool, bool) {
	bt := s.MarkedProperty()
	if bt == nil {
		return false, false
	}
	return bt.BooleanValue(), true
}

// AddUsageTerms sets the usage terms in the given language.
func (s *XMPRightsManagementSchema) AddUsageTerms(lang, value string) error {
	return s.SetUnqualifiedLanguagePropertyValue(RightsUsageTerms, lang, value)
}

// SetUsageTerms sets the usage terms in the default language.
func (s *XMPRightsManagementSchema) SetUsageTerms(terms string) error {
	return s.AddUsageTerms("", terms)
}

// UsageTermsProperty returns the usage terms array, or nil.
func (s *XMPRightsManagementSchema) UsageTermsProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, RightsUsageTerms)
}

// UsageTermsLanguages returns the languages the usage terms are written in.
func (s *XMPRightsManagementSchema) UsageTermsLanguages() ([]string, error) {
	return s.UnqualifiedLanguagePropertyLanguagesValue(RightsUsageTerms)
}

// UsageTermsOfLanguage returns the usage terms in the given language.
func (s *XMPRightsManagementSchema) UsageTermsOfLanguage(lang string) (string, error) {
	return s.UnqualifiedLanguagePropertyValue(RightsUsageTerms, lang)
}

// UsageTerms returns the usage terms in the default language.
func (s *XMPRightsManagementSchema) UsageTerms() (string, error) {
	return s.UsageTermsOfLanguage("")
}

// WebStatementProperty returns the web statement property, or nil.
func (s *XMPRightsManagementSchema) WebStatementProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, RightsWebStatement)
}

// WebStatement returns the URL of the web statement.
func (s *XMPRightsManagementSchema) WebStatement() string {
	return TextValueOf(&s.XMPSchema, RightsWebStatement)
}

// SetWebStatement sets the URL of the web statement.
func (s *XMPRightsManagementSchema) SetWebStatement(url string) error {
	return s.setURL(RightsWebStatement, url)
}

// SetWebStatementProperty sets the web statement property outright.
func (s *XMPRightsManagementSchema) SetWebStatementProperty(url *xmptype.URLValueType) {
	s.AddProperty(url)
}

// CertificateProperty returns the certificate property, or nil.
func (s *XMPRightsManagementSchema) CertificateProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, RightsCertificate)
}

// Certificate returns the URL of the certificate.
func (s *XMPRightsManagementSchema) Certificate() string {
	return TextValueOf(&s.XMPSchema, RightsCertificate)
}

// SetCertificate sets the URL of the certificate.
func (s *XMPRightsManagementSchema) SetCertificate(url string) error {
	return s.setURL(RightsCertificate, url)
}

// SetCertificateProperty sets the certificate property outright.
func (s *XMPRightsManagementSchema) SetCertificateProperty(url *xmptype.URLValueType) {
	s.AddProperty(url)
}

// setURL is Java's `addProperty((URLType) instanciateSimple(name, url))`.
func (s *XMPRightsManagementSchema) setURL(name, url string) error {
	property, err := s.InstanciateSimple(name, url)
	if err != nil {
		return err
	}
	s.AddProperty(property)
	return nil
}

// XMPPageTextSchema is the xmpTPg namespace: what a page of the document holds.
type XMPPageTextSchema struct{ XMPSchema }

// The fields XMPPageTextSchema declares.
const (
	// PageTextMaxPageSize is the size of the largest page in the document.
	PageTextMaxPageSize = "MaxPageSize"

	// PageTextNPages is the number of pages in the document.
	PageTextNPages = "NPages"

	// PageTextPlateNames is an ordered array of plate names that are needed to
	// print the document.
	PageTextPlateNames = "PlateNames"

	// PageTextColorants is an ordered array of colorants (swatches) that are
	// used in the document.
	PageTextColorants = "Colorants"

	// PageTextFonts is an unordered array of fonts that are used in the
	// document.
	PageTextFonts = "Fonts"
)

var pageTextInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: "xmpTPg",
	Namespace:      PageTextNamespace,
}

var pageTextProperties = describe(map[string]xmptype.PropertyType{
	PageTextMaxPageSize: simple(xmptype.Dimensions),
	PageTextNPages:      simple(xmptype.Integer),
	PageTextPlateNames:  card(xmptype.Text, xmptype.Seq),
	PageTextColorants:   card(xmptype.Colorant, xmptype.Seq),
	PageTextFonts:       card(xmptype.Font, xmptype.Bag),
}, PageTextMaxPageSize, PageTextNPages, PageTextPlateNames, PageTextColorants, PageTextFonts)

// NewXMPPageTextSchema returns the xmpTPg schema.
func NewXMPPageTextSchema(metadata xmptype.MetadataLike) (*XMPPageTextSchema, error) {
	return NewXMPPageTextSchemaPrefixed(metadata, "")
}

// NewXMPPageTextSchemaPrefixed returns the xmpTPg schema with the given prefix.
func NewXMPPageTextSchemaPrefixed(metadata xmptype.MetadataLike,
	prefix string) (*XMPPageTextSchema, error) {
	s := &XMPPageTextSchema{}
	s.setDescription(pageTextProperties, "XMPPageTextSchema")
	if err := s.InitSchema(metadata, pageTextInfo, "", prefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newXMPPageTextSchemaAs is the factory's constructor.
func newXMPPageTextSchemaAs(metadata xmptype.MetadataLike, prefix string) (Schema, error) {
	return NewXMPPageTextSchemaPrefixed(metadata, prefix)
}

// XMPBasicJobTicketSchema is the xmpBJ namespace: the jobs a document belongs
// to.
type XMPBasicJobTicketSchema struct {
	XMPSchema

	bagJobs *xmptype.ArrayProperty
}

// JobTicketJobRef is the one field XMPBasicJobTicketSchema declares.
const JobTicketJobRef = "JobRef"

var jobTicketInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: "xmpBJ",
	Namespace:      JobTicketNamespace,
}

var jobTicketProperties = describe(map[string]xmptype.PropertyType{
	JobTicketJobRef: card(xmptype.Job, xmptype.Bag),
}, JobTicketJobRef)

// NewXMPBasicJobTicketSchema returns the xmpBJ schema.
func NewXMPBasicJobTicketSchema(
	metadata xmptype.MetadataLike) (*XMPBasicJobTicketSchema, error) {
	return NewXMPBasicJobTicketSchemaPrefixed(metadata, "")
}

// NewXMPBasicJobTicketSchemaPrefixed returns the xmpBJ schema with the given
// prefix.
func NewXMPBasicJobTicketSchemaPrefixed(metadata xmptype.MetadataLike,
	ownPrefix string) (*XMPBasicJobTicketSchema, error) {
	s := &XMPBasicJobTicketSchema{}
	s.setDescription(jobTicketProperties, "XMPBasicJobTicketSchema")
	if err := s.InitSchema(metadata, jobTicketInfo, "", ownPrefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newXMPBasicJobTicketSchemaAs is the factory's constructor.
func newXMPBasicJobTicketSchemaAs(metadata xmptype.MetadataLike, prefix string) (Schema, error) {
	return NewXMPBasicJobTicketSchemaPrefixed(metadata, prefix)
}

// AddJobOf adds a job with the given identifier, name and URL.
func (s *XMPBasicJobTicketSchema) AddJobOf(id, name, url string) error {
	return s.AddJobOfPrefix(id, name, url, "")
}

// AddJobOfPrefix adds a job written with the given prefix.
func (s *XMPBasicJobTicketSchema) AddJobOfPrefix(id, name, url, fieldPrefix string) error {
	if s.bagJobs != nil && fieldPrefix == "" {
		// use same prefix for all jobs
		all := s.bagJobs.AllProperties()
		if len(all) > 0 {
			if first, isJob := all[0].(*xmptype.JobType); isJob && first.Prefix() != "" {
				fieldPrefix = first.Prefix()
			}
		}
	}
	job := xmptype.NewJobTypePrefixed(s.Metadata(), fieldPrefix)
	if err := job.SetID(id); err != nil {
		return err
	}
	if err := job.SetName(name); err != nil {
		return err
	}
	if err := job.SetURL(url); err != nil {
		return err
	}
	s.AddJob(job)
	return nil
}

// AddJob adds the given job.
func (s *XMPBasicJobTicketSchema) AddJob(job *xmptype.JobType) {
	prefix := s.NamespacePrefix(job.Namespace())
	if prefix != "" {
		// use same prefix for all jobs
		job.SetPrefix(prefix)
		if s.bagJobs != nil {
			for _, field := range s.bagJobs.AllProperties() {
				if existing, isJob := field.(*xmptype.JobType); isJob {
					existing.SetPrefix(prefix)
				}
			}
		}
	} else {
		// add prefix
		s.AddNamespace(job.Namespace(), job.Prefix())
	}
	// create bag if not existing
	if s.bagJobs == nil {
		s.bagJobs = s.CreateArrayProperty(JobTicketJobRef, xmptype.Bag)
		s.AddProperty(s.bagJobs)
	}
	// add job
	s.bagJobs.Container().AddProperty(job)
}

// Jobs returns the jobs, and nil where there are none.
func (s *XMPBasicJobTicketSchema) Jobs() ([]*xmptype.JobType, error) {
	tmp, err := s.UnqualifiedArrayList(JobTicketJobRef)
	if err != nil || tmp == nil {
		return nil, err
	}
	layers := make([]*xmptype.JobType, 0, len(tmp))
	for _, abstractField := range tmp {
		job, isJob := abstractField.(*xmptype.JobType)
		if !isJob {
			return nil, fmt.Errorf("%w: Job expected and %s found.",
				xmptype.ErrBadFieldValue, abstractField.TypeName())
		}
		layers = append(layers, job)
	}
	return layers, nil
}
