package tools

// A simple command line utility to get the version of PDFBox.
//
// Port of org.apache.pdfbox.tools.Version, which is both the `version` command
// and picocli's IVersionProvider for every other command's `-V`.

import (
	"flag"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// Version is the `version` command.
//
// Port of the package-private final class Version.
type Version struct {
	streams
}

var _ Command = (*Version)(nil)

// NewVersion returns the command.
func NewVersion() *Version { return &Version{} }

// Name is @Command(name = "version").
func (v *Version) Name() string { return "version" }

// Header is @Command(header = ...).
func (v *Version) Header() string { return "Gets the version of PDFBox" }

// Flags declares no options: the Java class has none of its own, and is not
// mixinStandardHelpOptions either -- but Execute adds those to every command,
// which is a small widening said here rather than hidden.
func (v *Version) Flags(*flag.FlagSet) {}

// Call prints the version.
//
// Port of call(), which prints getVersion()[0] and answers 0.
func (v *Version) Call() int {
	v.printlnOut(versionLine(v.Name()))
	return ExitOK
}

// versionLine is getVersion(), the IVersionProvider method: the qualified name
// of the command and the version in brackets, or "unknown" where PDFBox cannot
// say what version it is.
//
// Java's qualifiedName() is the command's name prefixed by its parents', so a
// subcommand of the dispatcher reads "pdfbox version" and the standalone
// command reads "version". The caller passes whichever it is.
func versionLine(qualifiedName string) string {
	version, known := util.Version()
	if !known {
		return "unknown"
	}
	return qualifiedName + " [" + version + "]"
}
