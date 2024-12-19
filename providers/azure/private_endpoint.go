// Copyright 2021 The Terraformer Authors.
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

package azure

import (
	"context"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
)

type PrivateEndpointGenerator struct {
	AzureService
}

func (az *PrivateEndpointGenerator) listServices() ([]network.PrivateLinkService, error) {
	subscriptionID, resourceGroup, authorizer, resourceManagerEndpoint := az.getClientArgs()
	client := network.NewPrivateLinkServicesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	var (
		iterator network.PrivateLinkServiceListResultIterator
		err      error
	)
	ctx := context.Background()
	var resources []network.PrivateLinkService
	if resourceGroup != "" {
		resourceGroups := strings.Split(resourceGroup, ",")
		// var resources []network.PrivateLinkService
		for _, rgName := range resourceGroups {
			iterator, err = client.ListComplete(ctx, rgName)

			if err != nil {
				return nil, err
			}
			for iterator.NotDone() {
				item := iterator.Value()
				resources = append(resources, item)
				if err := iterator.NextWithContext(ctx); err != nil {
					log.Println(err)
					return resources, err
				}
			}
		}
		return resources, nil

		// iterator, err = client.ListComplete(ctx, resourceGroup)
	} else {
		iterator, err = client.ListBySubscriptionComplete(ctx)
		if err != nil {
			return nil, err
		}

		for iterator.NotDone() {
			item := iterator.Value()
			resources = append(resources, item)
			if err := iterator.NextWithContext(ctx); err != nil {
				log.Println(err)
				return resources, err
			}
		}
		return resources, nil
	}

	// return resources, nil
}

func (az *PrivateEndpointGenerator) AppendServices(link *network.PrivateLinkService) {
	parts := strings.Split(*link.ID, "/")
	resourceGroup := parts[4]
	az.AppendSimpleResource(*link.ID, resourceGroup+"_"+*link.Name, "azurerm_private_link_service")
}

func (az *PrivateEndpointGenerator) listEndpoints() ([]network.PrivateEndpoint, error) {
	subscriptionID, resourceGroup, authorizer, resourceManagerEndpoint := az.getClientArgs()
	client := network.NewPrivateEndpointsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	var (
		iterator network.PrivateEndpointListResultIterator
		err      error
	)
	ctx := context.Background()
	var resources []network.PrivateEndpoint
	if resourceGroup != "" {
		resourceGroups := strings.Split(resourceGroup, ",")
		for _, rgName := range resourceGroups {
			iterator, err = client.ListComplete(ctx, rgName)
			if err != nil {
				return nil, err
			}
			for iterator.NotDone() {
				item := iterator.Value()
				resources = append(resources, item)
				if err := iterator.NextWithContext(ctx); err != nil {
					log.Println(err)
					return resources, err
				}
			}
		}
		return resources, nil

		// iterator, err = client.ListComplete(ctx, resourceGroup)
	} else {
		iterator, err = client.ListBySubscriptionComplete(ctx)
		if err != nil {
			return nil, err
		}

		for iterator.NotDone() {
			item := iterator.Value()
			resources = append(resources, item)
			if err := iterator.NextWithContext(ctx); err != nil {
				log.Println(err)
				return resources, err
			}
		}
		return resources, nil
	}

}

func (az *PrivateEndpointGenerator) AppendEndpoint(link *network.PrivateEndpoint) {
	parts := strings.Split(*link.ID, "/")
	resourceGroup := parts[4]

	az.AppendSimpleResource(*link.ID, resourceGroup+"_"+*link.Name, "azurerm_private_endpoint")
}

func (az *PrivateEndpointGenerator) InitResources() error {

	services, err := az.listServices()
	if err != nil {
		return err
	}
	for _, link := range services {
		az.AppendServices(&link)
	}
	endpoints, err := az.listEndpoints()
	if err != nil {
		return err
	}
	for _, endpoint := range endpoints {
		az.AppendEndpoint(&endpoint)
	}
	return nil
}
