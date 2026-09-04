// Package autodetect finds the fonts a machine has installed.
//
// Port of org.apache.fontbox.util.autodetect.
package autodetect

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// FontDirFinder finds the directories a machine keeps its fonts in.
//
// Port of the org.apache.fontbox.util.autodetect.FontDirFinder interface.
type FontDirFinder interface {
	// Find returns the font directories that exist and can be read.
	Find() []string
}

// nativeFontDirFinder keeps whichever of a fixed list of directories exist.
//
// Port of the abstract org.apache.fontbox.util.autodetect.NativeFontDirFinder;
// Java's three subclasses differ only in that list, so the port carries it.
type nativeFontDirFinder struct {
	searchableDirectories []string
}

var _ FontDirFinder = (*nativeFontDirFinder)(nil)

// Find returns the font directories that exist and can be read.
func (f *nativeFontDirFinder) Find() []string {
	var fontDirList []string
	for _, searchableDirectory := range f.searchableDirectories {
		if isReadableDirectory(searchableDirectory) {
			fontDirList = append(fontDirList, searchableDirectory)
		}
	}
	return fontDirList
}

// NewUnixFontDirFinder returns the finder for a Unix machine.
//
// Port of org.apache.fontbox.util.autodetect.UnixFontDirFinder.
func NewUnixFontDirFinder() FontDirFinder {
	return &nativeFontDirFinder{searchableDirectories: []string{
		userHome() + "/.fonts",     // user
		"/usr/local/fonts",         // local
		"/usr/local/share/fonts",   // local shared
		"/usr/share/fonts",         // system
		"/usr/X11R6/lib/X11/fonts", // X
		"/usr/share/X11/fonts",     // CentOS
	}}
}

// NewMacFontDirFinder returns the finder for a Mac.
//
// Port of org.apache.fontbox.util.autodetect.MacFontDirFinder.
func NewMacFontDirFinder() FontDirFinder {
	return &nativeFontDirFinder{searchableDirectories: []string{
		userHome() + "/Library/Fonts/", // user
		"/Library/Fonts/",              // local
		"/System/Library/Fonts/",       // system
		"/Network/Library/Fonts/",      // network
	}}
}

// NewOS400FontDirFinder returns the finder for an OS/400 machine.
//
// Port of org.apache.fontbox.util.autodetect.OS400FontDirFinder.
func NewOS400FontDirFinder() FontDirFinder {
	return &nativeFontDirFinder{searchableDirectories: []string{
		userHome() + "/.fonts", // user
		"/QIBM/ProdData/OS400/Fonts",
	}}
}

// windowsFontDirFinder finds the font directories of a Windows machine.
//
// Port of org.apache.fontbox.util.autodetect.WindowsFontDirFinder.
type windowsFontDirFinder struct{}

var _ FontDirFinder = (*windowsFontDirFinder)(nil)

// NewWindowsFontDirFinder returns the finder for a Windows machine.
func NewWindowsFontDirFinder() FontDirFinder { return &windowsFontDirFinder{} }

