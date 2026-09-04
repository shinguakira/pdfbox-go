package operator

import "fmt"

// nameAsBytes maps each operator name to its ASCII bytes, so that a writer does
// not encode the same string over and over.
//
// Port of the static nameAsBytes map of
// org.apache.pdfbox.contentstream.operator.OperatorName, in the order Java
// fills it. Java's names are ASCII, so the bytes of the Go string are the
// US_ASCII bytes of the Java one.
var nameAsBytes = func() map[string][]byte {
	m := map[string][]byte{}
	put := func(name string) { m[name] = []byte(name) }
	// non stroking color
	put(NonStrokingColor)
	put(NonStrokingColorN)
	put(NonStrokingRgb)
	put(NonStrokingGray)
	put(NonStrokingCmyk)
	put(NonStrokingColorspace)
	// stroking color
	put(StrokingColor)
	put(StrokingColorN)
	put(StrokingColorRgb)
	put(StrokingColorGray)
	put(StrokingColorCmyk)
	put(StrokingColorspace)
	// marked content
	put(BeginMarkedContentSeq)
	put(BeginMarkedContent)
	put(EndMarkedContent)
	put(MarkedContentPointWithProps)
	put(MarkedContentPoint)
	put(DrawObject)
	// state
	put(Concat)
	put(Restore)
	put(Save)
	put(SetFlatness)
	put(SetGraphicsStateParams)
	put(SetLineCapstyle)
	put(SetLineDashpattern)
	put(SetLineJoinstyle)
	put(SetLineMiterlimit)
	put(SetLineWidth)
	put(SetMatrix)
	put(SetRenderingintent)
	// graphics
	put(AppendRect)
	put(BeginInlineImage)
	put(BeginInlineImageData)
	put(EndInlineImage)
	put(ClipEvenOdd)
	put(ClipNonZero)
	put(CloseAndStroke)
	put(CloseFillEvenOddAndStroke)
	put(CloseFillNonZeroAndStroke)
	put(ClosePath)
	put(CurveTo)
	put(CurveToReplicateFinalPoint)
	put(CurveToReplicateInitialPoint)
	put(Endpath)
	put(FillEvenOddAndStroke)
	put(FillEvenOdd)
	put(FillNonZeroAndStroke)
	put(FillNonZero)
	put(LegacyFillNonZero)
	put(LineTo)
	put(MoveTo)
	put(ShadingFill)
	put(StrokePath)
	// text
	put(BeginText)
	put(EndText)
	put(MoveText)
	put(MoveTextSetLeading)
	put(NextLine)
	put(SetCharSpacing)
	put(SetFontAndSize)
	put(SetTextHorizontalScaling)
	put(SetTextLeading)
	put(SetTextRenderingmode)
	put(SetTextRise)
	put(SetWordSpacing)
	put(ShowText)
	put(ShowTextAdjusted)
	put(ShowTextLine)
	put(ShowTextLineAndSpace)
	// type3 font
	put(Type3D0)
	put(Type3D1)
	// compatibility section
	put(BeginCompatibilitySection)
	put(EndCompatibilitySection)
	return m
}()

// GetNameAsBytes returns the ASCII representation of the given operator name as
// a byte slice.
//
// Java throws IllegalArgumentException for an unknown name, which is unchecked,
// so the port panics.
func GetNameAsBytes(operatorName string) []byte {
	stringBytes, ok := nameAsBytes[operatorName]
	if !ok {
		panic(fmt.Sprintf("unknown operator %s", operatorName))
	}
	return stringBytes
}
