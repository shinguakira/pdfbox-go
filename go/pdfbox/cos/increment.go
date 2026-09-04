package cos

// Increment starts at a given UpdateInfo to collect updates that have been made
// to a Document and therefore should be added to its next increment.
//
// Port of org.apache.pdfbox.cos.COSIncrement, which implements
// Iterable<COSBase>; Objects returns the slice a Go caller ranges over instead.
type Increment struct {
	// objects contains the Bases that shall be added to the increment at top
	// level.
	//
	// Java uses a LinkedHashSet, so the order objects were collected in is the
	// order they are written in; the slice keeps that order and objectSet keeps
	// the membership test.
	objects   []Base
	objectSet map[Base]bool

	// excluded contains the direct Bases that are either written directly by
	// structures contained in objects or that must be excluded from being
	// written as indirect Objects for other reasons.
	excluded map[Base]bool

	// processedObjects contains all Objects that have already been processed by
	// this Increment and shall not be processed again.
	processedObjects map[*Object]bool

	// incrementOrigin contains the UpdateInfo that this Increment creates an
	// increment for.
	incrementOrigin UpdateInfo

	// initialized says whether this Increment has already been determined, or
	// must still be evaluated.
	initialized bool
}

// NewIncrement creates a new Increment for the given UpdateInfo. The increment
// uses its DocumentState as its own origin and collects all updates contained
// in the given UpdateInfo. Should the given object be nil, the resulting
// increment is empty.
func NewIncrement(incrementOrigin UpdateInfo) *Increment {
	return &Increment{
		objectSet:        map[Base]bool{},
		excluded:         map[Base]bool{},
		processedObjects: map[*Object]bool{},
		incrementOrigin:  incrementOrigin,
	}
}

// collect collects all updates made to the given Base and its contained
// structures, forwarding every UpdateInfo to the proper specialised collection
// method.
//
// It returns true if the Base represents a direct child structure that would
// require its parent to be updated instead.
func (in *Increment) collect(base Base) bool {
	if in.Contains(base) {
		return false
	}
	// handle updatable objects:
	switch value := base.(type) {
	case *Dictionary:
		return in.collectDictionary(base, value.Values())
	case *Stream:
		// COSStream is a COSDictionary in Java, so it takes the same branch.
		// The object added to the increment is the stream itself, not the
		// dictionary in it, or the stream data would not be written.
		return in.collectDictionary(base, value.Values())
	case *Object:
		in.collectObject(value)
		// COSObjects by definition are indirect and shall never cause a parent
		// structure to be updated.
		return false
	case *Array:
		return in.collectArray(value)
	}
	return false
}

// collectDictionary collects all updates made to the given dictionary and its
// contained structures. dictionary is the object itself and values are its
// entries, which for a Stream are the entries of the dictionary in it.
//
// It returns true if the dictionary represents a direct child structure that
// would require its parent to be updated instead.
//
// Port of collect(COSDictionary).
func (in *Increment) collectDictionary(dictionary Base, values []Base) bool {
	updateState := dictionary.(UpdateInfo).UpdateState()
	// Is definitely part of the increment?
	if !in.isExcluded(dictionary) && !in.Contains(dictionary) && updateState.IsUpdated() {
		in.add(dictionary)
	}
	childDemandsParentUpdate := false
	// Collect children:
	for _, entry := range values {
		// Primitives can not be part of an increment. (on top level)
		updatableEntry, isUpdatable := entry.(UpdateInfo)
		if !isUpdatable || in.Contains(entry) {
			continue
		}
		entryUpdateState := updatableEntry.UpdateState()
		// Entries with different document origin must be part of the increment!
		in.updateDifferentOrigin(entryUpdateState)
		// Always attempt to write COSArrays as direct objects.
		_, isReference := entry.(*Object)
		_, isArray := entry.(*Array)
		if updatableEntry.IsNeedToBeUpdated() &&
			((!isReference && entry.IsDirect()) || isArray) {
			// Exclude direct entries from the increment!
			in.Exclude(entry)
			childDemandsParentUpdate = true
		}
		// Collect descendants:
		childDemandsParentUpdate = in.collect(entry) || childDemandsParentUpdate
	}

	if in.isExcluded(dictionary) {
		return childDemandsParentUpdate
	}
	if childDemandsParentUpdate && !in.Contains(dictionary) {
		in.add(dictionary)
	}
	return false
}

