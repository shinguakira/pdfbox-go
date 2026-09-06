package schema

// The XMP basic schema.
//
// Port of XMPBasicSchema.

import (
	"fmt"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// XMPBasicSchema is the xmp namespace: when a document was made and changed,
// what made it, and its thumbnails.
type XMPBasicSchema struct {
	XMPSchema

	altThumbs *xmptype.ArrayProperty
}

// The fields XMPBasicSchema declares.
const (
	BasicAdvisory     = "Advisory"
	BasicBaseURL      = "BaseURL"
	BasicCreateDate   = "CreateDate"
	BasicCreatorTool  = "CreatorTool"
	BasicIdentifier   = "Identifier"
	BasicLabel        = "Label"
	BasicMetadataDate = "MetadataDate"
	BasicModifyDate   = "ModifyDate"
	BasicNickname     = "Nickname"
	BasicRating       = "Rating"
	BasicThumbnails   = "Thumbnails"
	BasicModifierDate = "ModifierDate"
)

var xmpBasicInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: "xmp",
	Namespace:      XMPBasicNamespace,
}

var xmpBasicProperties = describe(map[string]xmptype.PropertyType{
	BasicAdvisory:     card(xmptype.XPath, xmptype.Bag),
	BasicBaseURL:      card(xmptype.URL, xmptype.Simple),
	BasicCreateDate:   card(xmptype.Date, xmptype.Simple),
	BasicCreatorTool:  card(xmptype.AgentName, xmptype.Simple),
	BasicIdentifier:   card(xmptype.Text, xmptype.Bag),
	BasicLabel:        card(xmptype.Text, xmptype.Simple),
	BasicMetadataDate: card(xmptype.Date, xmptype.Simple),
	BasicModifyDate:   card(xmptype.Date, xmptype.Simple),
	BasicNickname:     card(xmptype.Text, xmptype.Simple),
	BasicRating:       card(xmptype.Integer, xmptype.Simple),
	BasicThumbnails:   card(xmptype.Thumbnail, xmptype.Alt),
	BasicModifierDate: card(xmptype.Date, xmptype.Simple),
}, BasicAdvisory, BasicBaseURL, BasicCreateDate, BasicCreatorTool, BasicIdentifier,
	BasicLabel, BasicMetadataDate, BasicModifyDate, BasicNickname, BasicRating,
	BasicThumbnails, BasicModifierDate)

// NewXMPBasicSchema returns the xmp schema.
func NewXMPBasicSchema(metadata xmptype.MetadataLike) (*XMPBasicSchema, error) {
	return NewXMPBasicSchemaPrefixed(metadata, "")
}

