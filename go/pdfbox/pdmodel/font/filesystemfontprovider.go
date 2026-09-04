package font

import (
	"bufio"
	"fmt"
	"hash/crc32"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/shinguakira/pdfbox-go/go/fontbox"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/fontbox/type1"
	"github.com/shinguakira/pdfbox-go/go/fontbox/util/autodetect"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// SkipChecksums changes publicly visible behaviour: the ".pdfbox.cache" file
// will have hash="-" for all files. After implementing FontHeaders, parsing
// font headers is faster than checksumming anyway.
//
// Port of the FileSystemFontProvider.SKIP_CHECKSUMS flag, which Java reads once
// from the pdfbox.fontcache.skipchecksums system property. Go has no system
// properties, so it is a package variable with the same default (false).
var SkipChecksums = false

// FontCacheDir is the directory the on-disk font cache is written to.
//
// Port of the pdfbox.fontcache system property. Empty, the default, means the
// user's home directory, and the temporary directory where that cannot be
// written to.
var FontCacheDir = ""

const checksumPlaceholder = "-"

// lineSeparator is what Java's BufferedWriter.newLine writes, which is the
// platform's line.separator; the cache file is shared with the Java on the same
// machine, so the port writes the same bytes.
var lineSeparator = func() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}()

// fileSystemFontProvider is a FontProvider which searches for fonts on the
// local filesystem.
//
// Port of the package-private final class
// org.apache.pdfbox.pdmodel.font.FileSystemFontProvider.
type fileSystemFontProvider struct {
	fontInfoList []*fsFontInfo
	cache        *FontCache
}

var _ FontProvider = (*fileSystemFontProvider)(nil)

// fsFontInfo is what the provider knows about one font file.
//
// Port of the private static class FileSystemFontProvider.FSFontInfo.
type fsFontInfo struct {
	postScriptName   string
	format           FontFormat
	cidSystemInfo    *CIDSystemInfo
	usWeightClass    int
	sFamilyClass     int
	ulCodePageRange1 int
	ulCodePageRange2 int
	macStyle         int
	panose           *PDPanoseClassification
	file             string
	parent           *fileSystemFontProvider
	hash             string
	lastModified     int64

	// mu stands for the synchronized on getFont.
	mu sync.Mutex
}

var _ FontInfo = (*fsFontInfo)(nil)

func newFSFontInfo(file string, format FontFormat, postScriptName string,
	cidSystemInfo *CIDSystemInfo, usWeightClass, sFamilyClass, ulCodePageRange1,
	ulCodePageRange2, macStyle int, panose []byte, parent *fileSystemFontProvider,
	hash string, lastModified int64) *fsFontInfo {
	info := &fsFontInfo{
		file:             file,
		format:           format,
		postScriptName:   postScriptName,
		cidSystemInfo:    cidSystemInfo,
		usWeightClass:    usWeightClass,
		sFamilyClass:     sFamilyClass,
		ulCodePageRange1: ulCodePageRange1,
		ulCodePageRange2: ulCodePageRange2,
		macStyle:         macStyle,
		parent:           parent,
		hash:             hash,
		lastModified:     lastModified,
	}
	if len(panose) >= PanoseClassificationLength {
		info.panose = NewPDPanoseClassification(panose)
	}
	return info
}

func (i *fsFontInfo) PostScriptName() string { return i.postScriptName }

func (i *fsFontInfo) Format() FontFormat { return i.format }

func (i *fsFontInfo) CIDSystemInfo() *CIDSystemInfo { return i.cidSystemInfo }

