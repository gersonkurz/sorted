# Sorted!

Sorted! is the programming language that won the esoteric language contest of
the year 2000. Its design criteria, quoting the
[original page](manual/Sorted!%20-%20p-nand-q.com.html):

- You should be able to sing a good programming language.
- Thou shalt not have the same cardinal more than once.
- Each program should contain exactly fourteen statements.

It is also bilingual: every statement can be written in English or in German,
and a program may mix both.

This repository is a Go port of the original C++ interpreter that Gerson Kurz
wrote in 2000. It is faithful to the original, quirks included: it accepts the
same programs, prints the same output and fails the same way. The one
deliberate exception is the original's `##` preprocessor (OPP), which the port
leaves out (see below). The original
source and the Win32 and Linux binaries are kept in [`legacy/`](legacy).

## Hello, World.

```
This code uses the numbers zero, one, seventy two, one hundred and one, one hundred eight, two hundred,
    one hundred eleven, fourtyfour, thirtytwo, eightyseven, twohundred one, onehundred fourteen,
    two hundred two, onehundred, fourtysix, and twohundredtwentytwo.
This code always goes to the first label, and sometimes goes to the second label if the first condition is true.
This code writes the cell indexed by the first number as a character.
This code cannot read.
This code uses the sum of the first number and the second number.
This code uses the condition that the sixteenth number is equal to the cell indexed by the first number.
This code uses two labels.
This code does not use any ordered differences.
This code assigns the fifth number to the sixth number, the first number to the sixteenth number, the first sum to
    the first number, the seventh number to the eleventh number, and the fifth number to the thirteenth number.
This code does not use any products.
This code implements the first assignment, the second assignment, the third assignment, the fourth assignment, the
    third assignment, the fifth assignment, the third assignment, the first label, the second jump, the first output,
    the third assignment, the first jump, and the second label.
This code does not use any ratios.
This code does not use any logical operations.
Cool.
```

```
$ sorted legacy/sorted.win32/hello.s
Hello, World.
```

The four sample programs live in `legacy/sorted.win32`: `hello.s`, its German
twin `hallo.s`, `fibo.s` (Fibonacci numbers as English words) and `itoa.s`
(prints an integer).

## Installation

With Go 1.26 or later:

```
go install github.com/gersonkurz/sorted/cmd/sorted@latest
```

