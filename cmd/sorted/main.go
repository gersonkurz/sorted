// Command sorted runs programs written in Sorted!, the esoteric language from
// 2000.
//
//	sorted [--dump FILE] [--to-c FILE] [--lang en|de] [--version] PROGRAM.s
//
// --dump writes the parsed tables and --to-c a translation into C, as the
// original's /D and /C do, before the program runs. --lang prints the program
// in English or German instead of running it.
//
// The flags are modern; what a program prints, and the diagnostics of the
// original (on stdout, byte for byte), are those of the Win32 Sorted.exe.
// Unlike the original, which always exits with 0, sorted exits with 1 when the
// program cannot be read or parsed, a requested file cannot be written (the
// original silently skips it), or the program fails at run time, and with 2
// on a usage error. Run-time errors have no counterpart in the original (it
// crashes), so they go to stderr.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/gersonkurz/sorted/internal/emit"
	"github.com/gersonkurz/sorted/internal/interp"
	"github.com/gersonkurz/sorted/internal/render"
	"github.com/gersonkurz/sorted/internal/syntax"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes the command line and returns the process exit code.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sorted", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print the version and exit")
	dumpFile := fs.String("dump", "", "write the parsed tables to `FILE` (legacy /D)")
	cFile := fs.String("to-c", "", "write a translation into C to `FILE` (legacy /C)")
	lang := fs.String("lang", "", "print the program in `LANG` (en or de) instead of running it")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: sorted [--dump FILE] [--to-c FILE] [--lang en|de] [--version] PROGRAM.s")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, "sorted", version)
		return 0
	}
	langs := map[string]render.Lang{"en": render.English, "de": render.German}
	if _, ok := langs[*lang]; fs.NArg() != 1 || *lang != "" && !ok {
		fs.Usage()
		return 2
	}
	name := fs.Arg(0)

	raw, err := os.ReadFile(name)
	if err != nil {
		// The original's preprocessor reports the file without a newline,
		// so both messages end up on one line.
		fmt.Fprintf(stdout, "*** ERROR, unable to open file %s for reading", name)
		fmt.Fprintf(stdout, "%s is not intelligible.\n", name)
		return 1
	}
	p, err := syntax.Parse(syntax.Filter(raw))
	if err != nil {
		fmt.Fprintf(stdout, "%v\n", err)
		fmt.Fprintf(stdout, "%s is not intelligible.\n", name)
		return 1
	}
	// Like the original: the C translation first, then the dump, then run.
	code := 0
	for _, out := range []struct {
		file   string
		render func(*syntax.Program) string
	}{{*cFile, emit.C}, {*dumpFile, emit.Dump}} {
		if out.file == "" {
			continue
		}
		if err := os.WriteFile(out.file, []byte(out.render(p)), 0o644); err != nil {
			fmt.Fprintf(stderr, "sorted: %v\n", err)
			code = 1
		}
	}
	if *lang != "" {
		text, err := render.Render(p, langs[*lang])
		if err != nil {
			fmt.Fprintf(stderr, "sorted: %v\n", err)
			return 1
		}
		fmt.Fprint(stdout, text)
		return code
	}
	if err := interp.Run(p, stdin, stdout, 0); err != nil {
		fmt.Fprintf(stderr, "sorted: %v\n", err)
		return 1
	}
	return code
}
