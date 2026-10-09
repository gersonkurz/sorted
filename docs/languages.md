# The language

*The reference: every sentence, every dialect, every tongue. The doctrine
behind them is the papal bull* Ordinata non errant
*([English](ordinata-non-errant.md), [Deutsch](ordinata-non-errant.de.md)).*

## The fourteen sentences

A program is fourteen sentences, always in this order. Each sentence can be
written in any of the languages, and a program may switch language from one
sentence to the next. The one gap is the German of 2000, which has no ratios,
no lists of ordered differences and no logical operations: a 2000 program
says those in English, and Very Sorted! has German for all three.

| # | Sentence | English | German | Italian (Very Very Sorted!) | French (Very Very Sorted!) | Portuguese (Very Very Sorted!) | Japanese (Very Very Sorted!) | Mandarin (Very Very Sorted!) |
|---|---|---|---|---|---|---|---|---|
| 1 | numbers | This code uses the numbers … | Dieses Programm benutzt die Zahlen … | Questo programma usa i numeri … | Ce programme utilise les nombres … | Este programa usa os números … | Kono puroguramu wa kazu … o tsukaimasu | 这个程序使用数字…… |
| 2 | jumps | This code always goes to … / sometimes goes to … if … is true | Dieses Programm springt immer an … | Questo programma va sempre alla … / va talvolta alla … se … è vera | Ce programme va toujours à la … / va parfois à la … si … est vraie | Este programa sempre vai para o … / às vezes vai para o … se … for verdadeira | Kono puroguramu wa itsumo … ni tobi, … ga shin nara … ni tobimasu | 这个程序总是跳到……，并在……为真时跳到…… |
| 3 | output | This code writes … as a character | Dieses Programm schreibt … als ein Zeichen | Questo programma scrive … come carattere | Ce programme écrit … comme caractère | Este programa escreve … como caractere | Kono puroguramu wa … o moji to shite kakimasu | 这个程序把……作为字符写出 |
| 4 | input | This code cannot read. | Dieses Programm kann nicht lesen. | Questo programma non può leggere. | Ce programme ne peut pas lire. | Este programa não pode ler. | Kono puroguramu wa yomemasen. | 这个程序不能读。 |
| 5 | sums | This code uses the sum of … and … | Dieses Programm benutzt die Summe aus … und … | Questo programma usa la somma del … e del … | Ce programme utilise la somme du … et du … | Este programa usa a soma do … e do … | Kono puroguramu wa … to … no wa o tsukaimasu | 这个程序使用……和……的和 |
| 6 | conditions | This code uses the condition that … is equal to … | … die Bedingung dass … ist gleich … | … la condizione che … sia uguale al … | … la condition que … soit égal au … | … a condição de que … seja igual ao … | … ga … to hitoshii to iu jōken … | ……等于……的条件 |
| 7 | labels | This code uses two labels. | Dieses Programm benutzt zwei Sprungziele. | Questo programma usa due etichette. | Ce programme utilise deux étiquettes. | Este programa usa dois rótulos. | Kono puroguramu wa raberu o niko tsukaimasu. | 这个程序使用两个标签。 |
| 8 | ordered differences | This code uses the ordered difference between … and … | … die geordnete Differenz zwischen … und … | … la differenza ordinata tra … e … | … la différence ordonnée entre … et … | … a diferença ordenada entre … e … | … to … no sa … | ……和……的差 |
| 9 | assignments | This code assigns … to … | Dieses Programm weisst zu … an … | Questo programma assegna … al … | Ce programme affecte … au … | Este programa atribui … ao … | Kono puroguramu wa … o … ni dainyū shimasu | 这个程序把……赋给…… |
| 10 | products | This code uses the product of … and … | Dieses Programm benutzt das Produkt von … und … | … il prodotto del … e del … | … le produit du … et du … | … o produto do … e do … | … to … no seki … | ……和……的积 |
| 11 | implementation | This code implements the first assignment, … | Dieses Programm implementiert … | Questo programma implementa il primo assegnamento, … | Ce programme implémente la première affectation, … | Este programa implementa a primeira atribuição, … | Kono puroguramu wa dai-ichi no dainyū, … o jissō shimasu | 这个程序实现第一个赋值、…… |
| 12 | ratios | This code uses the ratio of … to … | Dieses Programm benutzt das Verhältnis von … zu … (Very Sorted!) | … il rapporto tra … e … | … le rapport du … au … | … a razão entre … e … | … to … no hi … | ……和……的比 |
| 13 | logical operations | This code does not use any logical operations. | Dieses Programm ist unlogisch. | Questo programma è illogico. | Ce programme est illogique. | Este programa é ilógico. | Kono puroguramu wa hironriteki desu. | 这个程序不合逻辑。 |
| 14 | coolness | Cool. | Hervorragend. | Questo programma è molto molto figo. | Ce programme est très très chouette. | Este programa é muito muito legal. | Kono puroguramu wa totemo totemo kakkoii desu. | 这个程序非常非常酷。 |

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
- **Output** can be a character, or a number written out as a cardinal or
  ordinal: English or German in every dialect (see below for how to ask for
  an ordinal), any of the other languages in Very Very Sorted!.

