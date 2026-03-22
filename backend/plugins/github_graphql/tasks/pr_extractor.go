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
	"regexp"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/domainlayer/didgen"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/github/models"
	githubTasks "github.com/apache/incubator-devlake/plugins/github/tasks"
	graphqlModels "github.com/apache/incubator-devlake/plugins/github_graphql/model"
)

var _ plugin.SubTaskEntryPoint = ExtractPrs

var ExtractPrsMeta = plugin.SubTaskMeta{
	Name:             "Extract Pull Requests",
	EntryPoint:       ExtractPrs,
	EnabledByDefault: true,
	Description:      "Extract raw PullRequests data into tool layer table github_pull_requests",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_CODE_REVIEW},
}

func ExtractPrs(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*githubTasks.GithubTaskData)
	config := data.Options.ScopeConfig
	prDidGen := didgen.NewDomainIdGenerator(&models.GithubPullRequest{})
	var labelTypeRegex *regexp.Regexp
	var labelComponentRegex *regexp.Regexp
	var err errors.Error

	if config != nil && len(config.PrType) > 0 {
		labelTypeRegex, err = errors.Convert01(regexp.Compile(config.PrType))
		if err != nil {
			return errors.Default.Wrap(err, "regexp Compile prType failed")
		}
	}
	if config != nil && len(config.PrComponent) > 0 {
		labelComponentRegex, err = errors.Convert01(regexp.Compile(config.PrComponent))
		if err != nil {
			return errors.Default.Wrap(err, "regexp Compile prComponent failed")
		}
	}
	extractor, err := api.NewApiExtractor(api.ApiExtractorArgs{
		RawDataSubTaskArgs: api.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: githubTasks.GithubApiParams{
				ConnectionId: data.Options.ConnectionId,
				Name:         data.Options.Name,
			},
			Table: RAW_PRS_TABLE,
		},
		Extract: func(row *api.RawData) ([]interface{}, errors.Error) {
			rawL := &GraphqlQueryPr{}
			err := errors.Convert(json.Unmarshal(row.Data, rawL))
			if err != nil {
				return nil, err
			}

			results := make([]interface{}, 0, 1)
			githubPr, err := convertGithubPullRequest(rawL, data.Options.ConnectionId, data.Options.GithubId)
			if err != nil {
				return nil, err
			}
			extractGraphqlPreAccount(&results, rawL.Author, data.Options.GithubId, data.Options.ConnectionId)
			for _, label := range rawL.Labels.Nodes {
				results = append(results, &models.GithubPrLabel{
					ConnectionId: data.Options.ConnectionId,
					PullId:       githubPr.GithubId,
					LabelName:    label.Name,
				})
				// if pr.Type has not been set and prType is set in .env, process the below
				if labelTypeRegex != nil && labelTypeRegex.MatchString(label.Name) {
					githubPr.Type = label.Name
				}
				// if pr.Component has not been set and prComponent is set in .env, process
				if labelComponentRegex != nil && labelComponentRegex.MatchString(label.Name) {
					githubPr.Component = label.Name

				}
			}
			results = append(results, githubPr)

			for _, apiPullRequestReview := range rawL.Reviews.Nodes {
				if apiPullRequestReview.State != "PENDING" {
					githubPrReview := &models.GithubPrReview{
						ConnectionId:   data.Options.ConnectionId,
						GithubId:       apiPullRequestReview.DatabaseId,
						Body:           apiPullRequestReview.Body,
						State:          apiPullRequestReview.State,
						CommitSha:      apiPullRequestReview.Commit.Oid,
						GithubSubmitAt: apiPullRequestReview.SubmittedAt,

						PullRequestId: githubPr.GithubId,
					}

					if apiPullRequestReview.Author != nil {
						githubPrReview.AuthorUserId = apiPullRequestReview.Author.Id
						githubPrReview.AuthorUsername = apiPullRequestReview.Author.Login
						extractGraphqlPreAccount(&results, apiPullRequestReview.Author, data.Options.GithubId, data.Options.ConnectionId)
					}

					results = append(results, githubPrReview)
				}
			}
			for _, apiReviewRequests := range rawL.ReviewRequests.Nodes {
				githubReviewRequests := &models.GithubReviewer{
					ConnectionId:  data.Options.ConnectionId,
					PullRequestId: githubPr.GithubId,
					ReviewerId:    apiReviewRequests.RequestedReviewer.User.Id,
					Username:      apiReviewRequests.RequestedReviewer.User.Login,
				}
				results = append(results, githubReviewRequests)
			}

			for _, reviewReq := range rawL.ReviewRequests.Nodes {
				results = append(results, &graphqlModels.GithubPrReviewRequest{
					ConnectionId:  data.Options.ConnectionId,
					Repo:          data.Options.Name,
					PrNumber:      githubPr.Number,
					PrId:          prDidGen.Generate(data.Options.ConnectionId, githubPr.GithubId),
					PrGithubId:    githubPr.GithubId,
					PrUrl:         githubPr.Url,
					ReviewerId:    reviewReq.RequestedReviewer.User.Id,
					ReviewerLogin: reviewReq.RequestedReviewer.User.Login,
					ReviewerName:  reviewReq.RequestedReviewer.User.Name,
				})
			}
			for _, review := range rawL.Reviews.Nodes {
				if review.Author != nil && review.State != "PENDING" {
					results = append(results, &graphqlModels.GithubPrReviewRequest{
						ConnectionId:  data.Options.ConnectionId,
						Repo:          data.Options.Name,
						PrNumber:      githubPr.Number,
						PrId:          prDidGen.Generate(data.Options.ConnectionId, githubPr.GithubId),
						PrGithubId:    githubPr.GithubId,
						PrUrl:         githubPr.Url,
						ReviewerId:    review.Author.Id,
						ReviewerLogin: review.Author.Login,
						ReviewerName:  review.Author.Name,
					})
				}
			}
			for _, apiPullRequestCommit := range rawL.Commits.Nodes {
				githubCommit, err := convertPullRequestCommit(apiPullRequestCommit)
				if err != nil {
					return nil, err
				}
				results = append(results, githubCommit)

				githubPullRequestCommit := &models.GithubPrCommit{
					ConnectionId:       data.Options.ConnectionId,
					CommitSha:          apiPullRequestCommit.Commit.Oid,
					PullRequestId:      githubPr.GithubId,
					CommitAuthorName:   githubCommit.AuthorName,
					CommitAuthorEmail:  githubCommit.AuthorEmail,
					CommitAuthoredDate: githubCommit.AuthoredDate,
				}
				results = append(results, githubPullRequestCommit)
				results = append(results, &models.GithubRepoCommit{
					ConnectionId: data.Options.ConnectionId,
					RepoId:       data.Options.GithubId,
					CommitSha:    apiPullRequestCommit.Commit.Oid,
				})
				extractGraphqlPreAccount(&results, apiPullRequestCommit.Commit.Author.User, data.Options.GithubId, data.Options.ConnectionId)
			}

			for _, thread := range rawL.ReviewThreads.Nodes {
				rt := &graphqlModels.GithubPrReviewThread{
					ConnectionId: data.Options.ConnectionId,
					Repo:         data.Options.Name,
					PrNumber:     githubPr.Number,
					PrId:         prDidGen.Generate(data.Options.ConnectionId, githubPr.GithubId),
					PrGithubId:   githubPr.GithubId,
					PrUrl:        githubPr.Url,
					ThreadId:     thread.Id,
					IsResolved:   thread.IsResolved,
					IsOutdated:   thread.IsOutdated,
				}
				if len(thread.Comments.Nodes) > 0 {
					c := thread.Comments.Nodes[0]
					rt.Comment = c.Body
					rt.GithubCreatedAt = &c.CreatedAt
					if c.Author != nil {
						rt.Author = c.Author.Login
						inlineAccount := &GraphqlInlineAccountQuery{
							GithubAccountEdge: GithubAccountEdge{
								Login: c.Author.Login,
							},
						}
						extractGraphqlPreAccount(&results, inlineAccount, data.Options.GithubId, data.Options.ConnectionId) /// added by me
					}
				}
				results = append(results, rt)

				for _, c := range thread.Comments.Nodes {
					authorLogin := ""
					authorType := ""
					if c.Author != nil {
						authorLogin = c.Author.Login
						authorType = c.Author.TypeName /// User, Bot, Mannequin etc
					}
					results = append(results, &graphqlModels.GithubPrReviewThreadComment{
						ConnectionId:    data.Options.ConnectionId,
						Repo:            data.Options.Name,
						PrNumber:        githubPr.Number,
						ThreadId:        thread.Id,
						CommentId:       c.Id,
						PrUrl:           githubPr.Url,
						PrId:            prDidGen.Generate(data.Options.ConnectionId, githubPr.GithubId),
						AuthorLogin:     authorLogin,
						AuthorType:      authorType,
						Body:            c.Body,
						GithubCreatedAt: &c.CreatedAt,
					})
				}
			}
			return results, nil
		},
	})

	if err != nil {
		return err
	}

	return extractor.Execute()
}

