package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

type logger struct {
	m              sync.Mutex
	folder         string
	maxLinesInFile int
	linesInFile    int
	fileCounter    int
	fileList       []string
	filePosInList  int
	file           *os.File
	out            io.Writer
}

func NewLogger(folder string, maxLinesInFile int, maxFiles int, out io.Writer) *logger {
	l := &logger{
		folder:         folder,
		maxLinesInFile: maxLinesInFile,
		fileList:       make([]string, maxFiles),
		out:            out,
	}
	return l
}

func (l *logger) _checkFile() error {
	if l.linesInFile > l.maxLinesInFile && l.file != nil {
		l.file.Close()
		l.file = nil
	}
	if l.file == nil {
		date := time.Now().Format("2006-01-02_15-04-05")
		name := date + "_" + strconv.Itoa(l.fileCounter) + ".log"
		l.fileCounter++
		path := filepath.Join(l.folder, name)
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		l.linesInFile = 0
		l.file = f

		if l.fileList[l.filePosInList] != "" {
			err = os.Remove(l.fileList[l.filePosInList])
			if err != nil {
				return err
			}
		}
		l.fileList[l.filePosInList] = path
		l.filePosInList++
		if l.filePosInList == len(l.fileList) {
			l.filePosInList = 0
		}
	}
	return nil
}

func (l *logger) PipeToLogger(r io.Reader) {
	lr := bufio.NewReader(r)
	for {
		line, err := lr.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				lines := strings.Split(err.Error(), "\n")
				for _, li := range lines {
					li = strings.TrimRightFunc(li, unicode.IsSpace)
					l.WriteFile(li + "\n")
				}
			}
			l.Close()
			return
		} else {
			l.WriteFile(line)
		}
	}
}

// WriteFile writes to the log file.
// It expects the line to end with a newline
func (l *logger) WriteFile(li string) {
	l.m.Lock()
	defer l.m.Unlock()

	err := l._checkFile()
	if err != nil {
		l.Println(err)
	}
	if l.file != nil {
		_, err = l.file.WriteString(li)
		if err != nil {
			l.Println(err)
		}
		l.linesInFile++
	}

	l.Print(li)
}

func (l *logger) Close() {
	l.m.Lock()
	defer l.m.Unlock()

	if l.file != nil {
		l.Println("closing file")
		err := l.file.Close()
		if err != nil {
			l.Println(err)
		}
		l.file = nil
	}
}

func (l *logger) Print(s string) {
	l.out.Write([]byte(s))
}

func (l *logger) Println(s any) {
	fmt.Fprint(l.out, s)
	l.out.Write([]byte{'\n'})
}

// usage command 2>&1 | logWriter
func main() {
	maxLinesInFile := flag.Int("lines", 1000, "max lines in file")
	maxFiles := flag.Int("files", 10, "max files")
	errOut := flag.Bool("errOut", true, "output to stdErr")
	folder := flag.String("folder", ".", "folder")
	flag.Parse()

	out := os.Stdout
	if *errOut {
		out = os.Stderr
	}
	l := NewLogger(*folder, *maxLinesInFile, *maxFiles, out)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		l.Println("logger received os.Interrupt")
	}()

	l.PipeToLogger(os.Stdin)
}
