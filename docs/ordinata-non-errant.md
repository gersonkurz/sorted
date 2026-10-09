# ORDINATA NON ERRANT

*Bulla de proprietatibus linguae Sorted!, quae ab imperitis vitia appellantur*

*The bull on the properties of the language Sorted!, which the unlearned
call bugs*

[Deutsch](ordinata-non-errant.de.md)

---

**ORDINATA NON ERRANT.** The sorted do not err. Since the year 2000, when
this language won the esoteric language contest of its day, there have been
those who looked upon its interpreter and spoke of mistakes. Let it be known,
to all who sing their programs and to all who compile them, that what the
interpreter does is the language. It speaks *ex cathedra*. Where it seems to
stumble, it walks a path the faithful have not yet understood. These are its
properties, confirmed in the year 2026 by a port that reproduces every one of
them and pins each in a test, so that none may be lost.

## Caput I. De infallibilitate interpretis

*On the infallibility of the interpreter*

The interpreter of the year 2000, `Sorted.exe` for Win32, defines the
language. Its manual, venerable as it is, merely bears witness: it proclaims
that "each Sorted! program is, in fact, sorted, albeit randomly", whereas the
parser demands exactly fourteen sentences in a single fixed order: numbers,
jumps, output, input, sums, conditions, labels, ordered differences,
assignments, products, implementation, ratios, logical operations, and
"Cool." The program is sorted indeed, and sorted always the same way. The
randomness was a figure of speech.

Where a capture of the original's output exists, the capture is scripture.
Where none exists, its source is consulted, and never against a capture.

## Caput II. De cellis indexatis

*On the indexed cells*

"The cell indexed by" a value reads the cell one before the one it writes:
an indirect read takes `Data[v-1]`, an indirect write `Data[v]`. Let no one
call this an off-by-one. It is a teaching on humility: what you put down, you
do not pick up in the same place.

The sample `itoa.s` bears witness. It prints `41281927` preceded by a NUL,
because it writes its digits through one door and reads them through the
next. The faithful compiler of the year 2026 builds on this teaching: its
read pointers hold a cell's number plus one and its write pointers the number
itself, so that every C program it compiles honours the doctrine.

## Caput III. De numeris

*On the numbers*

The numbers are named as the interpreter names them, and its spelling is
canon: **fiveteen**, **fourty**, **nineth**, **twelveth**. "Fifteen" and
"twelfth" are not numbers in Sorted!, and a program that uses them will not
be understood. In the German tongue, one thousand is **einstausend** and one
million **einsmillionen**, and the millionth is **einsmillionenste**.

Written as a cardinal, zero is an empty line, for nothing is to be said
about it; so is any positive multiple of one thousand million, for the words
run out there, as words do. The ordinal of anything below one is the cry
**ERROR, ORDINALS ARE POSITIVE INTEGERS**, which is the truth.

What the ordinal of such a positive multiple is, the original does not
define: it looks for an ending before the start of its empty word. By the
rule of Caput IX the port gives it a plain result, the bare ending **th** or
**ste**, a suffix bereft of its number. That is the port's ruling, not the
interpreter's voice.

## Caput IV. De cursore communi

*On the shared cursor*

The parser tries its alternatives one after another with a single cursor,
and does not take it back when an alternative fails. Each failure thus
consumes a word. It follows that "as a english ordinal" cannot be read, but
**"as a english english ordinal"** can, for the first "english" is offered
up to the failed alternative. In German the sacrifice is greater: a German
cardinal is written **"als ein ein ein ein deutscher Kardinal"**. Let those
who find this strange consider that every request worth making is worth
repeating.

## Caput V. De sententiis singularibus

*On the single sentences*

An output or an input declared in the singular has no period check, and so
a program may declare only one of each. A single jump, condition, assignment
or statement that is followed by a comma is withdrawn, yet its count stays
counted: the dump of `hallo.s` reports **ELEMENTS=26** for its 23 entries.
Three entries were, and are not, and are remembered.

## Caput VI. De lectione quae non est

*On the reading that is not*

A program may declare that it reads, but no reference can name an input, so
no statement can ever perform one. **"This code cannot read."** is therefore
the only declaration of input that tells the whole truth. Had it been
possible to read, the character would have been stored into the never-filled
second operand of the read entry, which is the first cell. The language
spared its faithful this. The dialect Very Sorted! names "the first input"
and stores what it reads where the program says; for the year 2000, reading
remains what it was: declared, and never done.

## Caput VII. De operatione logica quae non est

*On the logical operation that is not*

Hear now the deepest of the mysteries, unnoticed for twenty-six years. The
language was born at p-nand-q.com and declares "logical operations", which
all took to be NAND. Yet the interpreter computes `~a & ~b`, which is **NOR**.
And behold: the sentence that declares it says so itself, "the logical
operation **of not** X **and not** Y". The program does exactly what it
says. It was the faithful who misread it.

Moreover, no reference can name a logical operation, so none was ever
evaluated. A NOR that could not be observed, called NAND by everyone, in a
language from p-nand-q.com: let this chapter be read aloud whenever someone
claims to understand Sorted! completely. The dialect Very Sorted! names a
true NAND, "of not both X and Y", and lets statements refer to it; this NOR
remains as it is, for the year 2000, and in Very Sorted! too.

