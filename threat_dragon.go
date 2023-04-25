package main

type ThreatModel struct {
	Summary Summary `json:"summary"`
	Detail  Detail  `json:"detail"`
	Version string  `json:"version"`
}

type Summary struct {
	Title       string `json:"title"`
	Owner       string `json:"owner"`
	Description string `json:"description"`
	ID          int    `json:"id"`
}

type Detail struct {
	Contributors []struct {
		Name string `json:"name"`
	} `json:"contributors"`
	Diagrams   []Diagram `json:"diagrams"`
	DiagramTop int       `json:"diagramTop"`
	Reviewer   string    `json:"reviewer"`
	ThreatTop  int       `json:"threatTop"`
}

type Diagram struct {
	Cells       []Cell `json:"cells"`
	Version     string `json:"version"`
	Title       string `json:"title"`
	Thumbnail   string `json:"thumbnail"`
	DiagramType string `json:"diagramType"`
	ID          int    `json:"id"`
}

type Attrs struct {
	Text       *Text `json:"text,omitempty"`
	Line       *Line `json:"line,omitempty"`
	TopLine    *Line `json:"topLine,omitempty"`
	BottomLine *Line `json:"bottomLine,omitempty"`
}

type Cell struct {
	Position  *CellPosition `json:"position,omitempty"`
	Size      *CellSize     `json:"size,omitempty"`
	Shape     string        `json:"shape"`
	Attrs     Attrs         `json:"attrs,omitempty"`
	Width     int           `json:"width,omitempty"`
	Height    int           `json:"height,omitempty"`
	ZIndex    int           `json:"zIndex"`
	Connector string        `json:"connector,omitempty"`
	Data      CellData      `json:"data"`
	ID        string        `json:"id"`
	Labels    []struct {
		Position float32 `json:"position,omitempty"`
		Attrs    Attrs   `json:"attrs"`
	} `json:"labels,omitempty"`
	Source   *CellName      `json:"source,omitempty"`
	Target   *CellName      `json:"target,omitempty"`
	Vertices []CellPosition `json:"vertices,omitempty"`
}

type CellPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type CellName struct {
	Cell string `json:"cell,omitempty"`
}

type CellSize struct {
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

type Threat struct {
	Status      string `json:"status"`
	Severity    string `json:"severity"`
	Mitigation  string `json:"mitigation"`
	Description string `json:"description"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	ModelType   string `json:"modelType"`
	ID          string `json:"id"`
}

type CellData struct {
	Type              string   `json:"type"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	OutOfScope        bool     `json:"outOfScope"`
	ReasonOutOfScope  string   `json:"reasonOutOfScope"`
	IsEncrypted       bool     `json:"isEncrypted"`
	HasOpenThreats    bool     `json:"hasOpenThreats"`
	Threats           []Threat `json:"threats"`
	IsALog            bool     `json:"isALog"`
	StoresCredentials bool     `json:"storesCredentials"`
	IsSigned          bool     `json:"isSigned"`
	IsTrustBoundary   bool     `json:"isTrustBoundary"`
}

type Text struct {
	Text string `json:"text,omitempty"`
}

type TargetMarker struct {
	Name string `json:"name,omitempty"`
}

type Line struct {
	Stroke          string        `json:"stroke,omitempty"`
	StrokeWidth     int           `json:"strokeWidth,omitempty"`
	TargetMarker    *TargetMarker `json:"targetMarker,omitempty"`
	StrokeDasharray interface{}   `json:"strokeDasharray,omitempty"`
}
