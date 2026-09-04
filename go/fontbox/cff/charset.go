package cff

// CFFCharset is a CFF charset: an array of SIDs/CIDs for all glyphs in the
// font.
//
// Port of the org.apache.fontbox.cff.CFFCharset interface.
type CFFCharset interface {
	// IsCIDFont indicates if the charset belongs to a CID font.
	IsCIDFont() bool

	// AddSID adds a new GID/SID/name combination to the charset, name being the
	// postscript name of the glyph.
	AddSID(gid, sid int, name string)

	// AddCID adds a new GID/CID combination to the charset.
	AddCID(gid, cid int)

	// SIDForGID returns the SID for a given GID. SIDs are internal to the font
	// and are not public.
	SIDForGID(gid int) int

	// GIDForSID returns the GID for the given SID. SIDs are internal to the
	// font and are not public.
	GIDForSID(sid int) int

	// GIDForCID returns the GID for a given CID, 0 if the CID is missing.
	GIDForCID(cid int) int

	// SID returns the SID for a given PostScript name; you would think this is
	// not needed, but some fonts have glyphs beyond their encoding with charset
	// SID names.
	SID(name string) int

	// NameForGID returns the PostScript glyph name for the given GID, the
	// second result being false where the charset has no name for it.
	NameForGID(gid int) (string, bool)

	// CIDForGID returns the CID for the given GID.
	CIDForGID(gid int) int
}

// CFFCharsetCID is a CFF charset: an array of CIDs for all glyphs in the font.
//
// Port of org.apache.fontbox.cff.CFFCharsetCID. The methods that make no sense
// for a CID font throw IllegalStateException, which is unchecked, so the port
// panics where Java throws.
type CFFCharsetCID struct {
	sidOrCidToGid map[int]int

	// inverse
	gidToCid map[int]int
}

const notAType1EquivalentFont = "Not a Type 1-equivalent font"

var _ CFFCharset = (*CFFCharsetCID)(nil)

// NewCFFCharsetCID returns an empty CID charset.
func NewCFFCharsetCID() *CFFCharsetCID {
	return &CFFCharsetCID{
		sidOrCidToGid: map[int]int{},
		gidToCid:      map[int]int{},
	}
}

// IsCIDFont indicates if the charset belongs to a CID font.
func (c *CFFCharsetCID) IsCIDFont() bool { return true }

// AddSID is not allowed on a CID charset.
func (c *CFFCharsetCID) AddSID(gid, sid int, name string) {
	panic(notAType1EquivalentFont)
}

// AddCID adds a new GID/CID combination to the charset.
func (c *CFFCharsetCID) AddCID(gid, cid int) {
	c.sidOrCidToGid[cid] = gid
	c.gidToCid[gid] = cid
}

// SIDForGID is not allowed on a CID charset.
func (c *CFFCharsetCID) SIDForGID(gid int) int { panic(notAType1EquivalentFont) }

// GIDForSID is not allowed on a CID charset.
func (c *CFFCharsetCID) GIDForSID(sid int) int { panic(notAType1EquivalentFont) }

// GIDForCID returns the GID for a given CID, 0 if the CID is missing.
func (c *CFFCharsetCID) GIDForCID(cid int) int { return c.sidOrCidToGid[cid] }

// SID is not allowed on a CID charset.
func (c *CFFCharsetCID) SID(name string) int { panic(notAType1EquivalentFont) }

// NameForGID is not allowed on a CID charset.
func (c *CFFCharsetCID) NameForGID(gid int) (string, bool) { panic(notAType1EquivalentFont) }

// CIDForGID returns the CID for the given GID.
func (c *CFFCharsetCID) CIDForGID(gid int) int { return c.gidToCid[gid] }

// CFFCharsetType1 is a CFF charset: an array of SIDs for all glyphs in the
// font.
//
// Port of org.apache.fontbox.cff.CFFCharsetType1. The methods that make no
// sense for a Type 1-equivalent font throw IllegalStateException, which is
// unchecked, so the port panics where Java throws.
type CFFCharsetType1 struct {
	sidOrCidToGid map[int]int
	gidToSid      map[int]int
	nameToSid     map[string]int

	// inverse
	gidToName map[int]string
}

const notACIDFont = "Not a CIDFont"

var _ CFFCharset = (*CFFCharsetType1)(nil)

// NewCFFCharsetType1 returns an empty Type 1-equivalent charset.
func NewCFFCharsetType1() *CFFCharsetType1 {
	return &CFFCharsetType1{
		sidOrCidToGid: map[int]int{},
		gidToSid:      map[int]int{},
		nameToSid:     map[string]int{},
		gidToName:     map[int]string{},
	}
}

// IsCIDFont indicates if the charset belongs to a CID font.
func (c *CFFCharsetType1) IsCIDFont() bool { return false }

