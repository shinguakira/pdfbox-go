package filespecification

import (
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDEmbeddedFile is a file embedded in a document.
//
// Port of PDEmbeddedFile, which extends PDStream.
//
// The four date accessors --- getCreationDate, setCreationDate, getModDate and
// setModDate --- are not here. They go through COSDictionary's embedded date
// accessors, which are the other half of DateConverter; slice 1 deferred both,
// and migration/tasks/README.md assigns DateConverter to this slice. They land
// with it. See migration/STATUS.md.
type PDEmbeddedFile struct {
	common.PDStream
}

var _ common.COSObjectable = (*PDEmbeddedFile)(nil)

// NewPDEmbeddedFile creates a new empty embedded file in the given document.
func NewPDEmbeddedFile(document common.COSDocumentLike) *PDEmbeddedFile {
	f := &PDEmbeddedFile{PDStream: *common.NewPDStreamOfDocument(document)}
	f.Stream().SetItem(cos.Type, cos.EmbeddedFile)
	return f
}

// NewPDEmbeddedFileOfStream wraps an existing stream.
func NewPDEmbeddedFileOfStream(str *cos.Stream) *PDEmbeddedFile {
	return &PDEmbeddedFile{PDStream: *common.NewPDStream(str)}
}

// NewPDEmbeddedFileOfInput creates an embedded file holding what the reader
// gives.
func NewPDEmbeddedFileOfInput(doc common.COSDocumentLike, str io.Reader) (*PDEmbeddedFile, error) {
	return NewPDEmbeddedFileOfInputFiltered(doc, str, nil)
}

// NewPDEmbeddedFileOfInputFiltered creates an embedded file holding what the
// reader gives, encoded with the given filter.
func NewPDEmbeddedFileOfInputFiltered(doc common.COSDocumentLike, input io.Reader,
	filter *cos.Name) (*PDEmbeddedFile, error) {
	var filters cos.Base
	if filter != nil {
		filters = filter
	}
	stream, err := common.NewPDStreamOfInput(doc, input, filters)
	if err != nil {
		return nil, err
	}
	f := &PDEmbeddedFile{PDStream: *stream}
	f.Stream().SetItem(cos.Type, cos.EmbeddedFile)
	return f, nil
}

// SetSubtype sets the /Subtype, which is the media type of the file.
func (f *PDEmbeddedFile) SetSubtype(mimeType string) {
	f.Stream().SetName(cos.Subtype, mimeType)
}

// Subtype returns the /Subtype.
func (f *PDEmbeddedFile) Subtype() string {
	return f.Stream().GetNameAsString(cos.Subtype, "")
}

// Size returns the /Params /Size entry.
func (f *PDEmbeddedFile) Size() int {
	return f.Stream().GetEmbeddedInt(cos.Params, cos.Size, -1)
}

// SetSize sets the /Params /Size entry.
func (f *PDEmbeddedFile) SetSize(size int) {
	f.Stream().SetEmbeddedInt(cos.Params, cos.Size, size)
}

// CheckSum returns the /Params /CheckSum entry.
func (f *PDEmbeddedFile) CheckSum() string {
	return f.Stream().GetEmbeddedString(cos.Params, cos.CheckSum, "")
}

// SetCheckSum sets the /Params /CheckSum entry.
func (f *PDEmbeddedFile) SetCheckSum(checksum string) {
	f.Stream().SetEmbeddedString(cos.Params, cos.CheckSum, checksum)
}

// MacSubtype returns the /Params /Mac /Subtype entry.
func (f *PDEmbeddedFile) MacSubtype() string {
	if params := f.Stream().GetCOSDictionary(cos.Params); params != nil {
		return params.GetEmbeddedString(cos.Mac, cos.Subtype, "")
	}
	return ""
}

// SetMacSubtype sets the /Params /Mac /Subtype entry.
func (f *PDEmbeddedFile) SetMacSubtype(macSubtype string) {
	f.setMacString(cos.Subtype, macSubtype)
}

// MacCreator returns the /Params /Mac /Creator entry.
func (f *PDEmbeddedFile) MacCreator() string {
	if params := f.Stream().GetCOSDictionary(cos.Params); params != nil {
		return params.GetEmbeddedString(cos.Mac, cos.Creator, "")
	}
	return ""
}

// SetMacCreator sets the /Params /Mac /Creator entry.
func (f *PDEmbeddedFile) SetMacCreator(macCreator string) {
	f.setMacString(cos.Creator, macCreator)
}

// MacResFork returns the /Params /Mac /ResFork entry.
func (f *PDEmbeddedFile) MacResFork() string {
	if params := f.Stream().GetCOSDictionary(cos.Params); params != nil {
		return params.GetEmbeddedString(cos.Mac, cos.ResFork, "")
	}
	return ""
}

// SetMacResFork sets the /Params /Mac /ResFork entry.
func (f *PDEmbeddedFile) SetMacResFork(macResFork string) {
	f.setMacString(cos.ResFork, macResFork)
}

// setMacString is the body the three /Mac setters share. The empty string is
// Java's null, which leaves an absent /Params alone.
func (f *PDEmbeddedFile) setMacString(key *cos.Name, value string) {
	params := f.Stream().GetCOSDictionary(cos.Params)
	if params == nil && value != "" {
		params = cos.NewDictionary()
		f.Stream().SetItem(cos.Params, params)
	}
	if params != nil {
		params.SetEmbeddedString(cos.Mac, key, value)
	}
}
