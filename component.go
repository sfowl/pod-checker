package main

import (
	"fmt"
	"strconv"
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
)

type Component struct {
	Name                string
	Namespace           string
	Group               string
	DeployedAs          string                   `yaml:"deployedAs"`
	RunsOn              string                   `yaml:"runsOn"`
	IsOperator          bool                     `yaml:"IsOperator"`
	SecurityContext     ComponentSecurityContext `yaml:"securityContext"`
	SCC                 string
	RunLevel            string           `yaml:"runLevel"`
	HostIPC             bool             `yaml:"hostIPC"`
	HostNetwork         bool             `yaml:"hostNetwork"`
	HostPID             bool             `yaml:"hostPID"`
	InboundTraffic      bool             `yaml:"inboundTraffic"`
	ExternallyExposed   bool             `yaml:"externallyExposed"`
	IncomingConnections []string         `yaml:"incomingConnections"`
	OutgoingConnections []string         `yaml:"outgoingConnections"`
	HostMounts          []string         `yaml:"hostMounts"`
	Pods                []corev1.Pod     `yaml:"-"`
	Services            []corev1.Service `yaml:"-"`
	Routes              []routev1.Route  `yaml:"-"`
}

func (c Component) Key() string {
	return strings.Join([]string{c.Group, c.Namespace, c.DeployedAs, c.Name}, "/")
}

func (c Component) Values() []string {
	runAsGroup := ""
	if c.SecurityContext.RunAsGroup != nil {
		runAsGroup = fmt.Sprintf("%d", uint64(*c.SecurityContext.RunAsGroup))
	}
	runAsUser := ""
	if c.SecurityContext.RunAsUser != nil {
		runAsUser = fmt.Sprintf("%d", uint64(*c.SecurityContext.RunAsUser))
	}
	return []string{
		c.Group,
		c.Namespace,
		c.Name,
		c.DeployedAs,
		c.RunsOn,
		strconv.FormatBool(c.IsOperator),
		c.SCC,
		c.RunLevel,
		strconv.FormatBool(c.HostIPC),
		strconv.FormatBool(c.HostNetwork),
		strconv.FormatBool(c.HostPID),
		strconv.FormatBool(c.SecurityContext.RunAsNonRoot),
		runAsGroup,
		runAsUser,
		c.SecurityContext.SELinuxOptions.String(),
		c.SecurityContext.SeccompProfile.String(),
		c.SecurityContext.Capabilities.String(),
		strconv.FormatBool(c.SecurityContext.Privileged),
		strconv.FormatBool(c.SecurityContext.ReadOnlyRootFilesystem),
		strconv.FormatBool(c.SecurityContext.AllowPrivilegeEscalation),
		strconv.FormatBool(c.InboundTraffic),
		strconv.FormatBool(c.ExternallyExposed),
		strings.Join(c.IncomingConnections, ","),
		strings.Join(c.OutgoingConnections, ","),
		strings.Join(c.HostMounts, ","),
	}
}
