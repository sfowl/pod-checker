package sslchecker

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"time"

	w "github.com/sfowl/pod-checker/pkg/cmdwrapper"
	n "github.com/sfowl/pod-checker/pkg/netutils"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

type SslChecker struct {
	namespace   string
	serviceName string
	port        int32
	reportDir   string
	cmds        []w.CmdWrapper
}

func NewSslChecker(namespace string, serviceName string, port int32, reportDir string) SslChecker {
	c := SslChecker{}
	c.namespace = namespace
	c.serviceName = serviceName
	c.port = port
	c.reportDir = reportDir

	return c
}

func SslCheckerForServices(namespace string, serviceName string, ports []corev1.ServicePort) {
	dir, err := ioutil.TempDir("/tmp", "sslchecker")
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Ssl checker reports directory: %s", dir)
	for _, p := range ports {
		s := NewSslChecker(namespace, serviceName, p.Port, dir)
		log.Infof("Ssl checker starting for %s:%d", s.fqdnSvc(), p.Port)
		s.Run()
	}

}

func (c *SslChecker) fqdnSvc() string {
	return fmt.Sprintf("%s.%s.svc", c.serviceName, c.namespace)
}

func (c *SslChecker) Run() {

	localPort, err := n.GetFreePort()
	c.cleanUp()

	if err != nil {
		panic(err)
	}

	gateway := n.GetDefaultOutboundIP().String()
	svc := fmt.Sprintf("%s:%d", c.fqdnSvc(), c.port)

	log.Infof("Starting port forward for service %s:%d in %s:%d", svc, c.port, gateway, localPort)

	portForwardArgs := []string{
		"port-forward",
		"--address", gateway,
		"-n", c.namespace,
		fmt.Sprintf("svc/%s", c.serviceName),
		fmt.Sprintf("%d:%d", localPort, c.port),
	}

	portForward := w.NewCmdWrapper("oc", portForwardArgs)

	if err := portForward.Start(); err != nil {
		panic(err)
	}

	log.Infof("Port forward for service %s started successfully", svc)

	c.cmds = append(c.cmds, portForward)
	// Wait for port-forward to start the server
	<-time.After(2 * time.Second)

	testSslArgs := []string{
		"run", "--rm", "-t",
		"--add-host", fmt.Sprintf("%s:%s", c.fqdnSvc(), gateway),
		"-v", fmt.Sprintf("%s:/reports:Z", c.reportDir),
		//@FIXME: quick solution to fix permissions issues with volumes mounted on rootless containers
		// ref: https://www.redhat.com/sysadmin/debug-rootless-podman-mounted-volumes
		"--user", "root",
		"drwetter/testssl.sh",
		"--jsonfile", fmt.Sprintf("/reports/%s_%d.json", c.fqdnSvc(), c.port),
		fmt.Sprintf("%s:%d", c.fqdnSvc(), localPort),
	}

	log.Infof("Starting sslchecker for service %s", svc)

	testssl := w.NewCmdWrapper("podman", testSslArgs)

	if err := testssl.Start(); err != nil {
		panic(err)
	}
	c.cmds = append(c.cmds, testssl)

	if err := testssl.StdOut(); err != nil {
		panic(err)
	}

	if err := testssl.Wait(); err != nil {
		panic(err)
	}

	log.Infof("Finished sslchecker for service %s", svc)

	if err := portForward.Kill(); err != nil {
		panic(err)
	}
}

func (c *SslChecker) cleanUp() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		for range sigChan {
			for _, cmd := range c.cmds {
				cmd.Kill()
			}
			os.Exit(1)
		}
	}()
}
