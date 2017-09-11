/*
Copyright 2017 caicloud authors. All rights reserved.

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

package manager

import (
	"fmt"
	"os"
	"strings"

	"github.com/caicloud/cyclone/pkg/api"
	"github.com/caicloud/cyclone/store"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zoumo/logdog"
	"gopkg.in/mgo.v2/bson"
)

// PipelineManager represents the interface to manage pipeline.
type PipelineManager interface {
	CreatePipeline(pipeline *api.Pipeline) (*api.Pipeline, error)
	GetPipeline(projectName string, pipelineName string) (*api.Pipeline, error)
	ListPipelines(projectName string, queryParams api.QueryParams) ([]api.Pipeline, int, error)
	UpdatePipeline(projectName string, pipelineName string, newPipeline *api.Pipeline) (*api.Pipeline, error)
	DeletePipeline(projectName string, pipelineName string) error
	ClearPipelinesOfProject(projectID string) error
}

// pipelineManager represents the manager for pipeline.
type pipelineManager struct {
	dataStore             *store.DataStore
	pipelineRecordManager PipelineRecordManager
}

// NewPipelineManager creates a pipeline manager.
func NewPipelineManager(dataStore *store.DataStore, pipelineRecordManager PipelineRecordManager) (PipelineManager, error) {
	if dataStore == nil {
		return nil, fmt.Errorf("Fail to new pipeline manager as data store is nil")
	}

	if pipelineRecordManager == nil {
		return nil, fmt.Errorf("Fail to new pipeline manager as pipeline record is nil")
	}

	return &pipelineManager{dataStore, pipelineRecordManager}, nil
}

// CreatePipeline creates a pipeline.
func (m *pipelineManager) CreatePipeline(pipeline *api.Pipeline) (*api.Pipeline, error) {
	createdPipeline, err := m.dataStore.CreatePipeline(pipeline)
	if err != nil {
		return nil, err
	}

	// Create the webhook for this pipeline.
	if pipeline.AutoTrigger != nil && pipeline.AutoTrigger.SCMTrigger != nil {
		for _, codeSourece := range pipeline.Build.Stages.CodeCheckout.CodeSources {
			// Only create webhook for main repository.
			if codeSourece.Main {
				if err := m.createWebhook(createdPipeline.ProjectID, createdPipeline.ID, codeSourece); err != nil {
					logdog.Error(err.Error())
				}
			}
		}
	}

	return createdPipeline, nil
}

// GetPipeline gets the pipeline by name in one project.
func (m *pipelineManager) GetPipeline(projectName string, pipelineName string) (*api.Pipeline, error) {
	project, err := m.dataStore.FindProjectByName(projectName)
	if err != nil {
		return nil, err
	}

	return m.dataStore.FindPipelineByName(project.ID, pipelineName)
}

// ListPipelines lists all pipelines in one project.
func (m *pipelineManager) ListPipelines(projectName string, queryParams api.QueryParams) ([]api.Pipeline, int, error) {
	project, err := m.dataStore.FindProjectByName(projectName)
	if err != nil {
		return nil, 0, err
	}

	return m.dataStore.FindPipelinesByProjectID(project.ID, queryParams)
}

// UpdatePipeline updates the pipeline by name in one project.
func (m *pipelineManager) UpdatePipeline(projectName string, pipelineName string, newPipeline *api.Pipeline) (*api.Pipeline, error) {
	pipeline, err := m.GetPipeline(projectName, pipelineName)
	if err != nil {
		return nil, err
	}

	// Update the properties of the pipeline.
	// TODO (robin) Whether need a method for this merge?
	if len(newPipeline.Name) > 0 {
		pipeline.Name = newPipeline.Name
	}

	if len(newPipeline.Description) > 0 {
		pipeline.Description = newPipeline.Description
	}

	if len(newPipeline.Owner) > 0 {
		pipeline.Owner = newPipeline.Owner
	}

	if newPipeline.Build != nil {
		pipeline.Build = newPipeline.Build
	}

	if newPipeline.AutoTrigger != nil {
		pipeline.AutoTrigger = newPipeline.AutoTrigger
	}

	if err = m.dataStore.UpdatePipeline(pipeline); err != nil {
		return nil, err
	}

	return pipeline, nil
}

// DeletePipeline deletes the pipeline by name in one project.
func (m *pipelineManager) DeletePipeline(projectName string, pipelineName string) error {
	pipeline, err := m.GetPipeline(projectName, pipelineName)
	if err != nil {
		return err
	}

	// Delete the pipeline records of this pipeline.
	if err = m.pipelineRecordManager.ClearPipelineRecordsOfPipeline(pipeline.ID); err != nil {
		logdog.Errorf("Fail to delete all pipeline records of the pipeline %s in the project %s as %s", pipelineName, projectName, err.Error())
		return err
	}

	if err = m.dataStore.DeletePipelineByID(pipeline.ID); err != nil {
		logdog.Errorf("Fail to delete the pipeline %s in the project %s as %s", pipelineName, projectName, err.Error())
		return err
	}

	return nil
}

// ClearPipelinesOfProject deletes all pipelines in one project.
func (m *pipelineManager) ClearPipelinesOfProject(projectID string) error {
	// Delete the pipeline records of this project.
	pipelines, count, err := m.dataStore.FindPipelinesByProjectID(projectID, api.QueryParams{})
	for i := 0; i < count; i++ {
		if err = m.pipelineRecordManager.ClearPipelineRecordsOfPipeline(pipelines[i].ID); err != nil {
			logdog.Errorf("Fail to delete all pipeline records of the project id is %s as %s", projectID, err.Error())
			return err
		}
	}
	return m.dataStore.DeletePipelinesByProjectID(projectID)
}

// createWebhook creates webhook for pipeline.
func (m *pipelineManager) createWebhook(projectID string, pipelineID string, codeSource *api.CodeSource) error {
	scmType := codeSource.Type
	switch scmType {
	case api.GitLab:
		gitSource := codeSource.GitLab
		url := gitSource.URL
		httpsPrefix := "https://"
		httpPrefix := "http://"
		scmServer := ""

		if strings.HasPrefix(url, httpsPrefix) {
			scmServer = httpsPrefix + strings.Split(strings.TrimPrefix(url, httpsPrefix), "/")[0]
		} else if strings.HasPrefix(url, httpPrefix) {
			scmServer = httpPrefix + strings.Split(strings.TrimPrefix(url, httpPrefix), "/")[0]
		} else {
			return fmt.Errorf("The url of code source %s is not correct", url)
		}

		// Find the SCM token.
		ds := m.dataStore
		query := bson.M{"projectId": projectID, "server": scmServer}
		accessToken := ""
		if token, err := ds.FindSCMTokenByQuery(query); err == nil {
			accessToken = token.AccessToken
		}

		// Create the webhook.
		client := gitlab.NewOAuthClient(nil, accessToken)
		client.SetBaseURL(scmServer + "/api/v3/")

		cycloneServer := os.Getenv("CYCLONE_SERVER")
		if cycloneServer == "" {
			cycloneServer = "http://127.0.0.1:7099"
		}
		state := true
		hookURL := fmt.Sprintf("%s/api/%s/webhooks/%s/pipelines/%s", cycloneServer, api.APIVersion, scmType, pipelineID)
		hook := &gitlab.AddProjectHookOptions{
			URL:                 &hookURL,
			PushEvents:          &state,
			MergeRequestsEvents: &state,
			TagPushEvents:       &state,
		}

		owner, name := parseURL(url)
		if _, _, err := client.Projects.AddProjectHook(owner+"/"+name, hook); err != nil {
			return fmt.Errorf("Fail to create webhook for pipeline %s as %s", err.Error())
		}

		return nil
	case api.GitHub, api.SVN:
		return fmt.Errorf("Webhook trigger of %s is still not implmented", scmType)
	}

	return fmt.Errorf("Webhook trigger of %s is not supported", scmType)
}

// parseURL is a helper func to parse the url,such as https://github.com/caicloud/test.git
// to return owner(caicloud) and name(test).
func parseURL(url string) (string, string) {
	strs := strings.SplitN(url, "/", -1)
	name := strings.SplitN(strs[4], ".", -1)
	return strs[3], name[0]
}
