package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	routeclientv1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func init() {
	// Only log the warning severity or above.
	log.SetLevel(log.WarnLevel)
}

type ClusterData struct {
	Namespaces          map[string]corev1.Namespace
	Pods                []corev1.Pod
	ServicesByNamespace map[string][]corev1.Service
	ReplicaSets         []appsv1.ReplicaSet
	Routes              []routev1.Route
}

type Component struct {
	Name                string
	Namespace           string
	Group               string
	DeployedAs          string
	RunsOn              string
	SCC                 string
	RunLevel            string
	HostNetwork         bool
	InboundTraffic      bool
	ExternallyExposed   bool
	IncomingConnections []string
	OutgoingConnections []string
	HostMounts          []string
	Pods                []corev1.Pod
	Services            []corev1.Service
	Routes              []routev1.Route
}

type FlowData struct {
	DstK8S_OwnerName string
	FlowDirection    string
	SrcK8S_Namespace string
	SrcK8S_OwnerName string
	App              string
	DstK8S_Namespace string
	AgentIP          string
	DstPort          string
	Etype            string
	SrcMac           string
	Proto            string
	Bytes            string
	DstK8S_Name      string
	DstK8S_OwnerType string
	SrcK8S_Type      string
	DstAddr          string
	Duplicate        string
	DstMac           string
	SrcK8S_OwnerType string
	DstK8S_Type      string
	Packets          string
	SrcPort          string
	DstK8S_HostName  string
	Interface        string
	Flags            string
	IfDirection      string
	DstK8S_HostIP    string
	SrcAddr          string
	SrcK8S_Name      string
}

// XXX this is super inefficient, lazy. Maybe should invert the map.
func getGroup(namespace string) string {
	for g, namespaces := range categorizedNamespaces {
		for _, n := range namespaces {
			if namespace == n {
				return g
			}
		}
	}
	return "other"
}

func printValues(writer *csv.Writer, values []string) {
	if err := writer.Write(values); err != nil {
		log.Fatalln("error writing output", err)
	}
}

func (c Component) Key() string {
	return strings.Join([]string{c.Group, c.Namespace, c.DeployedAs, c.Name}, "/")
}

func (c Component) Values() []string {
	return []string{
		c.Group,
		c.Namespace,
		c.Name,
		c.DeployedAs,
		c.RunsOn,
		c.SCC,
		c.RunLevel,
		strconv.FormatBool(c.HostNetwork),
		strconv.FormatBool(c.InboundTraffic),
		strconv.FormatBool(c.ExternallyExposed),
		strings.Join(c.HostMounts, ","),
		strings.Join(c.IncomingConnections, ","),
		strings.Join(c.OutgoingConnections, ","),
	}
}

// func (c Component) String() string {
// 	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%v", c.Namespace, c.Name, c.DeployedAs, c.RunsOn, c.SCC, c.HostNetwork)
// }

func getSCC(p corev1.Pod) string {
	if scc, ok := p.ObjectMeta.Annotations["openshift.io/scc"]; ok {
		return scc
	}
	for _, c := range p.Spec.Containers {
		if c.SecurityContext != nil {
			if c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
				return "privileged"
			}
		}
	}
	return ""
}

func getDeployedNodes(p corev1.Pod, ownerKind string) string {
	runsOn := ""
	for k, _ := range p.Spec.NodeSelector {
		if strings.HasPrefix(k, "node-role") {
			parts := strings.Split(k, "/")
			runsOn = parts[len(parts)-1]
		}
	}

	if runsOn == "" && (ownerKind == "DaemonSet" || ownerKind == "StatefulSet") {
		runsOn = "all"
	}
	if runsOn == "" && ownerKind == "Deployment" {
		runsOn = "worker"
	}
	if runsOn == "" && ownerKind == "StaticPods" {
		runsOn = strings.Split(p.Spec.NodeName, "-")[0]
	}
	runsOn += " nodes"

	return runsOn
}

func probePortMatches(probe *corev1.Probe, servPort int) bool {
	if probe != nil && probe.HTTPGet != nil && probe.HTTPGet.Port.IntValue() == servPort {
		return true
	}
	return false
}

