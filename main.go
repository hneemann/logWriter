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
	"syscall"
	"time"
	"unicode"
)

type Logger struct {
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

// NewLogger creates a new logger that writes to the folder [folder].
// The logger writes at most [maxLinesInFile] lines to a file.
// The logger keeps at most [maxFiles] files.
// The logger writes to the output stream [out].
func NewLogger(folder string, maxLinesInFile int, maxFiles int, out io.Writer) *Logger {
	l := &Logger{
		folder:         folder,
		maxLinesInFile: maxLinesInFile,
		fileList:       make([]string, maxFiles),
		out:            out,
	}
	return l
}

func (l *Logger) _checkFile() error {
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

// PipeToLogger reads from the reader [r] and writes to the log file
// and the output stream. The function returns when the reader is closed.
func (l *Logger) PipeToLogger(r io.Reader) {
	lr := bufio.NewReader(r)
	for {
		line, err := lr.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				l.WriteFile("logger: command terminated with error:\n")
				lines := strings.Split(err.Error(), "\n")
				for _, li := range lines {
					li = strings.TrimRightFunc(li, unicode.IsSpace)
					l.WriteFile(li + "\n")
				}
			} else {
				l.WriteFile("logger: command terminated with EOF\n")
			}
			l.Close()
			return
		} else {
			l.WriteFile(line)
		}
	}
}

// WriteFile writes to the log file and the output stream.
// It expects the line to end with a newline
func (l *Logger) WriteFile(li string) {
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

// Close closes the log file.
func (l *Logger) Close() {
	l.m.Lock()
	defer l.m.Unlock()

	if l.file != nil {
		l.Println("logger: closing file")
		err := l.file.Close()
		if err != nil {
			l.Println(err)
		}
		l.file = nil
	}
}

// Print writes to the output stream only.
func (l *Logger) Print(s string) {
	l.out.Write([]byte(s))
}

// Println writes to the output stream only.
// A newline is appended to the string.
func (l *Logger) Println(s any) {
	fmt.Fprint(l.out, s)
	l.out.Write([]byte{'\n'})
}

// usage command 2>&1 | logWriter
func main() {
	maxLinesInFile := flag.Int("lines", 1000, "max lines in file")
	maxFiles := flag.Int("files", 10, "max files")
	termDelay := flag.Duration("delay", 2*time.Second, "delay before exit")
	errOut := flag.Bool("errOut", true, "output to stdErr")
	folder := flag.String("folder", ".", "folder")
	flag.Parse()

	out := os.Stdout
	if *errOut {
		out = os.Stderr
	}
	l := NewLogger(*folder, *maxLinesInFile, *maxFiles, out)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-c
		l.WriteFile("logger: received signal: " + s.String() + "\n")
		time.Sleep(*termDelay)
		l.WriteFile("logger: command not terminated after " + (*termDelay).String() + "!, exit!\n")
		l.Close()
		os.Exit(0)
	}()

	l.PipeToLogger(os.Stdin)
}
