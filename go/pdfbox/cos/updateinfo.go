package cos

// UpdateInfo is implemented by the COS objects that remember whether they have
// been changed since the document was parsed, which is what an incremental save
// writes back.
//
// Port of the interface org.apache.pdfbox.cos.COSUpdateInfo, which extends
// COSObjectable. Dictionary, Array and Object implement it, as they do in Java,
// and so does Stream because COSStream extends COSDictionary.
type UpdateInfo interface {
	// COSObject returns the receiver. Port of the COSObjectable this extends.
	COSObject() Base

	// IsNeedToBeUpdated gets the update state for the COSWriter. This
	// indicates whether an object is to be written when there is an
	// incremental save.
	IsNeedToBeUpdated() bool

	// SetNeedToBeUpdated sets the update state of the dictionary for the
	// COSWriter. This indicates whether an object is to be written when there
	// is an incremental save.
	SetNeedToBeUpdated(flag bool)

	// ToIncrement uses this UpdateInfo as the base object of a new Increment.
	ToIncrement() *Increment

	// UpdateState returns the current UpdateState of this UpdateInfo.
	UpdateState() *UpdateState
}

// updateInfoState carries the COSUpdateState field of COSUpdateInfo's
// implementors. Dictionary, Array and Object embed it.
//
// Java's three default methods cannot be embedded here: they would need the
// owner, and an embedded struct in Go has no way back to the value that embeds
// it. Each implementor writes them out instead, one line each.
type updateInfoState struct {
	updateState *UpdateState
}

// state returns the owner's update state, creating it if the owner was built
// with a struct literal rather than a constructor. Java creates it in the
// constructor, which always runs; owner is what Java passes as `this`, so for a
// Stream it is the Stream and not the Dictionary embedded in it.
func (u *updateInfoState) state(owner UpdateInfo) *UpdateState {
	if u.updateState == nil {
		u.updateState = NewUpdateState(owner)
	}
	return u.updateState
}
