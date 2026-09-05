// Package digitalsignature models the digital signatures of a document.
//
// Port of org.apache.pdfbox.pdmodel.interactive.digitalsignature.
package digitalsignature

import (
	"bytes"
	"io"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// The signature filter values.
//
// Port of the FILTER_ constants of PDSignature.
var (
	FilterAdobePPKLite         = cos.AdobePPKLite
	FilterEntrustPPKEF         = cos.EntrustPPKEF
	FilterCICISignIt           = cos.CICISignIt
	FilterVeriSignPPKVS        = cos.VeriSignPPKVS
	SubFilterAdbeX509RSASha1   = cos.AdbeX509RsaSha1
	SubFilterAdbePkcs7Detached = cos.AdbePkcs7Detached
	SubFilterETSICAdESDetached = cos.GetPDFName("ETSI.CAdES.detached")
	SubFilterAdbePkcs7Sha1     = cos.AdbePkcs7Sha1
)

// PDSignature is a digital signature that can be attached to a document.
//
// To learn more about digital signatures, read Digital Signatures in a PDF by
// Adobe, at
// https://www.adobe.com/devnet-docs/acrobatetk/tools/DigSig/Acrobat_DigitalSignatures_in_PDF.pdf
//
// Port of PDSignature.
type PDSignature struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDSignature)(nil)

// NewPDSignature returns a new signature dictionary.
func NewPDSignature() *PDSignature {
	dictionary := cos.NewDictionary()
	dictionary.SetItem(cos.Type, cos.Sig)
	return &PDSignature{dictionary: dictionary}
}

// NewPDSignatureOf returns the signature the given dictionary holds.
func NewPDSignatureOf(dict *cos.Dictionary) *PDSignature {
	return &PDSignature{dictionary: dict}
}

// COSObject returns the signature dictionary.
func (s *PDSignature) COSObject() cos.Base { return s.dictionary }

// Dictionary returns the signature dictionary, typed.
func (s *PDSignature) Dictionary() *cos.Dictionary { return s.dictionary }

// SetType sets the /Type of the dictionary.
func (s *PDSignature) SetType(sigType *cos.Name) { s.dictionary.SetItem(cos.Type, sigType) }

// SetFilter sets the filter to be used.
func (s *PDSignature) SetFilter(filter *cos.Name) { s.dictionary.SetItem(cos.Filter, filter) }

// SetSubFilter sets the subfilter that says which signature shall be used.
func (s *PDSignature) SetSubFilter(subfilter *cos.Name) {
	s.dictionary.SetItem(cos.SubFilter, subfilter)
}

// SetName sets the name of the person or authority signing the document.
// According to the PDF specification, this value should be used only when it is
// not possible to extract the name from the signature.
func (s *PDSignature) SetName(name string) { s.dictionary.SetString(cos.NameKey, name) }

// SetLocation sets the CPU host name or physical location of the signing.
func (s *PDSignature) SetLocation(location string) {
	s.dictionary.SetString(cos.Location, location)
}

// SetReason sets the reason for the signing, such as (I agree...).
func (s *PDSignature) SetReason(reason string) { s.dictionary.SetString(cos.Reason, reason) }

// SetContactInfo sets the contact info provided by the signer to enable a
// recipient to contact the signer to verify the signature, e.g. a phone number.
func (s *PDSignature) SetContactInfo(contactInfo string) {
	s.dictionary.SetString(cos.ContactInfo, contactInfo)
}

// SetSignDate sets the date of the signing.
func (s *PDSignature) SetSignDate(cal time.Time) {
	util.SetDictionaryDate(s.dictionary, cos.M, cal)
}

// Filter returns the filter, or the empty string where there is none, which is
// the null Java answers.
func (s *PDSignature) Filter() string { return s.dictionary.GetNameAsString(cos.Filter, "") }

// SubFilter returns the subfilter, or the empty string where there is none.
func (s *PDSignature) SubFilter() string { return s.dictionary.GetNameAsString(cos.SubFilter, "") }

// Name returns the name of the person or authority signing the document.
// According to the PDF specification, this value should be used only when it is
// not possible to extract the name from the signature.
func (s *PDSignature) Name() string { return s.dictionary.GetString(cos.NameKey, "") }

// Location returns the CPU host name or physical location of the signing.
func (s *PDSignature) Location() string { return s.dictionary.GetString(cos.Location, "") }

// Reason returns the reason for the signing, such as (I agree...).
func (s *PDSignature) Reason() string { return s.dictionary.GetString(cos.Reason, "") }

// ContactInfo returns the contact info provided by the signer to enable a
// recipient to contact the signer to verify the signature, e.g. a phone number.
func (s *PDSignature) ContactInfo() string { return s.dictionary.GetString(cos.ContactInfo, "") }

// SignDate returns the date of the signing, and reports whether the dictionary
// carries one, which is the null Java answers.
func (s *PDSignature) SignDate() (time.Time, bool) {
	return util.DictionaryDate(s.dictionary, cos.M)
}

// SetByteRange sets the byte range, and does nothing where it does not have
// four entries.
func (s *PDSignature) SetByteRange(byteRange []int) {
	if len(byteRange) != 4 {
		return
	}
	ary := cos.NewArray()
	for _, i := range byteRange {
		ary.Add(cos.GetInteger(int64(i)))
	}

	s.dictionary.SetItem(cos.ByteRange, ary)
	ary.SetDirect(true)
}

