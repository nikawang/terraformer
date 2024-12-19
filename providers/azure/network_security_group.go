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

type NetworkSecurityGroupGenerator struct {
	AzureService
}

func (az *NetworkSecurityGroupGenerator) listResources() ([]network.SecurityGroup, error) {
	subscriptionID, resourceGroup, authorizer, resourceManagerEndpoint := az.getClientArgs()
	client := network.NewSecurityGroupsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	var (
		iterator network.SecurityGroupListResultIterator
		err      error
	)
	ctx := context.Background()
	var resources []network.SecurityGroup
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
		iterator, err = client.ListAllComplete(ctx)
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

func (az *NetworkSecurityGroupGenerator) appendResource(resource *network.SecurityGroup) {
	parts := strings.Split(*resource.ID, "/")
	resourceGroup := parts[4]
	az.AppendSimpleResource(*resource.ID, resourceGroup+"_"+*resource.Name, "azurerm_network_security_group")
}

func (az *NetworkSecurityGroupGenerator) appendRules(parent *network.SecurityGroup, resourceGroupID *ResourceID) error {
	subscriptionID, _, authorizer, resourceManagerEndpoint := az.getClientArgs()
	client := network.NewSecurityRulesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	ctx := context.Background()
	iterator, err := client.ListComplete(ctx, resourceGroupID.ResourceGroup, *parent.Name)
	if err != nil {
		return err
	}
	for iterator.NotDone() {
		item := iterator.Value()
		parts := strings.Split(*item.ID, "/")
		resourceGroup := parts[4]
		nsg := parts[8]
		az.AppendSimpleResource(*item.ID, resourceGroup+"_"+nsg+"_"+*item.Name, "azurerm_network_security_rule")
		if err := iterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}

func (az *NetworkSecurityGroupGenerator) InitResources() error {

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
		err = az.appendRules(&resource, resourceGroupID)
		if err != nil {
			return err
		}
	}
	return nil
}