## Caput VIII. De translatione reprobata

*On the rejected translation*

The interpreter of the year 2000 could also write its programs as C (`/C`).
That translation, however, did not do what the programs do: it printed every
number as a character, so that `fibo.s` uttered raw bytes in place of "one,
one, two, three", and it wrote indirect cells by the reading rule, moving the
NUL of `itoa.s` from the front to the back. Since the interpreter is
infallible and the translation contradicted it, the translation erred. It was
set aside in the year 2026, and `--to-c` now writes C that behaves exactly as
the interpreter does. The captures of the old translation are kept as relics.

## Caput IX. De rebus non definitis

*On undefined things*

Some of what the C++ of the year 2000 does is undefined behaviour: printing a
negative number as a cardinal indexes its word tables before their start;
"the cell indexed by the first sum" consults a table directory with an
unmasked type; the Win32 parser reads past the end of its input. Let it be
taught clearly: **these are neither properties nor errors**. Undefined
behaviour is not part of the language and is not imitated. Each such case
receives a plain, documented result: a crash for the negative cardinal, the
end of the input for the end of the input. Sorted! has properties, and it has
no bugs, and undefined behaviour is neither.

What was undefined may yet be defined, by a dialect that says so. Very
Sorted! defines "the cell indexed by the first sum" as the doctrine of
Caput II teaches it for every cell: it reads the cell before the one it
writes.

## Caput X. De lingua tertia

*On the third tongue*

The language counts its ages in verys, and in the second, Very Very Sorted!,
it learned a third tongue. Italian was born after the interpreter had
spoken, and so it inherited none of its properties: it has a form for every
sentence, its every form parses as written, and its numbers know zero
("zeresimo") and the negative ("meno sette"). Let no one take this for a
correction of the elder tongues. "fiveteen" remains fifteen, "as a english
english ordinal" remains how one asks for an ordinal, and a program of the
year 2000 that would sing otherwise in Italian is not translated at all.
The sorted do not err: neither the old, which keep their ways, nor the new,
which never strayed. In the same age the language learned a fourth tongue,
the French of Vaud, which counts in tens as the faithful always hoped
French would, and says "eh" and "voilà" where others pause; the parser
hears those as the silence they are. And a fifth, the Portuguese of
Brazil, which joins its numbers with the same "e" that joins its lists;
the parser takes each number as far as it will go, and where two would
become one, the renderer sets a comma between them, that each may stay
itself. And a sixth, the Japanese of the Latin letters, which says what it
does only at the end, as the patient do, and counts in tens of thousands;
in it the sum and the subject are one word, *wa*, and the parser tells
them apart by where they stand. And a seventh, Mandarin, in two scripts,
the characters that need no spaces and the letters that need their tones,
in which "and" and "sum" are one character, 和; the parser takes words
from a line without spaces by knowing them, and the old and the new
characters alike. **Questo programma è molto molto figo. Ce programme est
très très chouette. Este programa é muito muito legal. Kono puroguramu wa
totemo totemo kakkoii desu. 这个程序非常非常酷。**

## Caput XI. De matrimonio cum C

*On the marriage with C*

Sorted! and C have kept company since the year 2000. Their first
engagement was the translation of Caput VIII, which promised the bride a
faithful likeness and gave her another's face; it was set aside, as a
promise not kept must be. In the year 2026 the union was made in earnest,
and in both directions, as a marriage is: C comes to Sorted! (`--from-c`),
and Sorted! goes back to C (`--to-c`), and the one who makes the round trip
returns unchanged in deed, however changed in appearance. This is the vow,
and the tests witness it: every C program that is compiled is sung, run,
translated back, compiled again, and must say exactly what it said before.

The bride brings her law, and the groom keeps it. Every constant of C
becomes a declared number, each declared once, for **thou shalt not have
the same cardinal more than once**. The pointers of C honour the doctrine
of Caput II: a pointer that reads holds the cell's number and one more, a
pointer that writes the number itself, that the cell put down is not
sought in the same place. A function exists once, as all things in Sorted!
exist once, and returns by asking, at every place it was called from,
whether this is the place. What C leaves undefined it may not bring into
the house: a shift by a constant beyond the width of a word is refused at
the door, as Caput IX teaches.

Let none say that C was converted, or that Sorted! became C. Each keeps its
nature. C goes on compiling with its compilers, and Sorted! goes on singing.
What they share is what they do.

## Caput XII. De Pentecoste

*On Pentecost*

At Babel the tongues were confounded, that no one might understand the
other. In Sorted! it is the other way round: a program may be written in all
its tongues at once, each sentence in one of them, and it means one thing in
all of them, as on the day when each heard the apostles in their own language.
The tables are the meaning; the words are its garments. Whatever the mix,
at random, in turn, or by the fewest syllables, the text must read back
into the same tables, or it is not written. Hence the saying of the
faithful, which is no boast: **Sorted! is the only machine translator with
a hundred percent accuracy, as long as you only ever say one of fourteen
things.**

---

*Datum ad p-nand-q.com, anno linguae MM, confirmatum anno MMXXVI.*

*Given at p-nand-q.com in the year of the language 2000, confirmed in 2026.
The tests in this repository are its seals: each property above is pinned
by one, and none may be fixed.*
