package taggedpdf

import "slices"

// The standard structure types of a tagged PDF.
//
// Port of StandardStructureTypes, whose constants these are, one for one and in
// its order.
const (
	// Grouping Elements

	Document   = "Document"
	Part       = "Part"
	Art        = "Art"
	Sect       = "Sect"
	Div        = "Div"
	BlockQuote = "BlockQuote"
	Caption    = "Caption"
	TOC        = "TOC"
	TOCI       = "TOCI"
	Index      = "Index"
	NonStruct  = "NonStruct"
	Private    = "Private"

	// Block-Level Structure Elements

	P     = "P"
	H     = "H"
	H1    = "H1"
	H2    = "H2"
	H3    = "H3"
	H4    = "H4"
	H5    = "H5"
	H6    = "H6"
	L     = "L"
	LI    = "LI"
	LBL   = "Lbl"
	LBody = "LBody"
	Table = "Table"
	TR    = "TR"
	TH    = "TH"
	TD    = "TD"
	THead = "THead"
	TBody = "TBody"
	TFoot = "TFoot"

	// Inline-Level Structure Elements

	Span      = "Span"
	Quote     = "Quote"
	Note      = "Note"
	Reference = "Reference"
	BibEntry  = "BibEntry"
	Code      = "Code"
	Link      = "Link"
	Annot     = "Annot"
	Ruby      = "Ruby"
	RB        = "RB"
	RT        = "RT"
	RP        = "RP"
	Warichu   = "Warichu"
	WT        = "WT"
	WP        = "WP"

	// Illustration Elements

	Figure  = "Figure"
	Formula = "Formula"
	Form    = "Form"
)

// Types holds every standard structure type, sorted.
//
// Java fills its list by reflection over its own public final fields, which
// picks up the list itself as well: it adds the list's own toString, so the
// list holds one entry that is not a structure type, and which entry that is
// depends on the order Class.getFields answers in, which the JVM does not
// define. See migration/JAVA-BUGS.md entry 42. Go has no such reflection over
// constants, so the port names them and leaves the self-referential entry out.
var Types = slices.Sorted(slices.Values([]string{
	Document, Part, Art, Sect, Div, BlockQuote, Caption, TOC, TOCI, Index,
	NonStruct, Private,
	P, H, H1, H2, H3, H4, H5, H6, L, LI, LBL, LBody, Table, TR, TH, TD, THead,
	TBody, TFoot,
	Span, Quote, Note, Reference, BibEntry, Code, Link, Annot, Ruby, RB, RT, RP,
	Warichu, WT, WP,
	Figure, Formula, Form,
}))
