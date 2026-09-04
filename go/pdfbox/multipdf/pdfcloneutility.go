// Package multipdf merges, splits and overlays PDF documents.
//
// Port of org.apache.pdfbox.multipdf. PDFMergerUtility, LayerUtility and
// Overlay are not here: each of them reaches into pdmodel/interactive, which
// slice 8 brings. See migration/STATUS.md.
package multipdf

import (
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// cosObjectable is org.apache.pdfbox.pdmodel.common.COSObjectable, which
// cloneMerge takes. cos.Base satisfies it, and so does every pdmodel wrapper.
type cosObjectable interface {
	COSObject() cos.Base
}

// PDFCloneUtility clones PDF objects. It keeps track of objects it has already
// cloned.
//
// Port of org.apache.pdfbox.multipdf.PDFCloneUtility.
type PDFCloneUtility struct {
	destination   *pdmodel.PDDocument
	clonedVersion map[cos.Base]cos.Base
	clonedValues  map[cos.Base]bool
	// It might be useful to use IdentityHashMap like in PDFBOX-4477 for speed,
	// but we need a really huge file to test this. A test with the file from PDFBOX-4477
	// did not show a noticeable speed difference.
}

// NewPDFCloneUtility creates a new instance for the given target document.
//
// Java's constructor is protected; Go has no such access level, and the type is
// public, so this is exported. See migration/conventions/java-to-go.md.
func NewPDFCloneUtility(dest *pdmodel.PDDocument) *PDFCloneUtility {
	return &PDFCloneUtility{
		destination:   dest,
		clonedVersion: map[cos.Base]cos.Base{},
		clonedValues:  map[cos.Base]bool{},
	}
}

// Destination returns the destination PDF document this cloner instance is set
// up for.
func (c *PDFCloneUtility) Destination() *pdmodel.PDDocument { return c.destination }

// CloneForNewDocument deep-clones the given object for inclusion into a
// different PDF document identified by the destination parameter.
//
// Expert use only, don't use it if you don't know exactly what you are doing.
//
// Java is generic over the COSBase subtype and casts the result; Go returns the
// interface, and the callers that need a concrete type assert it, which is the
// same unchecked cast in a different place.
func (c *PDFCloneUtility) CloneForNewDocument(base cos.Base) (cos.Base, error) {
	if base == nil {
		return nil, nil
	}
	if retval, ok := c.clonedVersion[base]; ok {
		// we are done, it has already been converted.
		return retval, nil
	}
	if c.clonedValues[base] {
		// Don't clone a clone
		return base, nil
	}
	retval, err := c.cloneCOSBaseForNewDocument(base)
	if err != nil {
		return nil, err
	}
	c.clonedVersion[base] = retval
	c.clonedValues[retval] = true
	return retval, nil
}

// CloneDictionaryForNewDocument is CloneForNewDocument where the caller knows
// the result is a dictionary, which is Java's inferred TCOSBASE.
func (c *PDFCloneUtility) CloneDictionaryForNewDocument(base *cos.Dictionary) (*cos.Dictionary, error) {
	cloned, err := c.CloneForNewDocument(base)
	if err != nil {
		return nil, err
	}
	if cloned == nil {
		return nil, nil
	}
	// Java's cast is unchecked, and so is this assertion.
	return cloned.(*cos.Dictionary), nil
}

func (c *PDFCloneUtility) cloneCOSBaseForNewDocument(base cos.Base) (cos.Base, error) {
	switch value := base.(type) {
	case *cos.Object:
		return c.CloneForNewDocument(value.Object())
	case *cos.Array:
		return c.cloneCOSArray(value)
	case *cos.Stream:
		// COSStream is checked before COSDictionary in Java, because it is one.
		return c.cloneCOSStream(value)
	case *cos.Dictionary:
		return c.cloneCOSDictionary(value)
	}
	return base, nil
}

func (c *PDFCloneUtility) cloneCOSArray(array *cos.Array) (*cos.Array, error) {
	newArray := cos.NewArray()
	for i := 0; i < array.Size(); i++ {
		value := array.Get(i)
		if hasSelfReference(array, value) {
			newArray.Add(newArray)
		} else {
			cloned, err := c.CloneForNewDocument(value)
			if err != nil {
				return nil, err
			}
			newArray.Add(cloned)
		}
	}
	return newArray, nil
}

func (c *PDFCloneUtility) cloneCOSStream(stream *cos.Stream) (*cos.Stream, error) {
	newStream := c.destination.Document().CreateStream()
	output, err := newStream.CreateRawWriter()
	if err != nil {
		return nil, err
	}
	input, err := stream.CreateRawReader()
	if err != nil {
		output.Close()
		return nil, err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return nil, err
	}
	if err := output.Close(); err != nil {
		return nil, err
	}
	c.clonedVersion[stream] = newStream
	for _, key := range stream.KeySet() {
		value := stream.GetItem(key)
		if hasSelfReference(stream, value) {
			newStream.SetItem(key, newStream)
		} else {
			cloned, err := c.CloneForNewDocument(value)
			if err != nil {
				return nil, err
			}
			newStream.SetItem(key, cloned)
		}
	}
	return newStream, nil
}

func (c *PDFCloneUtility) cloneCOSDictionary(dictionary *cos.Dictionary) (*cos.Dictionary, error) {
	newDictionary := cos.NewDictionary()
	c.clonedVersion[dictionary] = newDictionary
	for _, key := range dictionary.KeySet() {
		value := dictionary.GetItem(key)
		if hasSelfReference(dictionary, value) {
			newDictionary.SetItem(key, newDictionary)
		} else {
			cloned, err := c.CloneForNewDocument(value)
			if err != nil {
				return nil, err
			}
			newDictionary.SetItem(key, cloned)
		}
	}
	return newDictionary, nil
}

// CloneMerge merges two objects of the same type by deep-cloning its members.
// Base and target must be instances of the same class.
//
// Java's method is package-private; the port exports it because Splitter and
// PDFMergerUtility, which are its callers, are in this package too and Go has
// no such access level.
func (c *PDFCloneUtility) CloneMerge(base, target cosObjectable) error {
	if base == nil || base == target {
		return nil
	}
	return c.cloneMergeCOSBase(base.COSObject(), target.COSObject())
}

func (c *PDFCloneUtility) cloneMergeCOSBase(source, target cos.Base) error {
	sourceBase := source
	if reference, ok := source.(*cos.Object); ok {
		sourceBase = reference.Object()
	}
	targetBase := target
	if reference, ok := target.(*cos.Object); ok {
		targetBase = reference.Object()
	}
	sourceArray, sourceIsArray := sourceBase.(*cos.Array)
	targetArray, targetIsArray := targetBase.(*cos.Array)
	if sourceIsArray && targetIsArray {
		for i := 0; i < sourceArray.Size(); i++ {
			cloned, err := c.CloneForNewDocument(sourceArray.Get(i))
			if err != nil {
				return err
			}
			targetArray.Add(cloned)
		}
		return nil
	}
	sourceDict, sourceIsDict := asDictionary(sourceBase)
	targetDict, targetIsDict := asDictionary(targetBase)
	if sourceIsDict && targetIsDict {
		for _, key := range sourceDict.KeySet() {
			value := sourceDict.GetItem(key)
			if item := targetDict.GetItem(key); item != nil {
				if err := c.CloneMerge(value, item); err != nil {
					return err
				}
			} else {
				cloned, err := c.CloneForNewDocument(value)
				if err != nil {
					return err
				}
				targetDict.SetItem(key, cloned)
			}
		}
	}
	return nil
}

// asDictionary is Java's `instanceof COSDictionary`, which a COSStream also
// satisfies.
func asDictionary(base cos.Base) (*cos.Dictionary, bool) {
	switch value := base.(type) {
	case *cos.Stream:
		return &value.Dictionary, true
	case *cos.Dictionary:
		return value, true
	}
	return nil, false
}

// hasSelfReference checks whether an element (of an array or a dictionary)
// points to its parent.
func hasSelfReference(parent, value cos.Base) bool {
	if cosObj, ok := value.(*cos.Object); ok {
		actual := cosObj.Object()
		if actual == parent {
			slog.Warn("multipdf: object has a reference to itself",
				"parent", simpleName(parent), "key", cosObj.Key())
			return true
		}
	}
	return false
}

// simpleName is Java's getClass().getSimpleName() for the two types that reach
// hasSelfReference.
func simpleName(base cos.Base) string {
	switch base.(type) {
	case *cos.Array:
		return "COSArray"
	case *cos.Stream:
		return "COSStream"
	case *cos.Dictionary:
		return "COSDictionary"
	}
	return "COSBase"
}
