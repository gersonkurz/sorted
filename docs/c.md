# C and Sorted!

*The practical side of the marriage that the bull blesses in
[Caput XI](ordinata-non-errant.md#caput-xi-de-matrimonio-cum-c).*

`sorted --from-c` compiles a C program into Sorted!, and `sorted --to-c`
writes any Sorted! program as C. Together they make a round trip: C →
Sorted! → C. The tests take every C program they compile around the whole
loop and compare its output with what clang makes of the original.

```
sorted --from-c fizzbuzz.c --english > fizzbuzz.s
sorted fizzbuzz.s
sorted --from-c fizzbuzz.c --to-c obfuscated.c --lang ja > fizzbuzz.ja.s
```

Unless you ask for a language, `--from-c` picks one at random. Ask for
several and you get [Babel mode](languages.md#babel-mode).

## Two songs

[`examples/99-bottles.c`](../examples/99-bottles.c) is the standard C version
of 99 Bottles of Beer. Next to it sits what the compiler makes of it in every
language: [English](../examples/99-bottles.s), [German](../examples/99-bottles.de.s),
[Italian](../examples/99-bottles.it.s), [French](../examples/99-bottles.fr.s),
[Portuguese](../examples/99-bottles.pt.s), [Japanese](../examples/99-bottles.ja.s),
and Mandarin in [hanzi](../examples/99-bottles.zh.s) and
[pinyin](../examples/99-bottles.pinyin.s). Each is about 630 lines, every
one of them singable, and every verse prints exactly as the C program
prints it.

[`examples/brainfuck.c`](../examples/brainfuck.c) is a Brainfuck
interpreter, and [`examples/brainfuck.s`](../examples/brainfuck.s) is that
interpreter in Sorted!: `sorted examples/brainfuck.s < examples/brainfuck.in`
prints "Hello World!". Brainfuck also translates to Sorted! command by
command, which makes Sorted! [Turing complete](turing-completeness.md), and
it already was in 2000.

## The C that works

A growing subset of C:

- **Functions:** functions with `int`, `char` and pointer parameters, returning `int`, `char`, a pointer or `void`; prototypes and recursion included.
- **Types:** `int` and `char` variables (also `unsigned`), structs, pointers to them, and arrays of those, also of arrays (`int m[3][4]`).
- **Initializers:** globals with constant or address initializers, locals in nested blocks, nested `{...}` lists and string literals.
- **Constants:** integer constants (decimal, hex, octal), character constants (`'a'`, `'\n'`, `'\x41'`) and string literals.
- **Expressions:**
  - `a[i]`, `&x`, `*p`, `s.m`, `p->m` and pointer arithmetic
  - `+ - * / %`, comparisons, and `&&`, `||` and `!`, which short-circuit as in C
  - the bitwise `& | ^ ~ << >>`
  - assignment including `+=`, `<<=` and friends, `++` and `--`
  - `?:` and the comma operator
- **Statements:** `if`/`else`, `while`, `do`/`while`, `for`, `switch` (fall-through included; Duff's device works), `break`, `continue` and `return`.
- **Input and output:** `putchar` and `getchar`, the only library calls.
- **Preprocessor:** `#define` (with and without parameters) and `#undef`. Array lengths and global initializers may be constant expressions. `#include <stdio.h>` is allowed, so the same file compiles with a C compiler too.

Anything else gets a precise "not supported in Sorted! (yet)" with its line and column. There is no `malloc`, just one big block of memory.

The front end is a Go port of [chibicc](https://github.com/rui314/chibicc),
Rui Ueyama's small C compiler (MIT license).

## How C becomes Sorted!

- **Constants** become the declared numbers, each declared once. "Thou shalt
  not have the same cardinal more than once" turns into constant pooling.
- **Variables** live in the cells after them. Arithmetic becomes sums,
  differences, products and ratios, comparisons become conditions, and `if`
  and `while` become labels and jumps.
- **Arrays** are runs of cells, reached through pointer cells. "The cell
  indexed by" reads the cell before the one it writes, the same off-by-one
  that makes `itoa.s` print a NUL, so the compiler keeps one pointer for
  reading and one for writing.
- **Functions** exist once, because Sorted! has no call stack. A call
  stores its arguments in the parameter cells and its own number in a
  return-address cell, and a return jumps through a chain of "go to the call
  site if the return address is ..." back to where it came from. A function
  that can call itself saves its parameters and locals on a stack before
  such a call and restores them afterwards. The stack is the free memory
  after the variables, so a recursion that goes too deep runs out of cells.
- **Bitwise operators** are arithmetic where they can be: `~x` is `-1 - x`,
  shifting by a constant multiplies or divides, and `x & 255` is a
  remainder. Shifting by a variable count calls a small helper written in
  the C subset itself. Any other `&`, `|` or `^` is built from NANDs, which
  only Very Sorted! has, so such a program comes out very.
- **`getchar`** needs Very Sorted! too: the Sorted! of 2000 cannot read.
- **Numbers** from 1000000000 on are built as `1000000 * q + r`.

Every program the compiler writes is parsed back to make sure it is the same
program.

## Translation

`--lang` also writes an existing Sorted! program back out, in any language.
Everything the compiler and the translator write in English or German uses
only forms the 2000 parser knows, so it also runs on the original
`Sorted.exe` (unless the program is very).

The other languages exist only in Very Very Sorted!, so writing a program in
one of them makes it Very Very Sorted!. The rare 2000 program that would
then behave differently is refused: one that prints German numbers, which
Very Sorted! spells in UTF-8, or one whose statements reach past their
tables.

A few programs that run fine cannot be written back. A declaration may add
up its parts to a number from 1000000000 on ("ninehundredmillion
onehundredmillion"), and the renderer cannot spell such values.

## And back to C

`sorted --from-c prog.c --to-c obfuscated.c` takes a program on the full
round trip. The result prints what `prog.c` prints, and nothing in it
resembles the original:

- every sum, difference, product, ratio and condition becomes a function named after where it sits in the tables (`S41`, `D7`, `C12`)
- every statement becomes a case of one big switch
- every variable becomes a cell of `_[193719]`
- the number words are ported along, in case the program prints cardinals

It is obfuscation with a warranty. `--to-c` works for any Sorted! program, quirks included: past-the-end references, jumps to labels that were never placed, 32-bit wrap-around and the run-time errors all come out the way the interpreter has them.

## Known kinks

Every program the compiler writes in English or German has to get past the
2000 parser unchanged, which takes the occasional detour:

- **"the eight number".** "the eighth number" does not read back: the parser
  matches the cardinal "eight" and trips over the leftover "h". So wherever an
  ordinal ends in "eighth", the compiler writes the cardinal instead: "the
  eight number", "the twentyeight number". `itoa.s` did that all along.
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
- **Code-switching.** The German of 2000 has no ratios, no logical operations
  and no lists of ordered differences, so in a German program those sentences
  are written in English. Sorted! has always allowed switching language from
  one sentence to the next. A program that is Very Sorted! anyway (one that
  calls `getchar` or needs a NAND) is German throughout.
- **Some tables cannot be written.** A program whose internal table layout
  could not have come from Sorted! text is refused rather than silently
  changed. That only happens with hand-crafted tables.