// Font returns a new FontBox font instance for the font, or nil if there was an
// error opening it.
func (i *fsFontInfo) Font() fontbox.FontBoxFont {
	// synchronized to avoid race condition on cache access,
	// which could result in an unreferenced but open font
	i.mu.Lock()
	defer i.mu.Unlock()

	if cached := i.parent.cache.GetFont(i); cached != nil {
		return cached
	}
	var font fontbox.FontBoxFont
	switch i.format {
	case FontFormatPFB:
		if t1 := getType1Font(i.postScriptName, i.file); t1 != nil {
			font = t1
		}
	case FontFormatTTF:
		if trueType := getTrueTypeFont(i.postScriptName, i.file); trueType != nil {
			font = trueType
		}
	case FontFormatOTF:
		if openType := getOTFFont(i.postScriptName, i.file); openType != nil {
			font = openType
		}
	default:
		panic("can't happen")
	}
	if font != nil {
		i.parent.cache.AddFont(i, font)
	}
	return font
}

func (i *fsFontInfo) FamilyClass() int { return i.sFamilyClass }

func (i *fsFontInfo) WeightClass() int { return i.usWeightClass }

func (i *fsFontInfo) CodePageRange1() int { return i.ulCodePageRange1 }

func (i *fsFontInfo) CodePageRange2() int { return i.ulCodePageRange2 }

func (i *fsFontInfo) MacStyle() int { return i.macStyle }

func (i *fsFontInfo) Panose() *PDPanoseClassification { return i.panose }

func (i *fsFontInfo) String() string {
	return fmt.Sprintf("%s %s %s %d", fontInfoString(i), i.file, i.hash, i.lastModified)
}

func getTrueTypeFont(postScriptName, file string) *ttf.TrueTypeFont {
	trueTypeFont, err := readTrueTypeFont(postScriptName, file)
	if err != nil {
		slog.Warn("Could not load font file", "file", file, "err", err)
		return nil
	}
	slog.Debug("Loaded font", "font", postScriptName, "file", file)
	return trueTypeFont
}

func readTrueTypeFont(postScriptName, file string) (*ttf.TrueTypeFont, error) {
	if strings.HasSuffix(strings.ToLower(filepath.Base(file)), ".ttc") {
		// ttc not closed here because it is needed later when ttf is accessed,
		// e.g. rendering PDF with non-embedded font which is in ttc file in our font directory
		collection, err := ttf.NewTrueTypeCollectionFile(file)
		if err != nil {
			return nil, err
		}
		trueTypeFont, err := collection.FontByName(postScriptName)
		if err != nil {
			collection.Close()
			return nil, err
		}
		if trueTypeFont == nil {
			collection.Close()
			return nil, fmt.Errorf("Font %s not found in %s", postScriptName, file)
		}
		return trueTypeFont, nil
	}
	source, err := pdfio.OpenBufferedFile(file)
	if err != nil {
		return nil, err
	}
	return ttf.NewParserEmbedded(false).Parse(source)
}

func getOTFFont(postScriptName, file string) *ttf.OpenTypeFont {
	openType, err := readOTFFont(postScriptName, file)
	if err != nil {
		slog.Warn("Could not load font file", "file", file, "err", err)
		return nil
	}
	return openType
}

func readOTFFont(postScriptName, file string) (*ttf.OpenTypeFont, error) {
	if strings.HasSuffix(strings.ToLower(filepath.Base(file)), ".ttc") {
		// ttc not closed here because it is needed later when ttf is accessed,
		// e.g. rendering PDF with non-embedded font which is in ttc file in our font directory
		collection, err := ttf.NewTrueTypeCollectionFile(file)
		if err != nil {
			return nil, err
		}
		trueTypeFont, err := collection.FontByName(postScriptName)
		if err != nil {
			// Java logs this one and returns null rather than warning like the
			// rest of the method does.
			slog.Error("Could not read the font collection", "file", file, "err", err)
			collection.Close()
			return nil, nil
		}
		if trueTypeFont == nil {
			collection.Close()
			return nil, fmt.Errorf("Font %s not found in %s", postScriptName, file)
		}
		openType := trueTypeFont.AsOpenType()
		if openType == nil {
			// Java casts to OpenTypeFont here, which throws ClassCastException
			// where the collection member is not one.
			panic(fmt.Sprintf("font: %s in %s is not an OpenType font", postScriptName, file))
		}
		return openType, nil
	}

	source, err := pdfio.OpenBufferedFile(file)
	if err != nil {
		return nil, err
	}
	openType, err := ttf.NewOTFParserEmbedded(false).Parse(source)
	if err != nil {
		return nil, err
	}
	slog.Debug("Loaded font", "font", postScriptName, "file", file)
	return openType, nil
}

