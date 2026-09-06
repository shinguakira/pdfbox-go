package xmptype

// How many values a property may hold.
//
// Port of the enum org.apache.xmpbox.type.Cardinality.

// Cardinality says whether a property is a single value or one of the three
// kinds of RDF array.
type Cardinality int

const (
	// Simple is a single value.
	Simple Cardinality = iota
	// Bag is an unordered array.
	Bag
	// Seq is an ordered array.
	Seq
	// Alt is an array of alternatives.
	Alt
)

// IsArray reports whether values of this cardinality are arrays.
func (c Cardinality) IsArray() bool { return c != Simple }

// String returns the enum constant's name, which is Java's Enum.toString and
// which the serialiser writes as the array element name.
func (c Cardinality) String() string {
	switch c {
	case Simple:
		return "Simple"
	case Bag:
		return "Bag"
	case Seq:
		return "Seq"
	case Alt:
		return "Alt"
	}
	return "Unknown"
}
