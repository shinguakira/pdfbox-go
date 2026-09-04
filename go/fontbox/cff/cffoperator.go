package cff

// CFFOperator represents a CFF operator.
//
// Port of org.apache.fontbox.cff.CFFOperator, whose whole content is one static
// map from a one or two byte encoding to the operator's name.
//
// Java's LinkedHashMap keeps the registration order; nothing here reads the map
// in order, so a plain map stands in for it.
var cffOperatorKeyMap = func() map[int]string {
	keyMap := make(map[int]string, 52)
	register := func(b0, b1 int, name string) {
		keyMap[cffOperatorKey(b0, b1)] = name
	}

	// Top DICT
	register(0, 0, "version")
	register(1, 0, "Notice")
	register(12, 0, "Copyright")
	register(2, 0, "FullName")
	register(3, 0, "FamilyName")
	register(4, 0, "Weight")
	register(12, 1, "isFixedPitch")
	register(12, 2, "ItalicAngle")
	register(12, 3, "UnderlinePosition")
	register(12, 4, "UnderlineThickness")
	register(12, 5, "PaintType")
	register(12, 6, "CharstringType")
	register(12, 7, "FontMatrix")
	register(13, 0, "UniqueID")
	register(5, 0, "FontBBox")
	register(12, 8, "StrokeWidth")
	register(14, 0, "XUID")
	register(15, 0, "charset")
	register(16, 0, "Encoding")
	register(17, 0, "CharStrings")
	register(18, 0, "Private")
	register(12, 20, "SyntheticBase")
	register(12, 21, "PostScript")
	register(12, 22, "BaseFontName")
	register(12, 23, "BaseFontBlend")
	register(12, 30, "ROS")
	register(12, 31, "CIDFontVersion")
	register(12, 32, "CIDFontRevision")
	register(12, 33, "CIDFontType")
	register(12, 34, "CIDCount")
	register(12, 35, "UIDBase")
	register(12, 36, "FDArray")
	register(12, 37, "FDSelect")
	register(12, 38, "FontName")

	// Private DICT
	register(6, 0, "BlueValues")
	register(7, 0, "OtherBlues")
	register(8, 0, "FamilyBlues")
	register(9, 0, "FamilyOtherBlues")
	register(12, 9, "BlueScale")
	register(12, 10, "BlueShift")
	register(12, 11, "BlueFuzz")
	register(10, 0, "StdHW")
	register(11, 0, "StdVW")
	register(12, 12, "StemSnapH")
	register(12, 13, "StemSnapV")
	register(12, 14, "ForceBold")
	register(12, 15, "LanguageGroup")
	register(12, 16, "ExpansionFactor")
	register(12, 17, "initialRandomSeed")
	register(19, 0, "Subrs")
	register(20, 0, "defaultWidthX")
	register(21, 0, "nominalWidthX")
	return keyMap
}()

func cffOperatorKey(b0, b1 int) int { return (b1 << 8) + b0 }

// GetOperator returns the operator name corresponding to the given one byte
// representation, the second result being false where there is none.
func GetOperator(b0 int) (string, bool) {
	return GetOperator2(b0, 0)
}

// GetOperator2 returns the operator name corresponding to the given two byte
// representation, the second result being false where there is none.
func GetOperator2(b0, b1 int) (string, bool) {
	name, ok := cffOperatorKeyMap[cffOperatorKey(b0, b1)]
	return name, ok
}
