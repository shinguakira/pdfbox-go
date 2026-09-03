package pdmodel

import "errors"

// ErrMissingResource reports a named resource that is not in the resource
// dictionary.
//
// Port of org.apache.pdfbox.pdmodel.MissingResourceException.
var ErrMissingResource = errors.New("missing resource")
