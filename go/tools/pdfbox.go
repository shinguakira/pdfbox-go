package tools

// Simple wrapper around all the command line utilities included in PDFBox.
// Used as the main class in the runnable standalone PDFBox jar.
//
// Port of org.apache.pdfbox.tools.PDFBox.

import (
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Subcommand is one entry of the dispatcher's table: the name it answers to and
// how to build a fresh instance of it.
//
// Java registers a Class and picocli instantiates it; repeatable subcommands
// mean one invocation can need two of the same command, each with its own
// options, so the port registers a constructor.
type Subcommand struct {
	Name string
	New  func() Command
}

// Subcommands is the table PDFBox.main builds, in the order it adds them.
//
// Java also adds "debug" for PDFDebugger when the JVM is not headless. The
// debugger is a module of its own and is not in this port's scope, so the name
// is not registered; see migration/STATUS.md.
func Subcommands() []Subcommand {
	return []Subcommand{
		{"decrypt", func() Command { return NewDecrypt() }},
		{"encrypt", func() Command { return NewEncrypt() }},
		{"decode", func() Command { return NewWriteDecodedDoc() }},
		{"fromimage", func() Command { return NewImageToPDF() }},
		{"export:images", func() Command { return NewExtractImages() }},
		{"export:xmp", func() Command { return NewExtractXMP() }},
		{"export:text", func() Command { return NewExtractText() }},
		{"export:fdf", func() Command { return NewExportFDF() }},
		{"export:xfdf", func() Command { return NewExportXFDF() }},
		{"import:fdf", func() Command { return NewImportFDF() }},
		{"import:xfdf", func() Command { return NewImportXFDF() }},
		{"overlay", func() Command { return NewOverlayPDF() }},
		{"merge", func() Command { return NewPDFMerger() }},
		{"split", func() Command { return NewPDFSplit() }},
		{"fromtext", func() Command { return NewTextToPDF() }},
		{"version", func() Command { return NewVersion() }},
	}
}

// PDFBox is the dispatcher.
type PDFBox struct {
	subcommands []Subcommand
	out, errw   io.Writer
}

// NewPDFBox returns the dispatcher over the commands that were built.
func NewPDFBox(out, errw io.Writer) *PDFBox {
	return &PDFBox{subcommands: Subcommands(), out: out, errw: errw}
}

// Run parses the arguments and runs each subcommand in them, answering the exit
// code of the last one that ran.
//
// Port of PDFBox.main. Two things of picocli's are reproduced here:
// setSubcommandsCaseInsensitive(true), and subcommandsRepeatable = true, which
// lets one invocation carry the same subcommand twice with different options --
// `export:text -i a -console export:text -i b -console` is two runs.
func (p *PDFBox) Run(args []string) int {
	if len(args) == 0 {
		// Java's run() throws ParameterException, which picocli turns into a
		// usage error.
		fmt.Fprintln(p.errw, "Error: Subcommand required")
		p.writeUsage(p.errw)
		return ExitUsage
	}

	groups, err := p.split(args)
	if err != nil {
		fmt.Fprintln(p.errw, err)
		p.writeUsage(p.errw)
		return ExitUsage
	}

	code := ExitOK
	for _, group := range groups {
		if group.name == "help" {
			p.writeHelp(group.args)
			continue
		}
		code = Execute(group.build(), group.args, p.out, p.errw)
	}
	return code
}

// invocation is one subcommand and the arguments that belong to it.
type invocation struct {
	name  string
	build func() Command
	args  []string
}

// split cuts the argument list at each subcommand name.
//
// A name is only a boundary where an option is not waiting for its value.
// picocli knows each option's arity, so `decrypt -i version` gives -i the file
// called "version" rather than starting the `version` command; the port asks
// the command's own flag set which of its options take a value.
//
// `help` is the other special case: picocli's HelpCommand takes the subcommand
// to describe as a parameter, so the name after it belongs to it.
func (p *PDFBox) split(args []string) ([]invocation, error) {
	byName := map[string]func() Command{}
	for _, sub := range p.subcommands {
		byName[strings.ToLower(sub.Name)] = sub.New
	}

	var groups []invocation
	// wantsValue is set while the previous argument was an option that takes
	// one, so that this argument is its value whatever it spells.
	wantsValue := false
	// helpWantsName is set while a `help` has just been opened, because its one
	// positional parameter is the subcommand to describe -- and that parameter
	// is a subcommand name, so it would otherwise be read as a boundary.
	helpWantsName := false
	// takesValue answers whether an option of the command being filled needs a
	// separate argument. It is nil for `help`, which has no options.
	var takesValue func(string) bool

	for _, arg := range args {
		lower := strings.ToLower(arg)

		if !wantsValue && !helpWantsName {
			if build, isSubcommand := byName[lower]; isSubcommand {
				groups = append(groups, invocation{name: lower, build: build})
				takesValue = valueTakingOptions(build())
				continue
			}
			if lower == "help" {
				groups = append(groups, invocation{name: "help"})
				takesValue = nil
				helpWantsName = true
				continue
			}
		}
		helpWantsName = false

		if len(groups) == 0 {
			return nil, fmt.Errorf("Unmatched argument at index 0: '%s'", arg)
		}
		last := len(groups) - 1
		groups[last].args = append(groups[last].args, arg)

		wantsValue = takesValue != nil && optionNeedsValue(arg, takesValue)
	}
	return groups, nil
}

// optionNeedsValue reports whether the argument is an option that will take the
// next one as its value: it has to look like an option, carry no "=", and name
// something other than a boolean.
func optionNeedsValue(arg string, takesValue func(string) bool) bool {
	if len(arg) < 2 || arg[0] != '-' {
		return false
	}
	name := strings.TrimPrefix(strings.TrimPrefix(arg, "-"), "-")
	if name == "" || strings.Contains(name, "=") {
		return false
	}
	return takesValue(name)
}

// valueTakingOptions answers which of a command's options need a separate
// argument, which is every one whose flag is not a boolean.
func valueTakingOptions(command Command) func(string) bool {
	set := flag.NewFlagSet(command.Name(), flag.ContinueOnError)
	set.SetOutput(io.Discard)
	// The two standard mixins, which Execute adds and which are boolean.
	var standard bool
	set.BoolVar(&standard, "h", false, "")
	set.BoolVar(&standard, "help", false, "")
	set.BoolVar(&standard, "V", false, "")
	set.BoolVar(&standard, "version", false, "")
	command.Flags(set)

	needsValue := map[string]bool{}
	set.VisitAll(func(f *flag.Flag) {
		boolFlag, isBool := f.Value.(interface{ IsBoolFlag() bool })
		needsValue[f.Name] = !(isBool && boolFlag.IsBoolFlag())
	})
	return func(name string) bool { return needsValue[name] }
}

// writeHelp is picocli's HelpCommand: with no argument it prints the global
// help, and with a subcommand name it prints that command's.
func (p *PDFBox) writeHelp(args []string) {
	if len(args) == 0 {
		p.writeUsage(p.out)
		return
	}
	wanted := strings.ToLower(args[0])
	for _, sub := range p.subcommands {
		if strings.ToLower(sub.Name) != wanted {
			continue
		}
		command := sub.New()
		// -h is how a command prints its own help, and Execute knows how.
		Execute(command, []string{"-h"}, p.out, p.errw)
		return
	}
	fmt.Fprintf(p.errw, "Unknown subcommand '%s'\n", args[0])
	p.writeUsage(p.errw)
}

// writeUsage prints the synopsis and the subcommands, which is what picocli's
// customSynopsis and footer give.
func (p *PDFBox) writeUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: pdfbox [COMMAND] [OPTIONS]")
	names := make([]string, 0, len(p.subcommands)+1)
	for _, sub := range p.subcommands {
		names = append(names, sub.Name)
	}
	names = append(names, "help")
	sort.Strings(names)
	fmt.Fprintln(w, "Commands: "+strings.Join(names, ", "))
	fmt.Fprintln(w, "See 'pdfbox help <command>' to read about a specific subcommand")

	// B10: the commands Java has and this port does not, so that a caller who
	// asks for one is told what it waits for rather than that it does not exist.
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Not built in this port:")
	for _, missing := range NotBuiltCommands {
		if missing.Name == "" {
			continue
		}
		fmt.Fprintf(w, "  %-14s %s (waiting for %s)\n",
			missing.Name, missing.Java, missing.Waiting)
	}
}
