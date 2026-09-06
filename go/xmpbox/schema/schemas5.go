package schema

// The EXIF schema.
//
// Port of ExifSchema, which is a table of field names with one accessor group:
// the UserComment alternative. Everything else is reached through XMPSchema.

import (
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// ExifSchema is the exif namespace: what a camera wrote about a photograph.
type ExifSchema struct{ XMPSchema }

// The fields ExifSchema declares.
const (
	ExifUserComment              = "UserComment"
	ExifVersion                  = "ExifVersion"
	ExifFlashPixVersion          = "FlashpixVersion"
	ExifColorSpace               = "ColorSpace"
	ExifComponentsConfiguration  = "ComponentsConfiguration"
	ExifCompressedBPP            = "CompressedBitsPerPixel"
	ExifPixelXDimension          = "PixelXDimension"
	ExifPixelYDimension          = "PixelYDimension"
	ExifRelatedSoundFile         = "RelatedSoundFile"
	ExifDateTimeOriginal         = "DateTimeOriginal"
	ExifDateTimeDigitized        = "DateTimeDigitized"
	ExifExposureTime             = "ExposureTime"
	ExifFNumber                  = "FNumber"
	ExifExposureProgram          = "ExposureProgram"
	ExifSpectralSensitivity      = "SpectralSensitivity"
	ExifISOSpeedRatings          = "ISOSpeedRatings"
	ExifShutterSpeedValue        = "ShutterSpeedValue"
	ExifApertureValue            = "ApertureValue"
	ExifBrightnessValue          = "BrightnessValue"
	ExifExposureBiasValue        = "ExposureBiasValue"
	ExifMaxApertureValue         = "MaxApertureValue"
	ExifSubjectDistance          = "SubjectDistance"
	ExifMeteringMode             = "MeteringMode"
	ExifLightSource              = "LightSource"
	ExifFlashEnergy              = "FlashEnergy"
	ExifFocalLength              = "FocalLength"
	ExifFocalPlaneXResolution    = "FocalPlaneXResolution"
	ExifFocalPlaneYResolution    = "FocalPlaneYResolution"
	ExifSubjectArea              = "SubjectArea"
	ExifFocalPlaneResolutionUnit = "FocalPlaneResolutionUnit"
	ExifSubjectLocation          = "SubjectLocation"
	ExifExposureIndex            = "ExposureIndex"
	ExifSensingMethod            = "SensingMethod"
	ExifFileSource               = "FileSource"
	ExifSceneType                = "SceneType"
	ExifCustomRendered           = "CustomRendered"
	ExifWhiteBalance             = "WhiteBalance"
	ExifExposureMode             = "ExposureMode"
	ExifDigitalZoomRatio         = "DigitalZoomRatio"
	ExifFocalLengthIn35mmFilm    = "FocalLengthIn35mmFilm"
	ExifSceneCaptureType         = "SceneCaptureType"
	ExifGainControl              = "GainControl"
	ExifContrast                 = "Contrast"
	ExifSaturation               = "Saturation"
	ExifSharpness                = "Sharpness"
	ExifSubjectDistanceRange     = "SubjectDistanceRange"
	ExifImageUniqueID            = "ImageUniqueID"
	ExifGPSVersionID             = "GPSVersionID"
	ExifGPSSatellites            = "GPSSatellites"
	ExifGPSStatus                = "GPSStatus"
	ExifGPSMeasureMode           = "GPSMeasureMode"
	ExifGPSMapDatum              = "GPSMapDatum"
	ExifGPSSpeedRef              = "GPSSpeedRef"
	ExifGPSTrackRef              = "GPSTrackRef"
	ExifGPSImgDirectionRef       = "GPSImgDirectionRef"
	ExifGPSDestBearingRef        = "GPSDestBearingRef"
	ExifGPSDestDistanceRef       = "GPSDestDistanceRef"
	ExifGPSProcessingMethod      = "GPSProcessingMethod"
	ExifGPSAreaInformation       = "GPSAreaInformation"
	ExifGPSAltitude              = "GPSAltitude"
	ExifGPSDOP                   = "GPSDOP"
	ExifGPSSpeed                 = "GPSSpeed"
	ExifGPSTrack                 = "GPSTrack"
	ExifGPSImgDirection          = "GPSImgDirection"
	ExifGPSDestBearing           = "GPSDestBearing"
	ExifGPSDestDistance          = "GPSDestDistance"
	ExifGPSAltitudeRef           = "GPSAltitudeRef"
	ExifGPSDifferential          = "GPSDifferential"
	ExifGPSTimeStamp             = "GPSTimeStamp"
	ExifOECF                     = "OECF"
	ExifSpatialFrequencyResponse = "SpatialFrequencyResponse"
	ExifGPSLatitude              = "GPSLatitude"
	ExifGPSLongitude             = "GPSLongitude"
	ExifGPSDestLatitude          = "GPSDestLatitude"
	ExifGPSDestLongitude         = "GPSDestLongitude"
	ExifCFAPattern               = "CFAPattern"
	ExifFlash                    = "Flash"
	ExifCFAPatternType           = "CFAPatternType"
	ExifDeviceSettingDescription = "DeviceSettingDescription"
)

var exifInfo = xmptype.StructuredTypeInfo{
	PreferedPrefix: "exif",
	Namespace:      "http://ns.adobe.com/exif/1.0/",
}

var exifProperties = describe(map[string]xmptype.PropertyType{
	ExifUserComment:              card(xmptype.LangAlt, xmptype.Simple),
	ExifVersion:                  card(xmptype.Text, xmptype.Simple),
	ExifFlashPixVersion:          card(xmptype.Text, xmptype.Simple),
	ExifColorSpace:               card(xmptype.Integer, xmptype.Simple),
	ExifComponentsConfiguration:  card(xmptype.Integer, xmptype.Seq),
	ExifCompressedBPP:            card(xmptype.Rational, xmptype.Simple),
	ExifPixelXDimension:          card(xmptype.Integer, xmptype.Simple),
	ExifPixelYDimension:          card(xmptype.Integer, xmptype.Simple),
	ExifRelatedSoundFile:         card(xmptype.Text, xmptype.Simple),
	ExifDateTimeOriginal:         card(xmptype.Date, xmptype.Simple),
	ExifDateTimeDigitized:        card(xmptype.Date, xmptype.Simple),
	ExifExposureTime:             card(xmptype.Rational, xmptype.Simple),
	ExifFNumber:                  card(xmptype.Rational, xmptype.Simple),
	ExifExposureProgram:          card(xmptype.Integer, xmptype.Simple),
	ExifSpectralSensitivity:      card(xmptype.Text, xmptype.Simple),
	ExifISOSpeedRatings:          card(xmptype.Integer, xmptype.Seq),
	ExifShutterSpeedValue:        card(xmptype.Rational, xmptype.Simple),
	ExifApertureValue:            card(xmptype.Rational, xmptype.Simple),
	ExifBrightnessValue:          card(xmptype.Rational, xmptype.Simple),
	ExifExposureBiasValue:        card(xmptype.Rational, xmptype.Simple),
	ExifMaxApertureValue:         card(xmptype.Rational, xmptype.Simple),
	ExifSubjectDistance:          card(xmptype.Rational, xmptype.Simple),
	ExifMeteringMode:             card(xmptype.Integer, xmptype.Simple),
	ExifLightSource:              card(xmptype.Integer, xmptype.Simple),
	ExifFlashEnergy:              card(xmptype.Rational, xmptype.Simple),
	ExifFocalLength:              card(xmptype.Rational, xmptype.Simple),
	ExifFocalPlaneXResolution:    card(xmptype.Rational, xmptype.Simple),
	ExifFocalPlaneYResolution:    card(xmptype.Rational, xmptype.Simple),
	ExifSubjectArea:              card(xmptype.Integer, xmptype.Seq),
	ExifFocalPlaneResolutionUnit: card(xmptype.Integer, xmptype.Simple),
	ExifSubjectLocation:          card(xmptype.Integer, xmptype.Seq),
	ExifExposureIndex:            card(xmptype.Rational, xmptype.Simple),
	ExifSensingMethod:            card(xmptype.Integer, xmptype.Simple),
	ExifFileSource:               card(xmptype.Integer, xmptype.Simple),
	ExifSceneType:                card(xmptype.Integer, xmptype.Simple),
	ExifCustomRendered:           card(xmptype.Integer, xmptype.Simple),
	ExifWhiteBalance:             card(xmptype.Integer, xmptype.Simple),
	ExifExposureMode:             card(xmptype.Integer, xmptype.Simple),
	ExifDigitalZoomRatio:         card(xmptype.Rational, xmptype.Simple),
	ExifFocalLengthIn35mmFilm:    card(xmptype.Integer, xmptype.Simple),
	ExifSceneCaptureType:         card(xmptype.Integer, xmptype.Simple),
	ExifGainControl:              card(xmptype.Integer, xmptype.Simple),
	ExifContrast:                 card(xmptype.Integer, xmptype.Simple),
	ExifSaturation:               card(xmptype.Integer, xmptype.Simple),
	ExifSharpness:                card(xmptype.Integer, xmptype.Simple),
	ExifSubjectDistanceRange:     card(xmptype.Integer, xmptype.Simple),
	ExifImageUniqueID:            card(xmptype.Text, xmptype.Simple),
	ExifGPSVersionID:             card(xmptype.Text, xmptype.Simple),
	ExifGPSSatellites:            card(xmptype.Text, xmptype.Simple),
	ExifGPSStatus:                card(xmptype.Text, xmptype.Simple),
	ExifGPSMeasureMode:           card(xmptype.Text, xmptype.Simple),
	ExifGPSMapDatum:              card(xmptype.Text, xmptype.Simple),
	ExifGPSSpeedRef:              card(xmptype.Text, xmptype.Simple),
	ExifGPSTrackRef:              card(xmptype.Text, xmptype.Simple),
	ExifGPSImgDirectionRef:       card(xmptype.Text, xmptype.Simple),
	ExifGPSDestBearingRef:        card(xmptype.Text, xmptype.Simple),
	ExifGPSDestDistanceRef:       card(xmptype.Text, xmptype.Simple),
	ExifGPSProcessingMethod:      card(xmptype.Text, xmptype.Simple),
	ExifGPSAreaInformation:       card(xmptype.Text, xmptype.Simple),
	ExifGPSAltitude:              card(xmptype.Rational, xmptype.Simple),
	ExifGPSDOP:                   card(xmptype.Rational, xmptype.Simple),
	ExifGPSSpeed:                 card(xmptype.Rational, xmptype.Simple),
	ExifGPSTrack:                 card(xmptype.Rational, xmptype.Simple),
	ExifGPSImgDirection:          card(xmptype.Rational, xmptype.Simple),
	ExifGPSDestBearing:           card(xmptype.Rational, xmptype.Simple),
	ExifGPSDestDistance:          card(xmptype.Rational, xmptype.Simple),
	ExifGPSAltitudeRef:           card(xmptype.Integer, xmptype.Simple),
	ExifGPSDifferential:          card(xmptype.Integer, xmptype.Simple),
	ExifGPSTimeStamp:             card(xmptype.Date, xmptype.Simple),
	ExifOECF:                     simple(xmptype.OECF),
	ExifSpatialFrequencyResponse: simple(xmptype.OECF),
	ExifGPSLatitude:              simple(xmptype.GPSCoordinate),
	ExifGPSLongitude:             simple(xmptype.GPSCoordinate),
	ExifGPSDestLatitude:          simple(xmptype.GPSCoordinate),
	ExifGPSDestLongitude:         simple(xmptype.GPSCoordinate),
	ExifCFAPattern:               simple(xmptype.CFAPattern),
	ExifFlash:                    simple(xmptype.Flash),
	ExifCFAPatternType:           simple(xmptype.CFAPattern),
	ExifDeviceSettingDescription: simple(xmptype.DeviceSettings),
}, ExifUserComment, ExifVersion, ExifFlashPixVersion, ExifColorSpace,
	ExifComponentsConfiguration, ExifCompressedBPP, ExifPixelXDimension, ExifPixelYDimension,
	ExifRelatedSoundFile, ExifDateTimeOriginal, ExifDateTimeDigitized, ExifExposureTime,
	ExifFNumber, ExifExposureProgram, ExifSpectralSensitivity, ExifISOSpeedRatings,
	ExifShutterSpeedValue, ExifApertureValue, ExifBrightnessValue, ExifExposureBiasValue,
	ExifMaxApertureValue, ExifSubjectDistance, ExifMeteringMode, ExifLightSource,
	ExifFlashEnergy, ExifFocalLength, ExifFocalPlaneXResolution, ExifFocalPlaneYResolution,
	ExifSubjectArea, ExifFocalPlaneResolutionUnit, ExifSubjectLocation, ExifExposureIndex,
	ExifSensingMethod, ExifFileSource, ExifSceneType, ExifCustomRendered, ExifWhiteBalance,
	ExifExposureMode, ExifDigitalZoomRatio, ExifFocalLengthIn35mmFilm, ExifSceneCaptureType,
	ExifGainControl, ExifContrast, ExifSaturation, ExifSharpness, ExifSubjectDistanceRange,
	ExifImageUniqueID, ExifGPSVersionID, ExifGPSSatellites, ExifGPSStatus, ExifGPSMeasureMode,
	ExifGPSMapDatum, ExifGPSSpeedRef, ExifGPSTrackRef, ExifGPSImgDirectionRef,
	ExifGPSDestBearingRef, ExifGPSDestDistanceRef, ExifGPSProcessingMethod,
	ExifGPSAreaInformation, ExifGPSAltitude, ExifGPSDOP, ExifGPSSpeed, ExifGPSTrack,
	ExifGPSImgDirection, ExifGPSDestBearing, ExifGPSDestDistance, ExifGPSAltitudeRef,
	ExifGPSDifferential, ExifGPSTimeStamp, ExifOECF, ExifSpatialFrequencyResponse,
	ExifGPSLatitude, ExifGPSLongitude, ExifGPSDestLatitude, ExifGPSDestLongitude,
	ExifCFAPattern, ExifFlash, ExifCFAPatternType, ExifDeviceSettingDescription)

// NewExifSchema returns the exif schema.
func NewExifSchema(metadata xmptype.MetadataLike) (*ExifSchema, error) {
	return NewExifSchemaPrefixed(metadata, "")
}

// NewExifSchemaPrefixed returns the exif schema with the given prefix.
func NewExifSchemaPrefixed(metadata xmptype.MetadataLike,
	ownPrefix string) (*ExifSchema, error) {
	s := &ExifSchema{}
	s.setDescription(exifProperties, "ExifSchema")
	if err := s.InitSchema(metadata, exifInfo, "", ownPrefix, ""); err != nil {
		return nil, err
	}
	return s, nil
}

// newExifSchemaAs is the factory's constructor.
func newExifSchemaAs(metadata xmptype.MetadataLike, prefix string) (Schema, error) {
	return NewExifSchemaPrefixed(metadata, prefix)
}

// UserCommentProperty returns the user comment alternative, or nil.
func (s *ExifSchema) UserCommentProperty() *xmptype.ArrayProperty {
	return PropertyAs[*xmptype.ArrayProperty](&s.XMPSchema, ExifUserComment)
}

// UserCommentLanguages returns the languages the user comment is written in,
// and nil where the property is absent.
func (s *ExifSchema) UserCommentLanguages() ([]string, error) {
	return s.UnqualifiedLanguagePropertyLanguagesValue(ExifUserComment)
}

// UserCommentIn returns the user comment in the given language.
func (s *ExifSchema) UserCommentIn(lang string) (string, error) {
	return s.UnqualifiedLanguagePropertyValue(ExifUserComment, lang)
}

// UserComment returns the user comment in the default language.
func (s *ExifSchema) UserComment() (string, error) { return s.UserCommentIn("") }
