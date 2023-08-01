package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var logFilePath = flag.String("path", "/var/log/cloud-init-output.log", "Path to the log file")

type FileState struct {
	LastPos   int64
	Mux       sync.Mutex
	StartTime time.Time
}

type CustomLogger struct {
	*log.Logger
	StartTime time.Time
}

func (cl *CustomLogger) Output(calldepth int, s string) error {
	now := time.Now()
	relativeTime := now.Sub(cl.StartTime).Round(time.Second)
	timestamp := now.Format("2006/01/02 15:04:05")
	s = fmt.Sprintf("%s [%s] %s", timestamp, relativeTime, s)
	return cl.Logger.Output(calldepth+1, s)
}

func NewCustomLogger(out io.Writer, flag int, startTime time.Time) *CustomLogger {
	return &CustomLogger{
		Logger:    log.New(out, "", flag),
		StartTime: startTime,
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func (fs *FileState) MonitorFile(logFilePath string, logger *CustomLogger) {
	fs.Mux.Lock()
	defer fs.Mux.Unlock()

	file, err := os.Open(logFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Output(2, fmt.Sprintf("File error: %s - %s", logFilePath, err.Error()))
		}
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		logger.Output(2, fmt.Sprintf("Stat error: %s - %s", logFilePath, err.Error()))
		return
	}

	if fileInfo.Size() < fs.LastPos {
		fs.LastPos = 0
	}

	_, err = file.Seek(fs.LastPos, 0)
	if err != nil {
		logger.Output(2, fmt.Sprintf("Seek error: %s - %s", logFilePath, err.Error()))
		return
	}

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				logger.Output(2, fmt.Sprintf("Read error: %s - %s", logFilePath, err.Error()))
			}
			break
		}
		line = strings.TrimSpace(line)
		logger.Output(2, fmt.Sprintf("%s - %s", logFilePath, line))
		fs.LastPos, err = file.Seek(0, io.SeekCurrent)
		if err != nil {
			logger.Output(2, fmt.Sprintf("Seek error: %s - %s", logFilePath, err.Error()))
			break
		}
	}
}

func main() {
	flag.Parse()
	ticker := time.NewTicker(500 * time.Millisecond)
	startTime := time.Now()
	fileState := &FileState{
		StartTime: startTime,
	}
	logger := NewCustomLogger(os.Stdout, log.LstdFlags|log.Lshortfile, startTime)
	go func() {
		for range ticker.C {
			fileState.MonitorFile(*logFilePath, logger)
		}
	}()

	select {}
}
