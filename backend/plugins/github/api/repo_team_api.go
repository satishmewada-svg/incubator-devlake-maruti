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

package api

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer/crossdomain"
	"github.com/apache/incubator-devlake/core/plugin"
)

// GetRepos returns all repos with their current team mapping
// @Summary get all repos
// @Tags plugins/github
// @Router /plugins/github/repos [GET]
func GetRepos(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	db := basicRes.GetDal()

	type RepoRow struct {
		Id   string `gorm:"column:id"`
		Name string `gorm:"column:name"`
		Url  string `gorm:"column:url"`
	}

	type RepoWithTeam struct {
		Id     string `json:"id"`
		Name   string `json:"name"`
		Url    string `json:"url"`
		TeamId string `json:"team_id"`
	}

	var repoRows []RepoRow
	err := db.All(&repoRows, dal.From("repos"))
	if err != nil {
		return nil, err
	}

	type TeamRepoRow struct {
		TeamId string `gorm:"column:team_id"`
		RepoId string `gorm:"column:repo_id"`
	}
	var teamRepos []TeamRepoRow
	err = db.All(&teamRepos, dal.From("team_repo"))
	if err != nil {
		return nil, err
	}

	repoTeamMap := make(map[string]string)
	for _, tr := range teamRepos {
		repoTeamMap[tr.RepoId] = tr.TeamId
	}

	// fetch teams for sorting
	var teams []crossdomain.Team
	err = db.All(&teams, dal.From(&crossdomain.Team{}))
	if err != nil {
		return nil, err
	}
	teamNameById := make(map[string]string, len(teams))
	for _, t := range teams {
		teamNameById[t.Id] = t.Name
	}

	result := make([]RepoWithTeam, 0, len(repoRows))
	for _, r := range repoRows {
		result = append(result, RepoWithTeam{
			Id:     r.Id,
			Name:   r.Name,
			Url:    r.Url,
			TeamId: repoTeamMap[r.Id],
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		ti := strings.ToLower(teamNameById[result[i].TeamId])
		tj := strings.ToLower(teamNameById[result[j].TeamId])
		if ti == tj {
			ri := strings.ToLower(strings.TrimSpace(result[i].Name))
			rj := strings.ToLower(strings.TrimSpace(result[j].Name))
			if ri == rj {
				return result[i].Id < result[j].Id
			}
			return ri < rj
		}
		return ti < tj
	})

	return &plugin.ApiResourceOutput{
		Body:   map[string]interface{}{"repos": result},
		Status: 200,
	}, nil
}

// SaveRepoTeamMapping saves team mapping for a repo
// @Summary save repo team mapping
// @Tags plugins/github
// @Router /plugins/github/repo-mapping [POST]
func SaveRepoTeamMapping(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	db := basicRes.GetDal()

	type SaveRequest struct {
		RepoId string `json:"repo_id"`
		TeamId string `json:"team_id"`
	}

	body, err := errors.Convert01(json.Marshal(input.Body))
	if err != nil {
		return nil, err
	}
	var req SaveRequest
	if jsonErr := json.Unmarshal(body, &req); jsonErr != nil {
		return nil, errors.Default.Wrap(jsonErr, "failed to parse request body")
	}

	if req.RepoId == "" {
		return nil, errors.Default.New("repo_id is required")
	}

	// delete ALL existing mappings for this repo first /// fix manual update
	delErr := db.Exec("DELETE FROM team_repo WHERE repo_id = ?", req.RepoId)
	if delErr != nil {
		return nil, delErr
	}

	// only insert if team selected
	if req.TeamId != "" {
		teamRepo := &crossdomain.TeamRepo{
			TeamId: req.TeamId,
			RepoId: req.RepoId,
		}
		teamRepo.NoPKModel.CreatedAt = time.Now()
		teamRepo.NoPKModel.UpdatedAt = time.Now()

		saveErr := db.CreateOrUpdate(teamRepo)
		if saveErr != nil {
			return nil, saveErr
		}
	}

	return &plugin.ApiResourceOutput{
		Body:   map[string]interface{}{"success": true},
		Status: 200,
	}, nil
}

// UploadRepoTeamMapping uploads CSV and maps repos to teams
// @Summary upload repo team mapping CSV
// @Tags plugins/github
// @Router /plugins/github/repo-mapping/upload [POST]
func UploadRepoTeamMapping(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	db := basicRes.GetDal()

	file, _, err2 := input.Request.FormFile("file")
	if err2 != nil {
		return nil, errors.Default.Wrap(err2, "no file provided")
	}
	defer file.Close()

	fileBytes, readErr := io.ReadAll(file)
	if readErr != nil {
		return nil, errors.Default.Wrap(readErr, "failed to read file")
	}

	reader := csv.NewReader(strings.NewReader(string(fileBytes)))
	rows, csvErr := reader.ReadAll()
	if csvErr != nil {
		return nil, errors.Default.Wrap(csvErr, "failed to parse CSV")
	}

	if len(rows) < 2 {
		return nil, errors.Default.New("file is empty or has no data rows")
	}

	headers := rows[0]
	colIndex := make(map[string]int)
	for i, h := range headers {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}

	repoCol, repoOk := colIndex["name"]
	if !repoOk {
		repoCol, repoOk = colIndex["repo"]
	}
	teamCol, teamOk := colIndex["team"]

	if !repoOk {
		return nil, errors.Default.New("CSV must have a 'name' or 'repo' column")
	}
	if !teamOk {
		return nil, errors.Default.New("CSV must have a 'team' column")
	}

	type RepoRow struct {
		Id   string `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	var repoRows []RepoRow
	err := db.All(&repoRows, dal.From("repos"))
	if err != nil {
		return nil, err
	}

	var teams []crossdomain.Team
	err = db.All(&teams, dal.From(&crossdomain.Team{}))
	if err != nil {
		return nil, err
	}

	// build lookup maps - match by full name AND short name /// fix not all repos mapped
	repoByName := make(map[string]RepoRow)
	for _, r := range repoRows {
		repoByName[strings.ToLower(r.Name)] = r
		// also index by short name only (after last /) /// added
		parts := strings.Split(r.Name, "/")
		if len(parts) > 1 {
			shortName := parts[len(parts)-1]
			repoByName[strings.ToLower(shortName)] = r
		}
	}

	teamByName := make(map[string]crossdomain.Team)
	for _, t := range teams {
		teamByName[strings.ToLower(t.Name)] = t
	}

	saved := 0
	skipped := 0
	notFound := make([]string, 0) /// added for debugging

	for _, cols := range rows[1:] {
		if len(cols) == 0 {
			continue
		}

		repoName := strings.TrimSpace(cols[repoCol])
		teamName := strings.TrimSpace(cols[teamCol])

		if repoName == "" || teamName == "" {
			skipped++
			continue
		}

		repo, repoFound := repoByName[strings.ToLower(repoName)]
		team, teamFound := teamByName[strings.ToLower(teamName)]

		if !repoFound || !teamFound {
			notFound = append(notFound, repoName) /// added for debugging
			skipped++
			continue
		}

		// delete existing first then insert /// fix re-upload issue
		db.Exec("DELETE FROM team_repo WHERE repo_id = ?", repo.Id)

		teamRepo := &crossdomain.TeamRepo{
			TeamId: team.Id,
			RepoId: repo.Id,
		}
		teamRepo.NoPKModel.CreatedAt = time.Now()
		teamRepo.NoPKModel.UpdatedAt = time.Now()

		saveErr := db.CreateOrUpdate(teamRepo)
		if saveErr != nil {
			skipped++
			continue
		}
		saved++
	}

	return &plugin.ApiResourceOutput{
		Body: map[string]interface{}{
			"success":   true,
			"saved":     saved,
			"skipped":   skipped,
			"not_found": notFound, /// added for debugging
		},
		Status: 200,
	}, nil
}
