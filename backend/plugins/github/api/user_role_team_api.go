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
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer"
	"github.com/apache/incubator-devlake/core/models/domainlayer/crossdomain"
	"github.com/apache/incubator-devlake/core/plugin"
)

// GetUsers returns all users with their current team and role
// @Summary get all users
// @Description get all users with team and role assignments
// @Tags plugins/github
// @Router /plugins/github/users [GET]
func GetUsers(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	db := basicRes.GetDal()

	type UserWithAssignment struct {
		Id           string `json:"id"`
		Name         string `json:"name"`
		Email        string `json:"email"`
		UserFullName string `json:"user_full_name"`
		TeamId       string `json:"team_id"`
		RoleId       string `json:"role_id"`
	}

	var users []crossdomain.User
	err := db.All(&users, dal.From(&crossdomain.User{}))
	if err != nil {
		return nil, err
	}

	// fetch all team_users mappings
	type TeamUserRow struct {
		UserId string `gorm:"column:user_id"`
		TeamId string `gorm:"column:team_id"`
		RoleId string `gorm:"column:role_id"`
	}
	var teamUsers []TeamUserRow
	err = db.All(&teamUsers, dal.From("team_users"))
	if err != nil {
		return nil, err
	}

	// build map userId -> teamUser
	tuMap := make(map[string]TeamUserRow)
	for _, tu := range teamUsers {
		tuMap[tu.UserId] = tu
	}

	result := make([]UserWithAssignment, 0, len(users))
	for _, u := range users {
		ua := UserWithAssignment{
			Id:           u.Id,
			Name:         u.Name,
			Email:        u.Email,
			UserFullName: u.UserFullName,
		}
		if tu, ok := tuMap[u.Id]; ok {
			ua.TeamId = tu.TeamId
			ua.RoleId = tu.RoleId
		}
		result = append(result, ua)
	}

	return &plugin.ApiResourceOutput{
		Body:   map[string]interface{}{"users": result},
		Status: 200,
	}, nil
}

// GetTeams returns all teams
// @Summary get all teams
// @Tags plugins/github
// @Router /plugins/github/teams [GET]
func GetTeams(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	db := basicRes.GetDal()
	var teams []crossdomain.Team
	err := db.All(&teams, dal.From(&crossdomain.Team{}))
	if err != nil {
		return nil, err
	}
	return &plugin.ApiResourceOutput{
		Body:   map[string]interface{}{"teams": teams},
		Status: 200,
	}, nil
}

// GetRoles returns all roles
// @Summary get all roles
// @Tags plugins/github
// @Router /plugins/github/roles [GET]
func GetRoles(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	db := basicRes.GetDal()
	var roles []crossdomain.Role
	err := db.All(&roles, dal.From(&crossdomain.Role{}))
	if err != nil {
		return nil, err
	}
	return &plugin.ApiResourceOutput{
		Body:   map[string]interface{}{"roles": roles},
		Status: 200,
	}, nil
}

// SaveUserMapping saves team and role for a user
// @Summary save user team/role mapping
// @Tags plugins/github
// @Router /plugins/github/user-mapping [POST]
func SaveUserMapping(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	db := basicRes.GetDal()

	type SaveRequest struct {
		UserId string `json:"user_id"`
		TeamId string `json:"team_id"`
		RoleId string `json:"role_id"`
	}

	body, err := errors.Convert01(json.Marshal(input.Body))
	if err != nil {
		return nil, err
	}
	var req SaveRequest
	if jsonErr := json.Unmarshal(body, &req); jsonErr != nil {
		return nil, errors.Default.Wrap(jsonErr, "failed to parse request body")
	}

	if req.UserId == "" {
		return nil, errors.Default.New("user_id is required")
	}

	// delete existing mapping first /// fix manual update
	delErr := db.Exec("DELETE FROM team_users WHERE user_id = ?", req.UserId)
	if delErr != nil {
		return nil, delErr
	}

	// only insert if team or role selected
	if req.TeamId != "" || req.RoleId != "" {
		teamUser := &crossdomain.TeamUser{
			TeamId: req.TeamId,
			UserId: req.UserId,
			RoleId: req.RoleId,
		}
		teamUser.NoPKModel.CreatedAt = time.Now()
		teamUser.NoPKModel.UpdatedAt = time.Now()

		saveErr := db.CreateOrUpdate(teamUser)
		if saveErr != nil {
			return nil, saveErr
		}
	}

	return &plugin.ApiResourceOutput{
		Body:   map[string]interface{}{"success": true},
		Status: 200,
	}, nil
}

