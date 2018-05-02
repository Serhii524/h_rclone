// +build go1.9

// Copyright 2018 Microsoft Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This code was auto-generated by:
// github.com/Azure/azure-sdk-for-go/tools/profileBuilder

package infrastructureinsights

import original "github.com/Azure/azure-sdk-for-go/services/azsadmin/mgmt/2016-05-01/infrastructureinsights"

type AlertsClient = original.AlertsClient

const (
	DefaultBaseURI = original.DefaultBaseURI
)

type BaseClient = original.BaseClient
type MetricsSourceType = original.MetricsSourceType

const (
	PhysicalNode     MetricsSourceType = original.PhysicalNode
	ResourceProvider MetricsSourceType = original.ResourceProvider
	VirtualMachine   MetricsSourceType = original.VirtualMachine
)

type MetricsUnit = original.MetricsUnit

const (
	B          MetricsUnit = original.B
	GB         MetricsUnit = original.GB
	KB         MetricsUnit = original.KB
	MB         MetricsUnit = original.MB
	One        MetricsUnit = original.One
	Percentage MetricsUnit = original.Percentage
	TB         MetricsUnit = original.TB
)

type Alert = original.Alert
type AlertList = original.AlertList
type AlertListIterator = original.AlertListIterator
type AlertListPage = original.AlertListPage
type AlertModel = original.AlertModel
type AlertSummary = original.AlertSummary
type BaseHealth = original.BaseHealth
type Metrics = original.Metrics
type RegionHealth = original.RegionHealth
type RegionHealthList = original.RegionHealthList
type RegionHealthListIterator = original.RegionHealthListIterator
type RegionHealthListPage = original.RegionHealthListPage
type RegionHealthModel = original.RegionHealthModel
type Resource = original.Resource
type ResourceHealth = original.ResourceHealth
type ResourceHealthList = original.ResourceHealthList
type ResourceHealthListIterator = original.ResourceHealthListIterator
type ResourceHealthListPage = original.ResourceHealthListPage
type ResourceHealthModel = original.ResourceHealthModel
type ServiceHealth = original.ServiceHealth
type ServiceHealthList = original.ServiceHealthList
type ServiceHealthListIterator = original.ServiceHealthListIterator
type ServiceHealthListPage = original.ServiceHealthListPage
type ServiceHealthModel = original.ServiceHealthModel
type UsageMetrics = original.UsageMetrics
type RegionHealthsClient = original.RegionHealthsClient
type ResourceHealthsClient = original.ResourceHealthsClient
type ServiceHealthsClient = original.ServiceHealthsClient

func NewAlertsClient(subscriptionID string) AlertsClient {
	return original.NewAlertsClient(subscriptionID)
}
func NewAlertsClientWithBaseURI(baseURI string, subscriptionID string) AlertsClient {
	return original.NewAlertsClientWithBaseURI(baseURI, subscriptionID)
}
func New(subscriptionID string) BaseClient {
	return original.New(subscriptionID)
}
func NewWithBaseURI(baseURI string, subscriptionID string) BaseClient {
	return original.NewWithBaseURI(baseURI, subscriptionID)
}
func PossibleMetricsSourceTypeValues() []MetricsSourceType {
	return original.PossibleMetricsSourceTypeValues()
}
func PossibleMetricsUnitValues() []MetricsUnit {
	return original.PossibleMetricsUnitValues()
}
func NewRegionHealthsClient(subscriptionID string) RegionHealthsClient {
	return original.NewRegionHealthsClient(subscriptionID)
}
func NewRegionHealthsClientWithBaseURI(baseURI string, subscriptionID string) RegionHealthsClient {
	return original.NewRegionHealthsClientWithBaseURI(baseURI, subscriptionID)
}
func NewResourceHealthsClient(subscriptionID string) ResourceHealthsClient {
	return original.NewResourceHealthsClient(subscriptionID)
}
func NewResourceHealthsClientWithBaseURI(baseURI string, subscriptionID string) ResourceHealthsClient {
	return original.NewResourceHealthsClientWithBaseURI(baseURI, subscriptionID)
}
func NewServiceHealthsClient(subscriptionID string) ServiceHealthsClient {
	return original.NewServiceHealthsClient(subscriptionID)
}
func NewServiceHealthsClientWithBaseURI(baseURI string, subscriptionID string) ServiceHealthsClient {
	return original.NewServiceHealthsClientWithBaseURI(baseURI, subscriptionID)
}
func UserAgent() string {
	return original.UserAgent() + " profiles/latest"
}
func Version() string {
	return original.Version()
}
