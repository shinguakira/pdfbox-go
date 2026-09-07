// Package tools is the command-line utilities.
//
// Port of the Java module `tools`, org.apache.pdfbox.tools. The commands are
// here as a library, and the single executable that dispatches to them is
// go/cmd/pdfbox.
//
// # The two substitutions this module rests on
//
// **The flag parser.** Java uses picocli, which is annotation-driven and has no
// Go equivalent worth transliterating. The port uses the standard library's
// `flag`, and it fits better than a POSIX parser would: picocli here declares
// long options with a *single* dash -- `-alwaysNext`, `-encoding`, `-startPage`
// -- with only a handful of `-i`/`--input` pairs, and Go's `flag` accepts
// exactly that, treating `-name` and `--name` alike and taking both
// `-name value` and `-name=value`. `pflag` or `cobra` would read `-alwaysNext`
// as a cluster of eleven single-letter flags.
//
// What picocli gives and `flag` does not is written out once, here: subcommands
// (in cmd/pdfbox), the standard help and version options that
// `mixinStandardHelpOptions` adds, required options, and picocli's exit codes.
//
// **The shape of a command.** Java's commands are `Callable<Integer>` with
// their options as annotated fields, run through
// `new CommandLine(app).execute(args)`, which returns the process exit code.
// The Java tests call exactly that and read System.out, so the port keeps the
// shape: a struct with its options as fields, a Call that answers the exit
// code, and Execute in place of CommandLine.execute. Every command takes its
// two streams rather than reaching for os.Stdout, which is what makes the exit
// code and the output testable.
package tools

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

// picocli's CommandLine.ExitCode constants, which are what execute() answers.
const (
	// ExitOK is ExitCode.OK: the command ran.
	ExitOK = 0

	// ExitSoftware is ExitCode.SOFTWARE: the command threw.
	ExitSoftware = 1

	// ExitUsage is ExitCode.USAGE: the arguments could not be parsed.
	ExitUsage = 2
)

// Command is one utility, in the shape picocli gives it.
type Command interface {
	// Name is @Command(name = ...).
	Name() string

	// Header is @Command(header = ...), the one line the help starts with.
	Header() string

	// Flags declares this command's options on the given set, which is
	// @Option on each field.
	Flags(set *flag.FlagSet)

	// Call is Callable<Integer>.call(): run, and answer the exit code.
	Call() int

	// setStreams hands the command the two streams it writes to. Java's
	// commands capture System.out and System.err in their constructor, and the
	// Java tests replace System.out before constructing one.
	setStreams(out, errw io.Writer)
}

// standardHelpOptions is implemented by a command whose @Command carries
// mixinStandardHelpOptions = true, which is what adds -h/--help and
// -V/--version. It is not on every command: `version` has neither, and
// `DecompressObjectstreams` declares its own -h/--help with usageHelp = true
// and so has no -V. Adding the pair everywhere would accept arguments the Java
// refuses, which is a difference no test would show.
type standardHelpOptions interface{ standardHelpOptions() }

// mixinStandardHelpOptions is embedded by a command that declares it. The
// method is the marker; there is nothing to hold.
type mixinStandardHelpOptions struct{}

func (mixinStandardHelpOptions) standardHelpOptions() {}

// usageHelpOption is implemented by a command that declares -h/--help with
// picocli's usageHelp = true but no version option.
type usageHelpOption interface{ usageHelpOption() }

// mixinUsageHelpOption is embedded by such a command.
type mixinUsageHelpOption struct{}

func (mixinUsageHelpOption) usageHelpOption() {}

// positional is implemented by a command that takes @Parameters rather than
// options. Only WriteDecodedDoc does.
type positional interface {
	// setPositional takes the arguments left after the options, and reports a
	// usage error where there are too many or too few.
	setPositional(args []string) error
}

// streams is the pair of writers every command holds. Java's field names are
// SYSOUT and SYSERR.
type streams struct {
	sysout io.Writer
	syserr io.Writer
}

func (s *streams) setStreams(out, errw io.Writer) { s.sysout, s.syserr = out, errw }

// out answers the standard output stream, defaulting to io.Discard so that a
// command built by hand and never given streams writes nowhere rather than
// panicking.
func (s *streams) out() io.Writer {
	if s.sysout == nil {
		return io.Discard
	}
	return s.sysout
}