func convertGithubPullRequest(pull *GraphqlQueryPr, connId uint64, repoId int) (*models.GithubPullRequest, errors.Error) {
	githubPull := &models.GithubPullRequest{
		ConnectionId:    connId,
		GithubId:        pull.DatabaseId,
		RepoId:          repoId,
		Number:          pull.Number,
		State:           pull.State,
		Title:           pull.Title,
		Url:             pull.Url,
		GithubCreatedAt: pull.CreatedAt,
		GithubUpdatedAt: pull.UpdatedAt,
		ClosedAt:        pull.ClosedAt,
		MergedAt:        pull.MergedAt,
		Body:            pull.Body,
		BaseRef:         pull.BaseRefName,
		BaseCommitSha:   pull.BaseRefOid,
		HeadRef:         pull.HeadRefName,
		HeadCommitSha:   pull.HeadRefOid,
		Additions:       pull.Additions,
		Deletions:       pull.Deletions,
		IsDraft:         pull.IsDraft,
	}
	if pull.MergedBy != nil {
		githubPull.MergedByName = pull.MergedBy.Login
		githubPull.MergedById = pull.MergedBy.Id
	}
	if pull.MergeCommit != nil {
		githubPull.MergeCommitSha = pull.MergeCommit.Oid
	}
	if pull.Author != nil {
		githubPull.AuthorName = pull.Author.Login
		githubPull.AuthorId = pull.Author.Id
	}
	return githubPull, nil
}

func convertPullRequestCommit(prCommit GraphqlQueryCommit) (*models.GithubCommit, errors.Error) {
	githubCommit := &models.GithubCommit{
		Sha:            prCommit.Commit.Oid,
		Message:        prCommit.Commit.Message,
		AuthorName:     prCommit.Commit.Author.Name,
		AuthorEmail:    prCommit.Commit.Author.Email,
		AuthoredDate:   prCommit.Commit.Author.Date,
		CommitterName:  prCommit.Commit.Committer.Name,
		CommitterEmail: prCommit.Commit.Committer.Email,
		CommittedDate:  prCommit.Commit.Committer.Date,
		Url:            prCommit.Url,
	}
	if prCommit.Commit.Author.User != nil {
		githubCommit.AuthorId = prCommit.Commit.Author.User.Id
		githubCommit.AuthorLogin = prCommit.Commit.Author.User.Login
	}
	return githubCommit, nil
}
