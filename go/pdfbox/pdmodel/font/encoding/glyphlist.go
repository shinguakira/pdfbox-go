package encoding

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/resources"
)

// GlyphList maps glyph names onto the Unicode they stand for, and back.
//
// Port of org.apache.pdfbox.pdmodel.font.encoding.GlyphList.
type GlyphList struct {
	// read-only mappings, never modified outside the constructors
	nameToUnicode map[string]string
	unicodeToName map[string]string

	// additional read/write cache for uniXXXX names
	uniNameToUnicodeCache sync.Map
}

// The two glyph lists the library ships with. Java builds them in a static
// initialiser and rethrows any failure as a RuntimeException; the port panics
// for the same reason, since neither file can be missing from the binary.
var (
	// Adobe Glyph List (AGL)
	defaultGlyphList = mustLoad("glyphlist.txt", 4281)

	// Zapf Dingbats has its own glyph list
	zapfDingbatsGlyphList = mustLoad("zapfdingbats.txt", 201)
)

// mustLoad reads one of the shipped glyph lists.
func mustLoad(filename string, numberOfEntries int) *GlyphList {
	path := "glyphlist/" + filename
	input, err := resources.Open(path)
	if err != nil {
		panic(fmt.Errorf("GlyphList '/org/apache/pdfbox/resources/%s' not found: %w", path, err))
	}
	defer input.Close()
	glyphList, err := NewGlyphList(input, numberOfEntries)
	if err != nil {
		panic(err)
	}
	return glyphList
}

// AdobeGlyphList returns the Adobe Glyph List.
func AdobeGlyphList() *GlyphList { return defaultGlyphList }

// ZapfDingbats returns the glyph list of the Zapf Dingbats font.
func ZapfDingbats() *GlyphList { return zapfDingbatsGlyphList }

// NewGlyphList returns the glyph list the given stream describes.
func NewGlyphList(input io.Reader, numberOfEntries int) (*GlyphList, error) {
	g := &GlyphList{
		nameToUnicode: make(map[string]string, numberOfEntries),
		unicodeToName: make(map[string]string, numberOfEntries),
	}
	if err := g.loadList(input); err != nil {
		return nil, err
	}
	return g, nil
}

// NewGlyphListFrom returns a glyph list that starts from another one and then
// reads the given stream over it.
func NewGlyphListFrom(glyphList *GlyphList, input io.Reader) (*GlyphList, error) {
	g := &GlyphList{
		nameToUnicode: make(map[string]string, len(glyphList.nameToUnicode)),
		unicodeToName: make(map[string]string, len(glyphList.unicodeToName)),
	}
	for name, unicode := range glyphList.nameToUnicode {
		g.nameToUnicode[name] = unicode
	}
	for unicode, name := range glyphList.unicodeToName {
		g.unicodeToName[unicode] = name
	}
	if err := g.loadList(input); err != nil {
		return nil, err
	}
	return g, nil
}

// loadList reads a glyph list file: one "name;codepoints" line per glyph, with
// comments starting at a hash.
func (g *GlyphList) loadList(input io.Reader) error {
	in := bufio.NewScanner(input)
	for in.Scan() {
		line := strings.TrimRight(in.Text(), "\r")
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ";")
		if len(parts) < 2 {
			return fmt.Errorf("encoding: Invalid glyph list entry: %s", line)
		}
		name := parts[0]
		unicodeList := strings.Split(parts[1], " ")
		codePoints := make([]rune, len(unicodeList))
		for i, hex := range unicodeList {
			codePoint, err := strconv.ParseInt(hex, 16, 32)
			if err != nil {
				// Java's Integer.parseInt throws NumberFormatException, which
				// nothing here catches.
				panic(err)
			}
			codePoints[i] = rune(codePoint)
		}
		str := string(codePoints)

		// forward mapping
		// Java logs where a name is already mapped; the later value wins either
		// way.
		g.nameToUnicode[name] = str

		// reverse mapping
		// PDFBOX-3884: take the various standard encodings as canonical,
		// e.g. tilde over ilde
		forceOverride := WinAnsiEncodingInstance.ContainsName(name) ||
			MacRomanEncodingInstance.ContainsName(name) ||
			MacExpertEncodingInstance.ContainsName(name) ||
			SymbolEncodingInstance.ContainsName(name) ||
			ZapfDingbatsEncodingInstance.ContainsName(name)
		if forceOverride {
			g.unicodeToName[str] = name
		} else if _, ok := g.unicodeToName[str]; !ok {
			g.unicodeToName[str] = name
		}
	}
	return in.Err()
}

// CodePointToName returns the name of the glyph the given code point is drawn
// with, or ".notdef" where the list has none.
func (g *GlyphList) CodePointToName(codePoint int) string {
	name, ok := g.unicodeToName[string(rune(codePoint))]
	if !ok {
		return ".notdef"
	}
	return name
}

// SequenceToName returns the name of the glyph the given sequence is drawn
// with, or ".notdef" where the list has none.
func (g *GlyphList) SequenceToName(unicodeSequence string) string {
	name, ok := g.unicodeToName[unicodeSequence]
	if !ok {
		return ".notdef"
	}
	return name
}

// ToUnicode returns what the named glyph stands for, or the empty string where
// the list does not know the name and it is not of a form that names a code
// point outright. Java returns null there.
func (g *GlyphList) ToUnicode(name string) string {
	if unicode, ok := g.nameToUnicode[name]; ok {
		return unicode
	}

	// separate read/write cache for thread safety
	if cached, ok := g.uniNameToUnicodeCache.Load(name); ok {
		return cached.(string)
	}

	unicode := ""
	// test if we have a suffix and if so remove it
	if dot := strings.IndexByte(name, '.'); dot > 0 {
		unicode = g.ToUnicode(name[:dot])
	} else if (len(name) == 7 && strings.HasPrefix(name, "uni")) ||
		(len(name) == 5 && strings.HasPrefix(name, "u")) {
		// test for Unicode name in the format uniXXXX/uXXXX where X is hex
		start := 1
		if len(name) == 7 {
			start = 3
		}
		codePoint, err := strconv.ParseInt(name[start:start+4], 16, 32)
		if err != nil {
			// Not a number in Unicode character name.
			return ""
		}
		if codePoint > 0xD7FF && codePoint < 0xE000 {
			// Unicode character name with disallowed code area.
			return ""
		}
		unicode = string(rune(codePoint))
	}

	if unicode != "" {
		// null value not allowed in ConcurrentHashMap
		g.uniNameToUnicodeCache.Store(name, unicode)
	}
	return unicode
}