func UploadUserMapping(input *plugin.ApiResourceInput) (*plugin.ApiResourceOutput, errors.Error) {
	db := basicRes.GetDal()

	// read file from multipart form
	file, _, err2 := input.Request.FormFile("file")
	if err2 != nil {
		return nil, errors.Default.Wrap(err2, "no file provided")
	}
	defer file.Close()

	// read all bytes
	fileBytes, readErr := io.ReadAll(file)
	if readErr != nil {
		return nil, errors.Default.Wrap(readErr, "failed to read file")
	}

	// parse CSV
	reader := csv.NewReader(strings.NewReader(string(fileBytes)))
	rows, csvErr := reader.ReadAll()
	if csvErr != nil {
		return nil, errors.Default.Wrap(csvErr, "failed to parse CSV")
	}

	if len(rows) < 2 {
		return nil, errors.Default.New("file is empty or has no data rows")
	}

	// ── STEP 1: collect unique teams and roles from CSV ──────────────────────

	// find header columns
	headers := rows[0]
	colIndex := make(map[string]int)
	for i, h := range headers {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}

	nameCol, nameOk := colIndex["name"]
	teamCol, teamOk := colIndex["team"]
	roleCol, roleOk := colIndex["role"]

	if !nameOk {
		return nil, errors.Default.New("CSV must have a 'name' column")
	}

	// collect unique team and role names from CSV
	uniqueTeams := make(map[string]bool)
	uniqueRoles := make(map[string]bool)
	for _, cols := range rows[1:] {
		if teamOk && teamCol < len(cols) {
			t := strings.TrimSpace(cols[teamCol])
			if t != "" {
				uniqueTeams[t] = true
			}
		}
		if roleOk && roleCol < len(cols) {
			r := strings.TrimSpace(cols[roleCol])
			if r != "" {
				uniqueRoles[r] = true
			}
		}
	}

	// ── STEP 2: fetch existing teams and roles, insert missing ones ──────────

	// fetch existing teams
	var existingTeams []crossdomain.Team
	err := db.All(&existingTeams, dal.From(&crossdomain.Team{}))
	if err != nil {
		return nil, err
	}
	teamByName := make(map[string]crossdomain.Team)
	for _, t := range existingTeams {
		teamByName[strings.ToLower(t.Name)] = t
	}

	// insert missing teams
	teamIndex := 0
	for _, t := range existingTeams {
		if t.SortingIndex > teamIndex {
			teamIndex = t.SortingIndex
		}
	}
	teamIndex++
	for teamName := range uniqueTeams {
		if _, exists := teamByName[strings.ToLower(teamName)]; !exists {
			newTeam := crossdomain.Team{
				DomainEntity: domainlayer.DomainEntity{
					Id: fmt.Sprintf("github:Team:%d", teamIndex),
				},
				Name:         teamName,
				SortingIndex: teamIndex,
			}
			newTeam.NoPKModel.CreatedAt = time.Now()
			newTeam.NoPKModel.UpdatedAt = time.Now()
			insertErr := db.CreateOrUpdate(&newTeam)
			if insertErr != nil {
				return nil, insertErr
			}
			teamByName[strings.ToLower(teamName)] = newTeam
			teamIndex++
		}
	}

	// fetch existing roles
	var existingRoles []crossdomain.Role
	err = db.All(&existingRoles, dal.From(&crossdomain.Role{}))
	if err != nil {
		return nil, err
	}
	roleByName := make(map[string]crossdomain.Role)
	for _, r := range existingRoles {
		roleByName[strings.ToLower(r.Name)] = r
	}

	// insert missing roles
	roleIndex := 0
	for _, r := range existingRoles {
		if r.SortingIndex > roleIndex {
			roleIndex = r.SortingIndex
		}
	}
	roleIndex++
	for roleName := range uniqueRoles {
		if _, exists := roleByName[strings.ToLower(roleName)]; !exists {
			newRole := crossdomain.Role{
				DomainEntity: domainlayer.DomainEntity{
					Id: fmt.Sprintf("github:Role:%d", roleIndex),
				},
				Name:         roleName,
				SortingIndex: roleIndex,
			}
			newRole.NoPKModel.CreatedAt = time.Now()
			newRole.NoPKModel.UpdatedAt = time.Now()
			insertErr := db.CreateOrUpdate(&newRole)
			if insertErr != nil {
				return nil, insertErr
			}
			roleByName[strings.ToLower(roleName)] = newRole
			roleIndex++
		}
	}

	// ── STEP 3: fetch users and map team/role ────────────────────────────────

	var users []crossdomain.User
	err = db.All(&users, dal.From(&crossdomain.User{}))
	if err != nil {
		return nil, err
	}
	userByName := make(map[string]crossdomain.User)
	for _, u := range users {
		userByName[strings.ToLower(u.Name)] = u
	}

	saved := 0
	skipped := 0

	for _, cols := range rows[1:] {
		if len(cols) == 0 {
			continue
		}

		name := strings.TrimSpace(cols[nameCol])
		teamName := ""
		roleName := ""

		if teamOk && teamCol < len(cols) {
			teamName = strings.TrimSpace(cols[teamCol])
		}
		if roleOk && roleCol < len(cols) {
			roleName = strings.TrimSpace(cols[roleCol])
		}

		// match user
		user, userFound := userByName[strings.ToLower(name)]
		if !userFound {
			skipped++
			continue
		}

		team, teamFound := teamByName[strings.ToLower(teamName)]
		role, roleFound := roleByName[strings.ToLower(roleName)]

		if !teamFound && !roleFound {
			skipped++
			continue
		}

		// delete existing first then insert /// fix re-upload issue
		db.Exec("DELETE FROM team_users WHERE user_id = ?", user.Id)

		teamUser := &crossdomain.TeamUser{
			UserId: user.Id,
		}
		if teamFound {
			teamUser.TeamId = team.Id
		}
		if roleFound {
			teamUser.RoleId = role.Id
		}
		teamUser.NoPKModel.CreatedAt = time.Now()
		teamUser.NoPKModel.UpdatedAt = time.Now()

		saveErr := db.CreateOrUpdate(teamUser)
		if saveErr != nil {
			skipped++
			continue
		}
		saved++
	}

	return &plugin.ApiResourceOutput{
		Body: map[string]interface{}{
			"success": true,
			"saved":   saved,
			"skipped": skipped,
		},
		Status: 200,
	}, nil
}
