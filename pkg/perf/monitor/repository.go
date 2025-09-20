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

package monitor

import (
	"fmt"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/pkg/errors"
)

// RepoMonitorState holds the last known state for repository monitoring
type RepoMonitorState struct {
	LastCommitSHA    string
	LastPRUpdate     time.Time
	LastReleaseTag   string
	LastIssueUpdate  time.Time
}

// RepositoryChange represents a change detected in the repository
type RepositoryChange struct {
	Type        string                 // "commit", "pr", "release", "issue"
	Description string                 // Human-readable description
	URL         string                 // Link to the change
	Timestamp   time.Time              // When the change occurred
	Metadata    map[string]interface{} // Additional data
}

// ListOpenPRs returns all open pull requests
func (g *Client) ListOpenPRs() ([]*github.PullRequest, error) {
	var allPRs []*github.PullRequest
	page := 0
	perPage := 100

	for {
		prs, _, err := g.Client.PullRequests.List(g.ctx, g.owner, g.repo, &github.PullRequestListOptions{
			State: "open",
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "listing pull requests")
		}

		allPRs = append(allPRs, prs...)
		if len(prs) < perPage {
			break
		}
		page++
	}

	return allPRs, nil
}

// GetLatestCommits returns the latest commits from a specific branch
func (g *Client) GetLatestCommits(branch string, since time.Time) ([]*github.RepositoryCommit, error) {
	opts := &github.CommitsListOptions{
		SHA:   branch,
		Since: since,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	commits, _, err := g.Client.Repositories.ListCommits(g.ctx, g.owner, g.repo, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "getting commits for branch %s", branch)
	}

	return commits, nil
}

// GetLatestReleases returns the latest releases
func (g *Client) GetLatestReleases(limit int) ([]*github.RepositoryRelease, error) {
	opts := &github.ListOptions{
		PerPage: limit,
	}

	releases, _, err := g.Client.Repositories.ListReleases(g.ctx, g.owner, g.repo, opts)
	if err != nil {
		return nil, errors.Wrap(err, "getting releases")
	}

	return releases, nil
}

