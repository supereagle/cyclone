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

package store

import (
	"time"

	"github.com/caicloud/cyclone/pkg/api"
	"gopkg.in/mgo.v2/bson"
)

// CreateSCMToken creates the SCM token, returns the SCM token created.
func (d *DataStore) CreateSCMToken(token *api.SCMToken) (*api.SCMToken, error) {
	token.ID = bson.NewObjectId().Hex()
	token.CreatedTime = time.Now()
	token.UpdatedTime = time.Now()

	if err := d.scmTokenCollection.Insert(token); err != nil {
		return nil, err
	}

	return token, nil
}

// FindSCMTokenByID finds the SCM token by id.
func (d *DataStore) FindSCMTokenByID(tokenID string) (*api.SCMToken, error) {
	token := &api.SCMToken{}
	if err := d.scmTokenCollection.FindId(tokenID).One(token); err != nil {
		return nil, err
	}

	return token, nil
}

// FindSCMTokenByQuery finds the SCM token by query.
func (d *DataStore) FindSCMTokenByQuery(query map[string]interface{}) (*api.SCMToken, error) {
	token := &api.SCMToken{}
	if err := d.scmTokenCollection.Find(query).One(token); err != nil {
		return nil, err
	}

	return token, nil
}

// FindSCMTokensByProjectID finds the SCM tokens by project id, returns all SCM tokens in this project.
func (d *DataStore) FindSCMTokensByProjectID(projectID string, queryParams api.QueryParams) ([]api.SCMToken, int, error) {
	tokens := []api.SCMToken{}
	query := bson.M{"projectId": projectID}
	collection := d.scmTokenCollection.Find(query)

	count, err := collection.Count()
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return tokens, count, nil
	}

	if queryParams.Start > 0 {
		collection.Skip(queryParams.Start)
	}
	if queryParams.Limit > 0 {
		collection.Limit(queryParams.Limit)
	}

	if err = collection.All(&tokens); err != nil {
		return nil, 0, err
	}

	return tokens, count, nil
}

// UpdateSCMToken updates the SCM token, please make sure the SCM token id is provided before call this method.
func (d *DataStore) UpdateSCMToken(token *api.SCMToken) error {
	token.UpdatedTime = time.Now()

	return d.scmTokenCollection.UpdateId(token.ID, token)
}

// DeleteSCMTokenByID deletes the SCM token by id.
func (d *DataStore) DeleteSCMTokenByID(tokenID string) error {
	return d.scmTokenCollection.RemoveId(tokenID)
}

// DeleteSCMTokensByProjectID deletes all the SCM tokens in one project by project id.
func (d *DataStore) DeleteSCMTokensByProjectID(projectID string) error {
	return d.scmTokenCollection.Remove(bson.M{"projectId": projectID})
}
