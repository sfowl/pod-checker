package main

var (
	categorizedNamespaces map[string][]string
)

func init() {
	// high level "group" -> namespace
	categorizedNamespaces = map[string][]string{
		"openshift": []string{
			"apiserver-operator",
			"apiserver",
			"controller-manager-operator",
			"controller-manager",
		},
		"kube": []string{
			"kube-apiserver-operator",
			"kube-apiserver",
			"kube-controller-manager-operator",
			"kube-controller-manager",
			"kube-scheduler-operator",
			"kube-scheduler",
			"etcd-operator",
			"etcd",
		},
		"auth": []string{
			"authentication-operator",
			"authentication",
			"oauth-apiserver",
		},
		"machine management": []string{
			"cluster-machine-approver",
			"cluster-node-tuning-operator",
			"machine-api",
			"machine-config-operator",
		},
		"observability": {
			"monitoring",
		},
		"OLM": {
			"marketplace",
			"operator-lifecycle-manager",
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
		},
		"storage": []string{
			"cluster-storage-operator",
			"kube-storage-version-migrator-operator",
			"kube-storage-version-migrator",
		},
		"console": []string{
			"console-operator",
			"console",
		},
		"other": []string{
			"cloud-controller-manager-operator",
			"cloud-credential-operator",
			"cluster-samples-operator",
			"cluster-version",
			"config-operator",
			"image-registry",
			"insights",
			"service-ca-operator",
			"service-ca",
		},
	}
}
