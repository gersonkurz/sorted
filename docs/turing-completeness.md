# Sorted! is Turing complete

*For the esolangs wiki. Every claim on this page is checked by a test in
this repository.*

**Sorted! is Turing complete up to its memory**, as any implemented
language is. With unbounded memory it would be Turing complete without
qualification (see the fine print for exactly what "unbounded" has to
cover). This was already true of the Sorted! of 2000. The proof is
a reduction from Brainfuck, and it is mechanical: a translation table and
the C-to-Sorted! compiler, which is tested against a native C compiler.

## The reduction

Brainfuck is Turing complete, given an unbounded tape. Each Brainfuck
command becomes one C statement (`t` is the tape of bytes, `p` the head):

| Brainfuck | C |
|---|---|
| `+` | `t[p]++;` |
| `-` | `t[p]--;` |
| `>` | `p++;` |
| `<` | `p--;` |
| `.` | `putchar(t[p]);` |
| `,` | `c = getchar(); t[p] = c == -1 ? 0 : c;` |
| `[` | `while (t[p]) {` |
| `]` | `}` |

The tape is `unsigned char t[30000]`, so a cell wraps around from 255 to 0.
`sorted --from-c` compiles this C to Sorted!, so every Brainfuck program
becomes a Sorted! program that prints the same thing. Two qualifications
apply. The program must stay on its tape of 30000 cells. Its input is read
as Sorted! reads all input, the way the Win32 C runtime's text mode does:
a CR LF pair arrives as LF, and a Ctrl-Z ends the input (`TestBrainfuckInput`
pins both).

The compiler is the only step that needs trust, and the tests check it the
same way for every program they compile: the Sorted! program and the C
program it came from, compiled natively, must print the same
(`internal/compile` `TestBrainfuck` does this for Brainfuck programs, among
them hello world, a squares printer, a reverser that reads its input, cell
wrap-around, and the end of the input).

What the C becomes in Sorted! terms:
- The tape is a run of memory cells.
- `t[p]` is "the cell indexed by" a pointer cell. Sorted! reads an indexed
  cell one before the one it writes, so a write goes through a cell
  holding the element's number and a read through one holding that number
  plus one. The papal bull *[Ordinata non errant](ordinata-non-errant.md)*
  explains why that is a doctrine and not a bug.
- `while` is a conditional jump to a label after the loop, and an
  unconditional jump back.
- The byte arithmetic is sums, differences and remainders.

All of these exist in the original grammar of 2000. **A Brainfuck program
without `,` therefore compiles to Sorted! as `Sorted.exe` accepted it in
2000** (the test checks that the output carries no Very Sorted! marker). The
language was Turing complete from the start; nobody had written the
compiler yet.

Input is the exception. The 2000 grammar has no way to name an input in a
statement, so `,` needs the Very Sorted! dialect ("This code is very
cool."), whose programs may implement "the first input".

## The interpreter

The reduction makes every Brainfuck program a Sorted! program. The other
direction runs a Brainfuck interpreter in Sorted!.
[`examples/brainfuck.c`](../examples/brainfuck.c) is one, written in the C
subset. Its Sorted! translations are
[`examples/brainfuck.s`](../examples/brainfuck.s) and
[`examples/brainfuck.de.s`](../examples/brainfuck.de.s), about 600 singable
lines each.

It reads a Brainfuck program from its input up to a `!`. It matches the
brackets, then runs the program, which reads whatever follows the `!`:

```sh
sorted examples/brainfuck.s < examples/brainfuck.in      # Hello World!
printf ',[.,]!Ordinata non errant' | sorted examples/brainfuck.s
```

Since it reads, it is a Very Sorted! program. The squares printer (every
square up to 10000) takes about half a second in it.

## The fine print

- **Memory.** A Sorted! program has 193719 cells (`MAX_DATA_PER_PROGRAM` in
  the original, a statically allocated array). The interpreter uses 30000
  of them for its tape, as Urban Müller's Brainfuck did. That bound is the
  "up to its memory". More cells alone would not lift it, though. A cell
  number is itself a value in a cell, and values are 32-bit, so the
  idealisation that makes Sorted! Turing complete without qualification
  removes both bounds: unboundedly many cells, holding unbounded integers
  (the head of the tape, and the addresses computed from it, then never
  wrap). No construct of the language depends on either bound; the tests
  demonstrate the bounded implementation, and the argument is about the
  idealised one, as for any language that runs on a real machine.
- **Cells.** Sorted! cells are 32-bit integers that wrap around. The
  byte-sized Brainfuck cells are built from them with remainders.
- **End of input.** `,` stores 0 at the end of the input, one of
  Brainfuck's customary conventions, so that `,[.,]` terminates.
- **Input bytes.** Input is read in the Win32 text mode of the original
  (CR LF as LF, Ctrl-Z as the end), so a Brainfuck program sees what it
  would see under a Win32 C runtime. That holds both translated and
  interpreted.
- **Errors.** The interpreter refuses a program longer than 5000 commands
  and unmatched brackets before it runs, and stops with a message when the
  head leaves the tape (`TestBrainfuckErrors`).
