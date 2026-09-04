package font

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox"
)

// Port of org.apache.pdfbox.pdmodel.font.CIDCharSetMatchTest.
//
// CIDSystemInfo-based candidate filtering for CID font substitution.
//
// isCharSetMatch is package-private in Java and unexported here, so the test
// lives in the package rather than beside it.

const (
	testChineseTraditional = 1 << 20
	testChineseSimplified  = 1 << 18
)

// charSetMatchFontInfo stands for the anonymous FontInfo the Java test builds.
type charSetMatchFontInfo struct {
	ros            *CIDSystemInfo
	codePageRange1 int
}

var _ FontInfo = (*charSetMatchFontInfo)(nil)

func (i *charSetMatchFontInfo) PostScriptName() string          { return "TestFont" }
func (i *charSetMatchFontInfo) Format() FontFormat              { return FontFormatOTF }
func (i *charSetMatchFontInfo) CIDSystemInfo() *CIDSystemInfo   { return i.ros }
func (i *charSetMatchFontInfo) Font() fontbox.FontBoxFont       { return nil }
func (i *charSetMatchFontInfo) FamilyClass() int                { return 0 }
func (i *charSetMatchFontInfo) WeightClass() int                { return 0 }
func (i *charSetMatchFontInfo) CodePageRange1() int             { return i.codePageRange1 }
func (i *charSetMatchFontInfo) CodePageRange2() int             { return 0 }
func (i *charSetMatchFontInfo) MacStyle() int                   { return 0 }
func (i *charSetMatchFontInfo) Panose() *PDPanoseClassification { return nil }

// charSetMatchInfo is the Java test's info(ros, codePageRange1) factory.
func charSetMatchInfo(ros *CIDSystemInfo, codePageRange1 int) FontInfo {
	return &charSetMatchFontInfo{ros: ros, codePageRange1: codePageRange1}
}

func TestCharSetMatch(t *testing.T) {
	mapper := newFontMapperImpl()
	cns1 := NewPDCIDSystemInfo("Adobe", "CNS1", 0)

	// exact ROS match
	if !mapper.isCharSetMatch(cns1,
		charSetMatchInfo(NewCIDSystemInfo("Adobe", "CNS1", 0), 0)) {
		t.Error("an exact ROS match should match")
	}

	// a different legacy ROS never matches
	if mapper.isCharSetMatch(cns1,
		charSetMatchInfo(NewCIDSystemInfo("Adobe", "Japan1", 0), testChineseTraditional)) {
		t.Error("a different legacy ROS should not match")
	}

	// Adobe-Identity-0 (Noto CJK, Source Han) matches via its OS/2 code page bits
	if !mapper.isCharSetMatch(cns1,
		charSetMatchInfo(NewCIDSystemInfo("Adobe", "Identity", 0), testChineseTraditional)) {
		t.Error("Adobe-Identity-0 with the traditional Chinese bit should match")
	}
	if mapper.isCharSetMatch(cns1,
		charSetMatchInfo(NewCIDSystemInfo("Adobe", "Identity", 0), testChineseSimplified)) {
		t.Error("Adobe-Identity-0 with only the simplified Chinese bit should not match")
	}

	// ROS-less TrueType fonts keep matching via code page bits
	if !mapper.isCharSetMatch(cns1, charSetMatchInfo(nil, testChineseTraditional)) {
		t.Error("a ROS-less font with the traditional Chinese bit should match")
	}
	if mapper.isCharSetMatch(cns1, charSetMatchInfo(nil, 0)) {
		t.Error("a ROS-less font with no code page bits should not match")
	}
}
