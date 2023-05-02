package main

import (
	"fmt"
	"strings"
	"time"

	tm "github.com/threagile/threagile/model"
)

func convertID(id string) string {
	// slash, space characters not allowed in ID
	return strings.ReplaceAll(strings.ReplaceAll(id, "/", "-"), " ", "-")
}

func genThreagile(components map[string]Component) tm.ModelInput {
	report := tm.ModelInput{
		Title:                title,
		Threagile_version:    tm.ThreagileVersion,
		Date:                 time.Now().String()[:10],
		Business_criticality: "important", // required
		Tags_available:       []string{"privileged", "hostNetwork"},
		// at least one data asset
		Data_assets: map[string]tm.InputDataAsset{
			"some-data": tm.InputDataAsset{
				ID:              "some-data",
				Usage:           "devops",       // required
				Quantity:        "many",         // required
				Confidentiality: "confidential", // required
				Integrity:       "operational",  // required
				Availability:    "operational",  // required
			},
		},
	}

	technicalAssets := make(map[string]tm.InputTechnicalAsset)
	for id, c := range components {
		assetID := convertID(id)
		ta := tm.InputTechnicalAsset{
			ID:                     assetID,
			Type:                   "process",
			Usage:                  "devops",
			Size:                   "component",
			Machine:                "container",
			Internet:               c.ExternallyExposed,
			Custom_developed_parts: true,
			Technology:             "unknown-technology", // required
			Encryption:             "none",               // required
			Confidentiality:        "confidential",       // required
			Integrity:              "operational",        // required
			Availability:           "operational",        // required
			Data_assets_processed:  []string{"some-data"},
		}
		comms := make(map[string]tm.InputCommunicationLink)
		for _, o := range c.OutgoingConnections {
			if _, ok := components[o]; !ok {
				// can't add links to unknown assets
				continue
			}

			targetID := convertID(o)
			l := tm.InputCommunicationLink{
				Target:         targetID,
				Authentication: "none",             // required
				Authorization:  "none",             // required
				Usage:          "devops",           // required
				Protocol:       "unknown-protocol", // required
			}

			comms[targetID] = l
		}
		ta.Communication_links = comms
		fmt.Println("--coms---")
		fmt.Println(len(comms))
		fmt.Println("--coms---")

		tags := make([]string, 0)
		if c.SCC == "privileged" {
			tags = append(tags, c.SCC)
		}
		if c.HostNetwork == true {
			tags = append(tags, "hostNetwork")
		}
		ta.Tags = tags

		technicalAssets[assetID] = ta
	}
	report.Technical_assets = technicalAssets
	fmt.Println("---")
	fmt.Println(len(report.Technical_assets))
	fmt.Println("---")

	return report
}
