package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type logger struct {
	folder         string
	maxLinesInFile int
	linesInFile    int
	fileNum        int
	counter        int
	fileList       []string
	file           *os.File
	collectorPort  int
}

func newLogger(folder string, maxLinesInFile int, maxFiles int) *logger {
	l := &logger{
		folder:         folder,
		maxLinesInFile: maxLinesInFile,
		fileList:       make([]string, maxFiles),
	}
	return l
}

func (l *logger) checkFile() error {
	if l.linesInFile > l.maxLinesInFile && l.file != nil {
		l.file.Close()
		l.file = nil
	}
	if l.file == nil {
		date := time.Now().Format("2006-01-02_15-04-05")
		name := date + "_" + strconv.Itoa(l.counter) + ".log"
		l.counter++
		path := filepath.Join(l.folder, name)
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		l.linesInFile = 0
		l.file = f

		if l.fileList[l.fileNum] != "" {
			err = os.Remove(l.fileList[l.fileNum])
			if err != nil {
				return err
			}
		}
		l.fileList[l.fileNum] = path
		l.fileNum++
		if l.fileNum == len(l.fileList) {
			l.fileNum = 0
		}
	}
	return nil
}

func (l *logger) pipeToLogger(r io.Reader) {
	lr := bufio.NewReader(r)
	for {
		line, err := lr.ReadString('\n')
		if err != nil {
			lines := strings.Split(err.Error(), "\n")
			for _, li := range lines {
				l.writeFile(li)
			}
			break
		} else {
			l.writeFile(line)
		}
	}
}

func (l *logger) writeFile(li string) {
	err := l.checkFile()
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

func main() {
	maxLinesInFile := flag.Int("lines", 500, "max lines in file")
	maxFiles := flag.Int("files", 5, "max files")
	folder := flag.String("folder", ".", "folder")
	flag.Parse()

	l := newLogger(*folder, *maxLinesInFile, *maxFiles)
	l.pipeToLogger(os.Stdin)
}
