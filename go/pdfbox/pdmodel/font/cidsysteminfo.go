package font

import "strconv"

// CIDSystemInfo is a CIDSystemInfo for the FontMapper API.
//
// Port of org.apache.pdfbox.pdmodel.font.CIDSystemInfo. It is the plain triple
// a system font carries; PDCIDSystemInfo is the one a PDF dictionary carries.
type CIDSystemInfo struct {
	registry   string
	ordering   string
	supplement int
}

// NewCIDSystemInfo returns the CIDSystemInfo naming the given collection.
func NewCIDSystemInfo(registry, ordering string, supplement int) *CIDSystemInfo {
	return &CIDSystemInfo{registry: registry, ordering: ordering, supplement: supplement}
}

// Registry returns the issuer of the character collection.
func (i *CIDSystemInfo) Registry() string { return i.registry }

// Ordering returns the name of the character collection.
func (i *CIDSystemInfo) Ordering() string { return i.ordering }

// Supplement returns the supplement number of the character collection.
func (i *CIDSystemInfo) Supplement() int { return i.supplement }

// String returns "Registry-Ordering-Supplement".
func (i *CIDSystemInfo) String() string {
	return i.Registry() + "-" + i.Ordering() + "-" + strconv.Itoa(i.Supplement())
}
