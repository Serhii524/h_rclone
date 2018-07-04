package adhybridhealthservice

// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"context"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"net/http"
)

// ReportsClient is the REST APIs for Azure Active Drectory Connect Health
type ReportsClient struct {
	BaseClient
}

// NewReportsClient creates an instance of the ReportsClient client.
func NewReportsClient() ReportsClient {
	return NewReportsClientWithBaseURI(DefaultBaseURI)
}

// NewReportsClientWithBaseURI creates an instance of the ReportsClient client.
func NewReportsClientWithBaseURI(baseURI string) ReportsClient {
	return ReportsClient{NewWithBaseURI(baseURI)}
}

// GetDevOps checks if the user is enabled for Dev Ops access.
func (client ReportsClient) GetDevOps(ctx context.Context) (result Result, err error) {
	req, err := client.GetDevOpsPreparer(ctx)
	if err != nil {
		err = autorest.NewErrorWithError(err, "adhybridhealthservice.ReportsClient", "GetDevOps", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetDevOpsSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "adhybridhealthservice.ReportsClient", "GetDevOps", resp, "Failure sending request")
		return
	}

	result, err = client.GetDevOpsResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "adhybridhealthservice.ReportsClient", "GetDevOps", resp, "Failure responding to request")
	}

	return
}

// GetDevOpsPreparer prepares the GetDevOps request.
func (client ReportsClient) GetDevOpsPreparer(ctx context.Context) (*http.Request, error) {
	const APIVersion = "2014-01-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPath("/providers/Microsoft.ADHybridHealthService/reports/DevOps/IsDevOps"),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetDevOpsSender sends the GetDevOps request. The method will close the
// http.Response Body if it receives an error.
func (client ReportsClient) GetDevOpsSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		autorest.DoRetryForStatusCodes(client.RetryAttempts, client.RetryDuration, autorest.StatusCodesForRetry...))
}

// GetDevOpsResponder handles the response to the GetDevOps request. The method always
// closes the http.Response Body.
func (client ReportsClient) GetDevOpsResponder(resp *http.Response) (result Result, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}
