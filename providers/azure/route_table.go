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

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-03-01/network"
)

type RouteTableGenerator struct {
	AzureService
}

func (az *RouteTableGenerator) listResources() ([]network.RouteTable, error) {
	subscriptionID, resourceGroup, authorizer, resourceManagerEndpoint := az.getClientArgs()
	client := network.NewRouteTablesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	var (
		iterator network.RouteTableListResultIterator
		err      error
	)
	ctx := context.Background()
	if resourceGroup != "" {
		resourceGroups := strings.Split(resourceGroup, ",")
		var resources []network.RouteTable
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
		iterator, err = client.ListAllComplete(ctx)
		if err != nil {
			return nil, err
		}
		var resources []network.RouteTable
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

	return nil, nil

}

func (az *RouteTableGenerator) appendResource(resource *network.RouteTable) {

	parts := strings.Split(*resource.ID, "/")
	resourceGroup := parts[4]
	// log.Println("resourceGroup:\t", resourceGroup)
	// log.Println("resourceID:\t", *resource.ID)
	// log.Println("resourceName:\t", *resource.Name)
	// vnetName := parts[8]
	az.AppendSimpleResource(*resource.ID, resourceGroup+"_"+*resource.Name, "azurerm_route_table")
}

func (az *RouteTableGenerator) appendRoutes(parent *network.RouteTable, resourceGroupID *ResourceID) error {
	subscriptionID, _, authorizer, resourceManagerEndpoint := az.getClientArgs()
	client := network.NewRoutesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	ctx := context.Background()
	iterator, err := client.ListComplete(ctx, resourceGroupID.ResourceGroup, *parent.Name)
	if err != nil {
		return err
	}
	for iterator.NotDone() {
		item := iterator.Value()
		// log.Println("routeID:\t", *item.ID)
		az.AppendSimpleResource(*item.ID, resourceGroupID.ResourceGroup+"_"+*item.Name, "azurerm_route")
		if err := iterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

func (az *RouteTableGenerator) listRouteFilters() ([]network.RouteFilter, error) {
	subscriptionID, resourceGroup, authorizer, resourceManagerEndpoint := az.getClientArgs()
	client := network.NewRouteFiltersClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	var (
		iterator network.RouteFilterListResultIterator
		err      error
	)
	ctx := context.Background()
	if resourceGroup != "" {
		iterator, err = client.ListByResourceGroupComplete(ctx, resourceGroup)
	} else {
		iterator, err = client.ListComplete(ctx)
	}
	if err != nil {
		return nil, err
	}
	var resources []network.RouteFilter
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

func (az *RouteTableGenerator) appendRouteFilters(resource *network.RouteFilter) {
	parts := strings.Split(*resource.ID, "/")
	resourceGroup := parts[4]
	az.AppendSimpleResource(*resource.ID, resourceGroup+"_"+*resource.Name, "azurerm_route_filter")
}

func (az *RouteTableGenerator) InitResources() error {

	resources, err := az.listResources()
	if err != nil {
		return err
	}
	for _, resource := range resources {
		az.appendResource(&resource)
		resourceGroupID, err := ParseAzureResourceID(*resource.ID)
		if err != nil {
			return err
		}
		err = az.appendRoutes(&resource, resourceGroupID)
		if err != nil {
			return err
		}
	}

	filters, err := az.listRouteFilters()
	if err != nil {
		return err
	}
	for _, resource := range filters {
		az.appendRouteFilters(&resource)
		if err != nil {
			return err
		}
	}
	return nil
}
