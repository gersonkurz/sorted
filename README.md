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
sorted [--dump FILE] [--to-c FILE] [--version] PROGRAM.s
```

`sorted` parses and runs the program. `--dump` writes the parsed tables and
`--to-c` a translation into C, the original's `/D` and `/C` options; both are
written before the program runs.

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
- **Output** can be a character, a number written out as an English or German
  cardinal, or (in German phrasing only: "als eine deutsche Ordinalzahl") a
  German ordinal.

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
  but never parse.
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

## Development

```
just test       # all tests
just verify     # format check, go vet, all tests
just check      # verify plus staticcheck
just run legacy/sorted.win32/fibo.s
```

The packages follow the original's pipeline: `internal/syntax` (source filter
and parser), `internal/numbers` (number words), `internal/interp`
(interpreter), `internal/emit` (`/D` and `/C` output), and the command in
`cmd/sorted`.

## License

MIT, see [LICENSE](LICENSE).
