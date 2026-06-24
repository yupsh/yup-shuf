package main

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	command "github.com/gloo-foo/cmd-shuf"
	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

// Error is the sentinel error type emitted by this package.
type Error string

func (e Error) Error() string { return string(e) }

// ErrBadRange reports an -i value that is not LO-HI with LO and HI integers.
const ErrBadRange Error = "invalid input range: want LO-HI"

const (
	flagCount = "head-count"
	flagRange = "input-range"
	flagEcho  = "echo"
	flagSeed  = "seed"
)

// usageText is the command's multi-line usage synopsis, shown in --help.
// cli/v3 indents the whole block by 3 spaces, so these lines are flush-left to
// stay aligned in the rendered output.
const usageText = `shuf [OPTIONS] [FILE]
shuf -e [OPTIONS] [ARG...]
shuf -i LO-HI [OPTIONS]

Write a random permutation of the input lines to standard output.
With no FILE, or when FILE is -, read standard input.`

// init replaces urfave/cli's default --version/-v flag with a --version-only
// flag, freeing the single-letter -v for command flags (e.g. grep -v) while
// still exposing the injected build version.
func init() {
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print version information and exit"}
}

// run builds and executes the shuf CLI against the injected version, I/O, and
// filesystem, returning the process exit code.
func run(version string, args []string, stdin io.Reader, stdout, stderr io.Writer, fs afero.Fs) int {
	cmd := newApp(version, stdin, stdout, fs)
	cmd.Writer = stdout
	cmd.ErrWriter = stderr
	if err := cmd.Run(context.Background(), args); err != nil {
		_, _ = fmt.Fprintf(stderr, "shuf: %v\n", err)
		return 1
	}
	return 0
}

func newApp(version string, stdin io.Reader, stdout io.Writer, fs afero.Fs) *cli.Command {
	return &cli.Command{
		Name:            "shuf",
		Version:         version,
		Usage:           "generate random permutations",
		UsageText:       usageText,
		HideHelpCommand: true,
		// Keep exit handling in run() rather than letting urfave/cli call
		// os.Exit, so the exit code stays testable.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Flags: []cli.Flag{
			&cli.IntFlag{Name: flagCount, Aliases: []string{"n"}, Usage: "output at most COUNT lines"},
			&cli.StringFlag{Name: flagRange, Aliases: []string{"i"}, Usage: "treat each number LO through HI as an input line"},
			&cli.BoolFlag{Name: flagEcho, Aliases: []string{"e"}, Usage: "treat each ARG as an input line"},
			&cli.Int64Flag{Name: flagSeed, Usage: "seed the shuffle for deterministic output"},
		},
		Action: action(stdin, stdout, fs),
	}
}

func action(stdin io.Reader, stdout io.Writer, fs afero.Fs) cli.ActionFunc {
	return func(_ context.Context, c *cli.Command) error {
		opts, err := options(c)
		if err != nil {
			return err
		}
		_, err = gloo.Run(source(c, stdin, fs), gloo.ByteWriteTo(stdout), command.Shuf(opts...))
		return err
	}
}

// source selects the input stream. -e and -i consume their data from flags and
// ignore stdin, so an empty reader is supplied; otherwise read files or stdin.
func source(c *cli.Command, stdin io.Reader, fs afero.Fs) any {
	if c.Bool(flagEcho) || c.IsSet(flagRange) {
		return gloo.ByteReaderSource([]io.Reader{strings.NewReader("")})
	}
	if c.NArg() == 0 {
		return gloo.ByteReaderSource([]io.Reader{stdin})
	}
	files := make([]gloo.File, c.NArg())
	for i := range files {
		files[i] = gloo.File(c.Args().Get(i))
	}
	return gloo.ByteFileSource(fs, files)
}

func options(c *cli.Command) ([]any, error) {
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

func echoOptions(c *cli.Command) []any {
	if !c.Bool(flagEcho) {
		return nil
	}
	return []any{command.ShufEcho(c.Args().Slice()...)}
}

func rangeOption(c *cli.Command) ([]any, error) {
	if !c.IsSet(flagRange) {
		return nil, nil
	}
	lo, hi, err := parseRange(c.String(flagRange))
	if err != nil {
		return nil, err
	}
	return []any{command.ShufRange(lo, hi)}, nil
}

func parseRange(spec string) (int, int, error) {
	lo, hi, ok := strings.Cut(spec, "-")
	if !ok {
		return 0, 0, ErrBadRange
	}
	return parseBounds(lo, hi)
}

func parseBounds(lo, hi string) (int, int, error) {
	low, err := strconv.Atoi(lo)
	if err != nil {
		return 0, 0, ErrBadRange
	}
	high, err := strconv.Atoi(hi)
	if err != nil {
		return 0, 0, ErrBadRange
	}
	return low, high, nil
}
