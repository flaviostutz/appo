package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	govulncheckVersion = "v1.1.4"
	cveAPIBaseURL      = "https://cveawg.mitre.org/api/cve/"
)

type severityRank int

const (
	severityUnknown severityRank = iota
	severityLow
	severityMedium
	severityHigh
	severityCritical
)

type govulnMessage struct {
	OSV     *govulnOSV     `json:"osv,omitempty"`
	Finding *govulnFinding `json:"finding,omitempty"`
}

type govulnOSV struct {
	ID       string        `json:"id"`
	Aliases  []string      `json:"aliases,omitempty"`
	Summary  string        `json:"summary,omitempty"`
	Severity []osvSeverity `json:"severity,omitempty"`
}

type osvSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

type govulnFinding struct {
	OSV   string        `json:"osv,omitempty"`
	Trace []govulnFrame `json:"trace,omitempty"`
}

type govulnFrame struct {
	Function string `json:"function,omitempty"`
	Receiver string `json:"receiver,omitempty"`
}

type vulnerability struct {
	ID         string
	Aliases    []string
	Summary    string
	DirectRank severityRank
}

type evaluatedVulnerability struct {
	ID       string
	Aliases  []string
	Summary  string
	Severity severityRank
}

type assessment struct {
	Critical []evaluatedVulnerability
	High     []evaluatedVulnerability
	Medium   []evaluatedVulnerability
	Low      []evaluatedVulnerability
	Unscored []evaluatedVulnerability
	Unknown  []evaluatedVulnerability
	Matched  []evaluatedVulnerability
}

type auditConfig struct {
	FailLevel severityRank
	Patterns  []string
}

type cveRecord struct {
	Containers struct {
		CNA cveContainer   `json:"cna"`
		ADP []cveContainer `json:"adp"`
	} `json:"containers"`
}

type cveContainer struct {
	Metrics []cveMetric `json:"metrics"`
}

type cveMetric struct {
	CVSSV20 *cvssMetric `json:"cvssV2_0,omitempty"`
	CVSSV30 *cvssMetric `json:"cvssV3_0,omitempty"`
	CVSSV31 *cvssMetric `json:"cvssV3_1,omitempty"`
	CVSSV40 *cvssMetric `json:"cvssV4_0,omitempty"`
}

type cvssMetric struct {
	BaseSeverity string  `json:"baseSeverity"`
	BaseScore    float64 `json:"baseScore"`
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:]))
}

func run(ctx context.Context, args []string) int {
	config, err := parseConfig(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse audit config: %v\n", err)
		return 1
	}
	config.Patterns = expandPatterns(config.Patterns)

	findings, err := runGovulncheckJSON(ctx, config.Patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect govulncheck findings: %v\n", err)
		return 1
	}

	vulnerabilities := collectReachableVulnerabilities(findings)
	aliasSeverities, unresolved, err := resolveAliasSeverities(ctx, vulnerabilities)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve vulnerability severities: %v\n", err)
		return 1
	}

	result := assessVulnerabilities(vulnerabilities, aliasSeverities)
	result.Matched = matchedVulnerabilities(result, config.FailLevel)
	writeAssessment(os.Stderr, result, config.FailLevel, unresolved)
	if len(result.Matched) > 0 {
		return 1
	}
	return 0
}

