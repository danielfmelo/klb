package log

import (
	"os"
	"strings"
	"testing"
)

type Logger struct {
	file *os.File
}

const logsdir = "./testdata/logs"

//New creates a new logger for the given testame.
//This will save the logs on the our common logs dir.
//This is not very usual on Go tests but we have pretty
//long running tests that may take some time to run and
//being able to tail some logs is useful.
func New(t *testing.T, testname string) *Logger {
	err := os.MkdirAll(logsdir, 0755)
	if err != nil {
		t.Fatalf("creating test logs dir: %s:", err)
	}
	logspath := strings.Join([]string{logsdir, testname + ".logs"}, "/")
	file, err := os.Create(logfilepath)
	if err != nil {
		t.Fatalf("error opening log file: %s", err)
	}
	return &Logger{
		logs: file,
	}
}

func (l *Logger) Write(b []byte) (n int, err error) {
	res, err := l.file.Write(b)
	l.file.Sync()
	return res, err
}
