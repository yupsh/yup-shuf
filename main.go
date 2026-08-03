// Command yup-shuf is the CLI wrapper around github.com/gloo-foo/cmd-shuf.
package main

import (
	"strconv"
	"strings"

	clix "github.com/gloo-foo/cli"
	command "github.com/gloo-foo/cmd-shuf"
	urf "github.com/urfave/cli/v3"
)

// version is the build version. It defaults to "dev" for local builds and is
// overridden at release time via the linker: -ldflags "-X main.version=<v>".
var version = "dev"

// Error is the sentinel error type emitted by this package.
type Error string

func (e Error) Error() string { return string(e) }

// ErrBadRange reports an -i value that is not LO-HI with LO and HI integers.
const ErrBadRange Error = "invalid input range: want LO-HI"

const (
	name      = "shuf"
	flagCount = "head-count"
	flagRange = "input-range"
	flagEcho  = "echo"
	flagSeed  = "seed"
)

// synopsis is the multi-line --help usage block; urfave/cli indents it three
// spaces, so the lines stay flush-left.
const synopsis = `shuf [OPTIONS] [FILE]
shuf -e [OPTIONS] [ARG...]
shuf -i LO-HI [OPTIONS]

Write a random permutation of the input lines to standard output.
With no FILE, or when FILE is -, read standard input.`

// spec declares the shuf wrapper: a filter whose source depends on its flags
// (-e echoes ARGs, -i enumerates a range, otherwise files or stdin).
var spec = clix.Spec{
	Name:     name,
	Summary:  "generate random permutations",
	Synopsis: synopsis,
	Build:    build,
	Flags:    flags(),
}

// flags builds a fresh set of the wrapper's flags. It is a constructor rather
// than a shared slice so each parse gets its own flag instances (urfave/cli
// retains a flag's set-state on its pointer across runs, which would otherwise
// leak between test invocations).
func flags() []urf.Flag {
	return []urf.Flag{
		&urf.IntFlag{
			Name:    flagCount,
			Aliases: []string{"n"},
			Usage:   "output at most COUNT lines",
			Sources: urf.EnvVars("YUP_SHUF_HEAD_COUNT"),
			Value:   0,
		},
		&urf.StringFlag{
			Name:    flagRange,
			Aliases: []string{"i"},
			Usage:   "treat each number LO through HI as an input line",
			Sources: urf.EnvVars("YUP_SHUF_INPUT_RANGE"),
			Value:   "",
		},
		&urf.BoolFlag{
			Name:    flagEcho,
			Aliases: []string{"e"},
			Usage:   "treat each ARG as an input line",
			Sources: urf.EnvVars("YUP_SHUF_ECHO"),
		},
		&urf.Int64Flag{
			Name:    flagSeed,
			Usage:   "seed the shuffle for deterministic output",
			Sources: urf.EnvVars("YUP_SHUF_SEED"),
			Value:   0,
		},
	}
}

// build maps the invocation to shuf's pipeline: a flag-dependent source into the
// shuf command configured by the flags. An invalid -i range is a usage error.
func build(inv clix.Invocation) (clix.Source, clix.Command, error) {
	opts, err := options(inv.Args)
	if err != nil {
		return nil, nil, err
	}
	return source(inv), command.Shuf(opts...), nil
}

// source selects the input stream. -e and -i consume their data from flags and
// ignore stdin, so an empty stream is supplied; otherwise read files or stdin.
func source(inv clix.Invocation) clix.Source {
	if inv.Args.Bool(flagEcho) || inv.Args.IsSet(flagRange) {
		return clix.Stdin(strings.NewReader(""))
	}
	return clix.OperandsOrStdin(inv)
}

func options(c *urf.Command) ([]any, error) {
	rangeOpt, err := rangeOption(c)
	if err != nil {
		return nil, err
	}
	opts := append([]any(nil), rangeOpt...)
	opts = append(opts, echoOptions(c)...)
	if c.IsSet(flagCount) {
		opts = append(opts, command.ShufCount(c.Int(flagCount)))
	}
	if c.IsSet(flagSeed) {
		opts = append(opts, command.ShufSeed(c.Int64(flagSeed)))
	}
	return opts, nil
}

func echoOptions(c *urf.Command) []any {
	if !c.Bool(flagEcho) {
		return nil
	}
	raw := c.Args().Slice()
	lines := make([]command.ShufLine, len(raw))
	for i, arg := range raw {
		lines[i] = command.ShufLine(arg)
	}
	return []any{command.ShufEcho(lines...)}
}

func rangeOption(c *urf.Command) ([]any, error) {
	if !c.IsSet(flagRange) {
		return nil, nil
	}
	lo, hi, err := parseRange(rangeSpec(c.String(flagRange)))
	if err != nil {
		return nil, err
	}
	return []any{command.ShufRange(command.ShufBound(lo), command.ShufBound(hi))}, nil
}

// rangeSpec is the textual -i/--input-range argument, "LO-HI".
type rangeSpec string

func parseRange(spec rangeSpec) (int, int, error) {
	lo, hi, ok := strings.Cut(string(spec), "-")
	if !ok {
		return 0, 0, ErrBadRange
	}
	return parseBounds(rangeBound(lo), rangeBound(hi))
}

// rangeBound is the textual lower or upper bound of the LO-HI input range.
type rangeBound string

func parseBounds(lo, hi rangeBound) (int, int, error) {
	low, err := strconv.Atoi(string(lo))
	if err != nil {
		return 0, 0, ErrBadRange
	}
	high, err := strconv.Atoi(string(hi))
	if err != nil {
		return 0, 0, ErrBadRange
	}
	return low, high, nil
}

// runMain is an indirection seam so main's wiring is testable without spawning
// the process; a test swaps it and restores it.
var runMain = clix.Main

func main() { runMain(spec, version) }