func getType1Font(postScriptName, file string) *type1.Type1Font {
	input, err := os.Open(file)
	if err != nil {
		slog.Warn("Could not load font file", "file", file, "err", err)
		return nil
	}
	defer input.Close()
	t1, err := type1.CreateWithPFB(input)
	if err != nil {
		slog.Warn("Could not load font file", "file", file, "err", err)
		return nil
	}
	slog.Debug("Loaded font", "font", postScriptName, "file", file)
	return t1
}

func (p *fileSystemFontProvider) createFSIgnored(file string, format FontFormat,
	postScriptName string) *fsFontInfo {
	var hash string
	if SkipChecksums {
		hash = checksumPlaceholder
	} else {
		computed, err := computeHashOfFile(file)
		if err != nil {
			hash = ""
		} else {
			hash = computed
		}
	}
	// JAVA-BUGS entry 21: Java passes null for the parent here, so Font() on an
	// ignored entry dereferences it. Ported as written; the Go panics where
	// Java throws NullPointerException.
	return newFSFontInfo(file, format, postScriptName, nil, 0, 0, 0, 0, 0, nil, nil, hash,
		lastModified(file))
}

// newFileSystemFontProvider scans the local system for fonts.
//
// Java catches AccessControlException around the whole body, which a security
// manager raises where the process may not read the file system; Go has no such
// manager and the walk simply finds nothing.
func newFileSystemFontProvider(cache *FontCache) *fileSystemFontProvider {
	p := &fileSystemFontProvider{cache: cache}

	slog.Debug("Will search the local system for fonts")

	// scan the local system for font files
	files := autodetect.NewFontFileFinder().Find()

	slog.Debug("Found fonts on the local system", "count", len(files))

	if len(files) > 0 {
		// load cached FontInfo objects
		cachedInfos := p.loadDiskCache(files)
		if len(cachedInfos) > 0 {
			p.fontInfoList = append(p.fontInfoList, cachedInfos...)
		} else {
			slog.Info("Building on-disk font cache, this may take a while")
			p.scanFonts(files)
			p.saveDiskCache()
			slog.Info("Finished building on-disk font cache",
				"fonts", len(p.fontInfoList))
		}
	}
	return p
}

func (p *fileSystemFontProvider) scanFonts(files []string) {
	// to force a specific font for debug, add code like this here:
	// files = Collections.singletonList(new File("font filename"))

	for _, file := range files {
		filePath := strings.ToLower(file)
		switch {
		case strings.HasSuffix(filePath, ".ttf") || strings.HasSuffix(filePath, ".otf"):
			p.addTrueTypeFont(file)
		case strings.HasSuffix(filePath, ".ttc") || strings.HasSuffix(filePath, ".otc"):
			p.addTrueTypeCollection(file)
		case strings.HasSuffix(filePath, ".pfb"):
			p.addType1Font(file)
		}
	}
}

func getDiskCacheFile() string {
	path := FontCacheDir
	if isBadPath(path) {
		path = userHome()
		if isBadPath(path) {
			path = os.TempDir()
		}
	}
	return filepath.Join(path, ".pdfbox.cache")
}

// isBadPath is Java's `path == null || !isDirectory() || !canWrite()`. Go
// reports the write permission of a directory in the mode bits on every
// platform this runs on, which is what File.canWrite reads too.
func isBadPath(path string) bool {
	if path == "" {
		return true
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return true
	}
	return info.Mode().Perm()&0o200 == 0
}

