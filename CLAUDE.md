# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

This repo is meant to hold a modern port of **Sorted!**, an esoteric language from 2000. No port code exists yet. The `.gitignore` uses the Go template, which suggests Go, but nothing confirms that yet. There are no build, lint, or test commands so far. Add them here once the port has a toolchain.

- `legacy/sorted.linux/` and `legacy/sorted.win32/` hold the original C++ implementation (around 4k lines, pre-standard C++). Treat it as the reference for semantics. The two trees are functionally the same: they differ in line endings, header case, and how `SortedSyntax::Parse` builds its dispatch table. Only Linux has `pmakex.ini` (build flags: `-fno-for-scope -fno-rtti -DTARGET_IS_UNIX`). Only Win32 has the MSVC6 `.dsp`/`.dsw` project files. The bundled `Sorted`/`Sorted.exe` binaries are 32-bit i386 builds and won't run on modern macOS.
- `legacy/*/*.s` are sample programs (`hello.s`, `hallo.s` in German, `fibo.s`, `itoa.s`). Use them as golden tests for the port.
- `manual/Sorted! - p-nand-q.com.html` is the archived overview page with the same examples. It is not a full specification, so the legacy source is the authority.

Legacy CLI: `Sorted /S<source> [/D<dumpfile>] [/C<c-output>]`. It interprets the program, can optionally dump the parsed tables, and can optionally emit an equivalent C program.

## How the legacy implementation works

Pipeline (`Sorted.cpp`): **OPP preprocessor → character filter → recursive-descent parser → table-driven interpreter**.

1. **OPP** (`OPP.cpp`) is a small C-preprocessor-like macro/include/conditional pass. It writes `<file>,opp` (the last `.` of `<file>.opp` is replaced by `,`), and that output is what gets parsed.
2. **Character filter**: every char that isn't `[A-Za-z.,]` becomes a space. Keywords match case-insensitively on whole words (`IS_KEYWORD`).
3. **Parser** (`SortedSyntax.cpp`): a program is exactly **14 sentences in a fixed order**. They are not actually "randomly sorted" despite what the manual says. Order: numbers, jumps, output, input, sums, conditions, labels, ordered differences, assignments, products, implementation, ratios, logical operations (NAND), "Cool." Each sentence has an English form ("This code …") and a German form ("Dieses Programm …"), and the two can be mixed per sentence. Each also has a "does not use any …" or "keine …" form. Cardinals parse through `EnglishNumbers.cpp` first, then `GermanNumbers.cpp`. These modules also print numbers as cardinals or ordinals.
4. **Data model** (`SortedSyntax.h`): everything lands in one `CODEINFO`:
   - `Data[]`: memory cells. The first N cells are pre-loaded with the declared numbers, in order. **A duplicate number is a parse error** ("cardinals only once"). "The k-th number" means cell `k-1`, even past the declared count.
   - `Code[]`: `SLIDE`s, each a pair of operands (`SLID {Type, Index}`) plus `Flags`. Every category (sums, diffs, products, ratios, nands, assigns, writes, reads, conditions, statements, jumps) is a contiguous block, located by `Type[SLIDE_INFO_*].{Index,Count}`.
   - An operand can reference a number/cell, or the k-th sum/diff/product/ratio/nand/condition/label. OR-ing in `SLID_TYPE_INDIRECT` (`0xF00000`) gives "the cell indexed by …".
   - Expressions are lazy. A sum or condition is a reusable *definition* that gets evaluated each time something references it (`GetDataValue`, recursive).
5. **Interpreter** (`SortedInterpreter::Interpret`): the "implementation" sentence lists statements (assignments, outputs, inputs, jumps, labels) by ordinal reference, and the same one can repeat. A label resolves to its statement index. A jump sets the PC to that index and the loop's `++` then continues *after* the label. "Assigns A to B" stores `_[0]=A` (source) and `_[1]=B` (target). Output formats: character, English or German cardinal, English or German ordinal. Cardinal and ordinal output appends a newline.

### Legacy quirks to decide on deliberately when porting

- Indirect **reads** use `Data[value-1]` (1-based), but indirect **writes** (`GetDataPointer`) use `Data[value]` (0-based). The sample programs may depend on this asymmetry, so check against `hello.s`/`itoa.s` before "fixing" it.
- Only plain or indirect number cells can be assignment/read targets. Any other target yields a null pointer in the original.
- Values are C `long`. Division truncates, and division by zero is undefined.
