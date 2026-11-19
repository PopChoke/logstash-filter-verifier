//go:build windows

package logstash

import (
	"errors"
	"os"
	"syscall"
)

func processSysProcAttr() *syscall.SysProcAttr {
	return nil
}

func stopProcess(process *os.Process) error {
	return process.Kill()
}

func reloadProcess(process *os.Process) error {
	return errors.New("configuration reload is not supported for Logstash on Windows")
}
