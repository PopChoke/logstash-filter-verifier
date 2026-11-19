//go:build !windows

package logstash

import (
	"os"
	"syscall"
)

func processSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func stopProcess(process *os.Process) error {
	return process.Signal(syscall.SIGTERM)
}

func reloadProcess(process *os.Process) error {
	return process.Signal(syscall.SIGHUP)
}
