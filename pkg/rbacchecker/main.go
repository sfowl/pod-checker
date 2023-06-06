package rbacchecker

import (
	"fmt"
	"os"

	w "github.com/sfowl/pod-checker/pkg/cmdwrapper"
	h "github.com/sfowl/pod-checker/pkg/helpers"
	log "github.com/sirupsen/logrus"
)

type RBACChecker struct {
	namespace          string
	serviceAccountName string
	group              string
}

func NewRBACChecker(namespace string, serviceAccountName string, group string) RBACChecker {
	c := RBACChecker{}
	c.namespace = namespace
	c.serviceAccountName = serviceAccountName
	c.group = group

	return c
}

func (c *RBACChecker) Run() {
	rbacToolArgs := []string{
		"rbac-tool",
		"policy-rules",
		"--output", "json",
		"-e", fmt.Sprintf("^%s$", c.serviceAccountName),
	}

	log.Infof("Starting rbac-tool for service account %s in namespace: %s", c.serviceAccountName, c.namespace)

	rbacTool := w.NewCmdWrapper("oc", rbacToolArgs)

	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	reportDir := fmt.Sprintf("%s/example/output/rbac_reports/%s", path, h.CanonicalGroup(c.group))
	if err := os.MkdirAll(reportDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	rbacTool.StdOutToFile(fmt.Sprintf("%s/sa_%s_%s.json", reportDir, c.serviceAccountName, c.namespace))

	if err := rbacTool.Start(); err != nil {
		log.Fatal(err)
	}

	if err := rbacTool.StdOut(); err != nil {
		log.Fatal(err)
	}

	if err := rbacTool.StdErr(); err != nil {
		log.Fatal(err)
	}

	if err := rbacTool.Wait(); err != nil {
		log.Fatal(err)
	}

	log.Infof("Finished execution of rbac-tool for service account %s", c.serviceAccountName)
}
