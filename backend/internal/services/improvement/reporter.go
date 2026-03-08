package improvement

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// IssueReport represents a reported issue to GitHub
type IssueReport struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	AnalysisID     string    `json:"analysis_id" gorm:"index"`
	IssueID        string    `json:"issue_id"` // Internal issue ID from analysis
	GitHubIssueID  int64     `json:"github_issue_id"`
	GitHubIssueURL string    `json:"github_issue_url"`
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	Labels         []string  `json:"labels" gorm:"serializer:json"`
	Status         string    `json:"status"` // pending, created, failed, closed
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ReporterConfig holds configuration for the issue reporter
type ReporterConfig struct {
	GitHubToken  string
	Owner        string
	Repo         string
	EnableCreate bool     // Whether to actually create issues
	DryRun       bool     // Log only, don't create
	Labels       []string // Default labels for all issues
}

// Reporter handles automatic GitHub issue creation
type Reporter struct {
	db     *gorm.DB
	client *github.Client
	config *ReporterConfig
	logger *log.Logger
}

// NewReporter creates a new GitHub issue reporter
func NewReporter(db *gorm.DB, config *ReporterConfig) *Reporter {
	var client *github.Client

	if config.GitHubToken != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: config.GitHubToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	}

	return &Reporter{
		db:     db,
		client: client,
		config: config,
		logger: log.Default(),
	}
}

// ReportFromAnalysis creates GitHub issues from an analysis result
func (r *Reporter) ReportFromAnalysis(ctx context.Context, analysis *AnalysisResult) ([]IssueReport, error) {
	r.logger.Printf("[Reporter] Creating issues from analysis %s", analysis.ID)

	var reports []IssueReport

	// Filter issues that should be reported (high severity or critical)
	for _, issue := range analysis.Issues {
		if issue.Severity != "high" && issue.Severity != "critical" {
			continue
		}

		// Check if already reported
		if r.isAlreadyReported(issue.ID) {
			r.logger.Printf("[Reporter] Issue %s already reported, skipping", issue.ID)
			continue
		}

		report, err := r.createIssue(ctx, analysis.ID, &issue)
		if err != nil {
			r.logger.Printf("[Reporter] Failed to create issue for %s: %v", issue.ID, err)
			continue
		}

		reports = append(reports, *report)
	}

	// Also report recommendations as enhancement issues
	for _, rec := range analysis.Recommendations {
		if rec.Priority != "high" {
			continue
		}

		// Check if already reported
		if r.isAlreadyReported(rec.ID) {
			continue
		}

		report, err := r.createRecommendationIssue(ctx, analysis.ID, &rec)
		if err != nil {
			r.logger.Printf("[Reporter] Failed to create recommendation issue for %s: %v", rec.ID, err)
			continue
		}

		reports = append(reports, *report)
	}

	r.logger.Printf("[Reporter] Created %d issues from analysis", len(reports))
	return reports, nil
}

