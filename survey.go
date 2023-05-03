package main

import (
	"fmt"
	"strings"
)

type Question struct {
	Question string
	Answer   string
	Hint     *string `yaml:"hint,omitempty"`
}

type Topic struct {
	Name      string
	Questions []Question
}

type SurveyComponent struct {
	Intro                  Topic
	Communications         Topic
	RBACandServiceAccounts Topic
	Encryption             Topic
	Authentication         Topic
	SecretManagement       Topic
	Logging                Topic
	PodSecurity            Topic
	SecurityContext        Topic
	Volumes                Topic
	DenialOfService        Topic
	NetworkPolicies        Topic
	Operators              Topic `yaml:"operators,omitempty"`
}

func genSurvey(components map[string]Component) []SurveyComponent {
	survey := make([]SurveyComponent, 0)

	for _, c := range components {
		s := SurveyComponent{}
		operatorHint := "Answer Yes → Go to “Operators” section"
		s.Intro = Topic{
			Name: "Intro",
			Questions: []Question{
				Question{
					Question: "Component Name",
					Answer:   c.Name,
				},
				Question{
					Question: "Owner(s)",
				},
				Question{
					Question: "Description",
				},
				Question{
					Question: "Functional Area",
					Answer:   c.Group,
				},
				Question{
					Question: "Is the component an operator ?",
					Answer:   fmt.Sprintf("%t", c.IsOperator),
					Hint:     &operatorHint,
				},
				Question{
					Question: "Document any sensitive data that the component might process",
				},
			},
		}
		outboundHint := "Even peer-to-peer communication within the same cluster"
		s.Communications = Topic{
			Name: "Communications",
			Questions: []Question{
				Question{
					Question: "Does the component present inbound non-encrypted interfaces (eg, HTTP) ? ",
				},
				Question{
					Question: "Does the component enforce encryption on outbound communications (eg. HTTPS, TLS, etc) ? ",
					Hint:     &outboundHint,
				},
				Question{
					Question: "Are the component's communications restricted by network policies, either ingress or egress?",
					Answer:   "no",
				},
			},
		}
		s.RBACandServiceAccounts = Topic{
			Name: "RBAC and Service Accounts",
			Questions: []Question{
				Question{
					Question: "Does the service account bound to the component use short-lived tokens? ",
				},
				Question{
					Question: "Is there any long-lived token bound to the service account ?",
				},
				Question{
					Question: "Is there any use of wildcards in the Roles and/or ClusterRoles assigned to the service account bound to the component? Both in the namespace field and in the permissions field",
				},
			},
		}
		s.Encryption = Topic{
			Name: "Encryption",
			Questions: []Question{
				Question{
					Question: "Does the component expose by default any non-secure cipher suite ?",
				},
				Question{
					Question: "Does the component disable certificate validation for any communication ?",
				},
				Question{
					Question: "Does the component expose a self-signed certificate ?",
				},
				Question{
					Question: "Does the component perform any cryptographic operations such as encryption, decryption, signing, verification, etc",
				},
				Question{
					Question: "Does the component expose any certificates that are not automatically renewed periodically?",
				},
			},
		}
		s.SecretManagement = Topic{
			Name: "Secret Management",
			Questions: []Question{
				Question{
					Question: "What means are used to pass secrets or credentials to the component's configuration ?",
				},
			},
		}
		s.Logging = Topic{
			Name: "Logging / Audit",
			Questions: []Question{
				Question{
					Question: "Are all the actions performed by the component audited with enough information to uniquely identify when, who and what action was performed ?",
				},
				Question{
					Question: "Is all the sensitive information masked or hashed in the logs ?",
				},
			},
		}
		seccompProfile := ""
		if c.SecurityContext.SeccompProfile != nil {
			seccompProfile = c.SecurityContext.SeccompProfile.String()
		}
		extraCaps := "no"
		if c.SecurityContext.Capabilities != nil && len(c.SecurityContext.Capabilities.Add) > 0 {
			extraCaps = fmt.Sprintf("yes: %v", c.SecurityContext.Capabilities.Add)
		}
		restrictedHint := "Answer No → Go to Security Context Section"
		isRestricted := "false"
		if c.SCC == "restricted-v2" {
			isRestricted = "true"
		}
		s.PodSecurity = Topic{
			Name: "Pod Security Profile",
			Questions: []Question{
				Question{
					Question: "Does the namespace where the component is implemented have a runlevel assigned to it?",
					Answer:   c.RunLevel,
				},
				Question{
					Question: "Do the pods that are part of the component have their security context restricted with the \"restricted-v2\" SCC? ",
					Hint:     &restrictedHint,
					Answer:   isRestricted,
				},
				Question{
					Question: "Does the component specify a SecComp profile ?",
					Answer:   seccompProfile,
				},
				Question{
					Question: "Does the component specify custom SeLinux options",
					Answer:   fmt.Sprintf("%s", c.SecurityContext.SELinuxOptions.String()),
				},
			},
		}
		ACLs := ""
		if c.SecurityContext.RunAsUser != nil {
			ACLs += fmt.Sprintf("runAsUser %d ", uint64(*c.SecurityContext.RunAsUser))
		}
		if c.SecurityContext.RunAsGroup != nil {
			ACLs += fmt.Sprintf("runAsGroup %d ", uint64(*c.SecurityContext.RunAsGroup))
		}
		if c.SecurityContext.FSGroup != nil {
			ACLs += fmt.Sprintf("FSGroup %d ", uint64(*c.SecurityContext.FSGroup))
		}
		s.SecurityContext = Topic{
			Name: "Security Context",
			Questions: []Question{
				Question{
					Question: "Does the component share any of the following host's namespaces ?",
					Answer:   fmt.Sprintf("hostIPC %t, hostNetwork %t, hostPID %t", c.HostIPC, c.HostNetwork, c.HostPID),
				},
				Question{
					Question: "Does the component enable privilege escalation by setting a \"true\" allowPrivilegeEscalation ?",
					Answer:   fmt.Sprintf("%t", c.SecurityContext.AllowPrivilegeEscalation),
				},
				Question{
					Question: "Do all containers run as non-root by enabling runAsNonRoot ?",
					Answer:   fmt.Sprintf("%t", c.SecurityContext.RunAsNonRoot),
				},
				Question{
					Question: "Do all containers run as non-privileged users ?",
					Answer:   ACLs,
				},
				Question{
					Question: "Does the component add additional Linux kernel capabilities to the default ones ? ",
					Answer:   extraCaps,
				},
				Question{
					Question: "Does the component unmask the /proc filesystem ? ",
					Answer:   fmt.Sprintf("%v", c.SecurityContext.ProcMount),
				},
			},
		}
		s.Volumes = Topic{
			Name: "Volumes",
			Questions: []Question{
				Question{
					Question: "Does the component mount, either read or read/write, any of the following \"sensitive\" hostPaths ?",
					Answer:   strings.Join(c.HostMounts, ", "),
				},
			},
		}
		survey = append(survey, s)
	}

	return survey
}