// ByteRange reads the byte range out of the file, and answers an empty slice
// where there is none.
func (s *PDSignature) ByteRange() []int {
	byteRange := s.dictionary.GetCOSArray(cos.ByteRange)
	if byteRange == nil {
		return []int{}
	}
	ary := make([]int, byteRange.Size())
	for i := 0; i < len(ary); i++ {
		ary[i] = byteRange.GetInt(i)
	}
	return ary
}

// Contents returns the /Contents string as bytes, that is, the embedded
// signature between the byte range gap, and an empty slice where there is none.
func (s *PDSignature) Contents() []byte {
	base := s.dictionary.GetDictionaryObject(cos.Contents)
	if str, isString := base.(*cos.StringObj); isString {
		return str.Bytes()
	}
	return []byte{}
}

// ContentsOfReader returns the embedded signature between the byte range gap,
// reading it out of the given signed PDF.
//
// Java takes an InputStream and closes it here; the port takes a reader and
// leaves closing to the caller, which is what Go expects of one.
//
// Java throws IndexOutOfBoundsException where the byte range array is not long
// enough, which is unchecked, so the port panics.
func (s *PDSignature) ContentsOfReader(pdfFile io.Reader) ([]byte, error) {
	byteRange := s.ByteRange()
	begin := byteRange[0] + byteRange[1] + 1
	length := byteRange[2] - begin

	filtered, err := NewCOSFilterInputStream(pdfFile, []int{begin, length})
	if err != nil {
		return nil, err
	}
	return convertedContents(filtered)
}

// ContentsOfBytes returns the embedded signature between the byte range gap of
// the given signed PDF.
//
// Java throws IndexOutOfBoundsException where the byte range array is not long
// enough, which is unchecked, so the port panics.
func (s *PDSignature) ContentsOfBytes(pdfFile []byte) ([]byte, error) {
	byteRange := s.ByteRange()
	begin := byteRange[0] + byteRange[1] + 1
	length := byteRange[2] - begin - 1

	return convertedContents(bytes.NewReader(pdfFile[begin : begin+length]))
}

// convertedContents reads the hex string the signature is written as and
// returns the bytes it stands for. Java declares it private.
func convertedContents(is io.Reader) ([]byte, error) {
	baos := &bytes.Buffer{}
	buffer := make([]byte, 1024)
	for {
		readLen, err := is.Read(buffer)
		if readLen > 0 {
			writeLen := readLen
			start := 0
			// Filter < and (
			if buffer[0] == 0x3C || buffer[0] == 0x28 {
				start++
				writeLen--
			}
			// Filter > and ) at the end
			if buffer[readLen-1] == 0x3E || buffer[readLen-1] == 0x29 {
				writeLen--
			}
			baos.Write(buffer[start : start+writeLen])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	// Java reads the buffer back as ISO-8859-1, which maps each byte to the
	// code point of the same value; parseHex then ignores everything that is
	// not a hex digit, so the bytes go in as they are.
	parsed, err := cos.ParseHexString(latin1String(baos.Bytes()))
	if err != nil {
		return nil, err
	}
	return parsed.Bytes(), nil
}

// latin1String decodes bytes the way new String(bytes, ISO_8859_1) does.
func latin1String(b []byte) string {
	runes := make([]rune, len(b))
	for i, c := range b {
		runes[i] = rune(c)
	}
	return string(runes)
}

// SetContents sets the contents.
func (s *PDSignature) SetContents(b []byte) {
	str := cos.NewStringObjBytesHex(b, true)
	s.dictionary.SetItem(cos.Contents, str)
}

// SignedContentOfReader returns the signed content of the document. This is not
// a PDF file, nor is it the PDF file before signing, it is the byte sequence
// made of the input minus the area where the signature bytes will be. See "The
// ByteRange and signature value" in Digital Signatures in a PDF, at
// https://www.adobe.com/content/dam/acom/en/devnet/acrobat/pdfs/DigitalSignaturesInPDF.pdf#page=5
//
// Java takes an InputStream and closes it here; the port takes a reader and
// leaves closing to the caller.
func (s *PDSignature) SignedContentOfReader(pdfFile io.Reader) ([]byte, error) {
	fis, err := NewCOSFilterInputStream(pdfFile, s.ByteRange())
	if err != nil {
		return nil, err
	}
	return fis.ToByteArray()
}

// SignedContentOfBytes returns the signed content of the given signed PDF; see
// SignedContentOfReader.
func (s *PDSignature) SignedContentOfBytes(pdfFile []byte) ([]byte, error) {
	fis, err := NewCOSFilterInputStream(bytes.NewReader(pdfFile), s.ByteRange())
	if err != nil {
		return nil, err
	}
	return fis.ToByteArray()
}

// PropBuild returns the PDF signature build dictionary, which says which
// signature handler was used, or nil where there is none.
func (s *PDSignature) PropBuild() *PDPropBuild {
	var propBuild *PDPropBuild
	propBuildDic := s.dictionary.GetCOSDictionary(cos.PropBuild)
	if propBuildDic != nil {
		propBuild = NewPDPropBuildOf(propBuildDic)
	}
	return propBuild
}

// SetPropBuild sets the PDF signature build dictionary.
func (s *PDSignature) SetPropBuild(propBuild *PDPropBuild) {
	s.dictionary.SetItem(cos.PropBuild, common.COSObjectOrNil(propBuild))
}