// userHome is Java's user.home system property.
func userHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// lastModified is Java's File.lastModified, which is zero where the file cannot
// be read.
func lastModified(file string) int64 {
	info, err := os.Stat(file)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixMilli()
}

// saveDiskCache saves the font metadata cache to disk.
func (p *fileSystemFontProvider) saveDiskCache() {
	file := getDiskCacheFile()

	out, err := os.Create(file)
	if err != nil {
		p.warnCacheNotWritten(err)
		return
	}
	writer := bufio.NewWriter(out)
	for _, fontInfo := range p.fontInfoList {
		if err := writeFontInfo(writer, fontInfo); err != nil {
			out.Close()
			p.warnCacheNotWritten(err)
			return
		}
	}
	if err := writer.Flush(); err != nil {
		out.Close()
		p.warnCacheNotWritten(err)
		return
	}
	if err := out.Close(); err != nil {
		p.warnCacheNotWritten(err)
	}
}

func (p *fileSystemFontProvider) warnCacheNotWritten(err error) {
	slog.Warn("Could not write to font cache", "err", err)
	slog.Warn("Installed fonts information will have to be reloaded for each start")
	slog.Warn("You can assign a directory to the 'pdfbox.fontcache' property")
}

func writeFontInfo(writer *bufio.Writer, fontInfo *fsFontInfo) error {
	writer.WriteString(javaTrim(fontInfo.postScriptName))
	writer.WriteString("|")
	writer.WriteString(fontInfo.format.String())
	writer.WriteString("|")
	if fontInfo.cidSystemInfo != nil {
		writer.WriteString(fontInfo.cidSystemInfo.Registry() + "-" +
			fontInfo.cidSystemInfo.Ordering() + "-" +
			strconv.Itoa(fontInfo.cidSystemInfo.Supplement()))
	}
	writer.WriteString("|")
	if fontInfo.usWeightClass > -1 {
		writer.WriteString(toHexString(fontInfo.usWeightClass))
	}
	writer.WriteString("|")
	if fontInfo.sFamilyClass > -1 {
		writer.WriteString(toHexString(fontInfo.sFamilyClass))
	}
	writer.WriteString("|")
	writer.WriteString(toHexString(fontInfo.ulCodePageRange1))
	writer.WriteString("|")
	writer.WriteString(toHexString(fontInfo.ulCodePageRange2))
	writer.WriteString("|")
	if fontInfo.macStyle > -1 {
		writer.WriteString(toHexString(fontInfo.macStyle))
	}
	writer.WriteString("|")
	if fontInfo.panose != nil {
		bytes := fontInfo.panose.Bytes()
		for i := 0; i < 10; i++ {
			// JAVA-BUGS entry 19: Java widens the signed byte before
			// Integer.toHexString, so a Panose value of 0x80 or more writes
			// eight hex digits where the reader expects two. Ported as written.
			str := toHexString(int(int8(bytes[i])))
			if len(str) == 1 {
				writer.WriteString("0")
			}
			writer.WriteString(str)
		}
	}
	writer.WriteString("|")
	absolute, err := filepath.Abs(fontInfo.file)
	if err != nil {
		absolute = fontInfo.file
	}
	writer.WriteString(absolute)
	writer.WriteString("|")
	writer.WriteString(fontInfo.hash)
	writer.WriteString("|")
	writer.WriteString(strconv.FormatInt(lastModified(fontInfo.file), 10))
	_, err = writer.WriteString(lineSeparator)
	return err
}

// javaTrim is String.trim, which strips every character up to and including the
// space rather than the Unicode whitespace strings.TrimSpace strips.
func javaTrim(s string) string {
	start := 0
	end := len(s)
	for start < end && s[start] <= ' ' {
		start++
	}
	for start < end && s[end-1] <= ' ' {
		end--
	}
	return s[start:end]
}

