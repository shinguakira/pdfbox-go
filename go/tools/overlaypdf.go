package tools

// Adds an overlay to an existing PDF document.
//
// Port of org.apache.pdfbox.tools.OverlayPDF. "Based on code contributed by
// Balazs Jerk."

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
)

// OverlayPDF adds an overlay to a PDF document.
type OverlayPDF struct {
	streams
	mixinStandardHelpOptions

	oddPageOverlay          string
	evenPageOverlay         string
	firstPageOverlay        string
	lastPageOverlay         string
	useAllPages             string
	adjustRotation          bool
	specificPageOverlayFile pageOverlays
	defaultOverlay          string
	position                string
	infile                  string
	outfile                 string
}

var _ Command = (*OverlayPDF)(nil)

// NewOverlayPDF returns the command with its one non-zero default.
func NewOverlayPDF() *OverlayPDF {
	return &OverlayPDF{
		position:                "BACKGROUND",
		specificPageOverlayFile: pageOverlays{},
	}
}

// Name is @Command(name = "overlaypdf").
func (o *OverlayPDF) Name() string { return "overlaypdf" }

// Header is @Command(header = ...).
func (o *OverlayPDF) Header() string { return "Adds an overlay to a PDF document" }

// Flags declares the eleven options.
func (o *OverlayPDF) Flags(set *flag.FlagSet) {
	set.StringVar(&o.oddPageOverlay, "odd", "", "overlay file used for odd pages")
	set.StringVar(&o.evenPageOverlay, "even", "", "overlay file used for even pages")
	set.StringVar(&o.firstPageOverlay, "first", "", "overlay file used for the first page")
	set.StringVar(&o.lastPageOverlay, "last", "", "overlay file used for the last page")
	set.StringVar(&o.useAllPages, "useAllPages", "",
		"overlay file used for overlay, all pages are used by simply repeating them")
	set.BoolVar(&o.adjustRotation, "adjustRotation", false,
		"adjust rotation for rotated source pages (applies only if default overlay file is used)")
	set.Var(&o.specificPageOverlayFile, "page",
		"overlay file used for the given page number, may occur more than once")
	set.StringVar(&o.defaultOverlay, "default", "", "the default overlay file")
	set.StringVar(&o.position, "position", "BACKGROUND",
		"where to put the overlay file: FOREGROUND or BACKGROUND (default: BACKGROUND)")
	set.StringVar(&o.infile, "i", "", "the PDF input file")
	set.StringVar(&o.infile, "input", "", "the PDF input file")
	set.StringVar(&o.outfile, "o", "", "the PDF output file")
	set.StringVar(&o.outfile, "output", "", "the PDF output file")
}

// validate is `required = true` on the input and the output, and picocli's
// conversion of the -position value to the enum.
func (o *OverlayPDF) validate(set *flag.FlagSet) error {
	if err := requireSet(set, "--input=<infile>", "i", "input"); err != nil {
		return err
	}
	if err := requireSet(set, "--output=<outfile>", "o", "output"); err != nil {
		return err
	}
	if _, err := overlayPosition(o.position); err != nil {
		return err
	}
	return nil
}

// Call overlays the document and writes out the result.
func (o *OverlayPDF) Call() int {
	retcode := ExitOK

	overlayer := multipdf.NewOverlay()
	position, err := overlayPosition(o.position)
	if err != nil {
		o.reportOverlayError(err)
		return 4
	}
	overlayer.SetOverlayPosition(position)

	for _, each := range []struct {
		file string
		set  func(string)
	}{
		{o.firstPageOverlay, overlayer.SetFirstPageOverlayFile},
		{o.lastPageOverlay, overlayer.SetLastPageOverlayFile},
		{o.oddPageOverlay, overlayer.SetOddPageOverlayFile},
		{o.evenPageOverlay, overlayer.SetEvenPageOverlayFile},
		{o.useAllPages, overlayer.SetAllPagesOverlayFile},
		{o.defaultOverlay, overlayer.SetDefaultOverlayFile},
		{o.infile, overlayer.SetInputFile},
	} {
		if each.file != "" {
			each.set(absolutePath(each.file))
		}
	}
	overlayer.SetAdjustRotation(o.adjustRotation)

	if err := o.overlay(overlayer); err != nil {
		o.reportOverlayError(err)
		retcode = 4
	}
	// close the input files AFTER saving the resulting file as some
	// streams are shared among the input and the output files
	if err := overlayer.Close(); err != nil {
		o.reportOverlayError(err)
		retcode = 4
	}
	return retcode
}

// overlay is the body of Java's try-with-resources, whose resource is the
// result document: it is closed however the save goes.
func (o *OverlayPDF) overlay(overlayer *multipdf.Overlay) error {
	result, err := overlayer.Overlay(o.specificPageOverlayFile)
	if err != nil {
		return err
	}
	defer result.Close()
	return result.SaveToFile(o.outfile)
}

// reportOverlayError is Java's two identical catch blocks.
func (o *OverlayPDF) reportOverlayError(err error) {
	o.printlnErr(fmt.Sprintf("Error adding overlay(s) to PDF [%T]: %v", err, err))
}

// overlayPosition is picocli's conversion of the -position value to the enum,
// which fails the command where the name is not one of the two.
func overlayPosition(name string) (multipdf.Position, error) {
	switch strings.ToUpper(name) {
	case "BACKGROUND":
		return multipdf.Background, nil
	case "FOREGROUND":
		return multipdf.Foreground, nil
	}
	return multipdf.Background, fmt.Errorf(
		"Invalid value for option '-position': expected one of [FOREGROUND, BACKGROUND] "+
			"but was '%s'", name)
}

// pageOverlays is the `Map<Integer, String>` of -page, which picocli fills from
// repeated `-page=<number>=<file>` arguments.
type pageOverlays map[int]string

func (p *pageOverlays) String() string {
	var pairs []string
	for page, file := range *p {
		pairs = append(pairs, strconv.Itoa(page)+"="+file)
	}
	return strings.Join(pairs, ",")
}

func (p *pageOverlays) Set(value string) error {
	page, file, found := strings.Cut(value, "=")
	if !found {
		return fmt.Errorf("expected <pageNumber>=<file> but was '%s'", value)
	}
	number, err := strconv.Atoi(page)
	if err != nil {
		return fmt.Errorf("expected <pageNumber>=<file> but was '%s'", value)
	}
	if *p == nil {
		*p = pageOverlays{}
	}
	(*p)[number] = file
	return nil
}