func getHostMounts(p corev1.Pod) []string {
	hostMounts := []string{}
	for _, c := range p.Spec.Containers {
		for _, m := range c.VolumeMounts {
			if !strings.HasPrefix(m.MountPath, "/var") {
				hostMounts = append(hostMounts, m.MountPath)
			}
		}
	}

	return hostMounts
}

// getServices that select a given pod, excluding metrics only services
func getServices(pod corev1.Pod, services map[string][]corev1.Service) []corev1.Service {
	matching := []corev1.Service{}
	sList := services[pod.Namespace]
	for _, s := range sList {
		labelsMatched := true
		ignore := false

		if strings.Contains(s.Name, "metrics") {
			continue
		}

		for k, v := range s.Spec.Selector {
			if pod.Labels[k] != v {
				labelsMatched = false
			}
		}
		if labelsMatched {
			for _, servicePort := range s.Spec.Ports {
				servPort := servicePort.TargetPort.IntValue()
				for _, container := range pod.Spec.Containers {
					ignore = probePortMatches(container.LivenessProbe, servPort)
					ignore = ignore || probePortMatches(container.ReadinessProbe, servPort)
					ignore = ignore || probePortMatches(container.StartupProbe, servPort)

					for _, containerPort := range container.Ports {
						if containerPort.ContainerPort == servicePort.TargetPort.IntVal {
							if !strings.Contains(containerPort.Name, "metrics") {
								ignore = false
								break
							}
						}
					}
				}
			}

			if !ignore {
				matching = append(matching, s)
			}
		}
	}

	return matching
}

func getRoutes(service corev1.Service, routeList []routev1.Route) []routev1.Route {
	matching := []routev1.Route{}
	for _, r := range routeList {
		if r.Namespace == service.Namespace {
			if r.Spec.To.Kind == "Service" && r.Spec.To.Name == service.Name {
				matching = append(matching, r)
			}
		}
	}

	return matching
}

