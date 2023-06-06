package cmdwrapper

import (
	"bufio"
	"io"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

type CmdWrapper struct {
	app    string
	args   []string
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
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

	for in.Scan() {
		if logError {
			log.Error(in.Text())
		} else {
			log.Info(in.Text())
		}
	}

	if err := in.Err(); err != nil {
		log.Errorf("%s", err)
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
