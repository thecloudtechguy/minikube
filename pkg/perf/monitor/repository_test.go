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
	"testing"
	"time"
)

func TestRepoMonitorState(t *testing.T) {
	state := &RepoMonitorState{
		LastCommitSHA:   "abc123",
		LastPRUpdate:    time.Now(),
		LastReleaseTag:  "v1.0.0",
		LastIssueUpdate: time.Now(),
	}

	if state.LastCommitSHA != "abc123" {
		t.Errorf("expected LastCommitSHA to be 'abc123', got %s", state.LastCommitSHA)
	}

	if state.LastReleaseTag != "v1.0.0" {
		t.Errorf("expected LastReleaseTag to be 'v1.0.0', got %s", state.LastReleaseTag)
	}
}

func TestRepositoryChange(t *testing.T) {
	change := RepositoryChange{
		Type:        "commit",
		Description: "Test commit",
		URL:         "https://github.com/test/repo/commit/abc123",
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"sha":    "abc123",
			"author": "testuser",
		},
	}

	if change.Type != "commit" {
		t.Errorf("expected Type to be 'commit', got %s", change.Type)
	}

	if change.Description != "Test commit" {
		t.Errorf("expected Description to be 'Test commit', got %s", change.Description)
	}

	sha, ok := change.Metadata["sha"]
	if !ok || sha != "abc123" {
		t.Errorf("expected metadata sha to be 'abc123', got %v", sha)
	}

	author, ok := change.Metadata["author"]
	if !ok || author != "testuser" {
		t.Errorf("expected metadata author to be 'testuser', got %v", author)
	}
}

func TestRepositoryChangeTypes(t *testing.T) {
	validTypes := []string{"commit", "pr", "release", "issue"}
	
	for _, changeType := range validTypes {
		change := RepositoryChange{
			Type:        changeType,
			Description: "Test " + changeType,
			URL:         "https://github.com/test/repo",
			Timestamp:   time.Now(),
		}

		if change.Type != changeType {
			t.Errorf("expected Type to be '%s', got %s", changeType, change.Type)
		}
	}
}