// Find returns the font directories that exist and can be read.
func (f *windowsFontDirFinder) Find() []string {
	var fontDirList []string
	// Java first asks for the system property env.windir, which nothing sets,
	// and then for the environment variable; only the second reaches anything.
	windir := os.Getenv("windir")
	if len(windir) > 2 {
		// remove any trailing '/'
		windir = strings.TrimSuffix(windir, "/")

		osFontsDir := filepath.Join(windir, "FONTS")
		if isReadableDirectory(osFontsDir) {
			fontDirList = append(fontDirList, osFontsDir)
		}
		psFontsDir := filepath.Join(windir[:2], "PSFONTS")
		if isReadableDirectory(psFontsDir) {
			fontDirList = append(fontDirList, psFontsDir)
		}
	} else {
		// Java reads os.name and picks WINNT where it ends in "NT", which no
		// version this port can run on does.
		const windowsDirName = "WINDOWS"
		// look for true type font folder
		for driveLetter := 'C'; driveLetter <= 'E'; driveLetter++ {
			osFontsDir := string(driveLetter) + `:\` + windowsDirName + `\FONTS`
			if isReadableDirectory(osFontsDir) {
				fontDirList = append(fontDirList, osFontsDir)
				break
			}
		}
		// look for type 1 font folder
		for driveLetter := 'C'; driveLetter <= 'E'; driveLetter++ {
			psFontsDir := string(driveLetter) + `:\PSFONTS`
			if isReadableDirectory(psFontsDir) {
				fontDirList = append(fontDirList, psFontsDir)
				break
			}
		}
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		localFontDir := filepath.Join(localAppData, "Microsoft", "Windows", "Fonts")
		if isReadableDirectory(localFontDir) {
			fontDirList = append(fontDirList, localFontDir)
		}
	}
	return fontDirList
}

// isReadableDirectory stands for Java's exists() && canRead(), which throws
// SecurityException where a security manager forbids the look; Go has no such
// manager and a failed stat simply means no.
func isReadableDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	entries, err := os.Open(path)
	if err != nil {
		return false
	}
	entries.Close()
	return true
}

// userHome is Java's user.home system property.
func userHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// FontFileFinder walks a machine's font directories for font files.
//
// Port of org.apache.fontbox.util.autodetect.FontFileFinder.
type FontFileFinder struct {
	fontDirFinder FontDirFinder
}

// NewFontFileFinder returns a finder that works out its directories itself.
func NewFontFileFinder() *FontFileFinder { return &FontFileFinder{} }

// determineDirFinder returns the finder this machine needs.
//
// Java reads the os.name system property; Go's runtime.GOOS says the same
// thing. There is no Go port for OS/400, so that finder is only reachable by
// naming it.
func determineDirFinder() FontDirFinder {
	switch runtime.GOOS {
	case "windows":
		return NewWindowsFontDirFinder()
	case "darwin":
		return NewMacFontDirFinder()
	default:
		return NewUnixFontDirFinder()
	}
}

// Find returns the font files of every font directory of the machine.
//
// Java gives back a list of URIs; the port gives the paths themselves, which is
// what every caller turns them back into.
func (f *FontFileFinder) Find() []string {
	if f.fontDirFinder == nil {
		f.fontDirFinder = determineDirFinder()
	}
	fontDirs := f.fontDirFinder.Find()
	var results []string
	for _, dir := range fontDirs {
		walk(dir, &results)
	}
	return results
}

// FindInDirectory returns the font files under the named directory.
func (f *FontFileFinder) FindInDirectory(dir string) []string {
	var results []string
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		walk(dir, &results)
	}
	return results
}

// walk searches for font files recursively in the given directory.
func walk(directory string, results *[]string) {
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return
	}
	filelist, err := os.ReadDir(directory)
	if err != nil {
		return
	}
	for _, entry := range filelist {
		path := filepath.Join(directory, entry.Name())
		if entry.IsDir() {
			// skip hidden directories
			if isHidden(entry) {
				slog.Debug("skip hidden directory", "directory", path)
				continue
			}
			walk(path, results)
			continue
		}
		if checkFontfile(entry.Name()) {
			slog.Debug("checkFontfile found", "file", path)
			*results = append(*results, path)
		}
	}
}

// isHidden is Java's File.isHidden, which on Unix means a leading dot and on
// Windows means the hidden attribute; the port takes the leading dot, which is
// what a font directory is ever likely to carry.
func isHidden(entry fs.DirEntry) bool { return strings.HasPrefix(entry.Name(), ".") }

func checkFontfile(fileName string) bool {
	name := strings.ToLower(fileName)
	return (strings.HasSuffix(name, ".ttf") || strings.HasSuffix(name, ".otf") ||
		strings.HasSuffix(name, ".pfb") || strings.HasSuffix(name, ".ttc")) &&
		// PDFBOX-3377 exclude weird files in AIX
		!strings.HasPrefix(name, "fonts.")
}
