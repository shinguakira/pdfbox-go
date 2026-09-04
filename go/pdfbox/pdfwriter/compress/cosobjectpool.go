package compress

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// ObjectPool maps cos.Base instances to cos.ObjectKeys and allows for a
// bidirectional lookup.
//
// Port of org.apache.pdfbox.pdfwriter.compress.COSObjectPool. Java's maps are
// keyed on COSObjectKey, which overrides equals and hashCode; the port keys on
// the key's internal hash, the way cos.Document's own pools do, and keeps the
// key object alongside so that a lookup can hand it back.
type ObjectPool struct {
	keyPool    map[int64]cos.Base
	keyObjects map[int64]*cos.ObjectKey
	objectPool map[cos.Base]*cos.ObjectKey

	highestXRefObjectNumber int64
}

// NewObjectPool creates a map of cos.Base instances to cos.ObjectKeys, allowing
// bidirectional lookups. This constructor can be used for pre-initialized
// structures to start the assignment of new object numbers starting from the
// hereby given offset.
func NewObjectPool(highestXRefObjectNumber int64) *ObjectPool {
	p := &ObjectPool{
		keyPool:    map[int64]cos.Base{},
		keyObjects: map[int64]*cos.ObjectKey{},
		objectPool: map[cos.Base]*cos.ObjectKey{},
	}
	p.highestXRefObjectNumber = max(p.highestXRefObjectNumber, highestXRefObjectNumber)
	return p
}

// Put updates the key and object maps, returning the actual key the object has
// been added for.
func (p *ObjectPool) Put(key *cos.ObjectKey, object cos.Base) (*cos.ObjectKey, error) {
	// to avoid to mixup indirect COSInteger objects holding the same value we have to check
	// if the given key is the same than the key which is stored for the "same" base object wihtin the object pool
	// the same is always true for COSFloat, COSBoolean and COSName and under certain circumstances for the remainig
	// types as well
	if object == nil || (p.ContainsObject(object) && p.Key(object).Equals(key)) {
		return nil, nil
	}
	actualKey := key
	if actualKey == nil || p.ContainsKey(actualKey) {
		p.highestXRefObjectNumber++
		var err error
		if actualKey, err = cos.NewObjectKey(p.highestXRefObjectNumber, 0); err != nil {
			return nil, err
		}
		object.SetKey(actualKey)
	} else {
		p.highestXRefObjectNumber = max(key.Number(), p.highestXRefObjectNumber)
	}
	p.keyPool[actualKey.InternalHash()] = object
	p.keyObjects[actualKey.InternalHash()] = actualKey
	p.objectPool[object] = actualKey
	return actualKey, nil
}

// Key returns the ObjectKey for a given registered cos.Base, or nil if such an
// object is not registered.
func (p *ObjectPool) Key(object cos.Base) *cos.ObjectKey {
	var key *cos.ObjectKey
	if reference, ok := object.(*cos.Object); ok {
		key = p.objectPool[reference.Object()]
	}
	if key == nil {
		return p.objectPool[object]
	}
	return key
}

// ContainsKey reports whether a cos.Base is registered for the given ObjectKey.
//
// Port of contains(COSObjectKey).
func (p *ObjectPool) ContainsKey(key *cos.ObjectKey) bool {
	_, ok := p.keyPool[key.InternalHash()]
	return ok
}

// Object returns the cos.Base that is registered for the given ObjectKey, or
// nil if no object is registered for that key.
func (p *ObjectPool) Object(key *cos.ObjectKey) cos.Base {
	return p.keyPool[key.InternalHash()]
}

// ContainsObject reports whether the given cos.Base is a registered object of
// this pool.
//
// Port of contains(COSBase).
func (p *ObjectPool) ContainsObject(object cos.Base) bool {
	if reference, ok := object.(*cos.Object); ok {
		if _, found := p.objectPool[reference.Object()]; found {
			return true
		}
	}
	_, found := p.objectPool[object]
	return found
}

// HighestXRefObjectNumber returns the highest known object number that is
// currently registered in this pool.
func (p *ObjectPool) HighestXRefObjectNumber() int64 {
	return p.highestXRefObjectNumber
}