// loadDiskCache loads the font metadata cache from disk, returning nil where
// the cache has to be rebuilt.
//
// Java parses the fields with valueOf and parseInt, which throw unchecked
// exceptions on a damaged line; those are not caught here any more than they
// are there, so the port panics where the Java would propagate.
func (p *fileSystemFontProvider) loadDiskCache(files []string) []*fsFontInfo {
	pending := make(map[string]bool, len(files))
	for _, file := range files {
		pending[absolutePath(file)] = true
	}

	var results []*fsFontInfo

	// Get the disk cache
	diskCacheFile := getDiskCacheFile()
	fileExists := false
	if _, err := os.Stat(diskCacheFile); err == nil {
		fileExists = true
	}

	if fileExists {
		lines, err := readLines(diskCacheFile)
		if err != nil {
			slog.Warn("Error loading font cache, will be re-built", "err", err)
			return nil
		}
		// consequent lines usually share the same font file (e.g. "Courier",
		// "Courier-Bold", "Courier-Oblique").
		// unused if SkipChecksums
		lastFile := ""
		lastHash := ""
		for _, line := range lines {
			parts := strings.SplitN(line, "|", 12)
			if len(parts) < 10 {
				slog.Warn("Incorrect line in font disk cache is skipped", "line", line)
				continue
			}

			var cidSystemInfo *CIDSystemInfo
			usWeightClass := -1
			sFamilyClass := -1
			macStyle := -1
			var panose []byte
			hash := ""
			var lastModifiedField int64

			postScriptName := parts[0]
			format, ok := ParseFontFormat(parts[1])
			if !ok {
				panic(fmt.Sprintf("font: no font format named %q", parts[1]))
			}
			if parts[2] != "" {
				ros := strings.Split(parts[2], "-")
				cidSystemInfo = NewCIDSystemInfo(ros[0], ros[1], parseInt(ros[2], 10))
			}
			if parts[3] != "" {
				usWeightClass = int(parseLong(parts[3], 16))
			}
			if parts[4] != "" {
				sFamilyClass = int(parseLong(parts[4], 16))
			}
			ulCodePageRange1 := int(parseLong(parts[5], 16))
			ulCodePageRange2 := int(parseLong(parts[6], 16))
			if parts[7] != "" {
				macStyle = int(parseLong(parts[7], 16))
			}
			if parts[8] != "" {
				panose = make([]byte, 10)
				for i := 0; i < 10; i++ {
					str := parts[8][i*2 : i*2+2]
					b := parseInt(str, 16)
					panose[i] = byte(b & 0xff)
				}
			}
			fontFile := parts[9]
			if len(parts) >= 12 && parts[10] != "" && parts[11] != "" {
				hash = parts[10]
				lastModifiedField = parseLong(parts[11], 10)
			}
			if _, err := os.Stat(fontFile); err == nil {
				// if the file exists, find out whether it's the same file.
				// first check whether time is different and if yes, whether hash is different
				keep := lastModified(fontFile) == lastModifiedField
				if !keep && !SkipChecksums {
					var newHash string
					if hash == lastHash && fontFile == lastFile {
						newHash = lastHash // already computed
					} else {
						computed, err := computeHashOfFile(fontFile)
						if err != nil {
							slog.Debug("Error reading font file",
								"file", absolutePath(fontFile), "err", err)
							newHash = "<err>"
						} else {
							newHash = computed
							lastFile = fontFile
							lastHash = newHash
						}
					}
					if hash == newHash {
						keep = true
						lastModifiedField = lastModified(fontFile)
					}
				}
				if keep {
					info := newFSFontInfo(fontFile, format, postScriptName, cidSystemInfo,
						usWeightClass, sFamilyClass, ulCodePageRange1, ulCodePageRange2,
						macStyle, panose, p, hash, lastModifiedField)
					results = append(results, info)
				} else {
					slog.Debug("Font file is different", "file", absolutePath(fontFile))
					continue // don't remove from "pending"
				}
			} else {
				slog.Debug("Font file not found, skipped", "file", absolutePath(fontFile))
			}
			delete(pending, absolutePath(fontFile))
		}
	}

	if len(pending) > 0 {
		// re-build the entire cache if we encounter un-cached fonts (could be optimised)
		slog.Info("New font files found, font cache will be re-built", "count", len(pending))
		return nil
	}

	return results
}

