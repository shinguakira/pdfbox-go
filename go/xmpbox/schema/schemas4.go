package schema

// The TIFF and EXIF schemas.
//
// Port of TiffSchema and ExifSchema. Both are mostly tables of field names: the
// only accessors either declares are for its language alternatives, plus TIFF's
// artist.

import (
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// TiffSchema is the tiff namespace: what a TIFF header would have carried.
type TiffSchema struct{ XMPSchema }

// The fields TiffSchema declares.
const (
	TiffImageDescription          = "ImageDescription"
	TiffCopyright                 = "Copyright"
	TiffArtist                    = "Artist"
	TiffImageWidth                = "ImageWidth"
	TiffImageLength               = "ImageLength"
	TiffBitsPerSample             = "BitsPerSample"
	TiffCompression               = "Compression"
	TiffPhotometricInterpretation = "PhotometricInterpretation"
	TiffOrientation               = "Orientation"
	TiffSamplesPerPixel           = "SamplesPerPixel"
	TiffPlanarConfiguration       = "PlanarConfiguration"
	TiffYCbCrSubSampling          = "YCbCrSubSampling"
	TiffYCbCrPositioning          = "YCbCrPositioning"
	TiffXResolution               = "XResolution"
	TiffYResolution               = "YResolution"
	TiffResolutionUnit            = "ResolutionUnit"
	TiffTransferFunction          = "TransferFunction"
	TiffWhitePoint                = "WhitePoint"
	TiffPrimaryChromaticities     = "PrimaryChromaticities"
	TiffYCbCrCoefficients         = "YCbCrCoefficients"
	TiffReferenceBlackWhite       = "ReferenceBlackWhite"
	TiffDateTime                  = "DateTime"
	TiffSoftware                  = "Software"
	TiffMake                      = "Make"
	TiffModel                     = "Model"
)

var tiffInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: "tiff",
	Namespace:      TiffNamespace,
}

var tiffProperties = describe(map[string]xmptype.PropertyType{
	TiffImageDescription:          card(xmptype.LangAlt, xmptype.Simple),
	TiffCopyright:                 card(xmptype.LangAlt, xmptype.Simple),
	TiffArtist:                    card(xmptype.ProperName, xmptype.Simple),
	TiffImageWidth:                card(xmptype.Integer, xmptype.Simple),
	TiffImageLength:               card(xmptype.Integer, xmptype.Simple),
	TiffBitsPerSample:             card(xmptype.Integer, xmptype.Seq),
	TiffCompression:               card(xmptype.Integer, xmptype.Simple),
	TiffPhotometricInterpretation: card(xmptype.Integer, xmptype.Simple),
	TiffOrientation:               card(xmptype.Integer, xmptype.Simple),
	TiffSamplesPerPixel:           card(xmptype.Integer, xmptype.Simple),
	TiffPlanarConfiguration:       card(xmptype.Integer, xmptype.Simple),
	TiffYCbCrSubSampling:          card(xmptype.Integer, xmptype.Seq),
	TiffYCbCrPositioning:          card(xmptype.Integer, xmptype.Simple),
	TiffXResolution:               card(xmptype.Rational, xmptype.Simple),
	TiffYResolution:               card(xmptype.Rational, xmptype.Simple),
	TiffResolutionUnit:            card(xmptype.Integer, xmptype.Simple),
	TiffTransferFunction:          card(xmptype.Integer, xmptype.Seq),
	TiffWhitePoint:                card(xmptype.Rational, xmptype.Seq),
	TiffPrimaryChromaticities:     card(xmptype.Rational, xmptype.Seq),
	TiffYCbCrCoefficients:         card(xmptype.Rational, xmptype.Seq),
	TiffReferenceBlackWhite:       card(xmptype.Rational, xmptype.Seq),
	TiffDateTime:                  card(xmptype.Date, xmptype.Simple),
	TiffSoftware:                  card(xmptype.AgentName, xmptype.Simple),
	TiffMake:                      card(xmptype.ProperName, xmptype.Simple),
	TiffModel:                     card(xmptype.ProperName, xmptype.Simple),
}, TiffImageDescription, TiffCopyright, TiffArtist, TiffImageWidth, TiffImageLength,
	TiffBitsPerSample, TiffCompression, TiffPhotometricInterpretation, TiffOrientation,
	TiffSamplesPerPixel, TiffPlanarConfiguration, TiffYCbCrSubSampling, TiffYCbCrPositioning,
	TiffXResolution, TiffYResolution, TiffResolutionUnit, TiffTransferFunction, TiffWhitePoint,
	TiffPrimaryChromaticities, TiffYCbCrCoefficients, TiffReferenceBlackWhite, TiffDateTime,
	TiffSoftware, TiffMake, TiffModel)

