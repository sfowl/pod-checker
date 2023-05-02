package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

func getComponentEdges(component Component) map[string][]string {
	edges := make(map[string][]string)
	for _, s := range component.OutgoingConnections {
		edgeName := fmt.Sprintf("%s to %s", component.Key(), s)
		if _, ok := edges[edgeName]; !ok {
			edges[edgeName] = []string{component.Key(), s}
		}
	}

	for _, s := range component.IncomingConnections {
		edgeName := fmt.Sprintf("%s to %s", s, component.Key())
		if _, ok := edges[edgeName]; !ok {
			edges[edgeName] = []string{s, component.Key()}
		}
	}

	return edges
}

func generateDiagram(group string, id int, components map[string]Component) Diagram {
	diagram := Diagram{
		Cells:       make([]Cell, 0),
		ID:          id,
		Title:       fmt.Sprintf("%s diagram (test)", group),
		DiagramType: "Generic",
		Version:     "2.0.1",
	}

	count := 0
	edges := make(map[string][]string)
	filteredComponents := make(map[string]Component)
	for k, c := range components {
		if c.Group != group {
			continue
		}

		filteredComponents[k] = c

		componentEdges := getComponentEdges(c)
		for edgeKey, edge := range componentEdges {
			////  log.Warnf("checking edge: %v", edge)
			nodesMatched := true
			for _, componentKey := range edge {
				// log.Warnf("checking ckey: %v", componentKey)
				if _, ok := filteredComponents[componentKey]; !ok {
					if externalComponent, ok := components[componentKey]; ok {
						filteredComponents[componentKey] = externalComponent
					} else {
						log.Warnf("edge points to unknown component: %s", componentKey)
						nodesMatched = false
					}
				}
			}
			if nodesMatched {
				edges[edgeKey] = edge
			}
		}
	}

	// byNamespaces := make(map[string][]Cell)

	for _, c := range filteredComponents {
		cell := Cell{
			Position: &CellPosition{
				X: 0 + ((count*100)%800)*2,
				Y: 0 + (100*(count/8))*2,
			},
			Size: &CellSize{
				Width:  160,
				Height: 80,
			},
			Attrs: Attrs{
				Text: &Text{
					Text: c.Name,
				},
				TopLine: &Line{
					Stroke:      "red",
					StrokeWidth: 3,
				},
				BottomLine: &Line{
					Stroke:      "red",
					StrokeWidth: 3,
				},
			},
			Shape: "process",
			// XXX uuid?
			ID:     c.Key(),
			ZIndex: 1,
			Data: CellData{
				Name:    c.Name,
				Type:    "tm.Process",
				Threats: []Threat{},
			},
		}
		diagram.Cells = append(diagram.Cells, cell)

		// if _, ok := byNamespac[c.Namespace]; ok {
		// 	byNamespace[c.Namespace] = append(byNamespace[c.Namespace], cell)
		// } else {
		// 	byNamespace[c.Namespace] = []Cell{cell}
		// }

		count++
	}

	for e, edge := range edges {
		cell := Cell{
			Attrs: Attrs{
				Line: &Line{
					Stroke: "red",
					TargetMarker: &TargetMarker{
						Name: "classic",
					},
					StrokeWidth: 1,
				},
			},
			Shape: "flow",
			// XXX uuid?
			ID:        e,
			ZIndex:    10,
			Width:     200,
			Height:    100,
			Connector: "smooth",
			Data: CellData{
				Name:    e,
				Type:    "tm.Flow",
				Threats: []Threat{},
			},
			Source: &CellName{
				Cell: edge[0],
			},
			Target: &CellName{
				Cell: edge[1],
			},
			Vertices: []CellPosition{},
		}

		diagram.Cells = append(diagram.Cells, cell)
	}

	return diagram
}

func generateThreatModel(components map[string]Component, outFilename string, excludedGroups []string) {
	tm := ThreatModel{
		Version: "2.0.1",
		Summary: Summary{
			Title:       title,
			Description: "test threat model",
		},
	}

	diagrams := make([]Diagram, 0)
	id := 0
	for group, _ := range categorizedNamespaces {
		if slices.Contains(excludedGroups, group) {
			continue
		}

		diagram := generateDiagram(group, id, components)
		diagrams = append(diagrams, diagram)
		id++
	}

	tm.Detail = Detail{Diagrams: diagrams}

	file, _ := json.MarshalIndent(tm, "", " ")

	_ = ioutil.WriteFile(outFilename, file, 0644)
}