// err answers the standard error stream.
func (s *streams) err() io.Writer {
	if s.syserr == nil {
		return io.Discard
	}
	return s.syserr
}

// printlnOut is SYSOUT.println.
func (s *streams) printlnOut(a ...any) { fmt.Fprintln(s.out(), a...) }

// printlnErr is SYSERR.println.
func (s *streams) printlnErr(a ...any) { fmt.Fprintln(s.err(), a...) }

// errMissingRequired is what a required option that was not given answers.
type errMissingRequired struct{ names string }

func (e *errMissingRequired) Error() string {
	return "Missing required option: '" + e.names + "'"
}

// requireSet reports an error where none of the given flag names was set, which
// is picocli's `required = true`.
func requireSet(set *flag.FlagSet, names string, given ...string) error {
	seen := map[string]bool{}
	set.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for _, name := range given {
		if seen[name] {
			return nil
		}
	}
	return &errMissingRequired{names: names}
}

// helpRequested is the sentinel a `-h` or `--help` raises, and versionRequested
// the one `-V` or `--version` raises. Both are what
// `mixinStandardHelpOptions = true` adds, and both exit 0 after printing.
var (
	helpRequested    = errors.New("tools: help requested")
	versionRequested = errors.New("tools: version requested")
)

// Execute runs the given command over the given arguments and answers the
// process exit code.
//
// Port of `new CommandLine(command).execute(args)`. Its three answers are
// picocli's: the value of call(), USAGE where the arguments do not parse, and
// SOFTWARE where call() threw -- which in Go is where it panicked, since this
// port renders an unchecked exception as a panic.
func Execute(command Command, args []string, out, errw io.Writer) (code int) {
	command.setStreams(out, errw)

	set := flag.NewFlagSet(command.Name(), flag.ContinueOnError)
	set.SetOutput(errw)
	// picocli prints usage itself, with the header first.
	set.Usage = func() { writeUsage(errw, command, set) }

	// mixinStandardHelpOptions = true adds both; usageHelp = true adds only the
	// first; `version` declares neither.
	var help, version *bool
	_, wantsStandard := command.(standardHelpOptions)
	_, wantsUsageHelp := command.(usageHelpOption)
	if wantsStandard || wantsUsageHelp {
		help = set.Bool("h", false, "Show this help message and exit.")
		set.BoolVar(help, "help", false, "Show this help message and exit.")
	}
	if wantsStandard {
		version = set.Bool("V", false, "Print version information and exit.")
		set.BoolVar(version, "version", false, "Print version information and exit.")
	}

	command.Flags(set)

	if err := set.Parse(args); err != nil {
		// flag prints the error and the usage itself.
		return ExitUsage
	}
	if help != nil && *help {
		writeUsage(out, command, set)
		return ExitOK
	}
	if version != nil && *version {
		fmt.Fprintln(out, versionLine(command.Name()))
		return ExitOK
	}
	if p, ok := command.(positional); ok {
		if err := p.setPositional(set.Args()); err != nil {
			fmt.Fprintln(errw, err)
			writeUsage(errw, command, set)
			return ExitUsage
		}
	} else if set.NArg() != 0 {
		// picocli refuses an argument a command declares no @Parameters for.
		fmt.Fprintf(errw, "Unmatched argument at index 0: '%s'\n", set.Arg(0))
		writeUsage(errw, command, set)
		return ExitUsage
	}
	if err := validate(command, set); err != nil {
		fmt.Fprintln(errw, err)
		writeUsage(errw, command, set)
		return ExitUsage
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Fprintln(errw, recovered)
			code = ExitSoftware
		}
	}()
	return command.Call()
}

// validator is implemented by a command that has a required option, which
// picocli checks before it calls call().
type validator interface {
	validate(set *flag.FlagSet) error
}

func validate(command Command, set *flag.FlagSet) error {
	if v, ok := command.(validator); ok {
		return v.validate(set)
	}
	return nil
}

// writeUsage prints the header, the synopsis and the options, which is what
// picocli's usage help does.
func writeUsage(w io.Writer, command Command, set *flag.FlagSet) {
	fmt.Fprintln(w, command.Header())
	fmt.Fprintf(w, "Usage: %s [OPTIONS]\n", command.Name())
	var options strings.Builder
	set.SetOutput(&options)
	set.PrintDefaults()
	set.SetOutput(w)
	fmt.Fprint(w, options.String())
}
