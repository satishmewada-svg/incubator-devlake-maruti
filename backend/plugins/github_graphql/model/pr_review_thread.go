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

package model

import (
	"time"

	"github.com/apache/incubator-devlake/core/models/common"
)

// GithubPrReviewThread stores pull request review thread summary collected via GitHub GraphQL.
// One row represents one thread (not each individual comment).
type GithubPrReviewThread struct {
	ConnectionId uint64 `gorm:"primaryKey"`
	Repo         string `gorm:"primaryKey;type:varchar(255)"`
	PrNumber     int    `gorm:"primaryKey;index"`
	// PrId is the DevLake domain id for pull request, e.g. github:GithubPullRequest:<connectionId>:<githubId>.
	PrId string `gorm:"type:varchar(255);index"`
	// PrGithubId is GitHub PullRequest databaseId (tool-layer id).
	PrGithubId int    `gorm:"index"`
	PrUrl      string `gorm:"type:varchar(255)"`

	ThreadId        string     `gorm:"primaryKey;type:varchar(255)"`
	IsResolved      bool       `gorm:"index"`
	IsOutdated      bool       `gorm:"index"`
	Author          string     `gorm:"type:varchar(255)"`
	Comment         string     `gorm:"type:longtext"`
	GithubCreatedAt *time.Time `gorm:"index"`

	common.NoPKModel
}

func (GithubPrReviewThread) TableName() string {
	return "_tool_github_pr_review_threads"
}