// collectArray collects all updates made to the given Array and its contained
// structures.
//
// It returns true if the Array's elements changed. An Array is always treated
// as a direct structure that would require its parent to be updated instead.
func (in *Increment) collectArray(array *Array) bool {
	updateState := array.UpdateState()
	childDemandsParentUpdate := updateState.IsUpdated()
	for _, entry := range array.ToList() {
		// Primitives can not be part of an increment. (on top level)
		updatableEntry, isUpdatable := entry.(UpdateInfo)
		if !isUpdatable || in.Contains(entry) {
			continue
		}
		entryUpdateState := updatableEntry.UpdateState()
		// Entries with different document origin must be part of the increment!
		in.updateDifferentOrigin(entryUpdateState)
		// Collect descendants:
		childDemandsParentUpdate = in.collect(entry) || childDemandsParentUpdate
	}
	return childDemandsParentUpdate
}

// collectObject collects all updates made to the given Object and its contained
// structures.
func (in *Increment) collectObject(object *Object) {
	if in.Contains(object) {
		return
	}
	in.addProcessedObject(object)
	updateState := object.UpdateState()
	// Objects with different document origin must be part of the increment!
	in.updateDifferentOrigin(updateState)
	// determine actual, if necessary or possible without dereferencing:
	var actual UpdateInfo
	if updateState.IsUpdated() || object.IsDereferenced() {
		if info, ok := object.Object().(UpdateInfo); ok {
			actual = info
		}
	}
	// Skip?
	if actual == nil || in.Contains(actual.COSObject()) {
		return
	}
	childDemandsParentUpdate := false
	actualUpdateState := actual.UpdateState()
	if actualUpdateState.IsUpdated() {
		childDemandsParentUpdate = true
	}
	in.Exclude(actual.COSObject())
	childDemandsParentUpdate = in.collect(actual.COSObject()) || childDemandsParentUpdate
	if updateState.IsUpdated() || childDemandsParentUpdate {
		in.add(actual.COSObject())
	}
}

// Contains reports whether the given Base is already known to and has been
// processed by this Increment.
func (in *Increment) Contains(base Base) bool {
	if in.objectSet[base] {
		return true
	}
	reference, isReference := base.(*Object)
	return isReference && in.processedObjects[reference]
}

// updateDifferentOrigin checks whether the given UpdateState's DocumentState
// differs from the Increment's known incrementOrigin. Should that be the case,
// the UpdateState originates from another Document and must be added to the
// Increment, hence calls update.
func (in *Increment) updateDifferentOrigin(updateState *UpdateState) {
	if in.incrementOrigin != nil && updateState != nil &&
		in.incrementOrigin.UpdateState().OriginDocumentState() != updateState.OriginDocumentState() {
		updateState.update()
	}
}

// add records that the given object shall be part of the increment. nil values
// are skipped.
func (in *Increment) add(object Base) {
	if object != nil && !in.objectSet[object] {
		in.objectSet[object] = true
		in.objects = append(in.objects, object)
	}
}

// addProcessedObject records that the given Object has been processed, or is
// being processed, so that it is skipped should it be encountered again. nil
// values are ignored.
func (in *Increment) addProcessedObject(base *Object) {
	if base != nil {
		in.processedObjects[base] = true
	}
}

// Exclude records that the given Bases are not fit for inclusion in an
// increment. nil values are ignored.
//
// It returns the Increment itself, to allow method chaining.
func (in *Increment) Exclude(base ...Base) *Increment {
	for _, b := range base {
		if b != nil {
			in.excluded[b] = true
		}
	}
	return in
}

// isExcluded reports whether the given Base has been excluded from the
// increment.
func (in *Increment) isExcluded(base Base) bool {
	return in.excluded[base]
}

// Objects returns all indirect Bases that shall be written to an increment as
// top level Objects. Calling this method causes the increment to be
// initialised.
func (in *Increment) Objects() []Base {
	if !in.initialized && in.incrementOrigin != nil {
		in.collect(in.incrementOrigin.COSObject())
		in.initialized = true
	}
	return in.objects
}