## The properties of 2000

Sorted! has no bugs, only features. The port keeps all of them, and the bull
declares them doctrine:

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
  A list of differences, introduced like a single one ("die geordnete
  Differenz zwischen …, und zwischen …"), stops at its first comma, so
  German can declare only one.
- Reading a cell indirectly counts from 1, writing one indirectly counts from
  0. This is why `itoa.s` prints a NUL byte before its digits.
- A jump to a label that was declared but never placed lands just after the
  first statement.

[`CLAUDE.md`](../CLAUDE.md) lists every known property, and the tests pin each one.

## The dialects

There is no version 1.0. The versions are the dialects, counted in verys, and
the last sentence of a program says which one it is written in: "Cool." for
the Sorted! of 2000, "This code is very cool." for Very Sorted!, "This code is
very very cool." for Very Very Sorted!. `sorted --version` says which one this
`sorted` reads, in its own words. Every dialect keeps everything the ones
before it had, and whatever the original accepts or rejects is accepted or
rejected exactly as before: a program is very only if the older grammar fails
and the newer one succeeds.

### Very Sorted!

The Sorted! of 2000 cannot read: a program may declare an
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

German gets what 2000 left out. Very Sorted! German has ratios, "das
Verhältnis von der ersten Zahl zu der zweiten Zahl" and "die Verhältnisse
von … zu …, und von … zu …", and lists of ordered differences, "die
geordneten Differenzen zwischen … und …, und zwischen … und …", so a very
program can be German from start to finish. A 2000 program still says those
sentences in English, to keep running on `Sorted.exe`.

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

### Very Very Sorted!

The second dialect of the very Fibonacci sequence has everything Very
Sorted! has, and five new tongues: Italian, the French of Vaud, Brazilian
Portuguese, Japanese and Mandarin. In a program that ends with "This code is
very very cool." (or "Questo programma è molto molto figo.", "Ce programme est
très très chouette.", and so on), every sentence may be written in any of
them, mixed with English and German as those two always mixed. None of them
inherited a quirk: every form parses as written, and their numbers know zero
and the negative, and print in every language.

#### Italian

Every sentence may be Italian:

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

#### French (Vaud)

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

#### Brazilian Portuguese

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

#### Japanese

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

#### Mandarin

And it speaks Mandarin, in hanzi and in pinyin, which Sorted! writes as two
languages and reads in any mix:

```
这个程序使用数字二十三、一万和一百零一。
这个程序总是跳到第一个标签，并在第一个条件为真时跳到第二个标签。
这个程序把第一个和作为中文序数写出。
这个程序使用第一个数字和第二个数字的和和第三个数字和第一个和的和。
这个程序非常非常酷。

Zhège chéngxù shǐyòng shùzì èrshísān, yīwàn hé yībǎilíngyī.
Zhège chéngxù fēicháng fēicháng kù.
```

Hanzi have no spaces between words, so the parser finds each word by its
characters, and reads 。，、 as a period and commas and traditional
characters (這個程式) as the simplified ones it writes. 和 is both "and" and
the sum, so a list of sums says 和和. Numbers are one word grouped by 万 and
亿, with 零 for a gap (一百零一) and 两 in counts (两个标签); ordinals are 第
and the number (第三个数字). Pinyin is read with or without its tones, which
cannot tell 一 (yī) from 亿 (yì), so "shiyi" is eleven. Chinese numbers print
in hanzi in every language ("as a chinese ordinal", "comme cardinal
chinois"), "第零" and "负七" included.

## Babel mode

Name several languages, `--lang zh,fr` or `--中文 --vaudois`, and each sentence
is written in one of them, in the order named:

- `--mix random` (the default) picks one at random; `--mix random:42` gives the same mix every time
- `--mix alternate` takes them in turn
- `--mix singable` takes whichever says the sentence in the fewest syllables (one per hanzi, otherwise one per run of vowels)

The program is written in the newest dialect any of the languages needs, and
every mixed text must still parse back into the same tables. The bull calls
this [Pentecost](ordinata-non-errant.md#caput-xii-de-pentecoste).

## Naming a language

Sorted! has no favourite language, not even for naming languages. `--lang
NAME`, or just `--NAME`, takes the language's name in any language Sorted!
speaks, ignoring case and accents: `--english`, `--Englisch`, `--anglais`,
`--lang inglês`, `--英語`, `--lang en`. `sorted --help` lists them all, and
the help itself comes in a language picked at random, or in the one you name
(`sorted --help --deutsch`).
