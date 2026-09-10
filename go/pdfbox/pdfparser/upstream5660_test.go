package pdfparser_test

// PDFBOX-5660, Apache de68eb3e3, in the sync of 2026-09-07.
//
// COSParser.parseObjectDynamically opened with
//
//	COSObject pdfObject = document.getObjectFromPool(objKey);
//	if (!pdfObject.isObjectNull())
//
// and getObjectFromPool answers null for a null key -- the method is written
// that way on purpose, `if (key != null)` around the whole body. So an
// indirect reference carrying no key threw NullPointerException out of a
// method that declares IOException. Apache added the check.
//
// COSObject(COSBase, ICOSParser) is the constructor that produces one: it
// takes a parser and no key, so a reference built through it and then
// dereferenced arrives here with objKey null. The port has it as
// cos.NewObjectLazy, and had the same defect with a nil dereference in place
// of the NullPointerException.

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// aFileParser is a parser over the smallest input that opens one. Nothing here
// reads the file: the object never gets as far as an offset.
func aFileParser(t *testing.T) *pdfparser.FileParser {
	t.Helper()
	parser, err := pdfparser.NewFileParser(
		pdfio.NewReadBufferBytes([]byte("%PDF-1.4\n%%EOF\n")), nil, nil)
	if err != nil {
		t.Fatalf("NewFileParser: %v", err)
	}
	return parser
}

// TestDereferenceObjectWithNoKeyIsAnError is the site.
func TestDereferenceObjectWithNoKeyIsAnError(t *testing.T) {
	parser := aFileParser(t)
	// A reference that holds a parser and no key, which is what
	// COSObject(COSBase, ICOSParser) builds.
	reference := cos.NewObjectLazy(nil, parser)
	if reference.Key() != nil {
		t.Fatalf("the reference has a key %v; the case needs one with none",
			reference.Key())
	}

	base, err := parser.DereferenceObject(reference)
	if err == nil {
		t.Fatalf("dereferencing a reference with no key answered %v, want an error",
			base)
	}
	// Java's message, which names the call that answered null.
	if !strings.Contains(err.Error(), "ObjectFromPool") {
		t.Errorf("err = %v, want one naming ObjectFromPool", err)
	}
}

// TestDereferenceObjectWithNoKeyIsLoggedNotFatal is the caller's half.
// COSObject.getObject has no throws clause: it swallows the IOException, logs
// it and answers whatever baseObject holds. So the fix turns a crash into a
// null object, which is what a caller of getObject on a broken reference has
// always been given for every other failure.
func TestDereferenceObjectWithNoKeyIsLoggedNotFatal(t *testing.T) {
	parser := aFileParser(t)
	reference := cos.NewObjectLazy(nil, parser)
	if got := reference.Object(); got != nil {
		t.Errorf("Object() = %v, want nil", got)
	}
}
