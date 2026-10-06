// Command sorted runs programs written in Sorted!, the esoteric language from
// 2000.
//
//	sorted [--dump FILE] [--to-c FILE] [--lang en|de] [--version] PROGRAM.s
//	sorted --from-c PROGRAM.c [--lang en|de] [--dump FILE] [--to-c FILE]
//
// --dump writes the parsed tables and --to-c a translation into C, as the
// original's /D and /C do, before the program runs. --lang prints the program
// in English or German instead of running it. --from-c compiles a C program
// into Sorted! and prints it (in English unless --lang says otherwise).
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

	"github.com/gersonkurz/sorted/internal/cc"
	"github.com/gersonkurz/sorted/internal/compile"
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
	fromC := fs.String("from-c", "", "compile the C program `FILE` into Sorted! and print it")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: sorted [--dump FILE] [--to-c FILE] [--lang en|de] [--version] PROGRAM.s")
		fmt.Fprintln(stderr, "       sorted --from-c PROGRAM.c [--lang en|de] [--dump FILE] [--to-c FILE]")
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
	positional := 1 // the Sorted! program, unless compiling from C
	if *fromC != "" {
		positional = 0
	}
	if _, ok := langs[*lang]; fs.NArg() != positional || *lang != "" && !ok {
		fs.Usage()
		return 2
	}
	if *fromC != "" {
		return translate(*fromC, langs[*lang], *dumpFile, *cFile, stdout, stderr)
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
	code := writeFiles(p, *cFile, *dumpFile, stderr)
	if *lang != "" {
		text, err := render.Render(p, langs[*lang])
		if err != nil {
			fmt.Fprintf(stderr, "sorted: %v\n", err)
			return 1
		}
		if _, err := fmt.Fprint(stdout, text); err != nil {
			fmt.Fprintf(stderr, "sorted: %v\n", err)
			return 1
		}
		return code
	}
	if err := interp.Run(p, stdin, stdout, 0); err != nil {
		fmt.Fprintf(stderr, "sorted: %v\n", err)
		return 1
	}
	return code
}

// writeFiles writes the requested legacy outputs, the C translation before
// the dump as the original does, and returns 1 if one cannot be written.
func writeFiles(p *syntax.Program, cFile, dumpFile string, stderr io.Writer) int {
	code := 0
	for _, out := range []struct {
		file   string
		render func(*syntax.Program) string
	}{{cFile, emit.C}, {dumpFile, emit.Dump}} {
		if out.file == "" {
			continue
		}
		if err := os.WriteFile(out.file, []byte(out.render(p)), 0o644); err != nil {
			fmt.Fprintf(stderr, "sorted: %v\n", err)
			code = 1
		}
	}
	return code
}

// translate compiles a C program into Sorted! and prints it. --dump and
// --to-c describe the compiled program as any Sorted! interpreter sees it.
func translate(name string, lang render.Lang, dumpFile, cFile string, stdout, stderr io.Writer) int {
	src, err := os.ReadFile(name)
	if err != nil {
		fmt.Fprintf(stderr, "sorted: %v\n", err)
		return 1
	}
	prog, err := cc.Parse(string(src))
	if err != nil {
		fmt.Fprintf(stderr, "sorted: %s:%v\n", name, err)
		return 1
	}
	compiled, err := compile.Compile(prog)
	if err != nil {
		fmt.Fprintf(stderr, "sorted: %s:%v\n", name, err)
		return 1
	}
	text, p, err := render.Compose(compiled, lang)
	if err != nil {
		fmt.Fprintf(stderr, "sorted: %s: %v\n", name, err)
		return 1
	}
	code := writeFiles(p, cFile, dumpFile, stderr)
	if _, err := fmt.Fprint(stdout, text); err != nil {
		fmt.Fprintf(stderr, "sorted: %v\n", err)
		return 1
	}
	return code
}
