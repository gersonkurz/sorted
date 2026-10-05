# Golden captures

Everything here except this README and `duplicate.s` was produced by the
original Win32 `Sorted.exe` (`legacy/sorted.win32/Sorted.exe`), run by Gerson
on Windows. The files keep their bytes exactly as captured, CRLF line ends
included; `.gitattributes` marks them `-text` so git never converts them.

Tests compare after turning CRLF into LF and, for program output, ignoring a
final newline (Gerson: "that final newline is negligible"). Expected output is
never derived from reading the C++ source.

| File | Produced by | Captured | Checked by |
|---|---|---|---|
| `hello.out`, `hallo.out` | `Sorted.exe /S<name>.s`, console output pasted | 2026-10-04 | `cmd/sorted` TestGolden, `internal/interp` |
| `fibo.out` | `Sorted.exe /Sfibo.s`, console output pasted, stored with CRLF | 2026-10-04 | as above, `internal/numbers` |
| `itoa.out` | `Sorted.exe /Sitoa.s > itoa.out` (redirected: the output starts with a NUL byte a console cannot show) | 2026-10-04 | as above |
| `hello.c`, `hallo.c`, `fibo.c`, `itoa.c` | `Sorted.exe /S<name>.s /C<name>-out.c` | 2026-10-04 | `cmd/sorted` TestGolden, `internal/emit`, `internal/numbers` (data initialisers) |
| `hallo.dump` | `Sorted.exe /Shallo.s /Deins` | 2026-10-04 | `cmd/sorted` TestGolden, `internal/emit`, `internal/numbers` (ordinals) |
| `hello.dump`, `fibo.dump`, `itoa.dump` | `Sorted.exe /S<name>.s /D<name>.dump > nul` | 2026-10-05 | `cmd/sorted` TestGolden, `internal/emit` |
| `missing.out` | `Sorted.exe /Snothere.s > missing.out`, run in this directory | 2026-10-05 | `cmd/sorted` TestMissingFile |
| `duplicate.out` | `Sorted.exe /Sduplicate.s > duplicate.out`, run in this directory | 2026-10-05 | `cmd/sorted` TestParseFailure |

`duplicate.s` is the input for `duplicate.out`: a program that declares the
number seven twice. The `.c` and `.dump` files were renamed after capture
(`<name>-out.c` → `<name>.c`, `eins` → `hallo.dump`).

Running `Sorted.exe` leaves a `<name>,opp` file next to each source (its
preprocessor's output); those are git-ignored.

To add a capture, run `Sorted.exe` from `legacy/sorted.win32` (or here, with
`..\..\legacy\sorted.win32\Sorted.exe`), redirect into this directory, and
add a row above.
