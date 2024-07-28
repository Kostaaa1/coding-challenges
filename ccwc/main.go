package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type line struct {
	counts  []int
	dstPath string
	err     error
}

type config struct {
	c     bool
	l     bool
	w     bool
	m     bool
	log   *log.Logger
	total []int
}

func getNBytes(content string) int {
	return len([]byte(content))
}
func getNLines(content string) int {
	lines := strings.Split(content, "\n")
	return len(lines) - 1
}
func getNWords(content string) int {
	words := strings.Fields(content)
	return len(words)
}
func getNChar(content string) int {
	c := utf8.RuneCountInString(content)
	return c
}

func (cfg *config) GetCounts(content string) []int {
	var counts []int
	addCount := func(condition bool, countF func(string) int) {
		if condition {
			counts = append(counts, countF(content))
		}
	}
	switch {
	case cfg.m:
		if checkLocaleSupportsUTF8() {
			addCount(true, getNChar)
		} else {
			addCount(true, getNBytes)
		}
	default:
		if !cfg.c && !cfg.l && !cfg.w {
			addCount(true, getNLines)
			addCount(true, getNWords)
			addCount(true, getNBytes)
		} else {
			addCount(cfg.c, getNBytes)
			addCount(cfg.l, getNLines)
			addCount(cfg.w, getNWords)
		}
	}
	for i, count := range counts {
		if i >= len(cfg.total) {
			cfg.total = append(cfg.total, count)
		} else {
			cfg.total[i] += count
		}
	}
	return counts
}

func (cfg *config) printLines(lines []line) {
	if len(lines) == 0 {
		log.Fatal("Unexpected error. 0 Lines")
	}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line.err != nil {
			fmt.Println(line.err)
			continue
		}
		var b strings.Builder
		for k, count := range line.counts {
			fmt.Fprintf(&b, " %s", strings.Repeat(" ", getLengthOfInt(cfg.total[k])-getLengthOfInt(count))+strconv.Itoa(count))
		}
		fmt.Printf("%s %s\n", b.String(), line.dstPath)
	}
	if len(lines) > 1 {
		fmt.Printf(" %s total\n", strings.Join(intsToStrings(cfg.total), " "))
	}
}

func main() {
	cfg := config{
		log:   log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
		c:     false,
		l:     false,
		w:     false,
		m:     false,
		total: []int{},
	}
	flag.BoolVar(&cfg.c, "c", false, "print the byte counts")
	flag.BoolVar(&cfg.m, "m", false, "print the character counts")
	flag.BoolVar(&cfg.l, "l", false, "print the newline counts")
	flag.BoolVar(&cfg.w, "w", false, "print the word counts")
	flag.Parse()

	dstPaths := extractPathsFromArgs()
	var lines = make([]line, len(dstPaths))

	if len(dstPaths) == 0 {
		content := readStdin()
		counts := cfg.GetCounts(content)
		var line = line{
			counts:  counts,
			dstPath: "",
			err:     nil,
		}
		lines = append(lines, line)
	} else {
		for i, p := range dstPaths {
			var line = line{
				counts:  []int{},
				dstPath: p,
				err:     nil,
			}
			b, err := openAndRead(p)
			if err != nil {
				line.err = err
			} else {
				content := string(b)
				c := cfg.GetCounts(content)
				line.counts = c
			}
			lines[i] = line
		}
	}
	cfg.printLines(lines)
}

func getLocale() (string, error) {
	cmd := exec.Command("locale")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func checkLocaleSupportsUTF8() bool {
	locale, err := getLocale()
	if err != nil {
		fmt.Println("error getting locale")
		return false
	}
	lines := strings.Split(locale, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "LC_CTYPE=") {
			value := strings.TrimPrefix(line, "LC_CTYPE=")
			value = strings.Trim(value, "\"")
			return strings.HasSuffix(value, "UTF-8")
		}
	}
	return false
}

func openAndRead(dst string) ([]byte, error) {
	f, err := os.Open(dst)
	if err != nil {
		return nil, fmt.Errorf("ccwc: '%s': No such file or directory", dst)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func extractPathsFromArgs() []string {
	var paths []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		paths = append(paths, arg)
	}
	return paths
}

func readStdin() string {
	ch := make(chan string)
	go func() {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			ch <- ""
			return
		}
		ch <- string(b)
	}()
	select {
	case result := <-ch:
		return result
	case <-time.After(5 * time.Second):
		fmt.Println("No stdin found")
		return ""
	}
}

func intsToStrings(ints []int) []string {
	var output = make([]string, len(ints))
	for i, v := range ints {
		output[i] = strconv.Itoa(v)
	}
	return output
}

func getLengthOfInt(v int) int {
	return len(strconv.Itoa(v))
}
