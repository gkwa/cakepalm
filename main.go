package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var logFilePath = flag.String("path", "/var/log/cloud-init-output.log", "Path to the log file")

type FileState struct {
	LastPos  int64
	Mux      sync.Mutex
	StartTime time.Time
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func (fs *FileState) MonitorFile(logFilePath string) {
	fs.Mux.Lock()
	defer fs.Mux.Unlock()

	file, err := os.Open(logFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("File error: %s - %s", logFilePath, err.Error())
		}
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("Stat error: %s - %s", logFilePath, err.Error())
		return
	}

	if fileInfo.Size() < fs.LastPos {
		fs.LastPos = 0
	}

	_, err = file.Seek(fs.LastPos, 0)
	if err != nil {
		log.Printf("Seek error: %s - %s", logFilePath, err.Error())
		return
	}

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error: %s - %s", logFilePath, err.Error())
			}
			break
		}
		relativeTime := time.Since(fs.StartTime).Round(time.Second)
		line = strings.TrimSpace(line)
		log.Printf("%s - %s - Relative Time: %s", logFilePath, relativeTime ,line )
		fs.LastPos, err = file.Seek(0, io.SeekCurrent)
		if err != nil {
			log.Printf("Seek error: %s - %s", logFilePath, err.Error())
			break
		}
	}
}

func main() {
	flag.Parse()
	ticker := time.NewTicker(500 * time.Millisecond)
	fileState := &FileState{
		StartTime: time.Now(),
	}
	go func() {
		for range ticker.C {
			fileState.MonitorFile(*logFilePath)
		}
	}()

	select {}
}
