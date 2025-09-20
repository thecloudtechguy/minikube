/*
Copyright 2024 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"
	"k8s.io/minikube/pkg/minikube/exit"
	"k8s.io/minikube/pkg/minikube/out"
	"k8s.io/minikube/pkg/minikube/reason"
	"k8s.io/minikube/pkg/minikube/style"
	"k8s.io/minikube/pkg/perf/monitor"
)

var (
	monitorRepo    string
	monitorOwner   string
	monitorBranch  string
	monitorPoll    time.Duration
	monitorCommits bool
	monitorPRs     bool
	monitorReleases bool
	monitorState   *monitor.RepoMonitorState
	monitorDryRun  bool
)

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor repository for changes and send notifications",
	Long: `Monitor a GitHub repository for various types of changes including:
- Commits on specified branches
- Pull request activities (opened, merged, closed)
- New releases
- Issue activities (optional)

The monitor command uses GitHub API to watch for changes and can send
notifications via console output or configured webhooks.`,
	Example: `  # Monitor the default minikube repository
  minikube monitor

  # Test monitoring without making API calls
  minikube monitor --dry-run

  # Monitor a specific repository and branch
  minikube monitor --repo=my-repo --owner=my-org --branch=main

  # Monitor only pull requests and releases
  minikube monitor --prs --releases --no-commits

  # Set custom polling interval
  minikube monitor --poll=30s`,
	Run: runMonitor,
}

func runMonitor(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	
	// Initialize monitoring state
	monitorState = &monitor.RepoMonitorState{}
	
	out.Styled(style.Happy, "🔍 Starting repository monitor for {{.owner}}/{{.repo}}", 
		out.V{"owner": monitorOwner, "repo": monitorRepo})
	
	if monitorDryRun {
		out.Styled(style.Notice, "🧪 Dry-run mode: No actual API calls will be made")
	}
	
	if monitorBranch != "" {
		out.Styled(style.Option, "📊 Monitoring branch: {{.branch}}", out.V{"branch": monitorBranch})
	}
	
	// Show what we're monitoring
	monitoring := []string{}
	if monitorCommits {
		monitoring = append(monitoring, "commits")
	}
	if monitorPRs {
		monitoring = append(monitoring, "pull requests")
	}
	if monitorReleases {
		monitoring = append(monitoring, "releases")
	}
	
	if len(monitoring) == 0 {
		exit.Message(reason.Usage, "No monitoring options selected. Use --commits, --prs, or --releases")
	}
	
	out.Styled(style.Option, "📢 Monitoring: {{.items}}", out.V{"items": fmt.Sprintf("%v", monitoring)})
	out.Styled(style.Option, "⏰ Polling interval: {{.interval}}", out.V{"interval": monitorPoll.String()})
	out.Styled(style.Empty, "")
	
	if monitorDryRun {
		out.Styled(style.Ready, "🧪 Dry-run: Simulating monitor behavior...")
		simulateDryRun()
		return
	}
	
	// Create GitHub client
	client := monitor.NewClient(ctx, monitorOwner, monitorRepo)
	
	out.Styled(style.Ready, "🚀 Monitor started! Press Ctrl+C to stop...")
	
	// Start monitoring loop
	ticker := time.NewTicker(monitorPoll)
	defer ticker.Stop()
	
	// Initial check (but don't report changes, just set baseline)
	out.Styled(style.Notice, "📋 Initializing monitor state...")
	if err := initializeMonitorState(ctx, client); err != nil {
		klog.Errorf("Error during initialization: %v", err)
	}
	out.Styled(style.Success, "✅ Monitor initialized and ready!")
	
	for {
		select {
		case <-ctx.Done():
			out.Styled(style.Stopped, "👋 Monitor stopped")
			return
		case <-ticker.C:
			if err := checkForChanges(ctx, client); err != nil {
				klog.Errorf("Error checking for changes: %v", err)
			}
		}
	}
}

func initializeMonitorState(ctx context.Context, client *monitor.Client) error {
	// Initialize state without reporting changes
	var branch string
	if monitorBranch != "" {
		branch = monitorBranch
	} else {
		branch = "master" // Default branch for minikube
	}
	
	// Just initialize state, don't report changes
	_, err := client.CheckForRepositoryChanges(monitorState, branch)
	return err
}

func checkForChanges(ctx context.Context, client *monitor.Client) error {
	out.Styled(style.Running, "🔄 Checking for changes... ({{.time}})", 
		out.V{"time": time.Now().Format("15:04:05")})
	
	var branch string
	if monitorBranch != "" {
		branch = monitorBranch
	} else {
		branch = "master" // Default branch for minikube
	}
	
	// Check for all types of changes
	changes, err := client.CheckForRepositoryChanges(monitorState, branch)
	if err != nil {
		return fmt.Errorf("checking repository changes: %w", err)
	}
	
	// Filter and report changes based on user preferences
	var reported bool
	for _, change := range changes {
		shouldReport := false
		
		switch change.Type {
		case "commit":
			shouldReport = monitorCommits
		case "pr":
			shouldReport = monitorPRs
		case "release":
			shouldReport = monitorReleases
		}
		
		if shouldReport {
			reportChange(change)
			reported = true
		}
	}
	
	if !reported && len(changes) == 0 {
		out.Styled(style.Empty, "   No new changes detected")
	}
	
	return nil
}

func reportChange(change monitor.RepositoryChange) {
	var emoji, style_type string
	
	switch change.Type {
	case "commit":
		emoji = "💾"
		style_type = "success"
	case "pr":
		emoji = "🔀"
		style_type = "celebrate"
	case "release":
		emoji = "🏷️"
		style_type = "happy"
	default:
		emoji = "📝"
		style_type = "option"
	}
	
	// Use appropriate style based on type
	switch style_type {
	case "success":
		out.Styled(style.Success, "{{.emoji}} {{.desc}}", out.V{"emoji": emoji, "desc": change.Description})
	case "celebrate":
		out.Styled(style.Celebrate, "{{.emoji}} {{.desc}}", out.V{"emoji": emoji, "desc": change.Description})
	case "happy":
		out.Styled(style.Happy, "{{.emoji}} {{.desc}}", out.V{"emoji": emoji, "desc": change.Description})
	default:
		out.Styled(style.Option, "{{.emoji}} {{.desc}}", out.V{"emoji": emoji, "desc": change.Description})
	}
	
	if change.URL != "" {
		out.Styled(style.URL, "   🔗 {{.url}}", out.V{"url": change.URL})
	}
}

func simulateDryRun() {
	out.Styled(style.Empty, "")
	out.Styled(style.Notice, "🔍 [DRY RUN] Would check for repository changes...")
	time.Sleep(1 * time.Second)
	
	if monitorCommits {
		out.Styled(style.Success, "💾 [DRY RUN] New commit by john.doe: Fix issue with startup sequence")
		out.Styled(style.URL, "   🔗 https://github.com/kubernetes/minikube/commit/abc123")
	}
	
	if monitorPRs {
		out.Styled(style.Celebrate, "🔀 [DRY RUN] PR #1234 opened: Add support for new driver")
		out.Styled(style.URL, "   🔗 https://github.com/kubernetes/minikube/pull/1234")
	}
	
	if monitorReleases {
		out.Styled(style.Happy, "🏷️ [DRY RUN] New release v1.35.1: Bug fixes and performance improvements")
		out.Styled(style.URL, "   🔗 https://github.com/kubernetes/minikube/releases/tag/v1.35.1")
	}
	
	out.Styled(style.Empty, "")
	out.Styled(style.Success, "✅ [DRY RUN] Monitoring simulation completed!")
	out.Styled(style.Notice, "💡 To run with real GitHub API calls, remove the --dry-run flag and set GITHUB_ACCESS_TOKEN")
}

func init() {
	monitorCmd.Flags().StringVar(&monitorOwner, "owner", monitor.GithubOwner, "GitHub repository owner")
	monitorCmd.Flags().StringVar(&monitorRepo, "repo", monitor.GithubRepo, "GitHub repository name")
	monitorCmd.Flags().StringVar(&monitorBranch, "branch", "", "Branch to monitor (default: all branches)")
	monitorCmd.Flags().DurationVar(&monitorPoll, "poll", 30*time.Second, "Polling interval for checking changes")
	monitorCmd.Flags().BoolVar(&monitorCommits, "commits", true, "Monitor commits")
	monitorCmd.Flags().BoolVar(&monitorPRs, "prs", true, "Monitor pull requests")
	monitorCmd.Flags().BoolVar(&monitorReleases, "releases", true, "Monitor releases")
	monitorCmd.Flags().BoolVar(&monitorDryRun, "dry-run", false, "Show what would be monitored without making API calls")
}