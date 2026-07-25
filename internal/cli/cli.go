// Package cli implements the image2shr command-line interface: subcommand
// dispatch and flag parsing on top of the standard library flag package.
//
// Hard rules (see docs/SPEC.md):
//   - stdout carries only the payload — binary output or the --json object.
//     Every log, warning, and progress message goes to stderr.
//   - Exit codes: 0 success, 1 runtime failure, 2 usage error.
//   - Same input + same flags + same version = byte-identical output.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

// Version is injected at build time via -ldflags (see Makefile).
var Version = "dev"

// Exit codes.
const (
	exitOK      = 0
	exitRuntime = 1
	exitUsage   = 2
)

// usageError distinguishes bad invocations (exit 2) from runtime failures
// (exit 1).
type usageError struct{ err error }

func (u usageError) Error() string { return u.err.Error() }

func usagef(format string, args ...any) error {
	return usageError{fmt.Errorf(format, args...)}
}

// env bundles the process's streams so every subcommand is testable without
// touching os.Stdout/os.Stderr.
type env struct {
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader
}

func (e *env) logf(format string, args ...any) {
	fmt.Fprintf(e.stderr, format+"\n", args...)
}

// Main runs the CLI and returns the process exit code.
func Main(args []string, stdout, stderr io.Writer, stdin io.Reader) int {
	e := &env{stdout: stdout, stderr: stderr, stdin: stdin}
	if len(args) == 0 {
		printUsage(stderr)
		return exitUsage
	}

	var err error
	switch cmd := args[0]; cmd {
	case "convert":
		err = cmdConvert(e, args[1:])
	case "preview":
		err = cmdPreview(e, args[1:])
	case "inspect":
		err = cmdInspect(e, args[1:])
	case "targets":
		err = cmdTargets(e, args[1:])
	case "version":
		err = cmdVersion(e, args[1:])
	case "help", "-h", "--help", "-help":
		printUsage(stdout)
	default:
		fmt.Fprintf(stderr, "image2shr: unknown command %q\n\n", cmd)
		printUsage(stderr)
		return exitUsage
	}

	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, flag.ErrHelp):
		return exitOK // -h/--help on a subcommand: usage already printed
	default:
		var ue usageError
		if errors.As(err, &ue) {
			fmt.Fprintf(stderr, "image2shr: %v\n", err)
			return exitUsage
		}
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(stderr, "image2shr: %v\n", err)
		}
		return exitRuntime
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `image2shr — convert modern images to Apple IIgs Super Hi-Res

Usage:
  image2shr <command> [flags] [args]

Commands:
  convert   convert an image to an SHR file
  preview   render an SHR file to PNG exactly as the IIgs would display it
  inspect   report an SHR file's modes, palettes, and SCB usage
  targets   list conversion targets
  version   print the version

Run "image2shr <command> --help" for flags and examples.
`)
}

// newFlagSet builds a subcommand FlagSet that reports parse errors as usage
// errors (exit 2) and prints its usage to the right stream.
func newFlagSet(e *env, name, usage string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	fs.Usage = func() {
		fmt.Fprint(e.stderr, usage)
		fmt.Fprintf(e.stderr, "\nFlags:\n")
		fs.PrintDefaults()
	}
	return fs
}

// parse wraps FlagSet.Parse, allowing flags and positional arguments to be
// interleaved ("image2shr preview title.shr -o out.png"): the stdlib parser
// stops at the first positional, so we collect it and keep parsing.
// Everything after a literal "--" is positional. Returns the positionals.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos, trailing []string
	for i, a := range args {
		if a == "--" {
			trailing = args[i+1:]
			args = args[:i]
			break
		}
	}
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, flag.ErrHelp
			}
			return nil, usageError{err}
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return append(pos, trailing...), nil
		}
		// rest[0] is a positional; keep parsing what follows as flags.
		pos = append(pos, rest[0])
		args = rest[1:]
	}
}
