// Package markedcontent holds the marked content a page's operators declare.
//
// Port of org.apache.pdfbox.pdmodel.documentinterchange.markedcontent.
package markedcontent

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDMarkedContent is one marked content sequence of a content stream.
//
// Port of
// org.apache.pdfbox.pdmodel.documentinterchange.markedcontent.PDMarkedContent.
type PDMarkedContent struct {
	tag        string
	hasTag     bool
	properties *cos.Dictionary
	contents   []any
}

// Create returns the marked content the given tag and properties describe.
func Create(tag *cos.Name, properties *cos.Dictionary) *PDMarkedContent {
	if cos.Artifact.Equals(tag) {
		return NewPDArtifactMarkedContent(properties)
	}
	return NewPDMarkedContent(tag, properties)
}

// NewPDMarkedContent returns a marked content sequence.
func NewPDMarkedContent(tag *cos.Name, properties *cos.Dictionary) *PDMarkedContent {
	m := &PDMarkedContent{properties: properties, contents: []any{}}
	if tag != nil {
		m.tag = tag.Name()
		m.hasTag = true
	}
	return m
}

// NewPDArtifactMarkedContent returns the marked content of an artifact.
//
// Java has a PDArtifactMarkedContent subclass carrying the artifact's own
// properties; only its tag matters to the text extractor, so the port keeps the
// one type. See migration/STATUS.md.
func NewPDArtifactMarkedContent(properties *cos.Dictionary) *PDMarkedContent {
	return NewPDMarkedContent(cos.Artifact, properties)
}

// Tag returns the tag of the sequence. The second result is false where the
// sequence has none, which is the null Java returns.
func (m *PDMarkedContent) Tag() (string, bool) { return m.tag, m.hasTag }

// Properties returns the property dictionary of the sequence, or nil.
func (m *PDMarkedContent) Properties() *cos.Dictionary { return m.properties }

// MCID returns the marked content identifier, or -1 where there is none.
func (m *PDMarkedContent) MCID() int {
	if m.properties == nil {
		return -1
	}
	return m.properties.GetInt(cos.MCID)
}

// Language returns the language the content is in. The second result is false
// where the sequence does not say.
func (m *PDMarkedContent) Language() (string, bool) {
	return m.stringProperty(cos.Lang, true)
}

// ActualText returns the text the sequence stands for, which replaces whatever
// its glyphs would say. The second result is false where the sequence gives
// none.
func (m *PDMarkedContent) ActualText() (string, bool) {
	return m.stringProperty(cos.ActualText, false)
}

// AlternateDescription returns the description of the content for a reader that
// cannot show it.
func (m *PDMarkedContent) AlternateDescription() (string, bool) {
	return m.stringProperty(cos.Alt, false)
}

// ExpandedForm returns what an abbreviation in the content stands for.
func (m *PDMarkedContent) ExpandedForm() (string, bool) {
	return m.stringProperty(cos.E, false)
}

// stringProperty reads one property, asName saying whether it is a name rather
// than a string.
func (m *PDMarkedContent) stringProperty(key *cos.Name, asName bool) (string, bool) {
	if m.properties == nil {
		return "", false
	}
	if !m.properties.ContainsKey(key) {
		return "", false
	}
	if asName {
		return m.properties.GetNameAsString(key, ""), true
	}
	return m.properties.GetString(key, ""), true
}

// Contents returns what the sequence holds: text positions, nested sequences
// and XObjects.
//
// Java declares the list as List<Object> and puts all three in it; the port
// keeps that, which also breaks the cycle a typed AddText would make -- the
// text package names this one.
func (m *PDMarkedContent) Contents() []any { return m.contents }

// AddText adds a text position to the sequence.
func (m *PDMarkedContent) AddText(text any) { m.contents = append(m.contents, text) }

// AddMarkedContent adds a nested sequence.
func (m *PDMarkedContent) AddMarkedContent(markedContent *PDMarkedContent) {
	m.contents = append(m.contents, markedContent)
}

// AddXObject adds an XObject to the sequence.
func (m *PDMarkedContent) AddXObject(xobject any) {
	m.contents = append(m.contents, xobject)
}

// String returns the sequence written out.
func (m *PDMarkedContent) String() string {
	return fmt.Sprintf("tag=%s, properties=%v, contents=%v", m.tag, m.properties, m.contents)
}