// GetRecentPRActivity returns PR activity since a given time
func (g *Client) GetRecentPRActivity(since time.Time) ([]*github.PullRequest, error) {
	// Get all PRs updated since the given time
	opts := &github.PullRequestListOptions{
		State:     "all", // Include both open and closed
		Sort:      "updated",
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var recentPRs []*github.PullRequest
	prs, _, err := g.Client.PullRequests.List(g.ctx, g.owner, g.repo, opts)
	if err != nil {
		return nil, errors.Wrap(err, "listing recent PRs")
	}

	// Filter PRs updated since the specified time
	for _, pr := range prs {
		if pr.UpdatedAt != nil && pr.UpdatedAt.After(since) {
			recentPRs = append(recentPRs, pr)
		} else {
			// Since results are sorted by updated date descending,
			// we can break once we find an older PR
			break
		}
	}

	return recentPRs, nil
}

// CheckForRepositoryChanges checks for various types of changes in the repository
func (g *Client) CheckForRepositoryChanges(state *RepoMonitorState, branch string) ([]RepositoryChange, error) {
	var changes []RepositoryChange
	
	// Check for new commits
	if branch != "" {
		commitChanges, err := g.checkCommitChanges(state, branch)
		if err != nil {
			return nil, errors.Wrap(err, "checking commit changes")
		}
		changes = append(changes, commitChanges...)
	}

	// Check for PR changes
	prChanges, err := g.checkPRChanges(state)
	if err != nil {
		return nil, errors.Wrap(err, "checking PR changes")
	}
	changes = append(changes, prChanges...)

	// Check for new releases
	releaseChanges, err := g.checkReleaseChanges(state)
	if err != nil {
		return nil, errors.Wrap(err, "checking release changes")
	}
	changes = append(changes, releaseChanges...)

	return changes, nil
}

func (g *Client) checkCommitChanges(state *RepoMonitorState, branch string) ([]RepositoryChange, error) {
	var changes []RepositoryChange
	
	// Get the latest commit from the branch
	ref, _, err := g.Client.Git.GetRef(g.ctx, g.owner, g.repo, fmt.Sprintf("heads/%s", branch))
	if err != nil {
		return nil, errors.Wrapf(err, "getting ref for branch %s", branch)
	}

	latestSHA := ref.Object.GetSHA()
	if state.LastCommitSHA != "" && latestSHA != state.LastCommitSHA {
		// Get commits since the last known commit
		since := time.Now().Add(-24 * time.Hour) // Fallback to last 24 hours
		commits, err := g.GetLatestCommits(branch, since)
		if err != nil {
			return nil, err
		}

		for _, commit := range commits {
			if commit.GetSHA() == state.LastCommitSHA {
				break // Stop when we reach the last known commit
			}
			
			changes = append(changes, RepositoryChange{
				Type:        "commit",
				Description: fmt.Sprintf("New commit by %s: %s", commit.GetCommit().GetAuthor().GetName(), commit.GetCommit().GetMessage()),
				URL:         commit.GetHTMLURL(),
				Timestamp:   commit.GetCommit().GetAuthor().GetDate().Time,
				Metadata: map[string]interface{}{
					"sha":     commit.GetSHA(),
					"author":  commit.GetCommit().GetAuthor().GetName(),
					"branch":  branch,
					"message": commit.GetCommit().GetMessage(),
				},
			})
		}
	}

	// Update state
	state.LastCommitSHA = latestSHA
	return changes, nil
}

func (g *Client) checkPRChanges(state *RepoMonitorState) ([]RepositoryChange, error) {
	var changes []RepositoryChange
	
	since := state.LastPRUpdate
	if since.IsZero() {
		since = time.Now().Add(-1 * time.Hour) // Default to last hour
	}

	recentPRs, err := g.GetRecentPRActivity(since)
	if err != nil {
		return nil, err
	}

	for _, pr := range recentPRs {
		var action string
		if pr.GetState() == "open" && pr.GetCreatedAt().After(since) {
			action = "opened"
		} else if pr.GetState() == "closed" {
			if pr.GetMerged() {
				action = "merged"
			} else {
				action = "closed"
			}
		} else {
			action = "updated"
		}

		changes = append(changes, RepositoryChange{
			Type:        "pr",
			Description: fmt.Sprintf("PR #%d %s: %s", pr.GetNumber(), action, pr.GetTitle()),
			URL:         pr.GetHTMLURL(),
			Timestamp:   pr.GetUpdatedAt().Time,
			Metadata: map[string]interface{}{
				"number": pr.GetNumber(),
				"title":  pr.GetTitle(),
				"action": action,
				"author": pr.GetUser().GetLogin(),
				"state":  pr.GetState(),
			},
		})
	}

	// Update state
	if len(recentPRs) > 0 {
		state.LastPRUpdate = time.Now()
	}

	return changes, nil
}

func (g *Client) checkReleaseChanges(state *RepoMonitorState) ([]RepositoryChange, error) {
	var changes []RepositoryChange
	
	releases, err := g.GetLatestReleases(10)
	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		if state.LastReleaseTag == "" {
			// First run, just update state
			state.LastReleaseTag = release.GetTagName()
			break
		}
		
		if release.GetTagName() == state.LastReleaseTag {
			break // Found the last known release
		}

		changes = append(changes, RepositoryChange{
			Type:        "release",
			Description: fmt.Sprintf("New release %s: %s", release.GetTagName(), release.GetName()),
			URL:         release.GetHTMLURL(),
			Timestamp:   release.GetPublishedAt().Time,
			Metadata: map[string]interface{}{
				"tag":        release.GetTagName(),
				"name":       release.GetName(),
				"draft":      release.GetDraft(),
				"prerelease": release.GetPrerelease(),
			},
		})
	}

	// Update state with the latest release
	if len(releases) > 0 {
		state.LastReleaseTag = releases[0].GetTagName()
	}

	return changes, nil
}