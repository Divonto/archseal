package archseal

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string             `json:"id"`
	ShortDescription sarifMessageObject `json:"shortDescription"`
}

type sarifMessageObject struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

func (r Report) SARIF() any {
	results := make([]sarifResult, 0, len(r.Violations)+len(r.Cycles))
	for _, v := range r.Violations {
		results = append(results, sarifResult{
			RuleID: "ARCH001",
			Level:  "error",
			Message: sarifMessage{
				Text: v.FromLayer + " must not depend on " + v.ToLayer + " via " + v.ImportPath,
			},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: v.File},
					Region:           sarifRegion{StartLine: v.Line},
				},
			}},
		})
	}
	for _, cycle := range r.Cycles {
		line := 1
		uri := ""
		if len(cycle.Files) > 0 {
			uri = cycle.Files[0]
		}
		results = append(results, sarifResult{
			RuleID: "ARCH002",
			Level:  "error",
			Message: sarifMessage{
				Text: "dependency cycle: " + joinCycle(cycle.Files),
			},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysicalLocation{
					ArtifactLocation: sarifArtifactLocation{URI: uri},
					Region:           sarifRegion{StartLine: line},
				},
			}},
		})
	}

	return sarifLog{
		Version: "2.1.0",
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "Archseal",
				InformationURI: "https://github.com/Divonto/archseal",
				Rules: []sarifRule{
					{ID: "ARCH001", ShortDescription: sarifMessageObject{Text: "Forbidden architecture dependency"}},
					{ID: "ARCH002", ShortDescription: sarifMessageObject{Text: "Dependency cycle"}},
				},
			}},
			Results: results,
		}},
	}
}

func joinCycle(files []string) string {
	if len(files) == 0 {
		return ""
	}
	out := files[0]
	for i := 1; i < len(files); i++ {
		out += " -> " + files[i]
	}
	if len(files) > 1 {
		out += " -> " + files[0]
	}
	return out
}
