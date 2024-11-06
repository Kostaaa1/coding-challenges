package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type config struct {
	r bool
	v bool
}

func print(dst, pattern string, includePath bool) error {
	info, err := os.Stat(dst)
	if os.IsNotExist(err) {
		return err
	}

	if info.IsDir() {
		str := fmt.Sprintf("ccgrep: %s: Is a directory", dst)
		fmt.Println(str)
		return nil
	}

	file, err := os.Open(dst)
	if err != nil {
		return err
	}
	defer file.Close()

	b, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	content := string(b)
	lines := strings.Split(content, "\n")

	var re *regexp.Regexp
	if pattern != "" {
		re, err = regexp.Compile(pattern)
		if err != nil {
			panic(fmt.Errorf("error compiling regex: %w", err))
		}
	}

	for _, line := range lines {
		if line == "" {
			continue
		}

		if re != nil {
			match := re.FindString(line)
			if match != "" {
				var str string
				highlighted := strings.Replace(line, match, redString(match), 1)
				if includePath {
					str = fmt.Sprintf("%s%s%s", magentaString(dst), blueString(":"), highlighted)
				} else {
					str = fmt.Sprintf("%s%s%s", magentaString(dst), blueString(":"), highlighted)
				}
				fmt.Println(str)
			}
		} else {
			fmt.Println(line)
		}
	}

	return nil
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
	flag.Parse()

	args := flag.Args()
	pattern := args[0]

	if len(args) > 2 {
		for _, path := range args[1:] {
			if err := print(path, pattern, true); err != nil {
				panic(err)
			}
		}
	} else {
		if err := print(args[1], pattern, false); err != nil {
			panic(err)
		}
	}
}

var (
	reset = "\033[0m"
)

func redString(text string) string {
	return fmt.Sprintf("%s%s%s", "\033[91m", text, reset)
}

func magentaString(text string) string {
	return fmt.Sprintf("%s%s%s", "\033[35m", text, reset)
}

func blueString(text string) string {
	return fmt.Sprintf("%s%s%s", "\033[36m", text, reset)
}
