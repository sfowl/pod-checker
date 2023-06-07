package sslchecker

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	w "github.com/sfowl/pod-checker/pkg/cmdwrapper"
	h "github.com/sfowl/pod-checker/pkg/helpers"
	n "github.com/sfowl/pod-checker/pkg/netutils"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const containerReportsDir string = "/reports"

type SslChecker struct {
	namespace     string
	serviceName   string
	port          int32
	hostReportDir string
	cmds          []w.CmdWrapper
}

func NewSslChecker(namespace string, serviceName string, port int32, hostReportDir string) SslChecker {
	c := SslChecker{}
	c.namespace = namespace
	c.serviceName = serviceName
	c.port = port
	c.hostReportDir = hostReportDir

	return c
}

func SslCheckerForServices(namespace string, serviceName string, ports []corev1.ServicePort, group string) {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	reportDir := fmt.Sprintf("%s/example/output/ssl_reports/%s", path, h.CanonicalGroup(group))

	if err := os.MkdirAll(reportDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	log.Infof("Ssl checker reports directory: %s", reportDir)
	for _, p := range ports {
		s := NewSslChecker(namespace, serviceName, p.Port, reportDir)
		log.Infof("Ssl checker starting for %s:%d", s.fqdnSvc(), p.Port)
		s.Run()
	}

}

func (c *SslChecker) fqdnSvc() string {
	return fmt.Sprintf("%s.%s.svc", c.serviceName, c.namespace)
}

func (c *SslChecker) Run() {

	localPort, err := n.GetFreePort()
	c.cleanUpOnSignal()

	if err != nil {
		panic(err)
	}

	// Support accessing to local host interfaces from a rootless container
	// This address will be used as a gateway between the host network and the sslchecker container's network
	gateway := n.GetDefaultOutboundIP().String()
	svc := fmt.Sprintf("%s:%d", c.fqdnSvc(), c.port)

	hostReportFile := c.reportFile(c.hostReportDir)

	if h.CheckFileExist(hostReportFile, fmt.Sprintf("Report file %s exists, it will not be overwritten. If you want to regenerate it, delete the old report", hostReportFile)) {
		return
	}

	log.Infof("Starting port forward for service %s in %s:%d", svc, gateway, localPort)

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
		"-v", fmt.Sprintf("%s:%s:Z", c.hostReportDir, containerReportsDir),
		"--network", "slirp4netns:allow_host_loopback=true",
		//@FIXME: quick solution to fix permissions issues with volumes mounted on rootless containers
		// ref: https://www.redhat.com/sysadmin/debug-rootless-podman-mounted-volumes
		"--user", "root",
		"drwetter/testssl.sh",
		"--jsonfile", c.reportFile(containerReportsDir),
		fmt.Sprintf("%s:%d", c.fqdnSvc(), localPort),
	}

	log.Infof("Starting sslchecker for service %s", svc)

	testssl := w.NewCmdWrapper("podman", testSslArgs)

	if err := testssl.Start(); err != nil {
		c.panic(err)
	}

	c.cmds = append(c.cmds, testssl)

	if err := testssl.StdOut(); err != nil {
		c.panic(err)
	}

	if err := testssl.StdErr(); err != nil {
		c.panic(err)
	}

	if err := testssl.Wait(); err != nil {
		c.panic(err)
	}

	log.Infof("Finished sslchecker for service %s", svc)

	if err := portForward.Kill(); err != nil {
		panic(err)
	}
}

func (c *SslChecker) cleanUp() {
	for _, cmd := range c.cmds {
		cmd.Kill()
	}
}

func (c *SslChecker) panic(err any) {
	c.cleanUp()
	panic(err)
}

func (c *SslChecker) cleanUpOnSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		for range sigChan {
			c.cleanUp()
			os.Exit(1)
		}
	}()
}

func (c *SslChecker) reportFile(dir string) string {
	return fmt.Sprintf("%s/%s_%d.json", dir, c.fqdnSvc(), c.port)
}
