// Command sorted runs programs written in Sorted!, the esoteric language from
// 2000.
//
//	sorted [--dump FILE] [--to-c FILE] [--lang en|de | --english | --german] [--version] PROGRAM.s
//	sorted --from-c PROGRAM.c [--lang en|de | --english | --german] [--dump FILE] [--to-c FILE]
//
// Before the program runs, --dump writes the parsed tables, as the
// original's /D does, and --to-c a C program that behaves exactly like it
// (unlike the original's /C, which is not ported), so C -> Sorted! -> C
// (--from-c with --to-c) turns a C program into an equivalent, thoroughly
// obfuscated one. --lang en or --english, and --lang de or --german, print
// the program in English or German instead of running it. --from-c
// compiles a C program into Sorted! and prints it; Sorted! is bilingual and
// does not prefer either language, so unless one is asked for, each run
// picks one at random.
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
	"math/rand/v2"
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

// pickLang chooses the language of a compiled program that nobody chose a
// language for (defaultPick: either, at random). Tests replace it.
var pickLang = defaultPick

func defaultPick() render.Lang {
	if rand.IntN(2) == 0 {
		return render.English
	}
	return render.German
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run executes the command line and returns the process exit code.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sorted", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print the version and exit")
	dumpFile := fs.String("dump", "", "write the parsed tables to `FILE` (legacy /D)")
	cFile := fs.String("to-c", "", "write a C program that behaves like this one to `FILE`")
	lang := fs.String("lang", "", "print the program in `LANG` (en or de) instead of running it")
	english := fs.Bool("english", false, "the same as --lang en")
	german := fs.Bool("german", false, "the same as --lang de")
	fromC := fs.String("from-c", "", "compile the C program `FILE` into Sorted! and print it")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: sorted [--dump FILE] [--to-c FILE] [--lang en|de | --english | --german] [--version] PROGRAM.s")
		fmt.Fprintln(stderr, "       sorted --from-c PROGRAM.c [--lang en|de | --english | --german] [--dump FILE] [--to-c FILE]")
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
	chosen := 0 // --lang, --english, --german: at most one
	for _, set := range []bool{*lang != "", *english, *german} {
		if set {
			chosen++
		}
	}
	switch {
	case *english:
		*lang = "en"
	case *german:
		*lang = "de"
	}
	if _, ok := langs[*lang]; fs.NArg() != positional || *lang != "" && !ok || chosen > 1 {
		fs.Usage()
		return 2
	}
	if *fromC != "" {
		l, ok := langs[*lang]
		if !ok {
			l = pickLang()
		}
		return translate(*fromC, l, outputs{*cFile, *dumpFile}, stdout, stderr)
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
	code := writeFiles(p, outputs{*cFile, *dumpFile}, stderr)
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

// outputs are the files a run writes besides the program's output.
type outputs struct {
	c, dump string // --to-c, --dump
}

// writeFiles writes the requested outputs, the C translation before the
// dump as the original does, and returns 1 if one cannot be written.
func writeFiles(p *syntax.Program, files outputs, stderr io.Writer) int {
	code := 0
	for _, out := range []struct {
		file   string
		render func(*syntax.Program) string
	}{{files.c, emit.Exact}, {files.dump, emit.Dump}} {
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
func translate(name string, lang render.Lang, files outputs, stdout, stderr io.Writer) int {
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
	code := writeFiles(p, files, stderr)
	if _, err := fmt.Fprint(stdout, text); err != nil {
		fmt.Fprintf(stderr, "sorted: %v\n", err)
		return 1
	}
	return code
}
