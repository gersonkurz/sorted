# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

This repo is a from-scratch **Go reimplementation of Sorted!**, the esoteric language Gerson Kurz (the repo owner) wrote in 2000. The goal is a port that is **faithful to the original, quirks included**: same accepted programs, same output, same failure behavior. When the original does something odd, reproduce it and document it in a test. Don't fix it. Everything needs unit tests.

- `legacy/sorted.linux/` and `legacy/sorted.win32/` hold the original C++ implementation (around 4k lines, pre-standard C++). It is the reference for semantics. The two trees are functionally the same: they differ in line endings, header case, and how `SortedSyntax::Parse` builds its dispatch table. Only Linux has `pmakex.ini` (build flags: `-fno-for-scope -fno-rtti -DTARGET_IS_UNIX`). Only Win32 has the MSVC6 `.dsp`/`.dsw` project files. The bundled `Sorted.exe` (Win32) and `Sorted` (Linux ELF) are 32-bit i386 builds. They don't run on macOS, but Gerson can run them on Windows. There is also an Amiga version, which isn't in the repo.
- `legacy/*/*.s` are sample programs (`hello.s`, `hallo.s` in German, `fibo.s`, `itoa.s`). They are the end-to-end golden tests (`cmd/sorted` `TestGolden`): output and `/D` of each are compared with captures from the original binary in `testdata/golden/` (whose `README.md` lists each capture's origin), and the `--to-c` translation, compiled, must print the captured output too. Expected output must come from actually running the original binary, so ask Gerson for it. Don't derive it from reading the C++.
- `manual/Sorted! - p-nand-q.com.html` is the archived overview page with the same examples. It is not a full specification, so the legacy source is the authority.

Port CLI: `sorted [--dump FILE] [--to-c FILE] [--lang en|de] [--version] PROGRAM.s` and `sorted --from-c PROGRAM.c [--lang en|de] [--dump FILE] [--to-c FILE]` (`cmd/sorted`); `--from-c` compiles the C subset (`internal/cc` front end, `internal/compile` lowering, `render.Compose`) and prints the Sorted! program, with C errors on stderr as `file:line:col: message`; `--lang` prints the program in that language instead of running it (`internal/render`, which checks every rendered text by parsing it back); `--dump` is the legacy `/D`, rendered by `internal/emit` byte for byte (LF where the original writes CRLF); `--to-c` (`emit.Exact`) writes C that behaves exactly like `internal/interp` (same stdout, the same runtime errors as `runtime error: …` on stderr with exit 1); both are written before the program runs. The original's `/C` translation is not ported (Gerson's decision, 2026-10-06): it ignores the output format (every write is `putchar`, so fibo prints raw bytes) and writes indirect cells 1-based (itoa's NUL moves to the end), so it does not do what the program does. Legacy messages go to stdout byte for byte; port-defined run-time errors go to stderr; exit 1 on any failure, 2 on usage errors (the original always exits 0).

Legacy CLI: `Sorted /S<source> [/D<dumpfile>] [/C<c-output>]`. It interprets the program, can optionally dump the parsed tables, and can optionally emit an equivalent C program. Flags start with `-` or `/` and are case-insensitive. The value follows the flag letter with no space. Every exit code is 0, including parse failures, which print `<file> is not intelligible.`

## Commands

Everything goes through the `justfile` (`just` with no arguments lists the recipes). It follows the maintainer's cross-platform style: Go commands sit under "Shared", and the few recipes that need a shell have native `[unix]` (sh) and `[windows]` (cmd) variants. **Keep double quotes out of Windows recipe lines.** They don't reliably survive the hand-off from just to cmd.exe, so linker flags use `-ldflags=-X=main.version=…`. The main package is `./cmd/sorted`; staticcheck is pinned as a `tool` in `go.mod`.

```sh
just build                 # → out/build/sorted (sorted.exe on Windows)
just run legacy/sorted.win32/hello.s   # build, then run with args passed through
just test                  # go test -count 1 ./...
just test-one 'TestA|TestB'   # go test -count 1 -v -run <regex> (the regex reaches go test via the environment, never the shell)
just verify                # fmt-check + go vet + test: the review loop's Verify step
just lint                  # fmt-check + go vet + staticcheck
just check                 # lint + test; run before committing
just coverage              # out/coverage.out + out/coverage.html
just examples              # regenerate examples/*.s (English, German) from examples/*.c
just fmt
just test-race             # unix only (the race detector needs cgo)
just package               # unix only: cross-compile all platforms → out/dist/*.tar.gz / *.zip + SHA256SUMS
just clean
```

Release archives contain the binary, `README.md`, `LICENSE`, and under `examples/` the legacy sample `.s` programs plus the repo's `examples/` (C programs and the Sorted! versions `--from-c` writes for them, `<name>.s` English and `<name>.de.s` German). `cmd/sorted` `TestExamples` fails when a committed `.s` is not what the compiler writes now (run `just examples`), and checks that it prints what the C program prints natively.

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

### Legacy quirks

The rule is to reproduce, not fix. These are the known ones:

- Indirect **reads** use `Data[value-1]` (1-based), but indirect **writes** (`GetDataPointer`) use `Data[value]` (0-based).
- Values are C `long`, which is 32-bit on Win32. Arithmetic wraps at int32. Division truncates toward zero. Expressions are re-evaluated on every reference; a reference past the end of its table reads the next slot of the static `Code` array (another table's entry or zeros).
- Number words keep the original's jokes and misspellings: "fiveteen", "fourty", "nineth", "twelveth" ("fifteen" and "twelfth" are *not* recognised), German "einstausend", "millionenste". Zero prints as an empty line. These live in `internal/numbers`, ported line by line.
- The parser shares one cursor across alternatives without restoring it, so some forms can never parse as written: "as a english ordinal", "as a german ordinal", and the German "ein englischer/deutscher Kardinal"/"ein englische Ordinalzahl" outputs ("ein Zeichen" and "eine deutsche Ordinalzahl" work). Each failed alternative eats one word, so repeating it gets through: "as a english english ordinal", "as a german german ordinal", "als ein ein englischer Kardinal", "als ein ein ein englische Ordinalzahl", "als ein ein ein ein deutscher Kardinal" (found by Codex). Outputs and inputs have no period check in their "single" form, so a program can declare only one of each; German ordered differences cannot be a list.
- A "single" jump, condition, assignment or statement followed by a comma is withdrawn (`Count--`) without undoing `TypeCount`, so the /D dump's `ELEMENTS` exceeds the real total (hallo: 26 for 23).
- References have no "logical operation" or "input" type, so logical operations can be declared but never used, and a program can never read input ("This code cannot read." is the only useful choice). If it could, `getchar` would store into the read entry's unfilled second operand, i.e. the first cell.
- itoa.s prints a NUL byte before its digits: the 0-based indirect write and the 1-based indirect read are one cell apart.
- **Undefined behaviour in the C code is not emulated** (Gerson's ruling: Sorted! has features, not bugs, and UB is neither). It gets a simple, documented, test-pinned result: a negative number printed as a cardinal is a crash (`numbers.ErrCrash`), and reads before a static buffer find NUL. Don't reverse-engineer the binary or ask for captures to pin UB.
- Win32's `isWhitespace()` has no NUL check, so `skipWhitespaces()` at the end of the input reads past the terminator (UB, see above). Linux has the check, and the port treats end of input as end of input.

## Next phase: C to Sorted! (decided with Gerson, 2026-10-05)

The faithful port is complete. The next phase compiles a growing subset of C into singable Sorted!, in the same binary (milestones M0–M5, issues #12–#17):

- `sorted --from-c prog.c [--lang en|de]` prints the Sorted! program on stdout; `--lang` also re-renders an existing `.s` file (translation English ↔ German). `--to-c` is the mirror of `--from-c`.
- No library calls and no `malloc`: one block of memory. `putchar` is the only intrinsic. The subset starts small and grows.
- The C front end is a Go port of chibicc's tokenizer, parser and type pass (MIT; keep its notice), slimmed to the subset.
- Output must be singable (that is the joke), in the style of the samples; German falls back to English per sentence where it has no form (ratios, lists of ordered differences).
- The parser stays faithful to 2000 (Gerson, 2026-10-05): generated programs use only forms the original accepts, so they also run on `Sorted.exe`. The renderer works around the parser's gaps ("the eight number" for "eighth", German "-n" endings only where the parser reads them back: "ersten" and "einhundertersten" do, "zwanzigsten" does not) instead of the grammar being extended.
- Generated programs never index a computed value ("the cell indexed by the first sum" is UB in the original: `Type[ps->Type-1]` with the unmasked type). Array elements are reached through pointer cells instead, a read pointer holding the cell number plus one and a write pointer the cell number (the indirect off-by-one); address constants are declared numbers, sized in one pass with fillers (`internal/compile` `program`).
- Pointers (#21): the front end has a type pass (`internal/cc/typecheck.go`, after chibicc's `add_type`), stricter than C compilers that only warn: pointers and integers do not mix except for the null constant 0, and pointer types must match. A pointer is a cell number (the write pointer), and pointer arithmetic counts cells (scaled by the size of what is pointed to, see Structs); `*p` reads through `p + 1`. `p[i]` on a pointer is `*(p + i)`; on an array variable it stays `NdIndex`. String literals are anonymous global char arrays (`.strN`, equal ones shared). Global initializers can be addresses of globals (`Obj.InitRef`). A function on a cycle of calls may not take the address of its own locals: the frame save/restore around a recursive call would undo writes through such a pointer.
- Structs (#22): members laid out one after another, each its size in cells (`Type.Size`: a scalar 1, an array `Len × element`, a struct the sum); a member is an offset, `p->m` is `(*p).m`, and pointer arithmetic, indices and pointer differences scale by the element size. Assigning a struct copies it cell by cell (`copyStruct`). Struct parameters and results by value, and `?:` between structs, are rejected (pass a pointer). Initializers are nested `{...}` lists with brace elision and zero-fill (`initValue`); global initializers are flattened to cells.
- Arrays of arrays and unsigned (#24): `int m[3][4]` is an array of arrays (sizes multiply), a parameter `int m[][4]` a pointer to arrays, and any array-typed lvalue decays (`arrayType`), so `m[i][j]` is `*(m[i] + j)`, a fixed cell when both indices are constant (`direct`, inner index bounds-checked). `unsigned` is the same 32-bit cell, typed by C's usual arithmetic conversions (`arith`, `promoted`) and folded unsigned (`Fold`); comparisons flip the sign bits, `/` `%` and variable `>>` call `udiv`/`umod`/`ushr` (written in the C subset, with the bitwise pass also run over the library itself), a constant `>>` masks, an `unsigned char` store wraps to 0..255 (`wrapTo`). Hexadecimal/octal literals above `INT_MAX` and the `u` suffix are `unsigned int`.
- Functions exist once (no call stack): a call stores the arguments in the parameter cells and its call-site number in a return-address cell, jumps to the function, and the return goes through a dispatch chain of conditional jumps back to the call site (a function with one call site returns straight there). Parameters and locals are plain cells. A function on a cycle of calls (recursion) saves its frame (return address, parameters, locals, preallocated) and the temporaries of the current expression on a stack before each call that can come back to it, and restores them after copying the result; the stack grows from the first free cell after the variables, through a stack pointer cell (writes through it, reads through a copy, the indirect off-by-one), and a recursion too deep for the remaining memory is the interpreter's "cell out of range".
- Bitwise operators are arithmetic (logical operations cannot be referenced): constant folding, `~x` = `-1 - x`, constant shifts as products or floored ratios, `x & (2^k-1)` as a remainder; the rest call runtime functions written in the C subset (`internal/compile/runtime.go`), compiled like user functions and emitted only when reached. Constant shift counts outside 0..31 are compile errors (UB in C); variable ones get whatever the loop gives. Numbers above 999999999 are `1000000 * q + r`.
- The preprocessor (`internal/cc/preprocess.go`) is minimal: `#define` (object-like and function-like, rescanning with Prosser's hide sets as in chibicc, an argument expanded only where it is substituted; a `(` with whitespace or a comment before it makes an object-like macro), `#undef`, the null directive, and `#include <stdio.h>`; `#`/`##`, variadic macros, `#if` and other includes are rejected. Array lengths and global initializers are constant expressions (`cc.Fold`, which also folds constants under bitwise operators).
- Every generated program must parse back into the same tables (round trip) and print what the C program prints when compiled natively (differential tests). The oracle is clang (`cc`) on macOS; Gerson uses only MSVC on Windows, where the tests skip, and declined setting up a C compiler there (2026-10-06).
- C → Sorted! → C is free obfuscation: `--from-c` with `--to-c` (`emit.Exact`, M5; it replaced the port of the original's `/C`, which did not preserve semantics). It resolves the tables while writing: one function per (operand type, code slot) an evaluation can reach (so past-the-end references read the same foreign slots), the statements as a switch inside the program-counter loop (with the slots before the statements when a jump target is not a label), wrapping int32 helpers, the 65536 evaluation-depth limit, the Win32 text-mode input, and the number words ported to C. Its tests compare it with the interpreter on the samples and on hand-built tables for every quirk and runtime error, and the differential tests take every C program through C → Sorted! → C.

## Porting decisions (made by Gerson)

- **Reference platform: Win32.** Where `sorted.win32/` and `sorted.linux/` differ, follow Win32. Golden output is captured from `Sorted.exe`, which Gerson runs on Windows on ARM. Only Gerson can add new captures, so ask for them. Golden comparisons normalise `\r\n` to `\n` and ignore a final newline (Gerson: "that final newline is negligible"), so console captures are good enough.
- **No OPP.** None of the sample programs use a `##` directive, so the preprocessor is not ported and no `<file>,opp` side file is written. One OPP behaviour stays visible: for a missing source file, OPP prints `*** ERROR, unable to open file <name> for reading` (no newline) before `<name> is not intelligible.`, and the port reproduces that.
- **CLI may be modernized.** The flag syntax doesn't have to be `/S<file>`. Program output and diagnostics stay byte-identical to the original.
- Module path: `github.com/gersonkurz/sorted`.

## Workflow

- Tasks are GitHub issues on `gersonkurz/sorted`. When work on an issue starts, assign it to `gersonkurz`. Post each Codex review verdict as an issue comment. The approved commit closes the issue with `Fixes #n`.
- Commit directly to `main` and push after every approved commit. No branches, no PRs.
- Until unit tests exist, the Verify step may be skipped (Gerson's explicit permission). From the first test onward it is mandatory.

## Review loop

One import line per machine — Claude Code skips an import whose path does not
exist and loads the others, so both can stand:

@C:/Projects/yaaadabi/protocol.md
@~/development/yaaadabi/protocol.md

Loop parameters:
- Verify: `just verify` (gofmt check, `go vet ./...`, `go test -count 1 ./...`; works on macOS, Linux and Windows)
- Yardstick docs: CLAUDE.md (faithfulness requirement, porting decisions, legacy pipeline, quirks); `legacy/sorted.win32/` as the semantic authority (`legacy/sorted.linux/` where identical); golden outputs captured from the original `Sorted.exe` under `testdata/`
- Review focus: fidelity to the legacy Win32 implementation: the same programs accepted and rejected, byte-identical stdout including error messages and newlines, every quirk preserved and pinned by a test rather than silently fixed; golden expectations must come from captures, never be derived from the C++; cross-platform (Windows + macOS) for the tool itself. The C-to-Sorted! differential tests use clang on macOS as the oracle and may skip where no `cc` exists; their behaviour on Windows is not a review criterion (Gerson, 2026-10-06)
- Task list: GitHub issues on gersonkurz/sorted, one issue per [task] finding, label `review-task`
