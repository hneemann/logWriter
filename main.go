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
}

func newLogger(folder string, maxLinesInFile int, maxFiles int) *logger {
	l := &logger{
		folder:         folder,
		maxLinesInFile: maxLinesInFile,
		fileList:       make([]string, maxFiles),
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

func (l *logger) pipeToLogger(r io.Reader) {
	lr := bufio.NewReader(r)
	for {
		line, err := lr.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				lines := strings.Split(err.Error(), "\n")
				for _, li := range lines {
					l.writeFile(li)
				}
			}
			l.close()
			return
		} else {
			l.writeFile(line)
		}
	}
}

func (l *logger) writeFile(li string) {
	l.m.Lock()
	defer l.m.Unlock()

	err := l._checkFile()
	if err != nil {
		fmt.Println(err)
	}
	if l.file != nil {
		_, err = l.file.WriteString(li)
		if err != nil {
			fmt.Println(err)
		}
		l.linesInFile++
	}
	fmt.Print(li)
}

func (l *logger) close() {
	l.m.Lock()
	defer l.m.Unlock()

	if l.file != nil {
		fmt.Println("closing file")
		err := l.file.Close()
		if err != nil {
			fmt.Println(err)
		}
		l.file = nil
	}
}

// usage command 2>&1 | logWriter
func main() {
	maxLinesInFile := flag.Int("lines", 500, "max lines in file")
	maxFiles := flag.Int("files", 5, "max files")
	folder := flag.String("folder", ".", "folder")
	flag.Parse()

	l := newLogger(*folder, *maxLinesInFile, *maxFiles)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Println("logger received os.Interrupt")
	}()

	l.pipeToLogger(os.Stdin)
}
