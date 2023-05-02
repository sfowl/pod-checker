package main

import (
	"encoding/csv"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

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

func getNetworkTraffic(networkCSV *string) []FlowData {
	file, err := os.Open(*networkCSV)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	var flows []FlowData

	// Skip first line
	if _, err := reader.Read(); err != nil {
		log.Fatal(err)
	}

	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		flow := FlowData{
			DstK8S_OwnerName: line[3],
			FlowDirection:    line[4],
			SrcK8S_Namespace: line[5],
			SrcK8S_OwnerName: line[6],
			App:              line[7],
			DstK8S_Namespace: line[8],
			AgentIP:          line[9],
			DstPort:          line[10],
			Etype:            line[11],
			SrcMac:           line[12],
			Proto:            line[13],
			Bytes:            line[14],
			DstK8S_Name:      line[15],
			DstK8S_OwnerType: line[16],
			SrcK8S_Type:      line[17],
			DstAddr:          line[18],
			Duplicate:        line[19],
			DstMac:           line[20],
			SrcK8S_OwnerType: line[21],
			DstK8S_Type:      line[22],
			Packets:          line[23],
			SrcPort:          line[24],
			DstK8S_HostName:  line[25],
			Interface:        line[26],
			Flags:            line[27],
			IfDirection:      line[28],
			DstK8S_HostIP:    line[29],
			SrcAddr:          line[30],
			SrcK8S_Name:      line[31],
		}
		flows = append(flows, flow)
	}

	return flows
}

func addNetworkDataToComponents(components map[string]Component, serviceToComponent map[string]string, flows []FlowData) map[string]Component {
	for _, f := range flows {
		srcNamespace := strings.TrimPrefix(f.SrcK8S_Namespace, "openshift-")
		srcComponentKey := getComponentKey(srcNamespace, f.SrcK8S_OwnerType, f.SrcK8S_OwnerName)
		if v, ok := serviceToComponent[srcComponentKey]; ok {
			srcComponentKey = v
		}

		dstNamespace := strings.TrimPrefix(f.DstK8S_Namespace, "openshift-")
		dstComponentKey := getComponentKey(dstNamespace, f.DstK8S_OwnerType, f.DstK8S_OwnerName)
		if v, ok := serviceToComponent[dstComponentKey]; ok {
			dstComponentKey = v
		}

		if srcComponent, ok := components[srcComponentKey]; !ok {
			log.Warnf("Unknown src component %s\n", srcComponentKey)
			// os.Exit(1)
		} else {
			// fmt.Println("adding", dstComponentKey, "to", srcComponentKey)
			if !slices.Contains(srcComponent.OutgoingConnections, dstComponentKey) {
				srcComponent.OutgoingConnections = append(srcComponent.OutgoingConnections, dstComponentKey)
				components[srcComponentKey] = srcComponent
			}
		}

		if dstComponent, ok := components[dstComponentKey]; !ok {
			log.Warnf("Unknown dst component %s\n", dstComponentKey)
			// os.Exit(1)
		} else {
			if !slices.Contains(dstComponent.IncomingConnections, srcComponentKey) {
				dstComponent.IncomingConnections = append(dstComponent.IncomingConnections, srcComponentKey)
				dstComponent.InboundTraffic = true
				components[dstComponentKey] = dstComponent
			}
		}
	}

	return components
}
