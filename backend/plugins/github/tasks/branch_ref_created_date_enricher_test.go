/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBranchNameFromRepoEvent(t *testing.T) {
	createEvent := githubRepoEvent{
		Type: "CreateEvent",
		Payload: githubRepoEventPayload{
			Ref:     "feature/foo",
			RefType: "branch",
		},
	}
	name, ok := branchNameFromRepoEvent(createEvent)
	require.True(t, ok)
	require.Equal(t, "feature/foo", name)

	pushEvent := githubRepoEvent{
		Type: "PushEvent",
		Payload: githubRepoEventPayload{
			Ref:    "refs/heads/feature/foo",
			Before: emptyGitObjectHash,
		},
	}
	name, ok = branchNameFromRepoEvent(pushEvent)
	require.True(t, ok)
	require.Equal(t, "feature/foo", name)

	_, ok = branchNameFromRepoEvent(githubRepoEvent{
		Type: "PushEvent",
		Payload: githubRepoEventPayload{
			Ref:    "refs/heads/feature/foo",
			Before: "abc",
		},
	})
	require.False(t, ok)
}

func TestRecordAndResolveBranchCreatedDate(t *testing.T) {
	branchDates := make(map[string]time.Time)
	first := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	later := first.Add(24 * time.Hour)

	recordBranchCreatedDate(branchDates, "feature/foo", later)
	recordBranchCreatedDate(branchDates, "feature/foo", first)

	resolved := resolveBranchCreatedDate("origin/feature/foo", branchDates)
	require.NotNil(t, resolved)
	require.Equal(t, first, *resolved)
}

func TestGithubRepoEventUnmarshalJSON(t *testing.T) {
	raw := []byte(`{
		"type":"CreateEvent",
		"created_at":"2024-06-01T10:00:00Z",
		"payload":{"ref":"main","ref_type":"branch"}
	}`)
	var event githubRepoEvent
	require.NoError(t, json.Unmarshal(raw, &event))
	require.Equal(t, "CreateEvent", event.Type)
	require.Equal(t, "main", event.Payload.Ref)
	require.Equal(t, "branch", event.Payload.RefType)
	require.Equal(t, time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC), event.CreatedAt.ToTime())
}
