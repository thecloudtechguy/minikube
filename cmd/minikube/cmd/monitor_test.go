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
	"testing"
	"time"

	"k8s.io/minikube/pkg/perf/monitor"
)

func TestMonitorCommandFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected map[string]interface{}
	}{
		{
			name: "default flags",
			args: []string{},
			expected: map[string]interface{}{
				"owner":    "kubernetes",
				"repo":     "minikube",
				"commits":  true,
				"prs":      true,
				"releases": true,
				"poll":     30 * time.Second,
			},
		},
		{
			name: "custom repository",
			args: []string{"--owner=myorg", "--repo=myrepo"},
			expected: map[string]interface{}{
				"owner": "myorg",
				"repo":  "myrepo",
			},
		},
		{
			name: "custom branch and poll interval",
			args: []string{"--branch=main", "--poll=60s"},
			expected: map[string]interface{}{
				"branch": "main",
				"poll":   60 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags to defaults
			monitorOwner = monitor.GithubOwner
			monitorRepo = monitor.GithubRepo
			monitorBranch = ""
			monitorPoll = 30 * time.Second
			monitorCommits = true
			monitorPRs = true
			monitorReleases = true

			// Parse the flags
			monitorCmd.ParseFlags(tt.args)

			// Check expected values
			if owner, ok := tt.expected["owner"]; ok && monitorOwner != owner.(string) {
				t.Errorf("expected owner %v, got %v", owner, monitorOwner)
			}
			if repo, ok := tt.expected["repo"]; ok && monitorRepo != repo.(string) {
				t.Errorf("expected repo %v, got %v", repo, monitorRepo)
			}
			if branch, ok := tt.expected["branch"]; ok && monitorBranch != branch.(string) {
				t.Errorf("expected branch %v, got %v", branch, monitorBranch)
			}
			if poll, ok := tt.expected["poll"]; ok && monitorPoll != poll.(time.Duration) {
				t.Errorf("expected poll %v, got %v", poll, monitorPoll)
			}
			if commits, ok := tt.expected["commits"]; ok && monitorCommits != commits.(bool) {
				t.Errorf("expected commits %v, got %v", commits, monitorCommits)
			}
			if prs, ok := tt.expected["prs"]; ok && monitorPRs != prs.(bool) {
				t.Errorf("expected prs %v, got %v", prs, monitorPRs)
			}
			if releases, ok := tt.expected["releases"]; ok && monitorReleases != releases.(bool) {
				t.Errorf("expected releases %v, got %v", releases, monitorReleases)
			}
		})
	}
}

func TestReportChange(t *testing.T) {
	testChange := monitor.RepositoryChange{
		Type:        "commit",
		Description: "Test commit message",
		URL:         "https://github.com/test/repo/commit/abc123",
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"sha":    "abc123",
			"author": "testuser",
		},
	}

	// This test just ensures the function doesn't panic
	// In a real test environment, you'd capture the output
	reportChange(testChange)
}