Or from a clone, with [just](https://github.com/casey/just):

```
just build      # out/build/sorted (sorted.exe on Windows)
just package    # macOS/Linux: release archives for all platforms in out/dist
```

## Usage

```
sorted [--dump FILE] [--to-c FILE] [--lang en|de] [--version] PROGRAM.s
sorted --from-c PROGRAM.c [--lang en|de] [--dump FILE] [--to-c FILE]
```

`sorted` parses and runs the program. `--dump` writes the parsed tables and
`--to-c` a translation into C, the original's `/D` and `/C` options; both are
written before the program runs. `--lang en` or `--lang de` prints the program
in English or German instead of running it, so `sorted --lang de hello.s` sings
Hello World in German. `--from-c` compiles a C program into Sorted! instead
(see "Young Adult Romance" below).

What the program prints, and the original's diagnostics, go to stdout exactly
as the original printed them:

```
$ sorted nothere.s
*** ERROR, unable to open file nothere.s for readingnothere.s is not intelligible.
```

Yes, on one line. The original's preprocessor forgot the newline, and so does
the port. The exit code is 0 on success, 1 if the program cannot be read or
parsed or fails at run time, and 2 for a usage error. (The original always
exited with 0.)

## The language in brief

A program is fourteen sentences, always in this order:

| # | Sentence | English | German |
|---|---|---|---|
| 1 | numbers | This code uses the numbers … | Dieses Programm benutzt die Zahlen … |
| 2 | jumps | This code always goes to … / sometimes goes to … if … is true | Dieses Programm springt immer an … |
| 3 | output | This code writes … as a character | Dieses Programm schreibt … als ein Zeichen |
| 4 | input | This code cannot read. | Dieses Programm kann nicht lesen. |
| 5 | sums | This code uses the sum of … and … | Dieses Programm benutzt die Summe aus … und … |
| 6 | conditions | This code uses the condition that … is equal to … | … die Bedingung dass … ist gleich … |
| 7 | labels | This code uses two labels. | Dieses Programm benutzt zwei Sprungziele. |
| 8 | ordered differences | This code uses the ordered difference between … and … | … die geordnete Differenz zwischen … und … |
| 9 | assignments | This code assigns … to … | Dieses Programm weisst zu … an … |
| 10 | products | This code uses the product of … and … | Dieses Programm benutzt das Produkt von … und … |
| 11 | implementation | This code implements the first assignment, … | Dieses Programm implementiert … |
| 12 | ratios | This code uses the ratio of … to … | (English only) |
| 13 | logical operations | This code does not use any logical operations. | Dieses Programm ist unlogisch. |
| 14 | coolness | Cool. | Hervorragend. |

Every sentence except the implementation and "Cool." also has a "none" form
("This code does not use any sums."). For more than one entry, use the plural
and a list, which needs a comma before the final "and": "the numbers seven,
eight, and nine", "the sums of … and …, and of … and …".

- **Numbers** are written as words and fill the memory cells in order. A
  number may appear only once.
- **References** use ordinals: "the third number" is cell 3, "the cell indexed
  by the first number" is the cell whose number the first cell holds.
- **Sums, ordered differences, products, ratios and conditions** are
  definitions, evaluated afresh every time something refers to them.
- **The implementation** is the program proper: the order in which
  assignments, outputs, jumps and labels run. The same assignment can appear
  many times.
- **Output** can be a character, or a number written out as an English or
  German cardinal or ordinal (see below for how to ask for an ordinal).

## Things worth knowing

Sorted! has no bugs, only features. The port keeps all of them:

- The number words have their own spelling: fifteen is "fiveteen", forty is
  "fourty", the ninth is the "nineth" and the twelfth the "twelveth".
  "fifteen" and "twelfth" are not numbers. In German, a thousand is
  "einstausend".
- Zero, written as a cardinal, is an empty line.
- "one hundred thousand" is 100. "onehundredthousand" is 100000.
- A program can have only one output and one input statement.
- A program cannot read: there is no way to refer to an input. "This code
  cannot read." is the only sensible input sentence.
- Logical operations can be declared but never used, for the same reason.
  They would compute `~a & ~b` anyway.
- "as a english ordinal" and "as a german ordinal" are part of the grammar,
  but do not parse: the cardinal alternative eats the "english" first. Say it
  twice and it works: "as a english english ordinal". In German, a cardinal
  takes four of them: "als ein ein ein ein deutscher Kardinal".
- German grammar is approximate: a reference to a difference must read
  "der ersten geordnete Differenz", because "geordneten" is not recognised.
- Reading a cell indirectly counts from 1, writing one indirectly counts from
  0. This is why `itoa.s` prints a NUL byte before its digits.
- A jump to a label that was declared but never placed lands just after the
  first statement.

[`CLAUDE.md`](CLAUDE.md) lists every known quirk, and the tests pin each one.

## How faithful is it?

The original `Sorted.exe` still runs on Windows. Its output for the sample
programs, its table dumps, its C translations and its error messages were
captured into [`testdata/golden`](testdata/golden); that directory's README
lists each capture's origin. The tests compare the port against them, and
expected output always comes from the original binary, never from reading its
source.

The original ran every program through a small C-style preprocessor (OPP,
with `##` directives for macros, includes and conditionals) before parsing it.
None of the sample programs uses it, so the port leaves it out: a program that
relies on `##` directives does not work here.

Where the C code has undefined behaviour (printing a negative number as a
cardinal, for example, indexes its word tables with a negative subscript), the
port does not try to reproduce what the 2000 compiler made of it. It gives a
simple, documented result instead: here, a run-time error.

## Young Adult Romance

Sorted! and C have been seeing each other since 2000: the original could
already turn any Sorted! program into C (`--to-c`). Now the relationship is
getting serious in the other direction: the same binary now compiles C into
Sorted!.

```
sorted --from-c fizzbuzz.c > fizzbuzz.s
sorted --from-c fizzbuzz.c --lang de > fizzbuzz-de.s
sorted fizzbuzz.s
```

**What works today.** A growing subset of C: a single `main`, `int` variables
(globals with constant initializers, and locals in nested blocks), integer
constants up to 999999999 and character constants (`'a'`, `'\n'`, `'\x41'`),
`+ - * / %`, comparisons, `&&`, `||` and `!` (short-circuiting, as in C),
assignment including `+=` and friends, `++` and `--`, `if`/`else`, `while`,
`for`, `break`, `continue`, `return`, and `putchar`. `#include <stdio.h>` is
allowed, so the same file compiles with a C compiler too; anything else gets a
precise "not supported in Sorted! (yet)" with its line and column. The front end is a Go
port of [chibicc](https://github.com/rui314/chibicc), Rui Ueyama's small C
compiler (MIT license).

Constants become the declared numbers, each declared once, which turns "thou
shalt not have the same cardinal more than once" into constant pooling.
Variables live in the cells after them, arithmetic becomes sums, differences,
products and ratios, comparisons become conditions, and `if` and `while`
become labels and jumps. Every program the compiler writes is parsed back to
make sure it is the same program, and the tests check that it prints exactly
what the C program prints when compiled with clang.

`--lang en` or `--lang de` also writes an existing Sorted! program back out, so
translating between English and German comes free. Everything the compiler
and the translator write uses only forms the 2000 parser knows, so it also
runs on the original `Sorted.exe`. A few programs that run fine cannot be
written back: a declaration may add up its parts to a number from 1000000000
on ("ninehundredmillion onehundredmillion"), and the current renderer cannot
spell such values.

**What is coming** (issues #14 to #17): arrays and strings, then functions,
then recursion. Some limits are part of the deal. There
are no library calls, `putchar` being the only one. There is no `malloc`, just
one big block of memory. And there is no input, because Sorted! cannot read.
Round trip a program through C → Sorted! → C, and you get C obfuscation for
free.

### Known kinks

Every program the compiler writes has to get past the 2000 parser unchanged,
which takes the occasional detour:

- **"the eight number".** "the eighth number" does not read back: the parser
  matches the cardinal "eight" and trips over the leftover "h". So wherever an
  ordinal ends in "eighth", the compiler writes the cardinal instead: "the
  eight number", "the twentyeight number", and so on. `itoa.s` did that all
  along.
- **German endings, sometimes.** "der ersten Zahl" and even "der
  einhundertersten Zahl" read back, but "der zwanzigsten Zahl" does not. The
  compiler tries the proper ending and falls back to the bare form ("der
  zwanzigste Zahl") wherever the parser would trip.
- **No "dem".** The parser knows the articles die, das, der and den, but not
  dem. Where German wants "dem", neuter nouns keep "das" ("von das erste
  Produkt") and masculine ones take "den" ("den ersten Sprungbefehl").
- **"der ersten geordnete Differenz".** The adjective must stay uninflected,
  or the parser does not recognise the difference.
- **"eins Sprungziel".** One label is "eins", because the parser reads "ein"
  as no number at all.
- **Ordinals in stereo.** Asking for an ordinal output means repeating
  yourself: "as a english english ordinal". The parser's alternatives share
  one cursor, and each failed one eats a word. German needs up to four:
  "als ein ein ein ein deutscher Kardinal".
- **Code-switching.** German has no ratios, no logical operations and no lists
  of ordered differences, so in a German program those sentences are written
  in English. Sorted! has always allowed switching language from one sentence
  to the next.
- **Some tables cannot be written.** A program whose internal table layout
  could not have come from Sorted! text, which only happens with hand-crafted
  tables, is refused rather than silently changed.

## Development

```
just test       # all tests
just verify     # format check, go vet, all tests
just check      # verify plus staticcheck
just run legacy/sorted.win32/fibo.s
```

The packages follow the original's pipeline: `internal/syntax` (source filter
and parser), `internal/numbers` (number words), `internal/interp`
(interpreter), `internal/emit` (`/D` and `/C` output), `internal/render`
(programs back to Sorted! text), and the command in `cmd/sorted`. The C
compiler adds `internal/cc` (the chibicc-derived front end) and
`internal/compile` (lowering C to Sorted! tables).

## License

MIT, see [LICENSE](LICENSE).
