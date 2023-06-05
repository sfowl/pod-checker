package cmdwrapper

import (
	"bufio"
	"io"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

type CmdWrapper struct {
	app    string
	args   []string
	cmd    *exec.Cmd
	stdout io.ReadCloser
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

	if err := cmd.Start(); err != nil {
		return err
	}

	c.cmd = cmd

	log.Debugf("Process started with PID: %d", c.Pid())

	return nil
}

func (c *CmdWrapper) Kill() error {
	err := c.cmd.Process.Kill()

	if err != nil {
		return err
	}
	log.Debugf("Process killed with PID: %d", c.Pid())

	return nil
}

func (c *CmdWrapper) Pid() int {
	return c.cmd.Process.Pid
}

func (c *CmdWrapper) StdOut() error {
	in := bufio.NewScanner(c.stdout)

	for in.Scan() {
		log.Debug(in.Text())
	}

	if err := in.Err(); err != nil {
		log.Errorf("error: %s", err)
	}
	return nil
}

func (c *CmdWrapper) Wait() error {
	if err := c.cmd.Wait(); err != nil {
		return err
	}
	return nil
}
