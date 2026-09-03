package cos

import (
	"errors"
	"strconv"
)

// Errors returned when constructing an ObjectKey. Java throws
// IllegalArgumentException for both.
var (
	ErrNegativeObjectNumber     = errors.New("cos: object number must not be a negative value")
	ErrNegativeGenerationNumber = errors.New("cos: generation number must not be a negative value")
)

const (
	// numberOffset is Short.SIZE in Java: the generation occupies the low 16
	// bits and the object number everything above.
	numberOffset = 16
	// generationMask covers those low 16 bits, so generations run 0-65535.
	generationMask = 1<<numberOffset - 1
)

// ObjectKey is the physical reference to an indirect PDF object — the
// "12 0 R" that appears throughout a file.
//
// Port of org.apache.pdfbox.cos.COSObjectKey. The number and generation are
// packed into a single int64 exactly as Java packs them, so that the packed
// value can be used directly as a map key and as the sort key.
type ObjectKey struct {
	// numberAndGeneration holds the generation in the low 16 bits and the
	// object number above them.
	numberAndGeneration int64
	// streamIndex is the index within a compressed object stream, or -1.
	streamIndex int
}

// NewObjectKey returns the key for the given object and generation number.
//
// Port of COSObjectKey(long, int), which defaults the stream index to -1.
func NewObjectKey(num int64, gen int) (*ObjectKey, error) {
	return NewObjectKeyInStream(num, gen, -1)
}

// NewObjectKeyInStream returns the key for an object at the given index within
// a compressed object stream.
//
// Port of COSObjectKey(long, int, int).
func NewObjectKeyInStream(num int64, gen, index int) (*ObjectKey, error) {
	if num < 0 {
		return nil, ErrNegativeObjectNumber
	}
	if gen < 0 {
		return nil, ErrNegativeGenerationNumber
	}
	return &ObjectKey{
		numberAndGeneration: ComputeInternalHash(num, gen),
		streamIndex:         index,
	}, nil
}

// ComputeInternalHash packs an object number and generation into the single
// value an ObjectKey stores.
//
// Port of the static COSObjectKey.computeInternalHash.
func ComputeInternalHash(num int64, gen int) int64 {
	return num<<numberOffset | int64(gen)&generationMask
}

// InternalHash returns the packed number and generation.
//
// Java also derives hashCode from this value. Go has no hashCode, so this is
// the value to use as a map key when one is needed.
func (k *ObjectKey) InternalHash() int64 { return k.numberAndGeneration }

// Number returns the object number.
func (k *ObjectKey) Number() int64 {
	// Java uses >>> here. The packed value is never negative — the constructor
	// rejects negative inputs — so a signed shift is equivalent.
	return k.numberAndGeneration >> numberOffset
}

// Generation returns the object generation number.
func (k *ObjectKey) Generation() int {
	return int(k.numberAndGeneration & generationMask)
}

// StreamIndex returns the index within a compressed object stream, or -1 when
// the object is not in one.
func (k *ObjectKey) StreamIndex() int { return k.streamIndex }

// Equals reports whether two keys refer to the same indirect object.
//
// As in Java, only the packed number and generation are compared — the stream
// index is deliberately not part of equality, since the same object can be
// reached both directly and through an object stream.
func (k *ObjectKey) Equals(other *ObjectKey) bool {
	return other != nil && other.numberAndGeneration == k.numberAndGeneration
}

// Compare orders keys by object number, then by generation.
//
// Port of compareTo. It returns exactly -1, 0 or 1 rather than an arbitrary
// negative or positive value, because the Java tests assert on those values.
func (k *ObjectKey) Compare(other *ObjectKey) int {
	switch {
	case k.numberAndGeneration < other.numberAndGeneration:
		return -1
	case k.numberAndGeneration > other.numberAndGeneration:
		return 1
	default:
		return 0
	}
}

// String returns the PDF reference form, "<number> <generation> R".
func (k *ObjectKey) String() string {
	return strconv.FormatInt(k.Number(), 10) + " " +
		strconv.Itoa(k.Generation()) + " R"
}
