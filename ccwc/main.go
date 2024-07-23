package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

type config struct {
	c    bool
	l    bool
	w    bool
	m    bool
	text string
	log  *log.Logger
}

func (cfg *config) getNBytes() string {
	return fmt.Sprintf("%d", len([]byte(cfg.text)))
}
func (cfg *config) getNLines() string {
	lines := strings.Split(cfg.text, "\n")
	return fmt.Sprintf("%d", len(lines))
}
func (cfg *config) getNWords() string {
	words := strings.Fields(cfg.text)
	return fmt.Sprintf("%d", len(words))
}
func (cfg *config) getNChar() string {
	chars := utf8.RuneCountInString(cfg.text)
	return fmt.Sprintf("%d", chars)
}

func findPathInArgs() string {
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		_, err := os.Stat(arg)
		if err == nil {
			return arg
		}
	}
	return ""
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

func getText() (string, string) {
	var text string
	var inputPath string

	p := findPathInArgs()
	if p == "" {
		text = readStdin()
	} else {
		b, err := openAndRead(p)
		if err != nil {
			log.Fatal(err)
		}
		text = string(b)
		inputPath = p
	}
	if text == "" {
		log.Fatal("the input is empty. Either provide a file path or stdin.")
	}
	return text, inputPath
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
	fmt.Println(locale)
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

func main() {
	cfg := config{
		log:  log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile),
		c:    false,
		l:    false,
		w:    false,
		m:    false,
		text: "",
	}
	flag.BoolVar(&cfg.c, "c", false, "Displays the number of bytes in a file.")
	flag.BoolVar(&cfg.l, "l", false, "Displays the number of lines in a file.")
	flag.BoolVar(&cfg.w, "w", false, "Displays the number of words in a file.")
	flag.BoolVar(&cfg.m, "m", false, "Displays the number of characters in a file.")
	flag.Parse()

	txt, inputPath := getText()
	cfg.text = txt

	var counts []string
	if cfg.m {
		if checkLocaleSupportsUTF8() {
			counts = append(counts, string(cfg.getNChar()))
		} else {
			counts = append(counts, string(cfg.getNBytes()))
		}
		fmt.Printf("%s %s\n", strings.Join(counts, " "), inputPath)
		return
	}

	if !cfg.c && !cfg.l && !cfg.w {
		counts = append(counts, string(cfg.getNBytes()))
		counts = append(counts, string(cfg.getNLines()))
		counts = append(counts, string(cfg.getNWords()))
	}
	if cfg.c {
		counts = append(counts, string(cfg.getNBytes()))
	}
	if cfg.l {
		counts = append(counts, string(cfg.getNLines()))
	}
	if cfg.w {
		counts = append(counts, string(cfg.getNWords()))
	}
	fmt.Printf("%s %s\n", strings.Join(counts, " "), inputPath)
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
