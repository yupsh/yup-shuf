# yup-shuf

```
NAME:
   shuf - generate random permutations

USAGE:
   shuf [OPTIONS] [FILE]
   shuf -e [OPTIONS] [ARG...]
   shuf -i LO-HI [OPTIONS]

   Write a random permutation of the input lines to standard output.
   With no FILE, or when FILE is -, read standard input.

VERSION:
   dev

GLOBAL OPTIONS:
   --head-count int, -n int         output at most COUNT lines (default: 0)
   --input-range string, -i string  treat each number LO through HI as an input line
   --echo, -e                       treat each ARG as an input line
   --seed int                       seed the shuffle for deterministic output (default: 0)
   --help, -h                       show help
   --version                        print version information and exit
```