func parseConfig(args []string) (auditConfig, error) {
	flagSet := flag.NewFlagSet("vulnaudit", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	failLevelFlag := flagSet.String("fail-level", severityCritical.String(), "minimum vulnerability severity that fails the audit")
	if err := flagSet.Parse(args); err != nil {
		return auditConfig{}, err
	}
	failLevel, err := parseSeverityLevel(*failLevelFlag)
	if err != nil {
		return auditConfig{}, err
	}
	patterns := flagSet.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	return auditConfig{FailLevel: failLevel, Patterns: patterns}, nil
}

func expandPatterns(patterns []string) []string {
	args := append([]string{"list"}, patterns...)
	cmd := exec.Command("go", args...)
	output, err := cmd.Output()
	if err != nil {
		return patterns
	}

	packages := strings.Fields(string(output))
	filtered := make([]string, 0, len(packages))
	for _, pkg := range packages {
		if strings.HasSuffix(pkg, "/cli/vulnaudit") {
			continue
		}
		filtered = append(filtered, pkg)
	}
	if len(filtered) == 0 {
		return patterns
	}
	return filtered
}

func runGovulncheckJSON(ctx context.Context, patterns []string) ([]govulnMessage, error) {
	args := append([]string{"run", "golang.org/x/vuln/cmd/govulncheck@" + govulncheckVersion, "-format", "json"}, patterns...)
	cmd := exec.CommandContext(ctx, "go", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}

	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	var messages []govulnMessage
	for {
		var msg govulnMessage
		if err := decoder.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}
func collectReachableVulnerabilities(messages []govulnMessage) map[string]vulnerability {
	osvByID := map[string]govulnOSV{}
	allFindings := map[string]struct{}{}
	symbolFindings := map[string]struct{}{}

	for _, msg := range messages {
		if msg.OSV != nil && msg.OSV.ID != "" {
			osvByID[msg.OSV.ID] = *msg.OSV
		}
		if msg.Finding == nil || msg.Finding.OSV == "" {
			continue
		}
		allFindings[msg.Finding.OSV] = struct{}{}
		if hasSymbolTrace(msg.Finding.Trace) {
			symbolFindings[msg.Finding.OSV] = struct{}{}
		}
	}

	considered := allFindings
	if len(symbolFindings) > 0 {
		considered = symbolFindings
	}

	vulnerabilities := map[string]vulnerability{}
	for osvID := range considered {
		entry := osvByID[osvID]
		vulnerabilities[osvID] = vulnerability{
			ID:         osvID,
			Aliases:    append([]string(nil), entry.Aliases...),
			Summary:    entry.Summary,
			DirectRank: rankOSVSeverity(entry.Severity),
		}
	}
	return vulnerabilities
}

func hasSymbolTrace(trace []govulnFrame) bool {
	for _, frame := range trace {
		if frame.Function != "" || frame.Receiver != "" {
			return true
		}
	}
	return false
}

func rankOSVSeverity(entries []osvSeverity) severityRank {
	best := severityUnknown
	for _, entry := range entries {
		rank := rankSeverity(entry.Score)
		if rank == severityUnknown {
			rank = rankCVSSScore(entry.Score)
		}
		if rank > best {
			best = rank
		}
	}
	return best
}

func resolveAliasSeverities(ctx context.Context, vulnerabilities map[string]vulnerability) (map[string]severityRank, []string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	aliasSeverities := map[string]severityRank{}
	var unresolved []string

	for _, vuln := range vulnerabilities {
		for _, alias := range vuln.Aliases {
			if _, ok := aliasSeverities[alias]; ok {
				continue
			}
			if !strings.HasPrefix(alias, "CVE-") {
				continue
			}
			rank, err := fetchCVESeverity(ctx, client, alias)
			if err != nil {
				unresolved = append(unresolved, alias)
				continue
			}
			aliasSeverities[alias] = rank
		}
	}

	sort.Strings(unresolved)
	return aliasSeverities, unresolved, nil
}

func fetchCVESeverity(ctx context.Context, client *http.Client, alias string) (severityRank, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, cveAPIBaseURL+alias, nil)
	if err != nil {
		return severityUnknown, err
	}
	response, err := client.Do(request)
	if err != nil {
		return severityUnknown, err
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode != http.StatusOK {
		return severityUnknown, fmt.Errorf("%s returned %s", alias, response.Status)
	}

	var record cveRecord
	if err := json.NewDecoder(response.Body).Decode(&record); err != nil {
		return severityUnknown, err
	}

	best := severityUnknown
	for _, metric := range record.Containers.CNA.Metrics {
		best = maxSeverity(best, rankCVSSMetric(metric))
	}
	for _, container := range record.Containers.ADP {
		for _, metric := range container.Metrics {
			best = maxSeverity(best, rankCVSSMetric(metric))
		}
	}
	return best, nil
}

func rankCVSSMetric(metric cveMetric) severityRank {
	for _, value := range []*cvssMetric{metric.CVSSV40, metric.CVSSV31, metric.CVSSV30, metric.CVSSV20} {
		if value == nil {
			continue
		}
		rank := rankSeverity(value.BaseSeverity)
		if rank != severityUnknown {
			return rank
		}
		rank = rankCVSSBaseScore(value.BaseScore)
		if rank != severityUnknown {
			return rank
		}
	}
	return severityUnknown
}

func assessVulnerabilities(vulnerabilities map[string]vulnerability, aliasSeverities map[string]severityRank) assessment {
	result := assessment{}
	ids := sortedKeys(vulnerabilities)
	for _, id := range ids {
		vuln := vulnerabilities[id]
		best := vuln.DirectRank
		for _, alias := range vuln.Aliases {
			best = maxSeverity(best, aliasSeverities[alias])
		}
		evaluated := evaluatedVulnerability{
			ID:       vuln.ID,
			Aliases:  append([]string(nil), vuln.Aliases...),
			Summary:  vuln.Summary,
			Severity: best,
		}
		switch best {
		case severityCritical:
			result.Critical = append(result.Critical, evaluated)
		case severityHigh, severityMedium, severityLow:
			switch best {
			case severityHigh:
				result.High = append(result.High, evaluated)
			case severityMedium:
				result.Medium = append(result.Medium, evaluated)
			case severityLow:
				result.Low = append(result.Low, evaluated)
			}
		case severityUnknown:
			if len(vuln.Aliases) == 0 {
				result.Unknown = append(result.Unknown, evaluated)
			} else {
				result.Unscored = append(result.Unscored, evaluated)
			}
		}
	}
	return result
}

func writeAssessment(output io.Writer, result assessment, failLevel severityRank, unresolved []string) {
	matched := result.Matched
	if len(matched) == 0 {
		matched = matchedVulnerabilities(result, failLevel)
	}
	total := len(result.Critical) + len(result.High) + len(result.Medium) + len(result.Low) + len(result.Unscored) + len(result.Unknown)
	_, _ = fmt.Fprintf(output,
		"severity-gated audit summary: total=%d critical=%d high=%d medium=%d low=%d unscored=%d unknown=%d fail-on>=%s matched=%d\n",
		total,
		len(result.Critical),
		len(result.High),
		len(result.Medium),
		len(result.Low),
		len(result.Unscored),
		len(result.Unknown),
		failLevel,
		len(matched),
	)
	for _, vuln := range matched {
		_, _ = fmt.Fprintf(output, "- [%s] %s: %s\n", vuln.Severity, vuln.ID, vulnerabilitySummary(vuln))
	}
	if len(unresolved) > 0 {
		_, _ = fmt.Fprintf(output, "severity lookup skipped for aliases: %s\n", strings.Join(unresolved, ", "))
	}
}

func sortedKeys(values map[string]vulnerability) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func maxSeverity(left, right severityRank) severityRank {
	if right > left {
		return right
	}
	return left
}

func (rank severityRank) String() string {
	switch rank {
	case severityLow:
		return "low"
	case severityMedium:
		return "medium"
	case severityHigh:
		return "high"
	case severityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

func parseSeverityLevel(value string) (severityRank, error) {
	rank := rankSeverity(value)
	if rank == severityUnknown {
		return severityUnknown, fmt.Errorf("unsupported fail level %q; expected one of: low, medium, high, critical", value)
	}
	return rank, nil
}

func rankSeverity(value string) severityRank {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "LOW":
		return severityLow
	case "MEDIUM", "MODERATE":
		return severityMedium
	case "HIGH":
		return severityHigh
	case "CRITICAL":
		return severityCritical
	default:
		return severityUnknown
	}
}

func rankCVSSScore(value string) severityRank {
	if value == "" {
		return severityUnknown
	}
	if score, err := strconv.ParseFloat(value, 64); err == nil {
		return rankCVSSBaseScore(score)
	}
	return severityUnknown
}

func rankCVSSBaseScore(score float64) severityRank {
	switch {
	case score >= 9.0:
		return severityCritical
	case score >= 7.0:
		return severityHigh
	case score >= 4.0:
		return severityMedium
	case score > 0:
		return severityLow
	default:
		return severityUnknown
	}
}

func matchedVulnerabilities(result assessment, failLevel severityRank) []evaluatedVulnerability {
	matched := make([]evaluatedVulnerability, 0)
	for _, vulnerabilities := range [][]evaluatedVulnerability{result.Critical, result.High, result.Medium, result.Low} {
		for _, vuln := range vulnerabilities {
			if vuln.Severity >= failLevel {
				matched = append(matched, vuln)
			}
		}
	}
	sort.Slice(matched, func(left, right int) bool {
		if matched[left].Severity != matched[right].Severity {
			return matched[left].Severity > matched[right].Severity
		}
		return matched[left].ID < matched[right].ID
	})
	return matched
}

func vulnerabilitySummary(vuln evaluatedVulnerability) string {
	if strings.TrimSpace(vuln.Summary) != "" {
		return vuln.Summary
	}
	if len(vuln.Aliases) > 0 {
		return strings.Join(vuln.Aliases, ", ")
	}
	return "no summary available"
}
