package encoding

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Port of org.apache.pdfbox.pdmodel.font.TestFontEncoding.
//
// testPDFBox3884 is not ported: it builds a document, saves it, loads it back
// and runs the text stripper over it, none of which this slice carries. See
// migration/tasks/slice-3-text-simple-fonts.md.

// TestAdd tests the add method of a font encoding.
func TestAdd(t *testing.T) {
	// see PDFDBOX-3332
	codeForSpace, ok := WinAnsiEncodingInstance.NameToCodeMap()["space"]
	if !ok {
		t.Fatal("WinAnsiEncoding has no code for space")
	}
	if codeForSpace != 32 {
		t.Errorf("WinAnsi code for space = %d, want 32", codeForSpace)
	}

	codeForSpace, ok = MacRomanEncodingInstance.NameToCodeMap()["space"]
	if !ok {
		t.Fatal("MacRomanEncoding has no code for space")
	}
	if codeForSpace != 32 {
		t.Errorf("MacRoman code for space = %d, want 32", codeForSpace)
	}
}

func TestOverwrite(t *testing.T) {
	// see PDFDBOX-3332
	dictEncodingDict := cos.NewDictionary()
	dictEncodingDict.SetItem(cos.Type, cos.Encoding)
	dictEncodingDict.SetItem(cos.BaseEncoding, cos.WinAnsiEncoding)
	differences := cos.NewArray()
	differences.Add(cos.GetInteger(32))
	differences.Add(cos.GetPDFName("a"))
	dictEncodingDict.SetItem(cos.Differences, differences)

	dictEncoding, err := NewDictionaryEncodingWithBuiltIn(dictEncodingDict, false, nil)
	if err != nil {
		t.Fatalf("new dictionary encoding: %v", err)
	}
	if _, ok := dictEncoding.NameToCodeMap()["space"]; ok {
		t.Error("space is still in the name to code map")
	}
	code, ok := dictEncoding.NameToCodeMap()["a"]
	if !ok {
		t.Fatal("a is not in the name to code map")
	}
	if code != 32 {
		t.Errorf("code for a = %d, want 32", code)
	}
}