// NewXMPBasicSchemaPrefixed returns the xmp schema with the given prefix.
func NewXMPBasicSchemaPrefixed(metadata xmptype.MetadataLike,
	ownPrefix string) (*XMPBasicSchema, error) {
	s := &XMPBasicSchema{}
	s.setDescription(xmpBasicProperties, "XMPBasicSchema")
	if err := s.InitSchema(metadata, xmpBasicInfo, "", ownPrefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newXMPBasicSchemaAs is the factory's constructor.
func newXMPBasicSchemaAs(metadata xmptype.MetadataLike, prefix string) (Schema, error) {
	return NewXMPBasicSchemaPrefixed(metadata, prefix)
}

// AddThumbnails appends a thumbnail of the given size, format and image.
func (s *XMPBasicSchema) AddThumbnails(height, width int, format, img string) error {
	if s.altThumbs == nil {
		s.altThumbs = s.CreateArrayProperty(BasicThumbnails, xmptype.Alt)
		s.AddProperty(s.altThumbs)
	}
	thumb := xmptype.NewThumbnailType(s.Metadata())
	if err := thumb.SetHeight(height); err != nil {
		return err
	}
	if err := thumb.SetWidth(width); err != nil {
		return err
	}
	if err := thumb.SetFormat(format); err != nil {
		return err
	}
	if err := thumb.SetImage(img); err != nil {
		return err
	}
	s.altThumbs.Container().AddProperty(thumb)
	return nil
}

// AddAdvisory adds an advisory XPath.
func (s *XMPBasicSchema) AddAdvisory(xpath string) error {
	return s.AddQualifiedBagValue(BasicAdvisory, xpath)
}

// RemoveAdvisory removes an advisory XPath.
func (s *XMPBasicSchema) RemoveAdvisory(xpath string) {
	s.RemoveUnqualifiedBagValue(BasicAdvisory, xpath)
}

// addSimple is Java's `addProperty(instanciateSimple(name, value))`, which every
// setter here goes through.
func (s *XMPBasicSchema) addSimple(name string, value any) error {
	property, err := s.InstanciateSimple(name, value)
	if err != nil {
		return err
	}
	s.AddProperty(property)
	return nil
}

// SetBaseURL sets the base URL.
func (s *XMPBasicSchema) SetBaseURL(url string) error { return s.addSimple(BasicBaseURL, url) }

// SetBaseURLProperty sets the base URL property outright.
func (s *XMPBasicSchema) SetBaseURLProperty(url *xmptype.URLValueType) { s.AddProperty(url) }

// SetCreateDate sets when the document was made.
func (s *XMPBasicSchema) SetCreateDate(date time.Time) error {
	return s.addSimple(BasicCreateDate, date)
}

// SetCreateDateProperty sets the creation date property outright.
func (s *XMPBasicSchema) SetCreateDateProperty(date *xmptype.DateType) { s.AddProperty(date) }

// SetCreatorTool sets what made the document.
func (s *XMPBasicSchema) SetCreatorTool(creatorTool string) error {
	return s.addSimple(BasicCreatorTool, creatorTool)
}

// SetCreatorToolProperty sets the creator tool property outright.
func (s *XMPBasicSchema) SetCreatorToolProperty(creatorTool *xmptype.AgentNameType) {
	s.AddProperty(creatorTool)
}

// AddIdentifier adds an identifier.
func (s *XMPBasicSchema) AddIdentifier(text string) error {
	return s.AddQualifiedBagValue(BasicIdentifier, text)
}

// RemoveIdentifier removes an identifier.
func (s *XMPBasicSchema) RemoveIdentifier(text string) {
	s.RemoveUnqualifiedBagValue(BasicIdentifier, text)
}

// SetLabel sets the label.
func (s *XMPBasicSchema) SetLabel(text string) error { return s.addSimple(BasicLabel, text) }

// SetLabelProperty sets the label property outright.
func (s *XMPBasicSchema) SetLabelProperty(text *xmptype.TextType) { s.AddProperty(text) }

// SetMetadataDate sets when the metadata last changed.
func (s *XMPBasicSchema) SetMetadataDate(date time.Time) error {
	return s.addSimple(BasicMetadataDate, date)
}

// SetMetadataDateProperty sets the metadata date property outright.
func (s *XMPBasicSchema) SetMetadataDateProperty(date *xmptype.DateType) { s.AddProperty(date) }

// SetModifyDate sets when the document last changed.
func (s *XMPBasicSchema) SetModifyDate(date time.Time) error {
	return s.addSimple(BasicModifyDate, date)
}

// SetModifierDate sets the modifier date.
func (s *XMPBasicSchema) SetModifierDate(date time.Time) error {
	return s.addSimple(BasicModifierDate, date)
}

// SetModifyDateProperty sets the modification date property outright.
func (s *XMPBasicSchema) SetModifyDateProperty(date *xmptype.DateType) { s.AddProperty(date) }

// SetModifierDateProperty sets the modifier date property outright.
func (s *XMPBasicSchema) SetModifierDateProperty(date *xmptype.DateType) { s.AddProperty(date) }

// SetNickname sets the nickname.
func (s *XMPBasicSchema) SetNickname(text string) error { return s.addSimple(BasicNickname, text) }

// SetNicknameProperty sets the nickname property outright.
func (s *XMPBasicSchema) SetNicknameProperty(text *xmptype.TextType) { s.AddProperty(text) }

// SetRating sets the rating.
func (s *XMPBasicSchema) SetRating(rate int) error { return s.addSimple(BasicRating, rate) }

// SetRatingProperty sets the rating property outright.
func (s *XMPBasicSchema) SetRatingProperty(rate *xmptype.IntegerType) { s.AddProperty(rate) }

// AdvisoryProperty returns the advisory array, or nil.
func (s *XMPBasicSchema) AdvisoryProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, BasicAdvisory)
}

// Advisory returns the advisory XPaths.
func (s *XMPBasicSchema) Advisory() []string { return s.UnqualifiedBagValueList(BasicAdvisory) }

// BaseURLProperty returns the base URL property, or nil.
func (s *XMPBasicSchema) BaseURLProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, BasicBaseURL)
}

// BaseURL returns the base URL.
func (s *XMPBasicSchema) BaseURL() string { return TextValueOf(&s.XMPSchema, BasicBaseURL) }

