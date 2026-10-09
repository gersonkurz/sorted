# Sorted!

Sorted! is the programming language that won the esoteric language contest of
the year 2000. Its design criteria, quoting the
[original page](https://p-nand-q.com/programming/languages/sorted/index.html):

- You should be able to sing a good programming language.
- Thou shalt not have the same cardinal more than once.
- Each program should contain exactly fourteen statements.

It is also multilingual: every statement can be written in English or in
German, and, since Very Very Sorted!, in Italian, the French of Vaud,
Brazilian Portuguese or Japanese (in romaji), and a program may mix them.

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
just cross      # check that it builds for all 19 release platforms
just package    # macOS/Linux: release archives for all platforms in out/dist
```

## Usage

```
sorted [--dump FILE] [--to-c FILE] [--lang NAME | --NAME] [--version] PROGRAM.s
sorted --from-c PROGRAM.c [--lang NAME | --NAME] [--dump FILE] [--to-c FILE]
```

`sorted` parses and runs the program. `--dump` writes the parsed tables, as
the original's `/D` does, and `--to-c` a C program that behaves exactly like
the Sorted! one; both are written before the program runs. The original's `/C`
had a C translation too, but it does not do what the program does: it prints
every number as a character, so `fibo.s` comes out as raw bytes instead of
"one, one, two, three", and it writes indirect cells one off, so `itoa.s`
moves its famous NUL. `--to-c` replaces it. `--lang NAME`, or just `--NAME`, prints the program in that
language instead of running it, so `sorted --deutsch hello.s` sings Hello
World in German, `sorted --italiano hello.s` in Italian, `sorted --vaudois
hello.s` in French, `sorted --brasileiro hello.s` in Portuguese and `sorted
--nihongo hello.s` in Japanese. Sorted! has no favourite language, not even for naming
languages: NAME may be the language's name in any language Sorted! speaks or
will, ignoring case and accents (`--english`, `--Englisch`, `--anglais`,
`--lang inglês`, `--英語`, `--lang en`; `sorted --help` lists them all).
`--from-c` compiles a C program into Sorted! instead (see "Young Adult
Romance" below), and unless you ask for a language, each run picks one at
random.

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

There is no version 1.0. The versions are the dialects, and `--version`
says which one this `sorted` reads, in its own words:

```
$ sorted --version
This code is very very cool.
```

A build that is not a release adds where it comes from, as in `This code is
very very cool. (a105f41-dirty)`. Releases are tagged the same way (`cool`,
`very-cool`, `very-very-cool`, ...), and the first one will be
`very-very-cool`: nothing is released before Very Very Sorted!.

## The language in brief

A program is fourteen sentences, always in this order:

| # | Sentence | English | German | Italian (Very Very Sorted!) | French (Very Very Sorted!) | Portuguese (Very Very Sorted!) | Japanese (Very Very Sorted!) |
|---|---|---|---|---|---|---|---|
| 1 | numbers | This code uses the numbers … | Dieses Programm benutzt die Zahlen … | Questo programma usa i numeri … | Ce programme utilise les nombres … | Este programa usa os números … | Kono puroguramu wa kazu … o tsukaimasu |
| 2 | jumps | This code always goes to … / sometimes goes to … if … is true | Dieses Programm springt immer an … | Questo programma va sempre alla … / va talvolta alla … se … è vera | Ce programme va toujours à la … / va parfois à la … si … est vraie | Este programa sempre vai para o … / às vezes vai para o … se … for verdadeira | Kono puroguramu wa itsumo … ni tobi, … ga shin nara … ni tobimasu |
| 3 | output | This code writes … as a character | Dieses Programm schreibt … als ein Zeichen | Questo programma scrive … come carattere | Ce programme écrit … comme caractère | Este programa escreve … como caractere | Kono puroguramu wa … o moji to shite kakimasu |
| 4 | input | This code cannot read. | Dieses Programm kann nicht lesen. | Questo programma non può leggere. | Ce programme ne peut pas lire. | Este programa não pode ler. | Kono puroguramu wa yomemasen. |
| 5 | sums | This code uses the sum of … and … | Dieses Programm benutzt die Summe aus … und … | Questo programma usa la somma del … e del … | Ce programme utilise la somme du … et du … | Este programa usa a soma do … e do … | Kono puroguramu wa … to … no wa o tsukaimasu |
| 6 | conditions | This code uses the condition that … is equal to … | … die Bedingung dass … ist gleich … | … la condizione che … sia uguale al … | … la condition que … soit égal au … | … a condição de que … seja igual ao … | … ga … to hitoshii to iu jōken … |
| 7 | labels | This code uses two labels. | Dieses Programm benutzt zwei Sprungziele. | Questo programma usa due etichette. | Ce programme utilise deux étiquettes. | Este programa usa dois rótulos. | Kono puroguramu wa raberu o niko tsukaimasu. |
| 8 | ordered differences | This code uses the ordered difference between … and … | … die geordnete Differenz zwischen … und … | … la differenza ordinata tra … e … | … la différence ordonnée entre … et … | … a diferença ordenada entre … e … | … to … no sa … |
| 9 | assignments | This code assigns … to … | Dieses Programm weisst zu … an … | Questo programma assegna … al … | Ce programme affecte … au … | Este programa atribui … ao … | Kono puroguramu wa … o … ni dainyū shimasu |
| 10 | products | This code uses the product of … and … | Dieses Programm benutzt das Produkt von … und … | … il prodotto del … e del … | … le produit du … et du … | … o produto do … e do … | … to … no seki … |
| 11 | implementation | This code implements the first assignment, … | Dieses Programm implementiert … | Questo programma implementa il primo assegnamento, … | Ce programme implémente la première affectation, … | Este programa implementa a primeira atribuição, … | Kono puroguramu wa dai-ichi no dainyū, … o jissō shimasu |
| 12 | ratios | This code uses the ratio of … to … | (English only) | … il rapporto tra … e … | … le rapport du … au … | … a razão entre … e … | … to … no hi … |
| 13 | logical operations | This code does not use any logical operations. | Dieses Programm ist unlogisch. | Questo programma è illogico. | Ce programme est illogique. | Este programa é ilógico. | Kono puroguramu wa hironriteki desu. |
| 14 | coolness | Cool. | Hervorragend. | Questo programma è molto molto figo. | Ce programme est très très chouette. | Este programa é muito muito legal. | Kono puroguramu wa totemo totemo kakkoii desu. |

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

Sorted! has no bugs, only features. The port keeps all of them, and the
papal bull *Ordinata non errant* ([English](docs/ordinata-non-errant.md),
[Deutsch](docs/ordinata-non-errant.de.md)) declares them doctrine (including
the logical operation that was never NAND):

- The number words have their own spelling: fifteen is "fiveteen", forty is
  "fourty", the ninth is the "nineth" and the twelfth the "twelveth".
  "fifteen" and "twelfth" are not numbers. In German, a thousand is
  "einstausend".
- Zero, written as a cardinal, is an empty line.
- "one hundred thousand" is 100. "onehundredthousand" is 100000.
- A program can have only one output and one input statement.
- A program cannot read: there is no way to refer to an input. "This code
  cannot read." is the only sensible input sentence. (Very Sorted!, below,
  can.)
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
already turn Sorted! programs into C (`/C`, if not always correctly). Now the relationship is
getting serious in the other direction: the same binary now compiles C into
Sorted!.

```
sorted --from-c fizzbuzz.c --english > fizzbuzz.s
sorted --from-c fizzbuzz.c --german > fizzbuzz-de.s
sorted fizzbuzz.s
```

The first song has to be the obvious one. [`examples/99-bottles.c`](examples/99-bottles.c)
is the standard C version of 99 Bottles of Beer, and next to it sit what the
compiler makes of it, [in English](examples/99-bottles.s),
[in German](examples/99-bottles.de.s), [in Italian](examples/99-bottles.it.s),
[in French](examples/99-bottles.fr.s), [in Portuguese](examples/99-bottles.pt.s)
and [in Japanese](examples/99-bottles.ja.s):
about 630 lines each, all singable, every verse printed exactly as the C
program prints it.

The second proves a point. [`examples/brainfuck.c`](examples/brainfuck.c) is
a Brainfuck interpreter, and [`examples/brainfuck.s`](examples/brainfuck.s)
is that interpreter in Sorted!: `sorted examples/brainfuck.s <
examples/brainfuck.in` prints "Hello World!". Brainfuck also translates to
Sorted! command by command, which makes Sorted! [Turing
complete](docs/turing-completeness.md), and it already was in 2000.

**What works today.** A growing subset of C: functions with `int`, `char`
and pointer parameters returning `int`, `char`, a pointer or `void`
(prototypes and recursion included), `int` and `char` variables (also
`unsigned`), structs, pointers to them and arrays of those, also of arrays
(`int m[3][4]`; globals with constant
or address initializers, locals in nested blocks, nested `{...}` lists and
string literals), integer constants (decimal, hex, octal), character constants (`'a'`,
`'\n'`, `'\x41'`) and string literals, `a[i]`, `&x`, `*p`, `s.m`, `p->m` and
pointer arithmetic, `+ - * / %`,
comparisons, `&&`, `||` and `!` (short-circuiting, as in C), the bitwise `& |
^ ~ << >>`, assignment including `+=`, `<<=` and friends, `++` and `--`,
`?:` and the comma operator, `if`/`else`, `while`, `do`/`while`, `for`,
`switch` (fall-through included, Duff's device works), `break`, `continue`,
`return`, `putchar` and `getchar`.
The preprocessor knows `#define` (with and without parameters) and `#undef`,
and array lengths and global initializers can be constant expressions.
`#include <stdio.h>` is allowed, so the same file compiles with a C compiler
too; anything else gets a precise "not supported in Sorted! (yet)" with its
line and column. The front end is a Go
port of [chibicc](https://github.com/rui314/chibicc), Rui Ueyama's small C
compiler (MIT license).

Constants become the declared numbers, each declared once, which turns "thou
shalt not have the same cardinal more than once" into constant pooling.
Variables live in the cells after them, arithmetic becomes sums, differences,
products and ratios, comparisons become conditions, and `if` and `while`
become labels and jumps. Arrays are runs of cells, reached through pointer
cells: for the same pointer, "the cell indexed by" reads the cell before the
one it writes, the same off-by-one that makes `itoa.s` print a NUL, so the
compiler keeps one pointer for reading and one for writing. Sorted! has no
call stack, so each function exists once: a call stores its arguments in the
parameter cells and its own number in a return-address cell, and a return
jumps through a chain of "go to the call site if the return address is ..."
back to where it came from. A function that can call itself, directly or
through others, saves its parameters and locals on a stack before such a
call and restores them afterwards; the stack is the free memory after the
variables, so a recursion that goes too deep runs out of cells. The
bitwise operators are arithmetic where they can be: `~x` is `-1 - x`,
shifting by a constant multiplies or divides, `x & 255` is a remainder, and
shifting by a variable count calls a small helper function, written in the
C subset itself. Any other `&`, `|` or `^` is built from NANDs, which only
Very Sorted! (below) has, so such a program comes out very.
Numbers from 1000000000 on are built as `1000000 * q + r`. Every program the compiler writes is parsed back to
make sure it is the same program, and the tests check that it prints exactly
what the C program prints when compiled with clang.

`--lang en` or `--lang de` also writes an existing Sorted! program back out, so
translating between English and German comes free. Everything the compiler
and the translator write in English or German uses only forms the 2000 parser
knows, so it also runs on the original `Sorted.exe` (unless the program is
very). Italian, French, Portuguese and Japanese are Very Very Sorted!
(below), so `--lang it`, `--lang fr`, `--lang pt` or `--lang ja` makes any
program Very Very Sorted!,
and refuses the rare 2000 program that would then do
something else: one that prints German numbers, which Very Sorted! spells in
UTF-8, or one whose statements reach past their tables. A few programs that run fine cannot be
written back: a declaration may add up its parts to a number from 1000000000
on ("ninehundredmillion onehundredmillion"), and the current renderer cannot
spell such values.

Some limits are part of the deal. There are no library calls, `putchar` and
`getchar` being the only ones. There is no `malloc`, just one big block of
memory.

**Very Sorted!** The Sorted! of 2000 cannot read: a program may declare an
input, but no statement can name it. Very Sorted!, the first dialect of the
very Fibonacci sequence, finishes that thought. A program that ends with
"This code is very cool." ("Dieses Programm ist ganz hervorragend.") may
implement "the first input" ("die erste Eingabe"), which reads a character
into the cell the input declares. A C program that calls `getchar()` compiles
to Very Sorted! by itself, so `wc`, `rot13` and friends now sing. Everything
the 2000 parser accepts or rejects stays exactly as it was: a program is
Very Sorted! only if the original's grammar fails and the very one succeeds.

Very Sorted! also has a NAND, at last: "the logical operation of not both
the first number and the second number" ("die logische Verknüpfung von
nicht beiden, der ersten Zahl und der zweiten Zahl"), and statements and
expressions can name it ("the first logical operation", "die erste logische
Verknüpfung"). The original's "of not X and not Y" stays what it always
was, a NOR, and now has German too ("von nicht X und nicht Y"). The
compiler builds `&`, `|` and `^` from two, three and four NANDs.

And Very Sorted! defines what 2000 left undefined: "the cell indexed by the
first sum" reads and writes the way indexing by a cell always has, the read
one cell before the write. A compiled program that is very anyway reaches
its array elements that way, without a pointer cell for every access.

Very Sorted! is also read as UTF-8, so German is finally German: "fünf",
"zwölf", "dreißig" and "Verhältnisse", in any case and either Unicode form,
and the 2000 spellings "fuenf" and "Verhaeltnisse" still work. German
numbers are printed that way too, and `--deutsch` writes a very program
with umlauts. On a Windows console, `chcp 65001` shows them as letters.
The 2000 dialect still reads bytes, where "fünf" is "f nf".

**Very Very Sorted!** The second dialect of the very Fibonacci sequence
speaks Italian. A program that ends with "This code is very very cool."
("Dieses Programm ist ganz ganz hervorragend.", "Questo programma è molto
molto figo.") has everything Very Sorted! has, and every sentence may be
Italian, mixed with English and German as those two always mixed:

```
Questo programma usa i numeri zero, uno, ventitré e un milione.
Questo programma va sempre alla prima etichetta e va talvolta alla seconda etichetta se la prima condizione è vera.
Questo programma usa le somme del primo numero e del secondo numero, e dell'ottava cella e della prima somma.
Questo programma usa la condizione che il primo numero sia uguale al secondo numero.
Questo programma usa l'operazione logica né il primo numero né il secondo numero.
```

Italian has a form for everything and no quirks: the articles fuse with
their prepositions and elide before a vowel ("dell'ottava cella"), the
ordinals agree with their nouns ("il primo numero", "la prima somma"), a
NOR is "né … né …" and a NAND "non entrambi … e …", and a program that uses
no logical operations "è illogico". Its numbers are single words, as on a
cheque ("unmilioneduecentomila"), written with or without accents, and
Italian numbers can be printed in any of the three languages ("come
cardinale italiano", "as an italian ordinal", "als eine italienische
Ordinalzahl"), for any number at all: "zero", "zeresimo", "meno sette".

Very Very Sorted! also speaks French, as spoken in Vaud, where counting is
decimal at last: septante, huitante, nonante. A program may end "Ce
programme est très très chouette.", and one without jumps says so the Vaudois
way, "Y'a pas le feu au lac.":

```
Ce programme utilise les nombres septante-et-un, huitante et deux-cent-vingt-et-un.
Y'a pas le feu au lac.
Ce programme, eh, écrit le premier nombre comme ordinal vaudois.
Ce programme utilise la condition que la première somme soit égale au troisième nombre.
Ce programme affecte la première somme à la cellule indexée par le deuxième nombre.
```

Numbers are one word, joined by hyphens as the 1990 spelling writes them,
which Very Very Sorted! keeps inside words. French loves its fillers, so the
parser reads past "eh", "hein", "quoi" and "voilà" (with their commas)
wherever they stand, and the renderer puts them in every third sentence and
ends the implementation with ", voilà". Vaudois numbers print in every
language too ("as a vaudois cardinal", "come ordinale vodese"), "zéroième"
and "moins sept" included.

And it speaks Brazilian Portuguese, Brazilian by name, because Brazil counts
*dezesseis, dezessete, dezenove* where Portugal says *dezasseis, dezassete,
dezanove*, and a billion is a *bilhão*. A program may end "Este programa é
muito muito legal.":

```
Este programa usa os números vinte e três, vinte, e um.
Este programa sempre vai para o primeiro rótulo.
Este programa escreve a vigésima terceira célula como ordinal brasileiro.
Este programa usa a condição de que a primeira soma seja menor que o terceiro número.
Este programa atribui a primeira soma à célula indexada pelo segundo número.
```

Numbers are several words joined by "e", as Brazilians write them ("duzentos
e trinta e quatro"), and so is the end of a list, so the parser reads a
number as far as it goes: "vinte e um" is 21, and the list of 20 and 1 is
written "vinte, e um". Ordinals agree with their nouns ("o vigésimo terceiro
número", "a vigésima terceira soma"), and Brazilian numbers print in every
language ("as a brazilian cardinal", "comme ordinal brésilien"), "zerésimo"
and "menos sete" included.

And it speaks Japanese, in romaji, which puts its verbs last. A list of
things ends with what the program does with them, a list of actions chains
the verb, and ordinals are a prefix, *dai-*:

```
Kono puroguramu wa kazu nijūsan, ichiman to roppyaku o tsukaimasu.
Kono puroguramu wa itsumo dai-ichi no raberu ni tobi, dai-ichi no jōken ga shin nara dai-ni no raberu ni tobimasu.
Kono puroguramu wa dai-ichi no wa o nihongo no josū to shite kakimasu.
Kono puroguramu wa raberu o niko tsukaimasu.
Kono puroguramu wa totemo totemo kakkoii desu.
```

A sum is *wa*, which is also the topic particle, so "Kono puroguramu wa wa o
tsukaimasen" says that the program uses no sums. Numbers are one word,
grouped by ten thousand (*man*) and a hundred million (*oku*), with their
sound changes (sanbyaku, roppyaku, happyaku, sanzen, hassen), labels are
counted with *-ko* (ikko, niko, rokko), and long vowels may be written
"jū", "juu" or "ju". Japanese numbers print in every language ("as a
japanese ordinal", "come cardinale giapponese"), "dai-zero" and "mainasu
nana" included.

**And back to C.** `sorted --from-c prog.c --to-c obfuscated.c` takes a
program on the full round trip: C → Sorted! → C. The result prints what
`prog.c` prints, and nothing in it resembles the original. Every sum,
difference, product, ratio and condition becomes a function named after where
it sits in the tables, `S41`, `D7`, `C12`, every statement a case of one big
switch, every variable a cell of `_[193719]`, and the number words are ported
along in case the program prints cardinals. The tests take every C program
they compile through the whole loop and compare its output with the original
C, so this is obfuscation with a warranty. `--to-c` works for any
Sorted! program, quirks included: past-the-end references, jumps to labels
that were never placed, 32-bit wrap-around and the run-time errors all come
out the way the interpreter has them.

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
and parser, `italian.go`, `french.go`, `portuguese.go` and `japanese.go` for
Very Very Sorted!), `internal/numbers` (number
words), `internal/interp`
(interpreter), `internal/emit` (`/D` dump and C translation), `internal/render`
(programs back to Sorted! text), and the command in `cmd/sorted`. The C
compiler adds `internal/cc` (the chibicc-derived front end) and
`internal/compile` (lowering C to Sorted! tables).

## License

MIT, see [LICENSE](LICENSE).
