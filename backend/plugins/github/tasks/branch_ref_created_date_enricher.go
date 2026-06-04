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
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/log"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/models/domainlayer/code"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/github/models"
)

const emptyGitObjectHash = "0000000000000000000000000000000000000000"

func init() {
	RegisterSubtaskMeta(&EnrichBranchRefCreatedDatesMeta)
}

var EnrichBranchRefCreatedDatesMeta = plugin.SubTaskMeta{
	Name:             "Enrich Branch Ref Created Dates",
	EntryPoint:       EnrichBranchRefCreatedDates,
	EnabledByDefault: false,
	Description:      "Set refs.created_date for GitHub branches using repo events and pull requests when available",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE},
	ProductTables:    []string{code.Ref{}.TableName()},
}

type githubRepoEvent struct {
	Type      string             `json:"type"`
	CreatedAt common.Iso8601Time `json:"created_at"`
	Payload   githubRepoEventPayload
}

type githubRepoEventPayload struct {
	Ref     string `json:"ref"`
	RefType string `json:"ref_type"`
	Before  string `json:"before"`
}

func EnrichBranchRefCreatedDates(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*GithubTaskData)
	db := taskCtx.GetDal()
	logger := taskCtx.GetLogger()

	domainRepoID := didgen.NewDomainIdGenerator(&models.GithubRepo{}).Generate(data.Options.ConnectionId, data.Options.GithubId)
	branchDates, err := collectGithubBranchCreatedDates(data, db, logger)
	if err != nil {
		return err
	}
	if len(branchDates) == 0 {
		logger.Info("no GitHub branch creation hints found for repo %s", data.Options.Name)
		return nil
	}

	refs := make([]code.Ref, 0)
	if err := db.All(
		&refs,
		dal.From(&code.Ref{}),
		dal.Where("repo_id = ? AND ref_type = ?", domainRepoID, "BRANCH"),
	); err != nil {
		return err
	}

	updated := 0
	for _, ref := range refs {
		createdDate := resolveBranchCreatedDate(ref.Name, branchDates)
		if createdDate == nil {
			continue
		}
		if ref.CreatedDate != nil && !createdDate.Before(*ref.CreatedDate) {
			continue
		}
		ref.CreatedDate = createdDate
		if err := db.CreateOrUpdate(&ref); err != nil {
			return err
		}
		updated++
	}
	logger.Info("updated created_date for %d GitHub branch refs in repo %s", updated, data.Options.Name)
	return nil
}

func collectGithubBranchCreatedDates(data *GithubTaskData, db dal.Dal, logger log.Logger) (map[string]time.Time, errors.Error) {
	branchDates := make(map[string]time.Time)

	page := 1
	for {
		query := url.Values{}
		query.Set("page", fmt.Sprintf("%d", page))
		query.Set("per_page", "100")
		res, err := data.ApiClient.Get(fmt.Sprintf("repos/%s/events", data.Options.Name), query, nil)
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			if res.StatusCode == http.StatusUnprocessableEntity || res.StatusCode == http.StatusNotFound {
				logger.Debug("repo events for %s returned %d, skipping branch created date enrichment", data.Options.Name, res.StatusCode)
				res.Body.Close()
				break
			}
			res.Body.Close()
			return nil, errors.HttpStatus(res.StatusCode).New(fmt.Sprintf("failed to list repo events for %s", data.Options.Name))
		}

		var events []githubRepoEvent
		if err := helper.UnmarshalResponse(res, &events); err != nil {
			res.Body.Close()
			return nil, err
		}
		res.Body.Close()

		if len(events) == 0 {
			break
		}
		for _, event := range events {
			branchName, ok := branchNameFromRepoEvent(event)
			if !ok {
				continue
			}
			recordBranchCreatedDate(branchDates, branchName, event.CreatedAt.ToTime())
		}

		if len(events) < 100 {
			break
		}
		page++
	}

	pullRequests := make([]models.GithubPullRequest, 0)
	if err := db.All(
		&pullRequests,
		dal.Select("head_ref, github_created_at"),
		dal.From(&models.GithubPullRequest{}),
		dal.Where(
			"connection_id = ? AND repo_id = ? AND head_ref <> ''",
			data.Options.ConnectionId,
			data.Options.GithubId,
		),
	); err != nil {
		return nil, err
	}
	for _, pr := range pullRequests {
		recordBranchCreatedDate(branchDates, pr.HeadRef, pr.GithubCreatedAt)
	}

	return branchDates, nil
}

func branchNameFromRepoEvent(event githubRepoEvent) (string, bool) {
	switch event.Type {
	case "CreateEvent":
		if event.Payload.RefType != "branch" || event.Payload.Ref == "" {
			return "", false
		}
		return normalizeGithubBranchName(event.Payload.Ref), true
	case "PushEvent":
		if !strings.HasPrefix(event.Payload.Ref, "refs/heads/") {
			return "", false
		}
		if event.Payload.Before != emptyGitObjectHash {
			return "", false
		}
		return normalizeGithubBranchName(strings.TrimPrefix(event.Payload.Ref, "refs/heads/")), true
	default:
		return "", false
	}
}

func normalizeGithubBranchName(name string) string {
	name = strings.TrimPrefix(name, "refs/heads/")
	return strings.TrimPrefix(name, "refs/remotes/origin/")
}

func branchNameCandidates(name string) []string {
	normalized := normalizeGithubBranchName(name)
	candidates := []string{normalized}
	if strings.HasPrefix(name, "origin/") {
		candidates = append(candidates, strings.TrimPrefix(name, "origin/"))
	}
	return candidates
}

func resolveBranchCreatedDate(name string, branchDates map[string]time.Time) *time.Time {
	for _, candidate := range branchNameCandidates(name) {
		if createdAt, ok := branchDates[candidate]; ok {
			return &createdAt
		}
	}
	return nil
}

func recordBranchCreatedDate(branchDates map[string]time.Time, branchName string, createdAt time.Time) {
	if branchName == "" || createdAt.IsZero() {
		return
	}
	branchName = normalizeGithubBranchName(branchName)
	if existing, ok := branchDates[branchName]; !ok || createdAt.Before(existing) {
		branchDates[branchName] = createdAt
	}
}

// Ensure payload fields are decoded when GitHub nests them under "payload".
func (e *githubRepoEvent) UnmarshalJSON(data []byte) error {
	type alias githubRepoEvent
	aux := struct {
		alias
		Payload json.RawMessage `json:"payload"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*e = githubRepoEvent(aux.alias)
	if len(aux.Payload) == 0 {
		return nil
	}
	return json.Unmarshal(aux.Payload, &e.Payload)
}