// CreateDateProperty returns the creation date property, or nil.
func (s *XMPBasicSchema) CreateDateProperty() *xmptype.DateType {
	return PropertyAs[*xmptype.DateType](&s.XMPSchema, BasicCreateDate)
}

// CreateDate returns when the document was made, the second result being false
// where it does not say.
func (s *XMPBasicSchema) CreateDate() (time.Time, bool) {
	return dateValueOf(s.CreateDateProperty())
}

// CreatorToolProperty returns the creator tool property, or nil.
func (s *XMPBasicSchema) CreatorToolProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, BasicCreatorTool)
}

// CreatorTool returns what made the document.
func (s *XMPBasicSchema) CreatorTool() string { return TextValueOf(&s.XMPSchema, BasicCreatorTool) }

// IdentifiersProperty returns the identifiers array, or nil.
func (s *XMPBasicSchema) IdentifiersProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, BasicIdentifier)
}

// Identifiers returns the identifiers.
func (s *XMPBasicSchema) Identifiers() []string {
	return s.UnqualifiedBagValueList(BasicIdentifier)
}

// LabelProperty returns the label property, or nil.
func (s *XMPBasicSchema) LabelProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, BasicLabel)
}

// Label returns the label.
func (s *XMPBasicSchema) Label() string { return TextValueOf(&s.XMPSchema, BasicLabel) }

// MetadataDateProperty returns the metadata date property, or nil.
func (s *XMPBasicSchema) MetadataDateProperty() *xmptype.DateType {
	return PropertyAs[*xmptype.DateType](&s.XMPSchema, BasicMetadataDate)
}

// MetadataDate returns when the metadata last changed.
func (s *XMPBasicSchema) MetadataDate() (time.Time, bool) {
	return dateValueOf(s.MetadataDateProperty())
}

// ModifyDateProperty returns the modification date property, or nil.
func (s *XMPBasicSchema) ModifyDateProperty() *xmptype.DateType {
	return PropertyAs[*xmptype.DateType](&s.XMPSchema, BasicModifyDate)
}

// ModifierDateProperty returns the modifier date property, or nil.
func (s *XMPBasicSchema) ModifierDateProperty() *xmptype.DateType {
	return PropertyAs[*xmptype.DateType](&s.XMPSchema, BasicModifierDate)
}

// ModifyDate returns when the document last changed.
func (s *XMPBasicSchema) ModifyDate() (time.Time, bool) {
	return dateValueOf(s.ModifyDateProperty())
}

// ModifierDate returns the modifier date.
func (s *XMPBasicSchema) ModifierDate() (time.Time, bool) {
	return dateValueOf(s.ModifierDateProperty())
}

// NicknameProperty returns the nickname property, or nil.
func (s *XMPBasicSchema) NicknameProperty() *xmptype.TextType {
	return TextPropertyOf(&s.XMPSchema, BasicNickname)
}

// Nickname returns the nickname.
func (s *XMPBasicSchema) Nickname() string { return TextValueOf(&s.XMPSchema, BasicNickname) }

// RatingProperty returns the rating property, or nil.
func (s *XMPBasicSchema) RatingProperty() *xmptype.IntegerType {
	return PropertyAs[*xmptype.IntegerType](&s.XMPSchema, BasicRating)
}

// Rating returns the rating, the second result being false where there is none.
func (s *XMPBasicSchema) Rating() (int, bool) {
	it := s.RatingProperty()
	if it == nil {
		return 0, false
	}
	return it.IntegerValue(), true
}

// ThumbnailsProperty returns the thumbnails, and nil where there are none.
func (s *XMPBasicSchema) ThumbnailsProperty() ([]*xmptype.ThumbnailType, error) {
	tmp, err := s.UnqualifiedArrayList(BasicThumbnails)
	if err != nil || tmp == nil {
		return nil, err
	}
	thumbs := make([]*xmptype.ThumbnailType, 0, len(tmp))
	for _, abstractField := range tmp {
		thumb, isThumb := abstractField.(*xmptype.ThumbnailType)
		if !isThumb {
			return nil, fmt.Errorf("%w: Thumbnail expected and %s found.",
				xmptype.ErrBadFieldValue, abstractField.TypeName())
		}
		thumbs = append(thumbs, thumb)
	}
	return thumbs, nil
}

// dateValueOf is Java's `dt == null ? null : dt.getValue()`.
//
// Both halves answer null: an absent property, and a property that is there and
// holds no date. See PDFBOX-6029.
func dateValueOf(date *xmptype.DateType) (time.Time, bool) {
	if date == nil {
		return time.Time{}, false
	}
	return date.DateValue()
}
