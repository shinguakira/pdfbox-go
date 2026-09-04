package logicalstructure

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// The keys of a mark information dictionary. Java writes them as bare strings,
// which COSDictionary interns through COSName.getPDFName.
var (
	keyMarked         = cos.GetPDFName("Marked")
	keyUserProperties = cos.GetPDFName("UserProperties")
	keySuspects       = cos.GetPDFName("Suspects")
)

// PDMarkInfo says whether a document is tagged.
//
// Port of PDMarkInfo.
type PDMarkInfo struct {
	dictionary *cos.Dictionary
}

var _ common.COSObjectable = (*PDMarkInfo)(nil)

// NewPDMarkInfo builds an empty mark information dictionary.
func NewPDMarkInfo() *PDMarkInfo {
	return &PDMarkInfo{dictionary: cos.NewDictionary()}
}

// NewPDMarkInfoOf builds one over the given dictionary.
func NewPDMarkInfoOf(dic *cos.Dictionary) *PDMarkInfo {
	return &PDMarkInfo{dictionary: dic}
}

// COSObject returns the dictionary.
func (m *PDMarkInfo) COSObject() cos.Base { return m.dictionary }

// Dictionary returns the dictionary, typed.
func (m *PDMarkInfo) Dictionary() *cos.Dictionary { return m.dictionary }

// IsMarked reports whether the document is marked.
func (m *PDMarkInfo) IsMarked() bool {
	return m.dictionary.GetBoolean(keyMarked, false)
}

// SetMarked sets whether the document is marked.
func (m *PDMarkInfo) SetMarked(value bool) {
	m.dictionary.SetBoolean(keyMarked, value)
}

// UsesUserProperties reports whether the document holds user properties.
func (m *PDMarkInfo) UsesUserProperties() bool {
	return m.dictionary.GetBoolean(keyUserProperties, false)
}

// SetUserProperties sets whether the document holds user properties.
func (m *PDMarkInfo) SetUserProperties(userProps bool) {
	m.dictionary.SetBoolean(keyUserProperties, userProps)
}

// IsSuspect reports whether the tagging is suspect.
func (m *PDMarkInfo) IsSuspect() bool {
	return m.dictionary.GetBoolean(keySuspects, false)
}

// SetSuspect sets whether the tagging is suspect.
//
// JAVA BUG: it ignores its argument and always writes false, so /Suspects can
// never be set through it. Ported as written; see migration/JAVA-BUGS.md entry 37.
func (m *PDMarkInfo) SetSuspect(suspect bool) {
	m.dictionary.SetBoolean(keySuspects, false)
}
