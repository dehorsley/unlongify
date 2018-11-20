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

var replacements = []struct {
	pat  *regexp.Regexp
	repl string
}{
	{regexp.MustCompile(`(((long|int) )+)unsigned\s*`), "unsigned ${1}"},
	{regexp.MustCompile(`(\s+)long\s+int(\s+)`), "${1}int${2}"},
	{regexp.MustCompile(`(\s+)unsigned\s+long\s+long(\s+)`), "${1}uint64_t${2}"},
	{regexp.MustCompile(`(\s+)long\s+long(\s+)`), "${1}int64_t${2}"},
	{regexp.MustCompile(`(\s+)long(\s+)`), "${1}int${2}"},
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
		line := i.val
		if i.typ == itemCode {
			for _, r := range replacements {
				line = r.pat.ReplaceAllString(line, r.repl)
			}
		}
		b.WriteString(line)
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
	err := filepath.Walk("fs", func(path string, info os.FileInfo, err error) error {
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
		if ext != ".c" && ext != ".h" {
			return nil
		}

		return processFile(path)
	})

	if err != nil {
		fmt.Printf("error walking the path %q: %v\n", "fs", err)
		return
	}

}
