package cmdwrapper

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	h "github.com/sfowl/pod-checker/pkg/helpers"
	log "github.com/sirupsen/logrus"
)

type CmdWrapper struct {
	app     string
	args    []string
	cmd     *exec.Cmd
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	logFile string
}

func NewCmdWrapper(app string, args []string) CmdWrapper {
	c := CmdWrapper{}
	c.app = app
	c.args = args

	return c
}

func (c *CmdWrapper) Start() error {

	cmd := exec.Command(c.app, c.args...)
	out, err := cmd.StdoutPipe()

	if err != nil {
		return err
	}

	c.stdout = out

	out, err = cmd.StderrPipe()

	if err != nil {
		return err
	}

	c.stderr = out

	if err := cmd.Start(); err != nil {
		return err
	}

	c.cmd = cmd

	log.Debugf("Process started with PID: %d", c.Pid())

	return nil
}

func (c *CmdWrapper) Kill() error {

	if _, err := os.FindProcess(c.cmd.Process.Pid); err != nil {
		log.Errorf("Failed to find process: %s\n", err)
		return err
	}

	if err := c.cmd.Process.Kill(); err != nil {
		return err
	}

	log.Debugf("Process killed with PID: %d", c.Pid())

	return nil
}

func (c *CmdWrapper) Pid() int {
	return c.cmd.Process.Pid
}

func (c *CmdWrapper) out(r io.Reader, logError bool) error {
	in := bufio.NewScanner(r)

	// Increase scanner buffer, rbac-tool returns pretty long data
	buf := make([]byte, 0, 64*1024)
	in.Buffer(buf, 1024*1024)

	var f *os.File
	var err error

	if c.logFile != "" {
		f, err = os.OpenFile(c.logFile, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()
	}

	for in.Scan() {
		s := in.Text()
		if logError {
			log.Error(s)
		} else {
			log.Info(s)
		}

		if c.logFile != "" {
			if _, err := f.WriteString(s); err != nil {
				log.Errorln(err)
			}
		}

	}

	if err := in.Err(); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *CmdWrapper) StdOut() error {
	return c.out(c.stdout, false)
}

func (c *CmdWrapper) StdErr() error {
	return c.out(c.stderr, true)
}

func (c *CmdWrapper) Wait() error {
	if err := c.cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func (c *CmdWrapper) StdOutToFile(file string) {
	if !h.CheckFileExist(file, fmt.Sprintf("File %s exists, it will not be overwritten. If you want to regenerate it, delete the old report", file)) {
		c.logFile = file
	}
}
