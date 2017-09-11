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

package router

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/caicloud/cyclone/pkg/api"
	httputil "github.com/caicloud/cyclone/pkg/util/http"
	"github.com/emicklei/go-restful"
	"github.com/zoumo/logdog"
)

// triggerPipelineByWebhook handles the request to trigger a pipeline by webhook.
func (router *router) triggerPipelineByWebhook(request *restful.Request, response *restful.Response) {
	scmType := request.PathParameter(scmTypePathParameterName)
	pipelineID := request.PathParameter(pipelineIDPathParameterName)

	requestBodyMap := make(map[string]interface{})
	if err := httputil.ReadEntityFromRequest(request, response, requestBodyMap); err != nil {
		return
	}

	params, err := parseTriggerParamsFromWebhook(api.SCMType(scmType), requestBodyMap)
	if err != nil {
		logdog.Error(err.Error())
		httputil.ResponseWithError(response, http.StatusInternalServerError, err)
		return
	}

	if err := router.webhookManager.TriggerPipeline(pipelineID, params); err != nil {
		httputil.ResponseWithError(response, http.StatusInternalServerError, err)
		return
	}

	response.WriteHeaderAndEntity(http.StatusOK, nil)
}

//parseTriggerParamsFromWebhook parses the trigger params from webhook request body.
func parseTriggerParamsFromWebhook(scmType api.SCMType, requestBodyMap map[string]interface{}) (*api.WebhookTriggerParams, error) {
	switch scmType {
	case api.GitLab:
		objectKind := requestBodyMap["object_kind"].(string)

		ref := ""
		if refParts := strings.Split(requestBodyMap["ref"].(string), "/"); len(refParts) == 3 {
			ref = refParts[2]
		}

		if objectKind == "push" {
			params := &api.WebhookTriggerParams{
				Ref:      ref,
				CommitID: requestBodyMap["checkout_sha"].(string),
			}

			return params, nil
		}

		return nil, fmt.Errorf("The event type %s from %s is still not implmented", objectKind, scmType)
	case api.GitHub, api.SVN:
		return nil, fmt.Errorf("Webhook trigger of %s is still not implmented", scmType)
	}

	return nil, fmt.Errorf("Webhook trigger of %s is not supported", scmType)
}
