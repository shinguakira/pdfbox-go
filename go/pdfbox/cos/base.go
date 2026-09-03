package cos

// Base is implemented by every object in a PDF document.
//
// Port of the abstract class org.apache.pdfbox.cos.COSBase. Java uses a single
// rooted hierarchy; the port splits it into this interface for the contract and
// the embedded object struct below for the state, per
// migration/conventions/java-to-go.md.
//
// Not yet ported: getKey and setKey, which carry the COSObjectKey of an
// indirect object. They arrive with ObjectKey. TestCOSBase does not exercise
// them, so nothing here is untested for their absence.
type Base interface {
	// Accept is the visitor double dispatch. Port of accept(ICOSVisitor).
	Accept(v Visitor) error

	// COSObject returns the receiver. Port of getCOSObject(), which COSBase
	// inherits from the COSObjectable interface so that a document-model
	// object and a COS object can be asked for their COS form alike.
	COSObject() Base

	// IsDirect reports whether this object is written inline rather than as an
	// indirect object.
	IsDirect() bool

	// SetDirect sets that flag.
	SetDirect(direct bool)
}

// object carries the state COSBase holds for every COS value. Concrete types
// embed it.
//
// Embedding promotes these methods but gives no virtual dispatch, so COSObject
// cannot live here — an embedded struct cannot return the value that embeds it.
// Each concrete type implements COSObject itself, one line each.
type object struct {
	direct bool
}

// IsDirect reports whether the object is written inline.
func (o *object) IsDirect() bool { return o.direct }

// SetDirect marks the object as written inline rather than indirectly.
func (o *object) SetDirect(direct bool) { o.direct = direct }
