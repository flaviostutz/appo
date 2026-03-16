package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCollectReachableVulnerabilitiesPrefersSymbolFindings(t *testing.T) {
	messages := []govulnMessage{
		{OSV: &govulnOSV{ID: "GO-1", Aliases: []string{"CVE-1"}}},
		{OSV: &govulnOSV{ID: "GO-2", Aliases: []string{"CVE-2"}}},
		{Finding: &govulnFinding{OSV: "GO-1", Trace: []govulnFrame{{Function: "DoThing"}}}},
		{Finding: &govulnFinding{OSV: "GO-2", Trace: []govulnFrame{{}}}},
	}

	vulnerabilities := collectReachableVulnerabilities(messages)
	if len(vulnerabilities) != 1 {
		t.Fatalf("expected 1 reachable vulnerability, got %d", len(vulnerabilities))
	}
	if _, ok := vulnerabilities["GO-1"]; !ok {
		t.Fatalf("expected GO-1 to be reachable")
	}
}

func TestAssessVulnerabilitiesUsesHighestAliasSeverity(t *testing.T) {
	vulnerabilities := map[string]vulnerability{
		"GO-1": {ID: "GO-1", Summary: "critical vuln", Aliases: []string{"CVE-1", "CVE-2"}},
		"GO-2": {ID: "GO-2", Summary: "high vuln", Aliases: []string{"CVE-3"}},
		"GO-3": {ID: "GO-3"},
	}
	aliasSeverities := map[string]severityRank{
		"CVE-1": severityMedium,
		"CVE-2": severityCritical,
		"CVE-3": severityHigh,
	}

	result := assessVulnerabilities(vulnerabilities, aliasSeverities)
	if len(result.Critical) != 1 || result.Critical[0].ID != "GO-1" {
		t.Fatalf("expected GO-1 to be critical, got %#v", result.Critical)
	}
	if len(result.High) != 1 || result.High[0].ID != "GO-2" {
		t.Fatalf("expected GO-2 to be high, got %#v", result.High)
	}
	if len(result.Unknown) != 1 || result.Unknown[0].ID != "GO-3" {
		t.Fatalf("expected GO-3 to be unknown, got %#v", result.Unknown)
	}
}

func TestParseSeverityLevel(t *testing.T) {
	rank, err := parseSeverityLevel("HIGH")
	if err != nil {
		t.Fatalf("expected high to parse: %v", err)
	}
	if rank != severityHigh {
		t.Fatalf("expected high rank, got %v", rank)
	}

	if _, err := parseSeverityLevel("bogus"); err == nil {
		t.Fatal("expected invalid severity level to fail")
	}
}

func TestWriteAssessmentOnlyShowsMatchedDetails(t *testing.T) {
	result := assessment{
		Critical: []evaluatedVulnerability{{ID: "GO-1", Summary: "critical vuln", Severity: severityCritical}},
		Medium:   []evaluatedVulnerability{{ID: "GO-2", Summary: "medium vuln", Severity: severityMedium}},
		Low:      []evaluatedVulnerability{{ID: "GO-3", Summary: "low vuln", Severity: severityLow}},
	}

	var output bytes.Buffer
	writeAssessment(&output, result, severityHigh, nil)

	text := output.String()
	if !strings.Contains(text, "fail-on>=high matched=1") {
		t.Fatalf("expected summary to mention high threshold and one match, got %q", text)
	}
	if !strings.Contains(text, "[critical] GO-1: critical vuln") {
		t.Fatalf("expected critical detail to be shown, got %q", text)
	}
	if strings.Contains(text, "GO-2") || strings.Contains(text, "GO-3") {
		t.Fatalf("expected below-threshold details to be omitted, got %q", text)
	}
}
