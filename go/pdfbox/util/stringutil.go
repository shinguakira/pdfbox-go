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
	for _, r := range s {
		if isJavaSpace(r) {
			pieces = append(pieces, string(current))
			current = current[:0]
			continue
		}
		current = append(current, r)
	}
	pieces = append(pieces, string(current))
	// String.split with a limit of zero drops the trailing empty strings, and
	// answers one empty string for an empty input.
	last := len(pieces)
	for last > 1 && pieces[last-1] == "" {
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
