package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	routeclientv1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"
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

const title = "OpenShift Threat Model"

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

// func (c Component) String() string {
// 	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%v", c.Namespace, c.Name, c.DeployedAs, c.RunsOn, c.SecurityContext, c.HostNetwork)
// }

func getSecurityInfo(p corev1.Pod) (string, ComponentSecurityContext) {
	scc := ""
	securityContext := ComponentSecurityContext{}
	securityContext.fromPodSC(p.Spec.SecurityContext)
	if _, ok := p.ObjectMeta.Annotations["openshift.io/scc"]; ok {
		scc = p.ObjectMeta.Annotations["openshift.io/scc"]
	}
	for _, c := range p.Spec.Containers {
		if c.SecurityContext != nil {
			if c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
				scc = "privileged"
			}
			securityContext.updateFromContainerSC(c.SecurityContext)
		}
	}
	return scc, securityContext
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
				// XXX inefficent, should use map/set
				if !slices.Contains(hostMounts, m.MountPath) {
					hostMounts = append(hostMounts, m.MountPath)
				}
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
		componentKey = "kube control plane/kube-apiserver/StaticPods/kube-apiserver"
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
		scc, securityContext := getSecurityInfo(p)
		podServices := getServices(p, clusterData.ServicesByNamespace)
		for _, s := range podServices {
			serviceToComponent[fmt.Sprintf("%s/%s/Service/%s", group, namespace, s.Name)] = componentKey
		}

		c.RunsOn = runsOn
		if strings.HasSuffix(c.Name, "operator") {
			c.IsOperator = true
		}

		c.PriorityClass = p.Spec.PriorityClassName

		if c.SCC == "" && scc != "" {
			c.SCC = scc
			c.SecurityContext = securityContext
		}
		if !c.HostIPC && p.Spec.HostIPC {
			c.HostIPC = p.Spec.HostIPC
		}
		if !c.HostNetwork && p.Spec.HostNetwork {
			c.HostNetwork = p.Spec.HostNetwork
		}
		if !c.HostNetwork && p.Spec.HostNetwork {
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

	// components := filterComponents(components, excludedGroups)

	printCSV(components, "example/output/components.tsv")
	// fmt.Printf("\nThere are %d pods\n", numPods)

	threagileReport := genThreagile(components)
	threagileYAML := marshalYAML(threagileReport)
	writeYAML(threagileYAML, "example/output/threagile_input.yaml")

	componentYAML := marshalYAML(components)
	writeYAML(componentYAML, "example/output/components.yaml")
	// also write one component per file
	writeComponents(components, "example/output/components")

	survey := genSurvey(components)
	surveyYAML := marshalYAML(survey)
	writeYAML(surveyYAML, "example/output/survey.yaml")
	writeSurvey(components, "example/output/survey")

	// generateThreatModel(components, "example/output/threat_dragon.json", excludedGroups)
}

// func filterComponents(components map[string]Component, excludedGroups []string) {
//
// }

func writeComponents(components map[string]Component, outputDir string) {
	for _, c := range components {
		componentYAML := marshalYAML(c)
		dir := fmt.Sprintf("%s/%s", outputDir, strings.ReplaceAll(c.Group, " ", "_"))
		os.MkdirAll(dir, 0755)
		writeYAML(componentYAML, fmt.Sprintf("%s/%s.yaml", dir, c.Name))
	}
}

func writeSurvey(components map[string]Component, outputDir string) {
	for k, c := range components {
		survey := genSurvey(map[string]Component{k: c})
		surveyYAML := marshalYAML(survey[0])
		dir := fmt.Sprintf("%s/%s", outputDir, strings.ReplaceAll(c.Group, " ", "_"))
		os.MkdirAll(dir, 0755)
		writeYAML(surveyYAML, fmt.Sprintf("%s/%s.yaml", dir, c.Name))
	}
}

func writeYAML(yamlData []byte, outputFile string) {
	err := ioutil.WriteFile(outputFile, yamlData, 0644)
	if err != nil {
		log.Errorf("Unable to write yaml data to %s: %s", outputFile, err)
	}
}

func marshalYAML(in interface{}) []byte {
	yamlData, err := yaml.Marshal(&in)
	if err != nil {
		panic(fmt.Errorf("Error while Marshaling. %v", err))
	}
	return yamlData
}

func printCSV(components map[string]Component, outputFile string) {
	f, err := os.Create(outputFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	writer.Comma = '\t'

	keys := make([]string, 0, len(components))
	for k := range components {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var c Component
	printValues(writer, []string{
		"Group",
		"Namespace",
		"Name",
		"DeployedAs",
		"RunsOn",
		"IsOperator",
		"Default SCC",
		"RunLevel",
		"HostIPC",
		"HostNetwork",
		"HostPID",
		// SecurityContext section
		"RunAsNonRoot",
		"RunAsGroup",
		"RunAsUser",
		"SELinuxOptions",
		"SeccompProfile",
		"Capabilities",
		"Privileged",
		"ReadOnlyRootFilesystem",
		"AllowPrivilegeEscalation",
		// end SecurityContext section
		"PriorityClass",
		"InboundTraffic?",
		"ExternallyExposed?",
		"IncomingConnections",
		"OutgoingConnections",
		"HostMounts",
	})
	for _, k := range keys {
		c = components[k]
		printValues(writer, c.Values())
	}

	writer.Flush()

}
