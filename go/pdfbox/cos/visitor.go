package cos

// Visitor visits a PDF document at the COS level.
//
// Port of org.apache.pdfbox.cos.ICOSVisitor. The Java method names are
// visitFromArray, visitFromBoolean and so on; the port drops the "From", which
// carried no meaning once the parameter type says what is being visited.
//
// This interface grows as the COS types are ported. Java declares eleven
// methods; the ones still to come are VisitArray, VisitDictionary, VisitFloat,
// VisitInteger, VisitStringObj, VisitStream, VisitDocument and VisitObject.
// Adding a method here is a breaking change for implementers, which is the
// intended signal — a visitor that has not been taught about a new COS type is
// incomplete.
type Visitor interface {
	VisitBoolean(obj *Boolean) error
	VisitFloat(obj *Float) error
	VisitInteger(obj *Integer) error
	VisitName(obj *Name) error
	VisitNull(obj *Null) error
	VisitObject(obj *Object) error
	VisitStringObj(obj *StringObj) error
}
