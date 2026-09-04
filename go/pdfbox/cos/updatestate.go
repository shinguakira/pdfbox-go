package cos

// UpdateState manages the update state for an UpdateInfo. Such states are used
// to create an Increment for the incremental saving of a Document.
//
// Port of org.apache.pdfbox.cos.COSUpdateState.
type UpdateState struct {
	// updateInfo is the UpdateInfo this UpdateState manages update states for.
	updateInfo UpdateInfo

	// originDocumentState is the DocumentState the updateInfo is linked to.
	originDocumentState *DocumentState

	// updated is the actual update state of updateInfo.
	//
	//   - true, if updateInfo has been updated after the document completed
	//     parsing.
	//   - false, if updateInfo has remained unaltered since the document
	//     completed parsing.
	updated bool
}

// NewUpdateState creates a new UpdateState for the given UpdateInfo.
func NewUpdateState(updateInfo UpdateInfo) *UpdateState {
	return &UpdateState{updateInfo: updateInfo}
}

// SetOriginDocumentState links the given DocumentState to the updated state of
// the managed updateInfo.
//
// This also initialises updated accordingly and sets the same DocumentState for
// all possibly contained substructures. Should originDocumentState already have
// been set, by a prior call to this method, this denies to overwrite it.
//
// DocumentState.IsAcceptingUpdates determines whether updates to updateInfo are
// allowed. As long as no DocumentState is linked to this UpdateState, it does
// not accept updates.
func (s *UpdateState) SetOriginDocumentState(originDocumentState *DocumentState) {
	s.setOriginDocumentState(originDocumentState, false)
}

// setOriginDocumentState is the private overload that additionally denies
// changing updated, should the flag dereferencing indicate that this is caused
// by dereferencing an Object.
func (s *UpdateState) setOriginDocumentState(originDocumentState *DocumentState, dereferencing bool) {
	if s.originDocumentState != nil || originDocumentState == nil {
		return
	}
	s.originDocumentState = originDocumentState
	if !dereferencing {
		s.update()
	}

	switch updateInfo := s.updateInfo.(type) {
	case *Dictionary:
		setOriginOfChildren(updateInfo.Values(), originDocumentState, dereferencing)
	case *Stream:
		// COSStream is a COSDictionary in Java, so it takes the same branch.
		setOriginOfChildren(updateInfo.Values(), originDocumentState, dereferencing)
	case *Array:
		setOriginOfChildren(updateInfo.ToList(), originDocumentState, dereferencing)
	case *Object:
		if updateInfo.IsDereferenced() {
			reference := updateInfo.Object()
			if info, ok := reference.(UpdateInfo); ok {
				info.UpdateState().setOriginDocumentState(originDocumentState, dereferencing)
			}
		}
	}
}

// setOriginOfChildren is the loop the two container branches above share.
func setOriginOfChildren(entries []Base, originDocumentState *DocumentState, dereferencing bool) {
	for _, entry := range entries {
		if info, ok := entry.(UpdateInfo); ok {
			info.UpdateState().setOriginDocumentState(originDocumentState, dereferencing)
		}
	}
}

// OriginDocumentState returns the originDocumentState that is linked to the
// managed updateInfo.
func (s *UpdateState) OriginDocumentState() *DocumentState {
	return s.originDocumentState
}

// isAcceptingUpdates reports whether the linked originDocumentState is
// accepting updates and such a DocumentState has been linked to this
// UpdateState.
func (s *UpdateState) isAcceptingUpdates() bool {
	return s.originDocumentState != nil && s.originDocumentState.IsAcceptingUpdates()
}

// IsUpdated returns the actual updated state of the managed updateInfo.
func (s *UpdateState) IsUpdated() bool {
	return s.updated
}

// update calls update(true). This only has an effect if isAcceptingUpdates
// returns true.
func (s *UpdateState) update() {
	s.updateTo(true)
}

// updateTo sets the updated state of the managed updateInfo to the given state.
// This only has an effect if isAcceptingUpdates returns true.
//
// Java names this update(boolean); Go has no overloading, and the four update
// overloads become updateTo, updateChild and updateChildren.
func (s *UpdateState) updateTo(updated bool) {
	if s.isAcceptingUpdates() {
		s.updated = updated
	}
}

// updateChild calls update for this UpdateState and sets the origin document
// state for the given child, initialising its updated state and
// originDocumentState.
//
// This has no effect for a child that is not an UpdateInfo.
//
// Port of update(COSBase).
func (s *UpdateState) updateChild(child Base) {
	s.update()
	if info, ok := child.(UpdateInfo); ok {
		info.UpdateState().SetOriginDocumentState(s.originDocumentState)
	}
}

// updateChildren calls update for this UpdateState and sets the origin document
// state for the given children, initialising their updated state and
// originDocumentState.
//
// This has no effect for a child that is not an UpdateInfo.
//
// Port of update(Iterable<COSBase>), which update(COSArray) delegates to.
func (s *UpdateState) updateChildren(children []Base) {
	s.update()
	if children == nil {
		return
	}
	for _, child := range children {
		if info, ok := child.(UpdateInfo); ok {
			info.UpdateState().SetOriginDocumentState(s.originDocumentState)
		}
	}
}

// dereferenceChild sets the origin document state for the dereferenced child,
// initialising its originDocumentState.
//
// This has no effect for a child that is not an UpdateInfo and never changes
// the child's updated state.
func (s *UpdateState) dereferenceChild(child Base) {
	if info, ok := child.(UpdateInfo); ok {
		info.UpdateState().setOriginDocumentState(s.originDocumentState, true)
	}
}

// toIncrement uses the managed updateInfo as the base object of a new
// Increment.
func (s *UpdateState) toIncrement() *Increment {
	return NewIncrement(s.updateInfo)
}
