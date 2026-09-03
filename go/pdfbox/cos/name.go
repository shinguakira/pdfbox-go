package cos

import (
	"bytes"
	"io"
	"runtime"
	"sync"
	"unicode/utf8"
	"weak"
)

// Name is a PDF name object, the /Type or /Kids that keys a dictionary.
//
// Port of org.apache.pdfbox.cos.COSName. Names are interned: two calls with the
// same bytes return the same *Name, so a *Name can be compared with == and used
// directly as a Go map key. Dictionary relies on that.
//
// A name holds raw bytes rather than a string. PDF names are byte sequences and
// need not be valid UTF-8, and writePDF has to reproduce the original bytes
// exactly — see the PDFBOX-6178 cases in name_test.go.
type Name struct {
	object
	nameBytes []byte
}

var _ Base = (*Name)(nil)

// The intern table. Java uses a ConcurrentHashMap of WeakReference plus a
// Cleaner so that names parsed out of one document do not pin memory for the
// life of the process; weak.Pointer and runtime.AddCleanup are the direct
// equivalents.
var (
	nameMapMu sync.Mutex
	nameMap   = make(map[string]weak.Pointer[Name])
)

// GetPDFName returns the interned name for the given string.
//
// Port of COSName.getPDFName(String). Well-formed PDF names are ASCII, for
// which the UTF-8 encoding here is an identity transform.
func GetPDFName(name string) *Name {
	return internName(name)
}

// GetPDFNameBytes returns the interned name for exactly these bytes.
//
// Port of COSName.getPDFName(byte[]). This is the factory the parser uses,
// after it has stripped the leading '/' and expanded any #XX escapes, because
// it preserves byte-level identity even when the bytes are not valid UTF-8.
// The slice is copied, so the caller may reuse it.
func GetPDFNameBytes(b []byte) *Name {
	return internName(string(b))
}

func internName(key string) *Name {
	nameMapMu.Lock()
	defer nameMapMu.Unlock()

	if wp, ok := nameMap[key]; ok {
		if n := wp.Value(); n != nil {
			return n
		}
	}

	n := &Name{nameBytes: []byte(key)}
	nameMap[key] = weak.Make(n)
	runtime.AddCleanup(n, func(k string) {
		nameMapMu.Lock()
		defer nameMapMu.Unlock()
		// Only drop the entry if it is still the dead one. A new name with the
		// same bytes may have been interned between collection and cleanup.
		if wp, ok := nameMap[k]; ok && wp.Value() == nil {
			delete(nameMap, k)
		}
	}, key)
	return n
}

// Name returns the name as a string.
//
// Port of getName(). Bytes that are not valid UTF-8 fall back to ISO-8859-1,
// which can decode any byte sequence, rather than producing U+FFFD. Java
// detects this by looking for U+FFFD in the decoded string; Go checks validity
// directly, which is the same test done earlier.
func (n *Name) Name() string {
	if utf8.Valid(n.nameBytes) {
		return string(n.nameBytes)
	}
	runes := make([]rune, len(n.nameBytes))
	for i, b := range n.nameBytes {
		runes[i] = rune(b)
	}
	return string(runes)
}

// Bytes returns the raw bytes of the name.
//
// Port of getBytes(), which returns the internal array directly. Names are
// interned and shared, so a caller that writes to the returned slice corrupts
// the name for every holder of it. That hazard exists in Java too — the byte
// array it hands back is equally shared and mutable — and it is carried over
// rather than closed, so that the two behave the same. Do not write to this.
func (n *Name) Bytes() []byte {
	return n.nameBytes
}

// IsEmpty reports whether the name is the empty string.
func (n *Name) IsEmpty() bool { return len(n.nameBytes) == 0 }

// COSObject returns the receiver.
func (n *Name) COSObject() Base { return n }

// Accept dispatches to the visitor.
func (n *Name) Accept(v Visitor) error { return v.VisitName(n) }

// Equals reports whether two names have the same bytes.
//
// Because names are interned, n == other is equivalent and cheaper. This method
// exists for parity with the Java API and for callers holding a Base.
func (n *Name) Equals(other *Name) bool {
	return other != nil && bytes.Equal(n.nameBytes, other.nameBytes)
}

// Compare orders names by their raw bytes, as Java's Arrays.compare does.
func (n *Name) Compare(other *Name) int {
	return bytes.Compare(n.nameBytes, other.nameBytes)
}

// String returns the Java toString form.
func (n *Name) String() string { return "COSName{" + n.Name() + "}" }

const hexDigits = "0123456789ABCDEF"

// WritePDF writes the name as a PDF object, escaping anything outside the
// permitted set as #XX.
//
// The permitted set is deliberately more restrictive than the PDF
// specification allows; see PDFBOX-2073 in COSName.writePDF.
func (n *Name) WritePDF(w io.Writer) error {
	buf := make([]byte, 0, len(n.nameBytes)+1)
	buf = append(buf, '/')
	for _, b := range n.nameBytes {
		switch {
		case b >= 'A' && b <= 'Z',
			b >= 'a' && b <= 'z',
			b >= '0' && b <= '9',
			b == '+', b == '-', b == '_', b == '@',
			b == '*', b == '$', b == ';', b == '.':
			buf = append(buf, b)
		default:
			buf = append(buf, '#', hexDigits[b>>4], hexDigits[b&0x0F])
		}
	}
	_, err := w.Write(buf)
	return err
}
