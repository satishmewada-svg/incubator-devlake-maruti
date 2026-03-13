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

package migrationscripts

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/migrationhelper"
)

var _ plugin.MigrationScript = (*addGithubPrReviewRequests)(nil)

type addGithubPrReviewRequests struct{}

type githubPrReviewRequest20260312000001 struct {
	ConnectionId  uint64 `gorm:"primaryKey"`
	Repo          string `gorm:"primaryKey;type:varchar(255)"`
	PrNumber      int    `gorm:"primaryKey;index"`
	PrId          string `gorm:"type:varchar(255);index"`
	PrGithubId    int    `gorm:"index"`
	PrUrl         string `gorm:"type:varchar(255)"`
	ReviewerId    int    `gorm:"primaryKey"`
	ReviewerLogin string `gorm:"type:varchar(255);index"`
	ReviewerName  string `gorm:"type:varchar(255)"`

	common.NoPKModel
}

func (githubPrReviewRequest20260312000001) TableName() string {
	return "_tool_github_pr_review_requests"
}

func (*addGithubPrReviewRequests) Up(basicRes context.BasicRes) errors.Error {
	return migrationhelper.AutoMigrateTables(
		basicRes,
		&githubPrReviewRequest20260312000001{},
	)
}

func (*addGithubPrReviewRequests) Version() uint64 {
	return 20260312000001
}

func (*addGithubPrReviewRequests) Name() string {
	return "add github pr review requests table"
}