// absolutePath is Java's File.getAbsolutePath, which falls back on the path as
// given where the working directory cannot be read.
func absolutePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absolute
}

// readLines is BufferedReader.readLine over the whole file: a line ends at
// "\n", "\r" or "\r\n", and the terminator is not part of it.
func readLines(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	start := 0
	for i := 0; i < len(content); i++ {
		switch content[i] {
		case '\n':
			lines = append(lines, string(content[start:i]))
			start = i + 1
		case '\r':
			lines = append(lines, string(content[start:i]))
			if i+1 < len(content) && content[i+1] == '\n' {
				i++
			}
			start = i + 1
		}
	}
	if start < len(content) {
		lines = append(lines, string(content[start:]))
	}
	return lines, nil
}

// parseInt is Integer.parseInt, which throws NumberFormatException for a string
// that is not a number in the given radix.
func parseInt(s string, radix int) int {
	value, err := strconv.ParseInt(s, radix, 64)
	if err != nil {
		panic(fmt.Sprintf("font: For input string: %q", s))
	}
	return int(value)
}

// parseLong is Long.parseLong, which throws NumberFormatException for a string
// that is not a number in the given radix.
func parseLong(s string, radix int) int64 {
	value, err := strconv.ParseInt(s, radix, 64)
	if err != nil {
		panic(fmt.Sprintf("font: For input string: %q", s))
	}
	return value
}

// addTrueTypeCollection adds a TTC or OTC to the file cache. To reduce memory,
// the parsed font is not cached.
func (p *fileSystemFontProvider) addTrueTypeCollection(ttcFile string) {
	hash := checksumPlaceholder
	if !SkipChecksums {
		computed, err := computeHashOfFile(ttcFile)
		if err != nil {
			slog.Warn("Could not load font file", "file", ttcFile, "err", err)
			p.fontInfoList = append(p.fontInfoList,
				p.createFSIgnored(ttcFile, FontFormatTTF, "*skipexception*"))
			return
		}
		hash = computed
	}
	err := ttf.ProcessAllFontHeaders(ttcFile, func(fontHeaders *ttf.FontHeaders) {
		p.addTrueTypeFontImpl(fontHeaders, ttcFile, hash)
	})
	if err != nil {
		slog.Warn("Could not load font file", "file", ttcFile, "err", err)
		p.fontInfoList = append(p.fontInfoList,
			p.createFSIgnored(ttcFile, FontFormatTTF, "*skipexception*"))
	}
}

// addTrueTypeFont adds an OTF or TTF font to the file cache. To reduce memory,
// the parsed font is not cached.
func (p *fileSystemFontProvider) addTrueTypeFont(ttfFile string) {
	// Java starts fontFormat at null, but assigns it before anything that can
	// throw, so the catch below never sees the null.
	var fontFormat FontFormat
	var parser *ttf.Parser
	if strings.HasSuffix(strings.ToLower(ttfFile), ".otf") {
		fontFormat = FontFormatOTF
		parser = ttf.NewOTFParserEmbedded(false).Parser
	} else {
		fontFormat = FontFormatTTF
		parser = ttf.NewParserEmbedded(false)
	}
	fontHeaders, err := p.readTableHeaders(parser, ttfFile)
	if err == nil {
		hash := checksumPlaceholder
		if !SkipChecksums {
			hash, err = computeHashOfFile(ttfFile)
		}
		if err == nil {
			p.addTrueTypeFontImpl(fontHeaders, ttfFile, hash)
			return
		}
	}
	slog.Warn("Could not load font file", "file", ttfFile, "err", err)
	p.fontInfoList = append(p.fontInfoList,
		p.createFSIgnored(ttfFile, fontFormat, "*skipexception*"))
}

