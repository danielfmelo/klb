package nash

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/NeowayLabs/klb/tests/lib/retrier"
	"github.com/NeowayLabs/nash"
)

type Shell struct {
	shell *nash.Shell
}

func New(t *testing.T, output io.Writer) *Shell {
	shell, err := nash.New()
	if err != nil {
		t.Fatal(err)
	}

	shell.SetStdout(output)
	shell.SetStderr(output)

	nashPath := os.Getenv("HOME") + "/.nash"
	os.MkdirAll(nashPath, 0655)
	shell.SetDotDir(nashPath)

	return &Shell{shell: shell}
}

func Run(
	ctx context.Context,
	t *testing.T,
	logger *log.Logger,
	scriptpath string,
	args ...string,
) {
	s := New(t, &logWriter{
		logger: logger,
	})
	retrier.Run(ctx, t, logger, "nash.Run:"+scriptpath, func() error {
		err := s.shell.ExecFile(scriptpath, args...)
		if err != nil {
			return fmt.Errorf(
				"error: %s, executing script: %s",
				err,
				scriptpath,
			)
		}
		return nil
	})
}

type logWriter struct {
	logger *log.Logger
}

func (l *logWriter) Write(b []byte) (int, error) {
	l.logger.Println(strings.TrimSuffix(string(b), "\n"))
	return len(b), nil
}
