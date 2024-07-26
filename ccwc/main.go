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

type config struct {
	c   bool
	l   bool
	w   bool
	m   bool
	log *log.Logger
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
	if cfg.m {
		if checkLocaleSupportsUTF8() {
			counts = append(counts, getNChar(content))
		} else {
			counts = append(counts, getNBytes(content))
		}
		return counts
	}
	if !cfg.c && !cfg.l && !cfg.w {
		counts = append(counts, getNLines(content))
		counts = append(counts, getNWords(content))
		counts = append(counts, getNBytes(content))
	}
	if cfg.c {
		counts = append(counts, getNBytes(content))
	}
	if cfg.l {
		counts = append(counts, getNLines(content))
	}
	if cfg.w {
		counts = append(counts, getNWords(content))
	}
	return counts
}

func printOutput(outputs []string, total []int) {
	for i := 0; i < len(outputs); i++ {
		o := outputs[i]
		parts := strings.SplitN(o, " ", len(total)+1)
		path := parts[len(total)]
		var b strings.Builder
		for k, n := range total {
			fmt.Fprintf(&b, "%s", " "+strings.Repeat(" ", getLengthOfInt(n)-len(parts[k]))+parts[k])
		}
		fmt.Printf("%s %s", b.String(), path)
	}
	if len(outputs) > 1 {
		fmt.Printf("%s%s total\n", " ", strings.Join(intsToStrings(total), " "))
	}
}

func (cfg *config) getCounts(dstPaths []string) {
	flagCount := getFlagCount()
	var total = make([]int, flagCount)
	var outputVals = make([]string, len(dstPaths))
	if len(dstPaths) == 0 {
		content := readStdin()
		counts := cfg.GetCounts(content)
		fmt.Println(counts)
		return
	}
	for i, p := range dstPaths {
		b, err := openAndRead(p)
		if err != nil {
			continue
		}
		c := cfg.GetCounts(string(b))
		for k, count := range c {
			total[k] += count
		}
		outputVals[i] = fmt.Sprintf("%s %s\n", strings.Join(intsToStrings(c), " "), p)
	}
	printOutput(outputVals, total)
}

func main() {
	cfg := config{
		log: log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
		c:   false,
		l:   false,
		w:   false,
		m:   false,
	}
	flag.BoolVar(&cfg.c, "c", false, "print the byte counts")
	flag.BoolVar(&cfg.m, "m", false, "print the character counts")
	flag.BoolVar(&cfg.l, "l", false, "print the newline counts")
	flag.BoolVar(&cfg.w, "w", false, "print the word counts")
	flag.Parse()
	paths := extractPathsFromArgs()
	cfg.getCounts(paths)
}

func getFlagCount() int {
	count := 0
	flag.Visit(func(f *flag.Flag) {
		count++
	})
	if count == 0 {
		return 3
	}
	return count
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
		log.Printf("error wile opening the file: %s", err)
		return nil, err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		log.Fatal("error reading bytes of the file")
		return nil, err
	}
	return b, nil
}

func extractPathsFromArgs() []string {
	var paths []string
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		_, err := os.Stat(arg)
		if err == nil {
			paths = append(paths, arg)
		}
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
