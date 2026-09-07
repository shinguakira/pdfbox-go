package tools

// Simple wrapper around all the command line utilities included in PDFBox.
// Used as the main class in the runnable standalone PDFBox jar.
//
// Port of org.apache.pdfbox.tools.PDFBox.

import (
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
		{"export:text", func() Command { return NewExtractText() }},
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
			p.writeUsage(p.out)
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
func (p *PDFBox) split(args []string) ([]invocation, error) {
	byName := map[string]func() Command{}
	for _, sub := range p.subcommands {
		byName[strings.ToLower(sub.Name)] = sub.New
	}

	var groups []invocation
	for _, arg := range args {
		lower := strings.ToLower(arg)
		if build, isSubcommand := byName[lower]; isSubcommand {
			groups = append(groups, invocation{name: lower, build: build})
			continue
		}
		if lower == "help" {
			groups = append(groups, invocation{name: "help"})
			continue
		}
		if len(groups) == 0 {
			return nil, fmt.Errorf("Unmatched argument at index 0: '%s'", arg)
		}
		last := len(groups) - 1
		groups[last].args = append(groups[last].args, arg)
	}
	return groups, nil
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
}
