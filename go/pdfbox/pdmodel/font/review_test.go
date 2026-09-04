package font

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
)

// Tests for the port defects the slice 4 adversarial review found. Each fails
// without its fix.

// TestSpaceWidthFromToUnicodeCMap pins the branch of PDFont.getSpaceWidth the
// port had left out:
//
//	if (toUnicodeCMap != null && dict.containsKey(COSName.TO_UNICODE))
//	{
//	    int spaceMapping = toUnicodeCMap.getSpaceMapping();
//	    if (spaceMapping > -1)
//	    {
//	        fontWidthOfSpace = getWidth(spaceMapping);
//	    }
//	}
//
// The font below maps code 5 to U+0020 through its /ToUnicode CMap, and gives
// code 5 and code 32 different widths. Java measures the space at code 5; the
// port had always measured it at code 32, through the encoding.
func TestSpaceWidthFromToUnicodeCMap(t *testing.T) {
	const toUnicode = `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CMapName /Test def
/CMapType 2 def
1 begincodespacerange
<00> <ff>
endcodespacerange
1 beginbfchar
<05> <0020>
endbfchar
endcmap
CMapName currentdict /CMap defineresource pop
end
end`

	stream := cos.NewStream(filter.Provider{})
	writer, err := stream.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	if _, err := writer.Write([]byte(toUnicode)); err != nil {
		t.Fatalf("writing the CMap: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing the CMap: %v", err)
	}

	// codes 5..32, so that /Widths covers both the space mapping and code 32
	widths := cos.NewArray()
	for code := 5; code <= 32; code++ {
		switch code {
		case 5:
			widths.Add(cos.NewFloat(999))
		default:
			widths.Add(cos.NewFloat(500))
		}
	}

	dict := cos.NewDictionary()
	dict.SetItem(cos.Type, cos.Font)
	dict.SetItem(cos.Subtype, cos.Type1)
	dict.SetName(cos.BaseFont, "Helvetica")
	dict.SetItem(cos.Encoding, cos.WinAnsiEncoding)
	dict.SetInt(cos.FirstChar, 5)
	dict.SetInt(cos.LastChar, 32)
	dict.SetItem(cos.Widths, widths)
	dict.SetItem(cos.ToUnicode, stream)

	font, err := NewPDType1FontFromDictionary(dict, nil)
	if err != nil {
		t.Fatalf("reading the font: %v", err)
	}

	if got := font.SpaceWidth(); got != 999 {
		t.Errorf("SpaceWidth = %v, want 999 (the width of the code the ToUnicode CMap "+
			"maps to U+0020)", got)
	}
}
