package util

// isJavaSpace reports whether r is one of the six characters Java's \s matches:
// space, tab, line feed, vertical tab, form feed and carriage return.
//
// Go's regexp \s leaves the vertical tab out, so the port does not use a
// pattern here.
func isJavaSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return false
}

// SplitOnSpace splits a string on each whitespace character, dropping the
// trailing empty pieces the way String.split does.
//
// Port of the static StringUtil.splitOnSpace, which splits on the \s pattern.
func SplitOnSpace(s string) []string {
	pieces := []string{}
	current := []rune{}
	matched := false
	for _, r := range s {
		if isJavaSpace(r) {
			matched = true
			pieces = append(pieces, string(current))
			current = current[:0]
			continue
		}
		current = append(current, r)
	}
	// Pattern.split answers the whole input, untrimmed, where the pattern never
	// matched -- which is why an empty input gives one empty string.
	if !matched {
		return []string{s}
	}
	pieces = append(pieces, string(current))
	// With a limit of zero it then drops *every* trailing empty string, so an
	// input that is nothing but separators gives an empty array rather than one
	// empty string. Java's own test asserts both shapes.
	last := len(pieces)
	for last > 0 && pieces[last-1] == "" {
		last--
	}
	return pieces[:last]
}

// TokenizeOnSpace splits a string into its runs of non-whitespace and its
// whitespace characters, each one on its own.
//
// Port of the static StringUtil.tokenizeOnSpace, which splits on the zero width
// pattern either side of a \s. Java has lookbehind and lookahead; Go's regexp
// has neither, so the port walks the string.
func TokenizeOnSpace(s string) []string {
	pieces := []string{}
	current := []rune{}
	for _, r := range s {
		if isJavaSpace(r) {
			if len(current) > 0 {
				pieces = append(pieces, string(current))
				current = current[:0]
			}
			pieces = append(pieces, string(r))
			continue
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		pieces = append(pieces, string(current))
	}
	if len(pieces) == 0 {
		// String.split answers one empty string for an empty input.
		return []string{""}
	}
	return pieces
}
