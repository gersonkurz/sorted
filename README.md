# Sorted!

Sorted! is the programming language that won the esoteric language contest of
the year 2000. Its design criteria, quoting the
[original page](https://p-nand-q.com/programming/languages/sorted/index.html):

- You should be able to sing a good programming language.
- Thou shalt not have the same cardinal more than once.
- Each program should contain exactly fourteen statements.

Twenty-six years later it sings in seven tongues: English, German, Italian, the
French of Vaud, Brazilian Portuguese, Japanese and Mandarin. It compiles C,
and it turns back into C. And it is the only machine translator with a
hundred percent accuracy, as long as you only ever say one of fourteen
things.

This repository is a Go port of the original C++ interpreter that Gerson Kurz
wrote in 2000, faithful to it in every respect, quirks included. Its quirks
are not bugs. The papal bull *Ordinata non errant*
([English](docs/ordinata-non-errant.md), [Deutsch](docs/ordinata-non-errant.de.md))
declares them doctrine, and this README is the shorter catechism.

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

The same program ends, depending on whom you ask:

```
Cool.                                          Hervorragend.
This code is very very cool.                   Questo programma è molto molto figo.
Ce programme est très très chouette.           Este programa é muito muito legal.
Kono puroguramu wa totemo totemo kakkoii desu. 这个程序非常非常酷。
```

## Installation

With Go 1.26 or later:

```
go install github.com/gersonkurz/sorted/cmd/sorted@latest
```

Or from a clone, with [just](https://github.com/casey/just): `just build`.
Release archives for 19 platforms are made with `just package`.

## Usage

```
sorted hello.s                     # run it
sorted --deutsch hello.s           # sing it in German instead
sorted --lang ja,zh hello.s        # in Japanese and Mandarin at once
sorted --from-c prog.c             # compile C into Sorted!
sorted --to-c out.c hello.s        # write the program as C, then run it
sorted --dump tables.txt hello.s   # write the parsed tables, as the original's /D
sorted --version                   # This code is very very cool.
sorted --help                      # in a language picked at random
```

What a program prints, and the original's diagnostics, go to stdout exactly
as the original printed them:

```
$ sorted nothere.s
*** ERROR, unable to open file nothere.s for readingnothere.s is not intelligible.
```

Yes, on one line: the original forgot the newline, and so does the port. The
exit code is 0 on success, 1 when a program cannot be read or parsed or fails
at run time, and 2 for a usage error. (The original always exited with 0.)

## The language

A program is fourteen sentences in a fixed order: numbers, jumps, output,
input, sums, conditions, labels, ordered differences, assignments, products,
implementation, ratios, logical operations, and "Cool.". The numbers fill the
memory cells, "the third number" is cell 3, the sums and conditions are
definitions evaluated afresh whenever something refers to them, and the
implementation is the program proper: the order in which its assignments,
outputs, jumps and labels run.

Every sentence can be written in every language Sorted! speaks (the German
of 2000 lacks three, which Very Sorted! supplies), and a program may switch
language from one sentence to the next. Name several languages
and `sorted` does the switching for you, at random, in turn, or by the
fewest syllables. [docs/languages.md](docs/languages.md) has every sentence
in every language.

## The ages of very

There is no version 1.0. The versions are the dialects, counted in verys,
and the last sentence of a program says which one it speaks:

- **"Cool."**: the Sorted! of 2000, exactly as `Sorted.exe` runs it.
- **"This code is very cool."**: Very Sorted!, which can finally read, has a
  true NAND, and spells German with umlauts.
- **"This code is very very cool."**: Very Very Sorted!, which speaks the five
  new tongues.

Each age keeps everything the ones before it had. Whatever the original
accepts or rejects is accepted or rejected exactly as before. Releases are
named the same way, and the first one is `very-very-cool`.

## The marriage with C

Sorted! and C have been seeing each other since 2000. The original could
already write its programs as C, if not faithfully: the bull rejected that
translation in [Caput VIII](docs/ordinata-non-errant.md#caput-viii-de-translatione-reprobata).
In 2026 the relationship became serious in both directions. `--from-c`
compiles a growing subset of C into Sorted!, and `--to-c` turns any Sorted!
program back into C. A program that makes the round trip prints exactly what
it printed before, and nothing in it resembles the original. The bull blesses
the union in [Caput XI](docs/ordinata-non-errant.md#caput-xi-de-matrimonio-cum-c).

The first song had to be the obvious one:
[99 Bottles of Beer](examples/99-bottles.c), compiled into
[English](examples/99-bottles.s), [German](examples/99-bottles.de.s),
[Italian](examples/99-bottles.it.s), [French](examples/99-bottles.fr.s),
[Portuguese](examples/99-bottles.pt.s), [Japanese](examples/99-bottles.ja.s)
and Mandarin, in [hanzi](examples/99-bottles.zh.s) and
[pinyin](examples/99-bottles.pinyin.s). Each is about 630 singable lines. The
second song is a [Brainfuck interpreter](examples/brainfuck.s), which proves
that Sorted! is [Turing complete](docs/turing-completeness.md), as it already
was in 2000.

[docs/c.md](docs/c.md) covers the C that works, how it becomes Sorted!, and the
kinks along the way.

## The doctrine

The original `Sorted.exe` still runs on Windows. Its output for the sample
programs, its table dumps and its error messages were captured into
[`testdata/golden`](testdata/golden), and the tests hold the port to them.
Expected output always comes from the original binary, never from reading its
source. Every property of the language is pinned by a test, for instance:

- Fifteen is "fiveteen", forty is "fourty", and the twelfth is the
  "twelveth". "fifteen" is not a number.
- "one hundred thousand" is 100. "onehundredthousand" is 100000.
- "as a english ordinal" does not parse, but "as a english english ordinal"
  does, because the cardinal alternative eats the first "english".
- Reading a cell indirectly counts from 1, writing one counts from 0, which
  is why `itoa.s` prints a NUL before its digits.
- The logical operation everyone took for a NAND is a NOR, and no program of
  2000 can ever use it.

There are two exceptions to faithfulness:
- **Undefined behaviour** in the C++ (printing a negative cardinal, say) is
  not imitated; it gets a plain, documented result, as the bull teaches in
  [Caput IX](docs/ordinata-non-errant.md#caput-ix-de-rebus-non-definitis).
- **The preprocessor** (OPP, with its `##` directives) is left out: no sample
  program uses it.

## Development

```
just test       # all tests
just verify     # format check, go vet, all tests
just lint       # format check, go vet, staticcheck
just check      # lint, tests, and every release platform builds
```

The packages follow the original's pipeline:
- `internal/syntax`: the source filter and parser, one file per language
- `internal/numbers`: number words
- `internal/interp`: the interpreter
- `internal/emit`: the `/D` dump and `--to-c`
- `internal/render`: tables back to Sorted! text

The C compiler adds `internal/cc` (a front end derived from
[chibicc](https://github.com/rui314/chibicc)) and `internal/compile`. The
command lives in `cmd/sorted`. [`CLAUDE.md`](CLAUDE.md) has the details,
every known quirk included.

## License

MIT, see [LICENSE](LICENSE).