func (p *fileSystemFontProvider) readTableHeaders(parser *ttf.Parser,
	ttfFile string) (*ttf.FontHeaders, error) {
	source, err := pdfio.OpenBufferedFile(ttfFile)
	if err != nil {
		return nil, err
	}
	return parser.ParseTableHeaders(source)
}

// addTrueTypeFontImpl adds an OTF or TTF font to the file cache. To reduce
// memory, the parsed font is not cached.
func (p *fileSystemFontProvider) addTrueTypeFontImpl(fontHeaders *ttf.FontHeaders,
	file string, hash string) {
	errorMessage := fontHeaders.Error()
	if errorMessage != "" {
		p.fontInfoList = append(p.fontInfoList,
			p.createFSIgnored(file, FontFormatTTF, "*skipexception*"))
		slog.Warn("Could not load font file", "file", file, "err", errorMessage)
		return
	}
	// read PostScript name, if any
	//
	// Java tests the name for null and takes the empty string as a name like
	// any other; FontHeaders.getName is a Go string, so the two cases are one
	// here. A font whose name record is present but empty is recorded as
	// *skipnoname* rather than under the empty name, which no lookup reaches
	// either way.
	name := fontHeaders.Name()
	if name == "" {
		p.fontInfoList = append(p.fontInfoList,
			p.createFSIgnored(file, FontFormatTTF, "*skipnoname*"))
		slog.Warn("Missing 'name' entry for PostScript name in font", "file", file)
		return
	}
	if strings.Contains(name, "|") {
		p.fontInfoList = append(p.fontInfoList,
			p.createFSIgnored(file, FontFormatTTF, "*skippipeinname*"))
		slog.Warn("Skipping font with '|' in name", "name", name, "file", file)
		return
	}

	// ignore bitmap fonts
	macStyle := fontHeaders.HeaderMacStyle()
	if macStyle == nil {
		p.fontInfoList = append(p.fontInfoList, p.createFSIgnored(file, FontFormatTTF, name))
		return
	}

	sFamilyClass := -1
	usWeightClass := -1
	ulCodePageRange1 := 0
	ulCodePageRange2 := 0
	var panose []byte
	os2WindowsMetricsTable := fontHeaders.OS2Windows()
	// Apple's AAT fonts don't have an OS/2 table
	if os2WindowsMetricsTable != nil {
		sFamilyClass = os2WindowsMetricsTable.FamilyClass()
		usWeightClass = os2WindowsMetricsTable.WeightClass()
		ulCodePageRange1 = int(int32(os2WindowsMetricsTable.CodePageRange1()))
		ulCodePageRange2 = int(int32(os2WindowsMetricsTable.CodePageRange2()))
		panose = os2WindowsMetricsTable.Panose()
	}

	var format FontFormat
	var ros *CIDSystemInfo
	if fontHeaders.IsOpenTypePostScript() {
		format = FontFormatOTF
		registry := fontHeaders.OtfRegistry()
		ordering := fontHeaders.OtfOrdering()
		if registry != "" || ordering != "" {
			ros = NewCIDSystemInfo(registry, ordering, fontHeaders.OtfSupplement())
		}
	} else {
		bytes := fontHeaders.NonOtfTableGCID142()
		if bytes != nil {
			// Apple's AAT fonts have a "gcid" table with CID info
			reg := string(bytes[10 : 10+64])
			registryName := reg[:strings.IndexByte(reg, 0)]
			ord := string(bytes[76 : 76+64])
			orderName := ord[:strings.IndexByte(ord, 0)]
			// JAVA-BUGS entry 20: Java ANDs the two halves of the supplement
			// where it means to OR them, so the value is always zero. Ported as
			// written.
			supplementVersion := int(int8(bytes[140])) << 8 & (int(bytes[141]) & 0xFF)
			ros = NewCIDSystemInfo(registryName, orderName, supplementVersion)
		}
		format = FontFormatTTF
	}
	p.fontInfoList = append(p.fontInfoList, newFSFontInfo(file, format, name, ros,
		usWeightClass, sFamilyClass, ulCodePageRange1, ulCodePageRange2,
		*macStyle, panose, p, hash, lastModified(file)))

	slog.Debug("Read font headers", "format", format, "name", name,
		"family", fontHeaders.FontFamily(), "subFamily", fontHeaders.FontSubFamily())
}

