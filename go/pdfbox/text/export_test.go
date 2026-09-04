package text

// The two package-private helpers the review tests reach.

// HasFontOrSizeChangedForTest exposes hasFontOrSizeChanged.
func HasFontOrSizeChangedForTest(current, last *TextPosition) bool {
	return hasFontOrSizeChanged(current, last)
}

// MultiplyFloatForTest exposes multiplyFloat.
func MultiplyFloatForTest(value1, value2 float32) float32 {
	return multiplyFloat(value1, value2)
}

// HandleDirectionForTest exposes handleDirection.
func HandleDirectionForTest(word string) string { return handleDirection(word) }