// AddSID adds a new GID/SID/name combination to the charset.
func (c *CFFCharsetType1) AddSID(gid, sid int, name string) {
	c.sidOrCidToGid[sid] = gid
	c.gidToSid[gid] = sid
	c.nameToSid[name] = sid
	c.gidToName[gid] = name
}

// AddCID is not allowed on a Type 1-equivalent charset.
func (c *CFFCharsetType1) AddCID(gid, cid int) { panic(notACIDFont) }

// SIDForGID returns the SID for a given GID.
func (c *CFFCharsetType1) SIDForGID(gid int) int { return c.gidToSid[gid] }

// GIDForSID returns the GID for the given SID.
func (c *CFFCharsetType1) GIDForSID(sid int) int { return c.sidOrCidToGid[sid] }

// GIDForCID is not allowed on a Type 1-equivalent charset.
func (c *CFFCharsetType1) GIDForCID(cid int) int { panic(notACIDFont) }

// SID returns the SID for a given PostScript name.
func (c *CFFCharsetType1) SID(name string) int { return c.nameToSid[name] }

// NameForGID returns the PostScript glyph name for the given GID, the second
// result being false where Java would return null.
func (c *CFFCharsetType1) NameForGID(gid int) (string, bool) {
	name, ok := c.gidToName[gid]
	return name, ok
}

// CIDForGID is not allowed on a Type 1-equivalent charset.
func (c *CFFCharsetType1) CIDForGID(gid int) int { panic(notACIDFont) }

// EmbeddedCharset represents an embedded CFF charset.
//
// Port of org.apache.fontbox.cff.EmbeddedCharset, which delegates to whichever
// of the two charsets suits the font.
type EmbeddedCharset struct {
	charset CFFCharset
}

var _ CFFCharset = (*EmbeddedCharset)(nil)

// NewEmbeddedCharset returns an embedded charset, isCIDFont choosing which kind
// it delegates to.
func NewEmbeddedCharset(isCIDFont bool) *EmbeddedCharset {
	if isCIDFont {
		return &EmbeddedCharset{charset: NewCFFCharsetCID()}
	}
	return &EmbeddedCharset{charset: NewCFFCharsetType1()}
}

// CIDForGID returns the CID for the given GID.
func (c *EmbeddedCharset) CIDForGID(gid int) int { return c.charset.CIDForGID(gid) }

// IsCIDFont indicates if the charset belongs to a CID font.
func (c *EmbeddedCharset) IsCIDFont() bool { return c.charset.IsCIDFont() }

// AddSID adds a new GID/SID/name combination to the charset.
func (c *EmbeddedCharset) AddSID(gid, sid int, name string) { c.charset.AddSID(gid, sid, name) }

// AddCID adds a new GID/CID combination to the charset.
func (c *EmbeddedCharset) AddCID(gid, cid int) { c.charset.AddCID(gid, cid) }

// SIDForGID returns the SID for a given GID.
func (c *EmbeddedCharset) SIDForGID(gid int) int { return c.charset.SIDForGID(gid) }

// GIDForSID returns the GID for the given SID.
func (c *EmbeddedCharset) GIDForSID(sid int) int { return c.charset.GIDForSID(sid) }

// GIDForCID returns the GID for a given CID.
func (c *EmbeddedCharset) GIDForCID(cid int) int { return c.charset.GIDForCID(cid) }

// SID returns the SID for a given PostScript name.
func (c *EmbeddedCharset) SID(name string) int { return c.charset.SID(name) }

// NameForGID returns the PostScript glyph name for the given GID.
func (c *EmbeddedCharset) NameForGID(gid int) (string, bool) { return c.charset.NameForGID(gid) }

// newStaticCharset builds one of the three charsets that come from a table, in
// which the row index is the GID.
func newStaticCharset(table []charsetEntry) *CFFCharsetType1 {
	charset := NewCFFCharsetType1()
	for gid, entry := range table {
		charset.AddSID(gid, entry.sid, entry.name)
	}
	return charset
}

// CFFISOAdobeCharset is the specialized CFFCharset used if the CharsetId of a
// font is set to 0.
//
// Port of org.apache.fontbox.cff.CFFISOAdobeCharset. Java exposes a singleton
// through getInstance; the port has the instance alone.
var CFFISOAdobeCharset = newStaticCharset(cffIsoAdobeCharsetTable)

// CFFExpertCharset is the specialized CFFCharset used if the CharsetId of a
// font is set to 1.
//
// Port of org.apache.fontbox.cff.CFFExpertCharset.
var CFFExpertCharset = newStaticCharset(cffExpertCharsetTable)

// CFFExpertSubsetCharset is the specialized CFFCharset used if the CharsetId of
// a font is set to 2.
//
// Port of org.apache.fontbox.cff.CFFExpertSubsetCharset.
var CFFExpertSubsetCharset = newStaticCharset(cffExpertSubsetCharsetTable)