// addType1Font adds a Type 1 font to the file cache. To reduce memory, the
// parsed font is not cached.
func (p *fileSystemFontProvider) addType1Font(pfbFile string) {
	t1, err := readType1FontFile(pfbFile)
	if err == nil {
		// Type1Font.Name never fails; the FontBoxFont interface it satisfies is
		// what makes it return an error at all, and Java declares the throws
		// there and not on the class.
		name, _ := t1.Name()
		if name == "" {
			p.fontInfoList = append(p.fontInfoList,
				p.createFSIgnored(pfbFile, FontFormatPFB, "*skipnoname*"))
			slog.Warn("Missing 'name' entry for PostScript name in font", "file", pfbFile)
			return
		}
		if strings.Contains(name, "|") {
			p.fontInfoList = append(p.fontInfoList,
				p.createFSIgnored(pfbFile, FontFormatPFB, "*skippipeinname*"))
			slog.Warn("Skipping font with '|' in name", "name", name, "file", pfbFile)
			return
		}
		hash := checksumPlaceholder
		if !SkipChecksums {
			hash, err = computeHashOfFile(pfbFile)
		}
		if err == nil {
			p.fontInfoList = append(p.fontInfoList, newFSFontInfo(pfbFile, FontFormatPFB,
				name, nil, -1, -1, 0, 0, -1, nil, p, hash, lastModified(pfbFile)))

			slog.Debug("Read PFB", "name", name, "family", t1.FamilyName(),
				"weight", t1.Weight())
			return
		}
	}
	p.fontInfoList = append(p.fontInfoList,
		p.createFSIgnored(pfbFile, FontFormatPFB, "*skipexception*"))
	slog.Warn("Could not load font file", "file", pfbFile, "err", err)
}

func readType1FontFile(pfbFile string) (*type1.Type1Font, error) {
	input, err := os.Open(pfbFile)
	if err != nil {
		return nil, err
	}
	defer input.Close()
	return type1.CreateWithPFB(input)
}

// ToDebugString returns every font the provider found, one per line.
func (p *fileSystemFontProvider) ToDebugString() string {
	var sb strings.Builder
	for _, info := range p.fontInfoList {
		sb.WriteString(info.Format().String())
		sb.WriteString(": ")
		sb.WriteString(info.PostScriptName())
		sb.WriteString(": ")
		sb.WriteString(info.file)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// GetFontInfo returns what the provider knows about each font on the system.
func (p *fileSystemFontProvider) GetFontInfo() []FontInfo {
	infos := make([]FontInfo, len(p.fontInfoList))
	for i, info := range p.fontInfoList {
		infos[i] = info
	}
	return infos
}

// computeHashOfFile is Java's computeHash over a stream it opens and closes.
//
// It doesn't use readAllBytes() because some fonts are huge (PDFBOX-5781).
func computeHashOfFile(file string) (string, error) {
	input, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer input.Close()
	return computeHash(input)
}

func computeHash(is io.Reader) (string, error) {
	crc := crc32.NewIEEE()

	buffer := make([]byte, 4096)
	for {
		readBytes, err := is.Read(buffer)
		if readBytes > 0 {
			crc.Write(buffer[:readBytes])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return strconv.FormatUint(uint64(crc.Sum32()), 16), nil
}
