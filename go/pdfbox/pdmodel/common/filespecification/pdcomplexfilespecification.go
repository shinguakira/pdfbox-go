package filespecification

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDComplexFileSpecification is a file specification dictionary, which can name
// the file several ways and can carry the file itself.
//
// Port of PDComplexFileSpecification.
type PDComplexFileSpecification struct {
	fs           *cos.Dictionary
	efDictionary *cos.Dictionary
}

var _ PDFileSpecification = (*PDComplexFileSpecification)(nil)

// NewPDComplexFileSpecification creates a specification over the given
// dictionary, or over a fresh one where it is nil.
//
// Java has a no-argument constructor and a COSDictionary one whose body is the
// no-argument one when the argument is null; the port is the second, since a
// Go caller passes nil for the first.
func NewPDComplexFileSpecification(dict *cos.Dictionary) *PDComplexFileSpecification {
	if dict == nil {
		fs := cos.NewDictionary()
		fs.SetItem(cos.Type, cos.Filespec)
		return &PDComplexFileSpecification{fs: fs}
	}
	return &PDComplexFileSpecification{fs: dict}
}

// COSObject returns the dictionary.
func (f *PDComplexFileSpecification) COSObject() cos.Base { return f.fs }

// Dictionary returns the dictionary, typed.
func (f *PDComplexFileSpecification) Dictionary() *cos.Dictionary { return f.fs }

// efDictionaryOf returns the /EF dictionary, caching it as Java does.
func (f *PDComplexFileSpecification) efDictionaryOf() *cos.Dictionary {
	if f.efDictionary == nil && f.fs != nil {
		f.efDictionary = f.fs.GetCOSDictionary(cos.EF)
	}
	return f.efDictionary
}

func (f *PDComplexFileSpecification) objectFromEFDictionary(key *cos.Name) cos.Base {
	if ef := f.efDictionaryOf(); ef != nil {
		return ef.GetDictionaryObject(key)
	}
	return nil
}

// Filename returns the first file name this specification carries, trying the
// unicode, DOS, Mac, Unix and plain entries in that order.
func (f *PDComplexFileSpecification) Filename() string {
	filename := f.FileUnicode()
	if filename == "" {
		filename = f.FileDos()
	}
	if filename == "" {
		filename = f.FileMac()
	}
	if filename == "" {
		filename = f.FileUnix()
	}
	if filename == "" {
		filename = f.File()
	}
	return filename
}

// FileUnicode returns the /UF entry.
func (f *PDComplexFileSpecification) FileUnicode() string {
	return f.fs.GetString(cos.UF, "")
}

// SetFileUnicode sets the /UF entry.
func (f *PDComplexFileSpecification) SetFileUnicode(file string) {
	f.fs.SetString(cos.UF, file)
}

// File returns the /F entry.
func (f *PDComplexFileSpecification) File() string {
	return f.fs.GetString(cos.F, "")
}

// SetFile sets the /F entry.
func (f *PDComplexFileSpecification) SetFile(file string) {
	f.fs.SetString(cos.F, file)
}

// FileDos returns the /DOS entry.
func (f *PDComplexFileSpecification) FileDos() string {
	return f.fs.GetString(cos.DOS, "")
}

// FileMac returns the /Mac entry.
func (f *PDComplexFileSpecification) FileMac() string {
	return f.fs.GetString(cos.Mac, "")
}

// FileUnix returns the /Unix entry.
func (f *PDComplexFileSpecification) FileUnix() string {
	return f.fs.GetString(cos.Unix, "")
}

// SetVolatile sets whether the file changes as it is read.
func (f *PDComplexFileSpecification) SetVolatile(fileIsVolatile bool) {
	f.fs.SetBoolean(cos.V, fileIsVolatile)
}

// IsVolatile reports whether the file changes as it is read.
func (f *PDComplexFileSpecification) IsVolatile() bool {
	return f.fs.GetBoolean(cos.V, false)
}

// EmbeddedFile returns the file embedded under /EF /F, or nil.
func (f *PDComplexFileSpecification) EmbeddedFile() *PDEmbeddedFile {
	return embeddedFileOf(f.objectFromEFDictionary(cos.F))
}

// embeddedFileOf is the `base instanceof COSStream` the five getters share.
func embeddedFileOf(base cos.Base) *PDEmbeddedFile {
	if stream, ok := base.(*cos.Stream); ok {
		return NewPDEmbeddedFileOfStream(stream)
	}
	return nil
}

// SetEmbeddedFile stores the file under /EF /F.
func (f *PDComplexFileSpecification) SetEmbeddedFile(file *PDEmbeddedFile) {
	f.setEmbedded(cos.F, file)
}

// EmbeddedFileDos returns the file embedded under /EF /DOS, or nil.
func (f *PDComplexFileSpecification) EmbeddedFileDos() *PDEmbeddedFile {
	return embeddedFileOf(f.objectFromEFDictionary(cos.DOS))
}

// EmbeddedFileMac returns the file embedded under /EF /Mac, or nil.
func (f *PDComplexFileSpecification) EmbeddedFileMac() *PDEmbeddedFile {
	return embeddedFileOf(f.objectFromEFDictionary(cos.Mac))
}

// EmbeddedFileUnix returns the file embedded under /EF /Unix, or nil.
func (f *PDComplexFileSpecification) EmbeddedFileUnix() *PDEmbeddedFile {
	return embeddedFileOf(f.objectFromEFDictionary(cos.Unix))
}

// EmbeddedFileUnicode returns the file embedded under /EF /UF, or nil.
func (f *PDComplexFileSpecification) EmbeddedFileUnicode() *PDEmbeddedFile {
	return embeddedFileOf(f.objectFromEFDictionary(cos.UF))
}

// SetEmbeddedFileUnicode stores the file under /EF /UF.
func (f *PDComplexFileSpecification) SetEmbeddedFileUnicode(file *PDEmbeddedFile) {
	f.setEmbedded(cos.UF, file)
}

// setEmbedded is the body the two embedded-file setters share.
func (f *PDComplexFileSpecification) setEmbedded(key *cos.Name, file *PDEmbeddedFile) {
	ef := f.efDictionaryOf()
	if ef == nil && file != nil {
		ef = cos.NewDictionary()
		f.fs.SetItem(cos.EF, ef)
		f.efDictionary = ef
	}
	if ef != nil {
		if file == nil {
			ef.SetItem(key, nil)
		} else {
			ef.SetItem(key, file.COSObject())
		}
	}
}

// SetFileDescription sets the /Desc entry.
func (f *PDComplexFileSpecification) SetFileDescription(description string) {
	f.fs.SetString(cos.Desc, description)
}

// FileDescription returns the /Desc entry.
func (f *PDComplexFileSpecification) FileDescription() string {
	return f.fs.GetString(cos.Desc, "")
}