func getClusterData() ClusterData {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	routev1Client, err := routeclientv1.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pods, err := clientset.CoreV1().Pods("").List(
		context.TODO(),
		metav1.ListOptions{
			FieldSelector: "status.phase=Running",
		},
	)
	if err != nil {
		panic(err.Error())
	}
	rs, err := clientset.AppsV1().ReplicaSets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	ns, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	namespaces := make(map[string]corev1.Namespace)
	for _, n := range ns.Items {
		namespaces[n.Name] = n
	}

	svcs, err := clientset.CoreV1().Services("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	services := make(map[string][]corev1.Service)
	for _, s := range svcs.Items {
		if _, ok := services[s.Namespace]; ok {
			services[s.Namespace] = append(services[s.Namespace], s)
		} else {
			services[s.Namespace] = []corev1.Service{s}
		}
	}

	routes, err := routev1Client.Routes("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	// deploys, err := clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	// if err != nil {
	// 	panic(err.Error())
	// }

	return ClusterData{
		Namespaces:          namespaces,
		Pods:                pods.Items,
		ReplicaSets:         rs.Items,
		Routes:              routes.Items,
		ServicesByNamespace: services,
	}
}

func getComponentKey(namespace string, ownerType string, ownerName string) string {
	if ownerType == "ConfigMap" {
		ownerType = "StaticPods"
		// heuristic based transformation for static pods
		if strings.HasPrefix(ownerName, "revision-status") {
			ownerName = namespace
		}
	}

	group := getGroup(strings.TrimPrefix(namespace, "openshift-"))
	componentKey := strings.Join([]string{group, namespace, ownerType, ownerName}, "/")

	// XXX special case
	if componentKey == "other/default/Service/kubernetes" {
		componentKey = "kube/kube-apiserver/StaticPods/kube-apiserver"
	}

	return componentKey
}

func main() {
	log.SetLevel(log.DebugLevel)

	networkCSV := flag.String("network-csv", "", "Path to the CSV file")
	exclude := flag.String("exclude", "", "list of groups to exclude (comma separated)")
	flag.Parse()

	// if *networkCSV == "" {
	// 	fmt.Println("Usage:", os.Args[0], "--network-csv <filename.csv>")
	// 	os.Exit(1)
	// }

	excludedGroups := strings.Split(*exclude, ",")

	clusterData := getClusterData()

	serviceToComponent := make(map[string]string)
	components := make(map[string]Component)
	for _, p := range clusterData.Pods {
		if appLabel, ok := p.Labels["app"]; ok && appLabel == "guard" {
			// guard pods have no owners, not interested in these
			continue
		}

		if len(p.OwnerReferences) != 1 {
			panic(fmt.Sprintf("Expected one owner, got %d", len(p.OwnerReferences)))
		}
		owner := p.OwnerReferences[0]
		ownerKey := ""
		ownerKind := owner.Kind
		ownerName := owner.Name
		if owner.Kind == "ReplicaSet" {
			for _, r := range clusterData.ReplicaSets {
				if r.Name == owner.Name {
					ownerName = r.OwnerReferences[0].Name
					ownerKind = r.OwnerReferences[0].Kind
				}
			}
		}
		if ownerKind == "Node" {
			name := strings.TrimPrefix(p.Labels["app"], "openshift-")
			ownerKey = fmt.Sprintf("StaticPods/%s", name)
			ownerKind = "StaticPods"
			ownerName = name
		} else {
			ownerKey = fmt.Sprintf("%s/%s", ownerKind, ownerName)
		}

		namespace := strings.TrimPrefix(p.GetNamespace(), "openshift-")
		group := getGroup(strings.TrimPrefix(namespace, "openshift-"))
		if slices.Contains(excludedGroups, group) {
			log.Debugf("Skipping %s due to exclusions", ownerKey)
			continue
		}

		componentKey := fmt.Sprintf("%s/%s/%s", group, namespace, ownerKey)
		hostMounts := getHostMounts(p)
		runLevel := clusterData.Namespaces[p.Namespace].Labels["openshift.io/run-level"]
		var c Component
		var ok bool
		if c, ok = components[componentKey]; !ok {
			c = Component{
				Name: ownerName,
				// everything starts with 'openshift-', noisy
				Namespace:   strings.TrimPrefix(p.Namespace, "openshift-"),
				Group:       group,
				DeployedAs:  ownerKind,
				HostNetwork: p.Spec.HostNetwork,
				HostMounts:  hostMounts,
				RunLevel:    runLevel,
				Pods:        []corev1.Pod{p},
			}
		} else {
			c.Pods = append(c.Pods, p)
			c.HostMounts = append(hostMounts)
		}

		runsOn := getDeployedNodes(p, ownerKind)
		scc := getSCC(p)
		podServices := getServices(p, clusterData.ServicesByNamespace)
		for _, s := range podServices {
			serviceToComponent[fmt.Sprintf("%s/%s/Service/%s", group, namespace, s.Name)] = componentKey
		}

		// TODO update component
		c.RunsOn = runsOn
		if c.SCC == "" && scc != "" {
			c.SCC = scc
		}
		if !c.HostNetwork && c.HostNetwork {
			c.HostNetwork = p.Spec.HostNetwork
		}
		if len(podServices) > 0 {
			c.InboundTraffic = true
			// c.Services = append(c.Services, podServices)
			for _, ps := range podServices {
				serviceRoutes := getRoutes(ps, clusterData.Routes)
				if len(serviceRoutes) > 0 {
					c.ExternallyExposed = true
					// c.Routes = append(c.Routes, serviceRoutes)
				}
			}
		}

		components[componentKey] = c
	}

	if *networkCSV != "" {
		flowData := getNetworkTraffic(networkCSV)
		components = addNetworkDataToComponents(components, serviceToComponent, flowData)
	} else {
		log.Warn("Can't generate a threat model diagram without network data")
	}

	// printResults(components, len(clusterData.Pods))

	generateThreatModel(components, "output.json", excludedGroups)
}

func printResults(components map[string]Component, numPods int) {
	writer := csv.NewWriter(os.Stdout)
	writer.Comma = '\t'

	keys := make([]string, 0, len(components))
	for k := range components {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var c Component
	printValues(writer, []string{"Group", "Namespace", "Name", "DeployedAs", "RunsOn", "SCC", "RunLevel", "HostNetwork", "InboundTraffic?", "ExternallyExposed?", "HostMounts", "IncomingConnections", "OutgoingConnections"})
	for _, k := range keys {
		c = components[k]
		printValues(writer, c.Values())
	}

	writer.Flush()

	// fmt.Printf("\nThere are %d pods\n", numPods)
}
