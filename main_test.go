package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	clix "github.com/gloo-foo/cli"
	"github.com/spf13/afero"
	urf "github.com/urfave/cli/v3"
)

// parse runs args through a bare command carrying a fresh set of the wrapper's
// flags and returns the parsed accessor.
func parse(t *testing.T, args ...string) *urf.Command {
	t.Helper()
	var got *urf.Command
	app := &urf.Command{
		Name:   name,
		Flags:  flags(),
		Action: func(_ context.Context, c *urf.Command) error { got = c; return nil },
	}
	if err := app.Run(context.Background(), args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return got
}

func invocation(t *testing.T, args ...string) clix.Invocation {
	return clix.Invocation{Args: parse(t, args...), Stdin: strings.NewReader(""), Fs: afero.NewMemMapFs()}
}

func TestOptions(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"none", []string{name}, 0},
		{"count", []string{name, "-n", "3"}, 1},
		{"seed", []string{name, "--seed", "7"}, 1},
		{"echo", []string{name, "-e", "a", "b"}, 1},
		{"range", []string{name, "-i", "1-9"}, 1},
		{"all", []string{name, "-e", "a", "-n", "2", "--seed", "5"}, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := options(parse(t, tc.args...))
			if err != nil {
				t.Fatalf("options err=%v", err)
			}
			if len(opts) != tc.want {
				t.Fatalf("options len=%d, want %d", len(opts), tc.want)
			}
		})
	}
}

func TestOptionsBadRange(t *testing.T) {
	for _, args := range [][]string{
		{name, "-i", "bad"},
		{name, "-i", "x-5"},
		{name, "-i", "5-y"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			if _, err := options(parse(t, args...)); !errors.Is(err, ErrBadRange) {
				t.Fatalf("err=%v, want ErrBadRange", err)
			}
		})
	}
}

func TestSource(t *testing.T) {
	for _, args := range [][]string{
		{name, "-e", "a"},
		{name, "-i", "1-9"},
		{name, "file.txt"},
		{name},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			if src := source(invocation(t, args...)); src == nil {
				t.Fatal("source is nil")
			}
		})
	}
}

func TestErrorMessage(t *testing.T) {
	if ErrBadRange.Error() != string(ErrBadRange) {
		t.Fatalf("ErrBadRange message=%q", ErrBadRange.Error())
	}
}

func TestBuild(t *testing.T) {
	src, filter, err := build(invocation(t, name))
	if err != nil || src == nil || filter == nil {
		t.Fatalf("build: src=%v filter=%v err=%v", src, filter, err)
	}
}

func TestBuild_BadRangeIsUsageError(t *testing.T) {
	src, filter, err := build(invocation(t, name, "-i", "bad"))
	if !errors.Is(err, ErrBadRange) {
		t.Fatalf("err=%v, want ErrBadRange", err)
	}
	if src != nil || filter != nil {
		t.Fatalf("src=%v filter=%v, want both nil on error", src, filter)
	}
}

func Test_main(t *testing.T) {
	orig := runMain
	t.Cleanup(func() { runMain = orig })
	var gotName clix.Name
	runMain = func(s clix.Spec, _ clix.Version) { gotName = s.Name }
	main()
	if gotName != name {
		t.Fatalf("main used spec %q, want %s", gotName, name)
	}
}
