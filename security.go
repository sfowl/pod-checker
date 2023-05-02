package main

import (
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
)

// ComponentSecurityContext is a custom merged struct of
// PodSecurityContext and container SecurityContext. Attributes in container
// SecurityContexts should override the Pod-level attributes.
type ComponentSecurityContext struct {
	// PodSecurityContext attributes
	FSGroup             *int64                         `yaml:"fsGroup,omitempty"`
	FSGroupChangePolicy *corev1.PodFSGroupChangePolicy `yaml:"fsGroupChanagePolicy,omitempty"`
	RunAsNonRoot        bool                           `yaml:"runAsNonRoot"`
	RunAsGroup          *int64                         `yaml:"runAsGroup"`
	RunAsUser           *int64                         `yaml:"runAsUser"`
	SELinuxOptions      *corev1.SELinuxOptions         `yaml:"selinuxOptions,omitempty"`
	SeccompProfile      *corev1.SeccompProfile         `yaml:"seccompProfile,omitempty"`
	SupplementalGroups  []int64                        `yaml:"suppplementalGroups,omitempty"`
	Sysctls             []corev1.Sysctl
	WindowsOptions      *corev1.WindowsSecurityContextOptions `yaml:"windowsOptions,omitempty"`

	// Container SecurityContext attributes
	Capabilities             *corev1.Capabilities  `yaml:"capabilities,omitempty"`
	Privileged               bool                  `yaml:"privileged"`
	ReadOnlyRootFilesystem   bool                  `yaml:"readOnlyRootFilesystem"`
	AllowPrivilegeEscalation bool                  `yaml:"allowPrivilegeEscalation"`
	ProcMount                *corev1.ProcMountType `yaml:"procMount,omitempty"`
}

func (c *ComponentSecurityContext) fromPodSC(sc *corev1.PodSecurityContext) {
	c.FSGroup = sc.FSGroup
	c.FSGroupChangePolicy = sc.FSGroupChangePolicy
	if sc.RunAsNonRoot != nil {
		c.RunAsNonRoot = *sc.RunAsNonRoot
	}
	c.RunAsGroup = sc.RunAsGroup
	c.RunAsUser = sc.RunAsUser
	c.SELinuxOptions = sc.SELinuxOptions
	c.SeccompProfile = sc.SeccompProfile
	c.SupplementalGroups = sc.SupplementalGroups
	c.Sysctls = sc.Sysctls
	c.WindowsOptions = sc.WindowsOptions
}

func (c *ComponentSecurityContext) updateFromContainerSC(sc *corev1.SecurityContext) {
	// Override PodSecurityContext attributes
	if sc.RunAsUser != nil {
		c.RunAsUser = sc.RunAsUser
	}
	if sc.RunAsGroup != nil {
		c.RunAsGroup = sc.RunAsGroup
	}
	if sc.RunAsNonRoot != nil {
		c.RunAsNonRoot = *sc.RunAsNonRoot
	}
	if sc.SELinuxOptions != nil {
		c.SELinuxOptions = sc.SELinuxOptions
	}
	if sc.SeccompProfile != nil {
		c.SeccompProfile = sc.SeccompProfile
	}
	if sc.WindowsOptions != nil {
		c.WindowsOptions = sc.WindowsOptions
	}

	// Container SecurityContext attributes
	if sc.Privileged != nil && *sc.Privileged == true {
		c.Privileged = *sc.Privileged
	}
	if sc.ReadOnlyRootFilesystem != nil && *sc.ReadOnlyRootFilesystem == true {
		c.ReadOnlyRootFilesystem = *sc.ReadOnlyRootFilesystem
	}
	if sc.AllowPrivilegeEscalation != nil && *sc.AllowPrivilegeEscalation == true {
		c.AllowPrivilegeEscalation = *sc.AllowPrivilegeEscalation
	}
	if sc.Capabilities != nil {
		if c.Capabilities == nil {
			c.Capabilities = sc.Capabilities
		} else {
			for _, a := range sc.Capabilities.Add {
				if !slices.Contains(c.Capabilities.Add, a) {
					c.Capabilities.Add = append(c.Capabilities.Add, a)
				}
			}
		}
	}
	// enum
	c.ProcMount = sc.ProcMount
}
