#!/bin/sh
# Integration checks for yup-shuf, run inside a Debian (GNU coreutils) container.
#
# shuf writes a RANDOM permutation of its input, so byte-for-byte parity with
# GNU `shuf` is impossible. Instead we assert the INVARIANTS that define shuf:
#   permutation — sorting the output equals sorting the input (no line is lost,
#                 duplicated, or invented).
#   count       — `-n N` yields exactly N lines (a subset of the input).
#   range       — `-i LO-HI | sort -n` equals `seq LO HI`.
#   echo        — `-e a b c | sort` equals the sorted arguments.
#
# perm  WANT  CMD...  — run yup-shuf CMD, sort its output, compare to WANT.
# count N     CMD...  — run yup-shuf CMD, assert it emitted exactly N lines.
set -eu

fails=0

# perm asserts the sorted output of `yup-shuf <args>` equals WANT (already
# sorted), proving the result is a permutation of the expected input set.
perm() {
	want=$1
	shift
	got=$(yup-shuf "$@" 2>/dev/null | sort || true)
	if [ "$got" = "$want" ]; then
		printf 'ok    perm   shuf %s\n' "$*"
	else
		printf 'FAIL  perm   shuf %s\n        want: %s\n        got:  %s\n' "$*" "$want" "$got"
		fails=$((fails + 1))
	fi
}

# permn is perm for numeric input: it sorts numerically (sort -n) so a range
# compares against `seq LO HI` rather than lexical order.
permn() {
	want=$1
	shift
	got=$(yup-shuf "$@" 2>/dev/null | sort -n || true)
	if [ "$got" = "$want" ]; then
		printf 'ok    perm   shuf %s\n' "$*"
	else
		printf 'FAIL  perm   shuf %s\n        want: %s\n        got:  %s\n' "$*" "$want" "$got"
		fails=$((fails + 1))
	fi
}

# count asserts `yup-shuf <args>` emits exactly N lines.
count() {
	want=$1
	shift
	got=$(yup-shuf "$@" 2>/dev/null | wc -l | tr -d ' ' || true)
	if [ "$got" = "$want" ]; then
		printf 'ok    count  shuf %s -> %s lines\n' "$*" "$want"
	else
		printf 'FAIL  count  shuf %s\n        want: %s lines\n        got:  %s lines\n' "$*" "$want" "$got"
		fails=$((fails + 1))
	fi
}

# stdin permutation: shuffling a fixed set, then sorting, returns the set.
input=$(printf 'alpha\nbeta\ngamma\ndelta\n')
sorted_input=$(printf '%s\n' "$input" | sort)
got=$(printf '%s\n' "$input" | yup-shuf 2>/dev/null | sort || true)
if [ "$got" = "$sorted_input" ]; then
	printf 'ok    perm   shuf < stdin\n'
else
	printf 'FAIL  perm   shuf < stdin\n        want: %s\n        got:  %s\n' "$sorted_input" "$got"
	fails=$((fails + 1))
fi

# stdin -n N: output is exactly N lines drawn from the input.
n=$(printf '%s\n' "$input" | yup-shuf -n 2 2>/dev/null | wc -l | tr -d ' ' || true)
if [ "$n" = "2" ]; then
	printf 'ok    count  shuf -n 2 < stdin -> 2 lines\n'
else
	printf 'FAIL  count  shuf -n 2 < stdin\n        want: 2 lines\n        got:  %s lines\n' "$n"
	fails=$((fails + 1))
fi

# --input-range (-i): `-i LO-HI | sort -n` equals `seq LO HI`.
permn "$(seq 1 9)" -i 1-9
permn "$(seq 5 12)" --input-range 5-12

# -i with -n: still a subset of the range, of the requested size.
count 3 -i 1-100 -n 3

# --echo (-e): shuffle the operands; sorted output equals the sorted operands.
perm "$(printf 'blue\ngreen\nred\n')" -e red green blue
perm "$(printf 'a\nb\nc\nd\n')" --echo d c b a

# -e with -n: N of the operands.
count 2 -e -n 2 one two three four

# Single-element permutation: identity.
perm "solo" -e solo
permn "$(seq 7 7)" -i 7-7

if [ "$fails" -ne 0 ]; then
	printf '\n%s check(s) failed\n' "$fails"
	exit 1
fi
printf '\nall checks passed\n'
