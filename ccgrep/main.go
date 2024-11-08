package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type config struct {
	r bool
	v bool
	i bool
}

var (
	reset      = "\033[0m"
	red        = "\033[91m"
	magenta    = "\033[35m"
	brightBlue = "\033[36m"
)

func (cfg *config) print(dst string, re *regexp.Regexp, includePath bool, wg *sync.WaitGroup, ch chan string) {
	defer wg.Done()

	info, err := os.Stat(dst)
	if os.IsNotExist(err) {
		fmt.Println("unexpected err: %w", err)
		return
	}

	if info.IsDir() {
		fmt.Printf("ccgrep: %s: Is a directory\n", dst)
		return
	}

	// THIS WILL WORK ONLY ON UNIX SYSTEMS! checking if the file is an executable (we want to use grep only on human-readble files). info.Mode() returns this: rwxrwxrwx. First 3 bits are for the owner, next 3 for the group, and last 3 are for other. ****
	mode := info.Mode()
	if mode&0100 != 0 {
		return
	}

	file, err := os.Open(dst)
	if err != nil {
		fmt.Printf("ccgrep: %s: No such file or directory\n", dst)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if cfg.v {
			if !re.MatchString(line) {
				highlighted := highlightMatches(line, re)
				if includePath {
					ch <- magentaString(dst) + blueString(":") + highlighted
				} else {
					ch <- highlighted
				}
			}
		} else {
			if re.MatchString(line) {
				highlighted := highlightMatches(line, re)
				if includePath {
					ch <- magentaString(dst) + blueString(":") + highlighted
				} else {
					ch <- highlighted
				}
			}
		}
	}
}

func highlightMatches(line string, re *regexp.Regexp) string {
	matches := re.FindAllStringIndex(line, -1)
	var highlighted strings.Builder
	lastIndex := 0
	for _, match := range matches {
		highlighted.WriteString(line[lastIndex:match[0]])
		highlighted.WriteString(redString(line[match[0]:match[1]]))
		lastIndex = match[1]
	}
	highlighted.WriteString(line[lastIndex:])
	return highlighted.String()
}

func (cfg *config) parseRegexPattern(pattern string) string {
	if cfg.i {
		return fmt.Sprintf("(?i)%s", pattern)
	}
	switch pattern {
	case "/d":
		return "[0-9]"
	case "/w":
		return "[a-zA-Z0-9_]"
	default:
		return pattern
	}
}

func (cfg *config) getAllPaths() []string {
	var allPaths []string

	pathArgs := flag.Args()[1:]

	if !cfg.r {
		allPaths = pathArgs
	} else {
		for _, path := range pathArgs {
			filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					allPaths = append(allPaths, p)
					return nil
				}
				if !d.IsDir() {
					allPaths = append(allPaths, p)
				}
				return nil
			})
		}
	}

	return allPaths
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println(`Usage: ccgrep [OPTION]... PATTERNS [FILE]...
Try 'ccgrep --help' for more information.`)
		os.Exit(2)
	}

	var cfg config
	flag.BoolVar(&cfg.r, "r", false, "--recursive like --directories=recurse")
	flag.BoolVar(&cfg.v, "v", false, "--invert-match select non-matching lines")
	flag.BoolVar(&cfg.i, "i", false, "--ignore-case ignore case distinctions in patterns and data")
	flag.Parse()

	pattern := cfg.parseRegexPattern(flag.Args()[0])

	///////////////////////
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Errorf("error compiling regex: %w", err))
	}
	///////////////////////

	paths := cfg.getAllPaths()

	var wg sync.WaitGroup
	ch := make(chan string, len(paths))

	for _, path := range paths {
		wg.Add(1)
		go cfg.print(path, re, len(paths) > 2, &wg, ch)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for result := range ch {
		fmt.Println(result)
	}
}

// ANSII
func redString(text string) string {
	return red + text + reset
}
func magentaString(text string) string {
	return magenta + text + reset
}
func blueString(text string) string {
	return brightBlue + text + reset
}
