package pdmodel

import "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"

// ErrMissingResource reports a named resource that is not in the resource
// dictionary.
//
// Port of org.apache.pdfbox.pdmodel.MissingResourceException. The sentinel
// itself is `common.ErrMissingResource`, so that the colour spaces -- which
// cannot import this package -- raise the same error and not a second one of
// their own; see the comment there.
var ErrMissingResource = common.ErrMissingResource
