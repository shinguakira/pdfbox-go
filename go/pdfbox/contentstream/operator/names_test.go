package operator

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdfwriter/OperatorNameTest.java.
//
// The Java test sits in the pdfwriter test package because ContentStreamWriter
// is what calls getNameAsBytes; the Go test sits with the names it tests, which
// is where a Go reader looks for it.

import (
	"bytes"
	"testing"
)

// assertNameAsBytes is the assertArrayEquals(X.getBytes(US_ASCII),
// getNameAsBytes(X)) the Java test repeats for every operator.
func assertNameAsBytes(t *testing.T, operatorName string) {
	t.Helper()
	if got := GetNameAsBytes(operatorName); !bytes.Equal([]byte(operatorName), got) {
		t.Errorf("GetNameAsBytes(%q) = %q, want %q", operatorName, got, operatorName)
	}
}

func TestNameAsByteMappingNonStrokingColor(t *testing.T) {
	assertNameAsBytes(t, NonStrokingColor)
	assertNameAsBytes(t, NonStrokingColorN)
	assertNameAsBytes(t, NonStrokingRgb)
	assertNameAsBytes(t, NonStrokingGray)
	assertNameAsBytes(t, NonStrokingCmyk)
	assertNameAsBytes(t, NonStrokingColorspace)
}

func TestNameAsByteMappingStrokingColor(t *testing.T) {
	assertNameAsBytes(t, StrokingColor)
	assertNameAsBytes(t, StrokingColorN)
	assertNameAsBytes(t, StrokingColorRgb)
	assertNameAsBytes(t, StrokingColorGray)
	assertNameAsBytes(t, StrokingColorCmyk)
	assertNameAsBytes(t, StrokingColorspace)
}

func TestNameAsByteMappingMarkedContent(t *testing.T) {
	assertNameAsBytes(t, BeginMarkedContentSeq)
	assertNameAsBytes(t, BeginMarkedContent)
	assertNameAsBytes(t, EndMarkedContent)
	assertNameAsBytes(t, MarkedContentPointWithProps)
	assertNameAsBytes(t, MarkedContentPoint)
	assertNameAsBytes(t, DrawObject)
}

func TestNameAsByteMappingState(t *testing.T) {
	assertNameAsBytes(t, Concat)
	assertNameAsBytes(t, Restore)
	assertNameAsBytes(t, Save)
	assertNameAsBytes(t, SetFlatness)
	assertNameAsBytes(t, SetGraphicsStateParams)
	assertNameAsBytes(t, SetLineCapstyle)
	assertNameAsBytes(t, SetLineDashpattern)
	assertNameAsBytes(t, SetLineJoinstyle)
	assertNameAsBytes(t, SetLineMiterlimit)
	assertNameAsBytes(t, SetLineWidth)
	assertNameAsBytes(t, SetMatrix)
	assertNameAsBytes(t, SetRenderingintent)
}

func TestNameAsByteGraphics(t *testing.T) {
	assertNameAsBytes(t, AppendRect)
	assertNameAsBytes(t, BeginInlineImage)
	assertNameAsBytes(t, BeginInlineImageData)
	assertNameAsBytes(t, EndInlineImage)
	assertNameAsBytes(t, ClipEvenOdd)
	assertNameAsBytes(t, ClipNonZero)
	assertNameAsBytes(t, CloseAndStroke)
	assertNameAsBytes(t, CloseFillEvenOddAndStroke)
	assertNameAsBytes(t, CloseFillNonZeroAndStroke)
	assertNameAsBytes(t, ClosePath)
	assertNameAsBytes(t, CurveTo)
	assertNameAsBytes(t, CurveToReplicateFinalPoint)
	assertNameAsBytes(t, CurveToReplicateInitialPoint)
	assertNameAsBytes(t, Endpath)
	assertNameAsBytes(t, FillEvenOddAndStroke)
	assertNameAsBytes(t, FillEvenOdd)
	assertNameAsBytes(t, FillNonZeroAndStroke)
	assertNameAsBytes(t, FillNonZero)
	assertNameAsBytes(t, LegacyFillNonZero)
	assertNameAsBytes(t, LineTo)
	assertNameAsBytes(t, MoveTo)
	assertNameAsBytes(t, ShadingFill)
	assertNameAsBytes(t, StrokePath)
}

func TestNameAsByteText(t *testing.T) {
	assertNameAsBytes(t, BeginText)
	assertNameAsBytes(t, EndText)
	assertNameAsBytes(t, MoveText)
	assertNameAsBytes(t, MoveTextSetLeading)
	assertNameAsBytes(t, NextLine)
	assertNameAsBytes(t, SetCharSpacing)
	assertNameAsBytes(t, SetFontAndSize)
	assertNameAsBytes(t, SetTextHorizontalScaling)
	assertNameAsBytes(t, SetTextLeading)
	assertNameAsBytes(t, SetTextRenderingmode)
	assertNameAsBytes(t, SetTextRise)
	assertNameAsBytes(t, SetWordSpacing)
	assertNameAsBytes(t, ShowText)
	assertNameAsBytes(t, ShowTextAdjusted)
	assertNameAsBytes(t, ShowTextLine)
	assertNameAsBytes(t, ShowTextLineAndSpace)
}

func TestNameAsByteType3(t *testing.T) {
	assertNameAsBytes(t, Type3D0)
	assertNameAsBytes(t, Type3D1)
}

func TestNameAsByteCompatibility(t *testing.T) {
	assertNameAsBytes(t, BeginCompatibilitySection)
	assertNameAsBytes(t, EndCompatibilitySection)
}

// TestUnkownOperator is testUnkownOperator, spelling and all. Java throws
// IllegalArgumentException, which is unchecked, so the port panics.
func TestUnkownOperator(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("GetNameAsBytes(\"UNKNOWN\") returned instead of panicking")
		}
	}()
	GetNameAsBytes("UNKNOWN")
}
