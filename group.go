package main

var (
	categorizedNamespaces map[string][]string
)

func init() {
	// high level "group" -> namespace
	categorizedNamespaces = map[string][]string{
		"kube control plane": []string{
			"kube-apiserver-operator",
			"kube-apiserver",
			"kube-controller-manager-operator",
			"kube-controller-manager",
			"kube-scheduler-operator",
			"kube-scheduler",
			"etcd-operator",
			"etcd",
		},
		"openshift control plane": []string{
			"apiserver-operator",
			"apiserver",
			"controller-manager-operator",
			"controller-manager",
			"cloud-controller-manager-operator",
			"cloud-credential-operator",
			"cluster-version",
			"config-operator",
		},
		"machine management": []string{
			"cluster-machine-approver",
			"cluster-node-tuning-operator",
			"machine-api",
			"machine-config-operator",
		},
		"OLM": {
			"marketplace",
			"operator-lifecycle-manager",
		},
		"auth": []string{
			"authentication-operator",
			"authentication",
			"oauth-apiserver",
		},
		"console": []string{
			"console-operator",
			"console",
		},
		"networking": {
			"dns-operator",
			"dns",
			"multus",
			"network-diagnostics",
			"network-operator",
			"ingress-canary",
			"ingress-operator",
			"ingress",
			"route-controller-manager",
			"sdn",
			"service-ca-operator",
			"service-ca",
		},
		"observability": {
			"monitoring",
		},
		"storage": []string{
			"cluster-storage-operator",
			"kube-storage-version-migrator-operator",
			"kube-storage-version-migrator",
			"image-registry",
		},
		"other": []string{
			"cluster-samples-operator",
			"insights",
		},
	}
}
