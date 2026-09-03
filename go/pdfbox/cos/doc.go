// Package cos implements the COS object model — the typed values a PDF file is
// built from: booleans, numbers, names, strings, arrays, dictionaries, streams
// and indirect references.
//
// It is the Go port of org.apache.pdfbox.cos. Names drop the COS prefix the Java
// classes carry, since the package supplies it: COSDictionary becomes
// cos.Dictionary. The exception is COSString, which becomes cos.StringObj —
// cos.String(x) would read as a conversion.
//
// The Java model is an abstract COSBase with a visitor over a mutable,
// reference-identity object graph. Go has neither inheritance nor a natural
// visitor, so Base is an interface and the shared state lives in an embedded
// struct; see migration/conventions/java-to-go.md.
//
// Objects here are not safe for concurrent use. The Java model is mutable and
// shared by reference throughout a document, and the port keeps that.
package cos
