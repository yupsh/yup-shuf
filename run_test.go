package main

import (
	"bytes"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestRun(t *testing.T) {
	cases := []struct {
		files      map[string]string
		name       string
		version    string
		stdin      string
		wantOut    string
		wantErrSub string
		args       []string
		wantLines  []string
		wantCount  int
		wantCode   int
	}{
		{
			name:      "stdin permutation seeded",
			args:      []string{"shuf", "--seed", "42"},
			stdin:     "alpha\nbeta\ngamma\n",
			wantLines: []string{"alpha", "beta", "gamma"},
			wantCount: 3,
		},
		{
			name:      "count limit",
			args:      []string{"shuf", "-n", "2", "--seed", "7"},
			stdin:     "a\nb\nc\nd\ne\n",
			wantCount: 2,
		},
		{
			name:      "input range",
			args:      []string{"shuf", "-i", "1-4", "--seed", "3"},
			wantLines: []string{"1", "2", "3", "4"},
			wantCount: 4,
		},
		{
			name:      "echo args",
			args:      []string{"shuf", "-e", "--seed", "9", "red", "green", "blue"},
			wantLines: []string{"blue", "green", "red"},
			wantCount: 3,
		},
		{
			name:      "file source",
			args:      []string{"shuf", "--seed", "5", "/in.txt"},
			files:     map[string]string{"/in.txt": "one\ntwo\nthree\n"},
			wantLines: []string{"one", "three", "two"},
			wantCount: 3,
		},
		{
			name:    "version flag reports injected version",
			version: "1.2.3",
			args:    []string{"shuf", "--version"},
			wantOut: "shuf version 1.2.3\n",
		},
		{
			name:       "bad range errors",
			args:       []string{"shuf", "-i", "1to4"},
			wantCode:   1,
			wantErrSub: "shuf:",
		},
		{
			name:       "bad range bound errors",
			args:       []string{"shuf", "-i", "x-4"},
			wantCode:   1,
			wantErrSub: "shuf:",
		},
		{
			name:       "bad range high bound errors",
			args:       []string{"shuf", "-i", "1-y"},
			wantCode:   1,
			wantErrSub: "shuf:",
		},
		{
			name:       "unknown flag errors",
			args:       []string{"shuf", "--nope"},
			wantCode:   1,
			wantErrSub: "shuf:",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			for path, content := range tc.files {
				if err := afero.WriteFile(fs, path, []byte(content), 0o644); err != nil {
					t.Fatalf("write fixture %s: %v", path, err)
				}
			}

			var out, errOut bytes.Buffer
			code := run(tc.version, tc.args, strings.NewReader(tc.stdin), &out, &errOut, fs)

			if code != tc.wantCode {
				t.Fatalf("exit code = %d, want %d (stderr=%q)", code, tc.wantCode, errOut.String())
			}
			if tc.wantErrSub != "" {
				if !strings.Contains(errOut.String(), tc.wantErrSub) {
					t.Fatalf("stderr = %q, want substring %q", errOut.String(), tc.wantErrSub)
				}
				return
			}
			if tc.wantOut != "" {
				if out.String() != tc.wantOut {
					t.Fatalf("stdout = %q, want %q", out.String(), tc.wantOut)
				}
				return
			}

			got := outputLines(out.String())
			if len(got) != tc.wantCount {
				t.Fatalf("got %d lines, want %d (out=%q)", len(got), tc.wantCount, out.String())
			}
			if tc.wantLines != nil {
				assertPermutation(t, got, tc.wantLines)
			}
		})
	}
}

func outputLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

// assertPermutation verifies got is a permutation of want (order-independent),
// since shuf output order varies even when seeded across architectures.
func assertPermutation(t *testing.T, got, want []string) {
	t.Helper()
	gotSorted := append([]string(nil), got...)
	wantSorted := append([]string(nil), want...)
	sort.Strings(gotSorted)
	sort.Strings(wantSorted)
	if strings.Join(gotSorted, ",") != strings.Join(wantSorted, ",") {
		t.Fatalf("output %v is not a permutation of %v", got, want)
	}
}

func Test_main(t *testing.T) {
	origExit, origRun := osExit, runCLI
	t.Cleanup(func() { osExit, runCLI = origExit, origRun })

	gotCode := -1
	osExit = func(code int) { gotCode = code }
	runCLI = func(string, []string, io.Reader, io.Writer, io.Writer, afero.Fs) int { return 7 }

	main()

	if gotCode != 7 {
		t.Fatalf("main propagated exit code %d, want 7", gotCode)
	}
}
