package common

import "errors"

// ErrMissingResource reports a named resource that is not in the resource
// dictionary.
//
// Port of org.apache.pdfbox.pdmodel.MissingResourceException, which is one
// class thrown from two places: PDResources.getXObject and the like, and
// PDColorSpace.create for a colour space name the resources do not hold.
// `pdmodel` cannot hold the sentinel for both, because
// `pdmodel/graphics/color` cannot import `pdmodel` -- PDColorSpace.create takes
// a PDResources in the Java, and in Go that is an import cycle, which the
// colour spaces break with an interface of their own. So the sentinel lives
// here, where both already import from: `pdmodel.ErrMissingResource` and
// `color.ErrMissingResource` are this one error, and code that recognises the
// exception -- PDFStreamEngine.OperatorException above all -- sees both.
//
// One class in the Java has to be one sentinel here. Two of them is how a page
// whose colour space was missing ended where PDFBox logs it and walks on:
// PDFium's testing/resources/bug_481363.pdf, and
// TestAMissingColourSpaceDoesNotEndThePage in `pdfbox/rendering`.
var ErrMissingResource = errors.New("missing resource")