// createIssue creates a GitHub issue for an identified issue
func (r *Reporter) createIssue(ctx context.Context, analysisID string, issue *IdentifiedIssue) (*IssueReport, error) {
	title := fmt.Sprintf("[Auto] %s", issue.Title)
	body := r.formatIssueBody(issue)
	labels := r.determineLabels(issue)

	report := &IssueReport{
		ID:         fmt.Sprintf("report_%d", time.Now().UnixNano()),
		AnalysisID: analysisID,
		IssueID:    issue.ID,
		Title:      title,
		Body:       body,
		Labels:     labels,
		Status:     "pending",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if r.config.DryRun {
		r.logger.Printf("[Reporter][DryRun] Would create issue: %s", title)
		report.Status = "dry_run"
	} else if r.client != nil && r.config.EnableCreate {
		ghIssue, _, err := r.client.Issues.Create(ctx, r.config.Owner, r.config.Repo, &github.IssueRequest{
			Title:  &title,
			Body:   &body,
			Labels: &labels,
		})

		if err != nil {
			report.Status = "failed"
			r.db.Create(report)
			return report, fmt.Errorf("failed to create GitHub issue: %w", err)
		}

		report.GitHubIssueID = *ghIssue.ID
		report.GitHubIssueURL = *ghIssue.HTMLURL
		report.Status = "created"
		r.logger.Printf("[Reporter] Created GitHub issue #%d: %s", *ghIssue.Number, title)
	} else {
		report.Status = "disabled"
	}

	r.db.Create(report)
	return report, nil
}

// createRecommendationIssue creates a GitHub issue for a recommendation
func (r *Reporter) createRecommendationIssue(ctx context.Context, analysisID string, rec *Recommendation) (*IssueReport, error) {
	title := fmt.Sprintf("[Auto Enhancement] %s", rec.Title)
	body := r.formatRecommendationBody(rec)
	labels := append(r.config.Labels, "enhancement", "auto-generated")

	report := &IssueReport{
		ID:         fmt.Sprintf("report_%d", time.Now().UnixNano()),
		AnalysisID: analysisID,
		IssueID:    rec.ID,
		Title:      title,
		Body:       body,
		Labels:     labels,
		Status:     "pending",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if r.config.DryRun {
		r.logger.Printf("[Reporter][DryRun] Would create recommendation issue: %s", title)
		report.Status = "dry_run"
	} else if r.client != nil && r.config.EnableCreate {
		ghIssue, _, err := r.client.Issues.Create(ctx, r.config.Owner, r.config.Repo, &github.IssueRequest{
			Title:  &title,
			Body:   &body,
			Labels: &labels,
		})

		if err != nil {
			report.Status = "failed"
			r.db.Create(report)
			return report, fmt.Errorf("failed to create GitHub issue: %w", err)
		}

		report.GitHubIssueID = *ghIssue.ID
		report.GitHubIssueURL = *ghIssue.HTMLURL
		report.Status = "created"
	} else {
		report.Status = "disabled"
	}

	r.db.Create(report)
	return report, nil
}

// formatIssueBody formats the GitHub issue body
func (r *Reporter) formatIssueBody(issue *IdentifiedIssue) string {
	var sb strings.Builder

	sb.WriteString("## Automatically Generated Issue\n\n")
	sb.WriteString("This issue was automatically generated by the self-improvement analysis system.\n\n")

	sb.WriteString("### Description\n")
	sb.WriteString(issue.Description)
	sb.WriteString("\n\n")

	sb.WriteString("### Details\n")
	sb.WriteString(fmt.Sprintf("- **Category:** %s\n", issue.Category))
	sb.WriteString(fmt.Sprintf("- **Severity:** %s\n", issue.Severity))
	sb.WriteString(fmt.Sprintf("- **Frequency:** %d occurrences\n", issue.Frequency))
	if issue.Impact != "" {
		sb.WriteString(fmt.Sprintf("- **Impact:** %s\n", issue.Impact))
	}
	sb.WriteString("\n")

	if len(issue.Examples) > 0 {
		sb.WriteString("### Examples\n")
		for i, ex := range issue.Examples {
			if i >= 3 {
				sb.WriteString(fmt.Sprintf("- ... and %d more\n", len(issue.Examples)-3))
				break
			}
			sb.WriteString(fmt.Sprintf("- %s\n", ex))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("*This issue was automatically created. Please review and prioritize accordingly.*\n")

	return sb.String()
}

// formatRecommendationBody formats the recommendation issue body
func (r *Reporter) formatRecommendationBody(rec *Recommendation) string {
	var sb strings.Builder

	sb.WriteString("## Improvement Recommendation\n\n")
	sb.WriteString("This enhancement was suggested by the self-improvement analysis system.\n\n")

	sb.WriteString("### Description\n")
	sb.WriteString(rec.Description)
	sb.WriteString("\n\n")

	sb.WriteString("### Details\n")
	sb.WriteString(fmt.Sprintf("- **Priority:** %s\n", rec.Priority))
	sb.WriteString(fmt.Sprintf("- **Estimated Effort:** %s\n", rec.Effort))
	sb.WriteString("\n")

	if len(rec.ActionItems) > 0 {
		sb.WriteString("### Suggested Action Items\n")
		for _, item := range rec.ActionItems {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", item))
		}
		sb.WriteString("\n")
	}

	if len(rec.IssueRefs) > 0 {
		sb.WriteString("### Related Issues\n")
		sb.WriteString(fmt.Sprintf("This recommendation addresses: %s\n", strings.Join(rec.IssueRefs, ", ")))
		sb.WriteString("\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("*This enhancement suggestion was automatically created. Please review and prioritize accordingly.*\n")

	return sb.String()
}

// determineLabels determines appropriate labels for an issue
func (r *Reporter) determineLabels(issue *IdentifiedIssue) []string {
	labels := append([]string{}, r.config.Labels...)
	labels = append(labels, "auto-generated")

	// Add category label
	switch issue.Category {
	case "error":
		labels = append(labels, "bug")
	case "performance":
		labels = append(labels, "performance")
	case "quality":
		labels = append(labels, "quality")
	case "usability":
		labels = append(labels, "usability", "enhancement")
	}

	// Add priority label based on severity
	switch issue.Severity {
	case "critical":
		labels = append(labels, "priority: critical")
	case "high":
		labels = append(labels, "priority: high")
	case "medium":
		labels = append(labels, "priority: medium")
	case "low":
		labels = append(labels, "priority: low")
	}

	return labels
}

// isAlreadyReported checks if an issue has already been reported
func (r *Reporter) isAlreadyReported(issueID string) bool {
	var count int64
	r.db.Model(&IssueReport{}).
		Where("issue_id = ? AND status IN ('created', 'pending')", issueID).
		Count(&count)
	return count > 0
}

// GetReports returns all issue reports
func (r *Reporter) GetReports(limit int) ([]IssueReport, error) {
	var reports []IssueReport
	err := r.db.Order("created_at DESC").
		Limit(limit).
		Find(&reports).Error
	return reports, err
}

// GetReportsByAnalysis returns reports for a specific analysis
func (r *Reporter) GetReportsByAnalysis(analysisID string) ([]IssueReport, error) {
	var reports []IssueReport
	err := r.db.Where("analysis_id = ?", analysisID).
		Order("created_at DESC").
		Find(&reports).Error
	return reports, err
}

// UpdateReportStatus updates the status of a report
func (r *Reporter) UpdateReportStatus(reportID, status string) error {
	return r.db.Model(&IssueReport{}).
		Where("id = ?", reportID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// CloseGitHubIssue closes a GitHub issue
func (r *Reporter) CloseGitHubIssue(ctx context.Context, report *IssueReport) error {
	if r.client == nil {
		return fmt.Errorf("GitHub client not configured")
	}

	closed := "closed"
	_, _, err := r.client.Issues.Edit(ctx, r.config.Owner, r.config.Repo, int(report.GitHubIssueID), &github.IssueRequest{
		State: &closed,
	})

	if err != nil {
		return err
	}

	return r.UpdateReportStatus(report.ID, "closed")
}