// NewTiffSchema returns the tiff schema.
func NewTiffSchema(metadata xmptype.MetadataLike) (*TiffSchema, error) {
	return NewTiffSchemaPrefixed(metadata, "")
}

// NewTiffSchemaPrefixed returns the tiff schema with the given prefix.
func NewTiffSchemaPrefixed(metadata xmptype.MetadataLike,
	prefix string) (*TiffSchema, error) {
	s := &TiffSchema{}
	s.setDescription(tiffProperties, "TiffSchema")
	if err := s.InitSchema(metadata, tiffInfo, "", prefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newTiffSchemaAs is the factory's constructor.
func newTiffSchemaAs(metadata xmptype.MetadataLike, prefix string) (Schema, error) {
	return NewTiffSchemaPrefixed(metadata, prefix)
}

// ArtistProperty returns the artist property, or nil.
//
// The field is declared as a ProperName and this getter asks for one, but
// SetArtist stores a TextType, which is not a ProperNameType; so the value set
// through SetArtist is never visible here. Ported as written; see
// migration/JAVA-BUGS.md.
func (s *TiffSchema) ArtistProperty() *xmptype.ProperNameType {
	return PropertyAs[*xmptype.ProperNameType](&s.XMPSchema, TiffArtist)
}

// Artist returns the artist, and the empty string where there is none.
func (s *TiffSchema) Artist() string {
	tt := s.ArtistProperty()
	if tt == nil {
		return ""
	}
	return tt.StringValue()
}

// SetArtist sets the name of the artist.
func (s *TiffSchema) SetArtist(text string) error {
	return SetTextValue(&s.XMPSchema, TiffArtist, text)
}

// ImageDescriptionProperty returns the image description alternative, or nil.
func (s *TiffSchema) ImageDescriptionProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, TiffImageDescription)
}

// ImageDescriptionLanguages returns the languages the image description is
// written in, and nil where the property is absent.
func (s *TiffSchema) ImageDescriptionLanguages() ([]string, error) {
	return s.UnqualifiedLanguagePropertyLanguagesValue(TiffImageDescription)
}

// ImageDescriptionIn returns the image description in the given language.
func (s *TiffSchema) ImageDescriptionIn(lang string) (string, error) {
	return s.UnqualifiedLanguagePropertyValue(TiffImageDescription, lang)
}

// ImageDescription returns the image description in the default language.
func (s *TiffSchema) ImageDescription() (string, error) {
	return s.ImageDescriptionIn("")
}

// AddImageDescription sets the image description for a language.
func (s *TiffSchema) AddImageDescription(lang, value string) error {
	return s.SetUnqualifiedLanguagePropertyValue(TiffImageDescription, lang, value)
}

// CopyrightProperty returns the copyright alternative, or nil.
func (s *TiffSchema) CopyrightProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, TiffCopyright)
}

// CopyrightLanguages returns the languages the copyright is written in, and nil
// where the property is absent.
func (s *TiffSchema) CopyrightLanguages() ([]string, error) {
	return s.UnqualifiedLanguagePropertyLanguagesValue(TiffCopyright)
}

// CopyrightIn returns the copyright in the given language.
func (s *TiffSchema) CopyrightIn(lang string) (string, error) {
	return s.UnqualifiedLanguagePropertyValue(TiffCopyright, lang)
}

// Copyright returns the copyright in the default language.
func (s *TiffSchema) Copyright() (string, error) { return s.CopyrightIn("") }

// AddCopyright sets the copyright for a language.
func (s *TiffSchema) AddCopyright(lang, value string) error {
	return s.SetUnqualifiedLanguagePropertyValue(TiffCopyright, lang, value)
}
