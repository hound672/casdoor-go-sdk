// Copyright 2021 The Casdoor Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package casdoorsdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

var (
	// client is a shared http Client.
	client HttpClient = &http.Client{}
)

// SetHttpClient sets custom http Client.
func SetHttpClient(httpClient HttpClient) {
	client = httpClient
}

// HttpClient interface has the method required to use a type as custom http client.
// The net/*http.Client type satisfies this interface.
type HttpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Response struct {
	Status string      `json:"status"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
	Data2  interface{} `json:"data2"`
}

// DoGetResponse is a general function to get response from param url through HTTP Get method.
func DoGetResponse(url string) (*Response, error) {
	respBytes, err := DoGetBytesRaw(url)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return nil, err
	}

	if response.Status != "ok" {
		return nil, fmt.Errorf(response.Msg)
	}

	return &response, nil
}

// DoGetBytes is a general function to get response data in bytes from param url through HTTP Get method.
func DoGetBytes(url string) ([]byte, error) {
	response, err := DoGetResponse(url)
	if err != nil {
		return nil, err
	}

	res, err := json.Marshal(response.Data)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// DoGetBytesRaw is a general function to get response from param url through HTTP Get method.
func DoGetBytesRaw(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(authConfig.ClientId, authConfig.ClientSecret)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respBytes, nil
}

func DoPost(action string, queryMap map[string]string, postBytes []byte, isForm, isFile bool) (*Response, error) {
	url := GetUrl(action, queryMap)

	var err error
	var contentType string
	var body io.Reader
	if isForm {
		if isFile {
			contentType, body, err = createFormFile(map[string][]byte{"file": postBytes})
			if err != nil {
				return nil, err
			}
		} else {
			var params map[string]string
			err = json.Unmarshal(postBytes, &params)
			if err != nil {
				return nil, err
			}

			contentType, body, err = createForm(params)
			if err != nil {
				return nil, err
			}
		}
	} else {
		contentType = "text/plain;charset=UTF-8"
		body = bytes.NewReader(postBytes)
	}

	respBytes, err := DoPostBytesRaw(url, contentType, body)
	if err != nil {
		return nil, err
	}

	var response Response
	err = json.Unmarshal(respBytes, &response)
	if err != nil {
		return nil, err
	}

	if response.Status != "ok" {
		return nil, fmt.Errorf(response.Msg)
	}

	return &response, nil
}

// DoPostBytesRaw is a general function to post a request from url, body through HTTP Post method.
func DoPostBytesRaw(url string, contentType string, body io.Reader) ([]byte, error) {
	if contentType == "" {
		contentType = "text/plain;charset=UTF-8"
	}

	var resp *http.Response

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(authConfig.ClientId, authConfig.ClientSecret)
	req.Header.Set("Content-Type", contentType)

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	respByte, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respByte, nil
}

// modifyUser is an encapsulation of user CUD(Create, Update, Delete) operations.
// possible actions are `add-user`, `update-user`, `delete-user`,
func modifyUser(action string, user *User, columns []string) (*Response, bool, error) {
	return modifyUserById(action, user.GetId(), user, columns)
}

func modifyUserById(action string, id string, user *User, columns []string) (*Response, bool, error) {
	queryMap := map[string]string{
		"id": id,
	}

	if len(columns) != 0 {
		queryMap["columns"] = strings.Join(columns, ",")
	}

	user.Owner = authConfig.OrganizationName
	postBytes, err := json.Marshal(user)
	if err != nil {
		return nil, false, err
	}

	resp, err := DoPost(action, queryMap, postBytes, false, false)
	if err != nil {
		return nil, false, err
	}

	return resp, resp.Data == "Affected", nil
}

// modifyPermission is an encapsulation of permission CUD(Create, Update, Delete) operations.
// possible actions are `add-permission`, `update-permission`, `delete-permission`,
func modifyPermission(action string, permission *Permission, columns []string) (*Response, bool, error) {
	queryMap := map[string]string{
		"id": fmt.Sprintf("%s/%s", permission.Owner, permission.Name),
	}

	if len(columns) != 0 {
		queryMap["columns"] = strings.Join(columns, ",")
	}

	permission.Owner = authConfig.OrganizationName
	postBytes, err := json.Marshal(permission)
	if err != nil {
		return nil, false, err
	}

	resp, err := DoPost(action, queryMap, postBytes, false, false)
	if err != nil {
		return nil, false, err
	}

	return resp, resp.Data == "Affected", nil
}

// modifyRole is an encapsulation of role CUD(Create, Update, Delete) operations.
// possible actions are `add-role`, `update-role`, `delete-role`,
func modifyRole(action string, role *Role, columns []string) (*Response, bool, error) {
	queryMap := map[string]string{
		"id": fmt.Sprintf("%s/%s", role.Owner, role.Name),
	}

	if len(columns) != 0 {
		queryMap["columns"] = strings.Join(columns, ",")
	}

	role.Owner = authConfig.OrganizationName
	postBytes, err := json.Marshal(role)
	if err != nil {
		return nil, false, err
	}

	resp, err := DoPost(action, queryMap, postBytes, false, false)
	if err != nil {
		return nil, false, err
	}

	return resp, resp.Data == "Affected", nil
}
