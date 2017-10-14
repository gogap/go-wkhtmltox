package wkhtmltox

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"syscall"
	"time"
)

func execCommand(timeout time.Duration, data []byte, name string, args ...string) (result []byte, err error) {

	cmd := exec.Command(name, args...)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}

	outBuf := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)

	if err != nil {
		return
	}

	err = cmd.Start()

	if err != nil {
		return
	}

	if len(data) > 0 {
		_, err = io.Copy(stdin, bytes.NewBuffer(data))
		if err != nil {
			return
		}
	}

	stdin.Close()

	go io.Copy(outBuf, stdout)
	go io.Copy(errBuf, stderr)

	ch := make(chan struct{})

	go func(cmd *exec.Cmd) {
		defer close(ch)
		cmd.Wait()
	}(cmd)

	select {
	case <-ch:
	case <-time.After(timeout):
		cmd.Process.Kill()
		err = errors.New("execute timeout")
		return
	}

	if outBuf.Len() > 0 {
		return outBuf.Bytes(), nil
	}

	errStr := errBuf.String()

	if len(errStr) > 0 {
		return nil, errors.New(errStr)
	}

	return
}
