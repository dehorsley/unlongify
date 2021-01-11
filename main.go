package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

func matches(path string, res []*regexp.Regexp) bool {
	for _, re := range res {
		if re.MatchString(path) {
			return true
		}
	}
	return false

}

type replacement struct {
	pat  *regexp.Regexp
	repl string
}

var codeReplacements = []replacement{
	{regexp.MustCompile(`(((long|int) )+)unsigned\s*`), "unsigned ${1}"},
	{regexp.MustCompile(`(?P<l>(^|\s|\(|,)+)long\s+int(?P<r>(\s|[)*])+)`), "${l}int${r}"},
	{regexp.MustCompile(`(?P<l>(^|\s|\(|,)+)unsigned\s+long\s+long(?P<r>(\s|[)*])+)`), "${l}TMP_uint64_t${r}"},
	{regexp.MustCompile(`(?P<l>(^|\s|\(|,)+)long\s+long(?P<r>(\s|[)*])+)`), "${l}TMP_int64_t${r}"},
	{regexp.MustCompile(`(?P<l>(^|\s|\(|,)+)long(?P<r>(\s|[)*])+)`), "${l}int${r}"},
	{regexp.MustCompile(`(?P<l>(^|\s|\()+)TMP_int64_t(?P<r>(\s|[)*])+)`), "${l}long long${r}"},
	{regexp.MustCompile(`(?P<l>(^|\s|\()+)TMP_uint64_t(?P<r>(\s|[)*])+)`), "${l}unsigned long long${r}"},
}

var stringReplacements = []replacement{
	{regexp.MustCompile(`(?P<l>%(?P<flag>[-+ 0#'I]*)(?P<width>\*?\d*\$?)(?P<precision>\.\*?\d*\$?)?)l?(?P<r>[idouxX])`), "${l}${r}"},
}

var skipDirs = []*regexp.Regexp{}

func processFile(path string) error {
	var b bytes.Buffer

	s, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading file %s: %v", path, err)
	}

	_, items := lex(string(s))

	for i := range items {
		s := i.val
		switch i.typ {
		case itemCode:
			for _, r := range codeReplacements {
				s = r.pat.ReplaceAllString(s, r.repl)
			}
		case itemString:
			for _, r := range stringReplacements {
				s = r.pat.ReplaceAllString(s, r.repl)
			}
		case itemError:
			return fmt.Errorf("error processing file %s: %s", path, i.val)
		}
		b.WriteString(s)
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("error opening file %s for writing: %v", path, err)
	}
	defer f.Close()

	_, err = b.WriteTo(f)

	if err != nil {
		return fmt.Errorf("error writing to file %s: %v", path, err)
	}

	return nil
}

const usage = `
usage: unlongify <PATH>

Eg:

	unlongify /usr2/st

This recursively scans a directory tree for C source files and headers and
modifies them to change "long" type declarations to "int". Care is taken to
avoid false positives elsewhere in the source. Printf/scanf format options
are also updated to use "int".

This tool aimed at updating 32-bit x86 C code to dual 32/64-bit
(x86/x86_64) code. GCC on x86 processors compiles both "int" and "long" to
32-bit integers, whereas on x86_64 "int" compiles to 32-bit and "long"
compiles to 64-bit. The tool doesn't really understand C, and should be 
treated at a blunt object to get the code close to correct.

Users writing modern code should consider updating any program interfaces
to use fixed width integers defined in "stdint.h". This is more portable
between compilers and architectures.

Note: not all longs are bad, specifically some system calls explicitly
require and return "long" arguments. Of particular note is "mtype" field in
the struct argument to "msgrcv" must be of type "long". THIS TOOL WILL
BLINDLY CONVERT THESE TO "ints"!

Users should check their code after using this tool. Modern versions of GCC
will warn in at least some cases check the compilers output.

WARNING: this does not make backups before editing files
`

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}

	err := filepath.Walk(os.Args[1], func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && matches(path, skipDirs) {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(info.Name())
		if ext != ".c" && ext != ".h" && ext != ".cpp" && ext != ".hpp" {
			return nil
		}

		return processFile(path)
	})

	if err != nil {
		fmt.Printf("error walking the path %q: %v\n", "fs", err)
		return
	}

}
