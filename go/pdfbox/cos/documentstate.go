package cos

// DocumentState collects all known states a Document may have and allows their
// evaluation.
//
// Port of org.apache.pdfbox.cos.COSDocumentState.
type DocumentState struct {
	// parsing is the parsing state of the document.
	//
	//   - true, if the document is currently being parsed. (initial state)
	//   - false, if the document's parsing completed and it may be edited and
	//     updated.
	parsing bool
}

// NewDocumentState returns a state whose document is still being parsed, which
// is Java's field initialiser.
func NewDocumentState() *DocumentState {
	return &DocumentState{parsing: true}
}

// SetParsing sets the parsing state of the document.
func (s *DocumentState) SetParsing(parsing bool) {
	s.parsing = parsing
}

// IsAcceptingUpdates reports whether the document's parsing is completed and it
// may be updated.
func (s *DocumentState) IsAcceptingUpdates() bool {
	return !s.parsing
}
