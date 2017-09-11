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

	"github.com/caicloud/cyclone/pkg/api"
	"github.com/caicloud/cyclone/store"
	"github.com/zoumo/logdog"
)

// WebhookManager represents the interface to manage webhook.
type WebhookManager interface {
	TriggerPipeline(pipelineID string, params *api.WebhookTriggerParams) error
}

// webhookManager represents the manager for webhook.
type webhookManager struct {
	dataStore *store.DataStore
}

// NewWebhookManager creates a webhook manager.
func NewWebhookManager(dataStore *store.DataStore) (WebhookManager, error) {
	if dataStore == nil {
		return nil, fmt.Errorf("Fail to new webhook manager as data store is nil")
	}

	return &webhookManager{dataStore}, nil
}

// TriggerPipeline triggers the pipeline with webhook params.
func (m *webhookManager) TriggerPipeline(pipelineID string, params *api.WebhookTriggerParams) error {
	ds := m.dataStore
	pipeline, err := ds.FindPipelineByID(pipelineID)
	if err != nil {
		return err
	}

	logdog.Infof("The pipeline %s is triggered by webhook with params %v\n", pipeline.Name, params)

	return nil
}
