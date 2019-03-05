package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	{regexp.MustCompille(`(?P<l>%(?P<flag>[-+ 0#'I]*)(?P<width>\*?\d*\$?)(?P<precision>\.\*?\d*\$?)?)l?(?P<r>[idouxX])`), "${l}${r}"},
}

var skipDirs = []*regexp.Regexp{}

func processFile(path string) error {
	b := &strings.Builder{}

	s, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
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
		}
		b.WriteString(s)
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(b.String())

	if err != nil {
		return err
	}

	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("which directory to process")
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
