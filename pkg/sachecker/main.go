package sachecker

import (
	"fmt"
	"os"

	w "github.com/sfowl/pod-checker/pkg/cmdwrapper"
	h "github.com/sfowl/pod-checker/pkg/helpers"
	log "github.com/sirupsen/logrus"
)

type SAChecker struct {
	namespace          string
	serviceAccountName string
	group              string
}

func NewSAChecker(namespace string, serviceAccountName string, group string) SAChecker {
	c := SAChecker{}
	c.namespace = namespace
	c.serviceAccountName = serviceAccountName
	c.group = group

	return c
}

func (c *SAChecker) Run() {
	c.runRBACTool()
	c.getSATokens()
}

func (c *SAChecker) runRBACTool() {
	rbacToolArgs := []string{
		"rbac-tool",
		"policy-rules",
		"--output", "json",
		"-e", fmt.Sprintf("^system:serviceaccounts:%s$", c.serviceAccountName),
	}

	log.Infof("Starting rbac-tool for service account %s in namespace: %s", c.serviceAccountName, c.namespace)

	c.runOC(rbacToolArgs, "rbac")

	log.Infof("Finished execution of rbac-tool for service account %s", c.serviceAccountName)
}

func (c *SAChecker) getSATokens() {
	getSAParams := []string{
		"get",
		"sa",
		"--output", "jsonpath={range .secrets[*]}{.name}{\"\\n\"}{.end}",
		"-n", c.namespace,
		c.serviceAccountName,
	}

	log.Infof("Getting tokens for service account %s in namespace: %s", c.serviceAccountName, c.namespace)

	c.runOC(getSAParams, "sa_tokens")

	log.Infof("Finished getting tokens for service account %s", c.serviceAccountName)
}

func (c *SAChecker) runOC(args []string, verification string) {
	w := w.NewCmdWrapper("oc", args)

	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	reportDir := fmt.Sprintf("%s/example/output/%s_reports/%s", path, verification, h.CanonicalGroup(c.group))
	if err := os.MkdirAll(reportDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	w.StdOutToFile(fmt.Sprintf("%s/sa_%s_%s.json", reportDir, c.serviceAccountName, c.namespace))

	if err := w.Start(); err != nil {
		log.Fatal(err)
	}

	if err := w.StdOut(); err != nil {
		log.Fatal(err)
	}

	if err := w.StdErr(); err != nil {
		log.Fatal(err)
	}

	if err := w.Wait(); err != nil {
		log.Fatal(err)
	}
}
