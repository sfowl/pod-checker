package main

import (
	"time"

	tm "github.com/threagile/threagile/model"
)

func genThreagile(components map[string]Component) tm.ModelInput {
	report := tm.ModelInput{
		Title:             title,
		Threagile_version: tm.ThreagileVersion,
		Date:              time.Now().Format(time.RFC822Z),
	}

	technicalAssets := make(map[string]tm.InputTechnicalAsset)
	for id, c := range components {
		ta := tm.InputTechnicalAsset{
			ID:                     id,
			Type:                   "process",
			Usage:                  "devops",
			Size:                   "component",
			Machine:                "container",
			Internet:               c.ExternallyExposed,
			Custom_developed_parts: true,
		}
		comms := make(map[string]tm.InputCommunicationLink)
		for _, o := range c.OutgoingConnections {
			l := tm.InputCommunicationLink{
				Target: o,
			}

			comms[o] = l
		}
		ta.Communication_links = comms

		tags := make([]string, 0)
		if c.SCC == "privileged" {
			tags = append(tags, c.SCC)
		}
		if c.HostNetwork == true {
			tags = append(tags, "hostNetwork")
		}
		ta.Tags = tags

		technicalAssets[id] = ta
	}
	report.Technical_assets = technicalAssets

	return report
}
