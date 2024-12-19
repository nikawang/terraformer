// Copyright 2019 The Terraformer Authors.
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
	"github.com/Azure/go-autorest/autorest"
	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
	"github.com/hashicorp/go-azure-helpers/authentication"
)

type VirtualNetworkGenerator struct {
	AzureService
}

func (g VirtualNetworkGenerator) createResources(ctx context.Context, iterator network.VirtualNetworkListResultIterator) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for iterator.NotDone() {
		virtualNetwork := iterator.Value()
		tferName := terraformutils.TfSanitize(*virtualNetwork.Name)

		for _, resource := range resources {
			if tferName == resource.ResourceName {
				*virtualNetwork.Name = *virtualNetwork.Name + "_" + *virtualNetwork.ID
			}
		}
		iObj, _ := ParseAzureResourceID(*virtualNetwork.ID)
		resourceGroup := iObj.ResourceGroup
		resources = append(resources, terraformutils.NewSimpleResource(
			*virtualNetwork.ID,
			resourceGroup+"_"+*virtualNetwork.Name,
			"azurerm_virtual_network",
			g.ProviderName,
			[]string{}))

		peeringClient := network.NewVirtualNetworkPeeringsClientWithBaseURI(g.Args["config"].(authentication.Config).CustomResourceManagerEndpoint, g.Args["config"].(authentication.Config).SubscriptionID)
		peeringClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

		// 列出当前虚拟网络的 Peering
		peeringIterator, err := peeringClient.ListComplete(ctx, resourceGroup, *virtualNetwork.Name)
		if err != nil {
			log.Println(err)
			return resources, err
		}
		for peeringIterator.NotDone() {
			peering := peeringIterator.Value()
			resources = append(resources, terraformutils.NewSimpleResource(
				*peering.ID,
				resourceGroup+"_"+*virtualNetwork.Name+"_"+*peering.Name,
				"azurerm_virtual_network_peering",
				g.ProviderName,
				[]string{}))
			if err := peeringIterator.NextWithContext(ctx); err != nil {
				log.Println(err)
				return resources, err
			}
		}
		if err := iterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

func (g *VirtualNetworkGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(authentication.Config).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(authentication.Config).CustomResourceManagerEndpoint
	virtualNetworkClient := network.NewVirtualNetworksClientWithBaseURI(resourceManagerEndpoint, subscriptionID)

	virtualNetworkClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	if rg := g.Args["resource_group"].(string); rg != "" {
		// 将资源组字符串按逗号分割
		resourceGroups := strings.Split(rg, ",")
		for _, rgName := range resourceGroups {
			rgName = strings.TrimSpace(rgName)
			output, err := virtualNetworkClient.ListComplete(ctx, rgName)
			if err != nil {
				return err
			}
			resources, err := g.createResources(ctx, output)
			if err != nil {
				return err
			}
			g.Resources = append(g.Resources, resources...)
		}
	} else {
		output, err := virtualNetworkClient.ListAllComplete(ctx)
		if err != nil {
			return err
		}
		g.Resources, err = g.createResources(ctx, output)
		if err != nil {
			return err
		}
	}
	return nil
}

// func (g *VirtualNetworkGenerator) InitResources() error {
// 	ctx := context.Background()
// 	subscriptionID := g.Args["config"].(authentication.Config).SubscriptionID
// 	resourceManagerEndpoint := g.Args["config"].(authentication.Config).CustomResourceManagerEndpoint
// 	virtualNetworkClient := network.NewVirtualNetworksClientWithBaseURI(resourceManagerEndpoint, subscriptionID)

// 	virtualNetworkClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

// 	var (
// 		output network.VirtualNetworkListResultIterator
// 		err    error
// 	)

// 	if rg := g.Args["resource_group"].(string); rg != "" {
// 		output, err = virtualNetworkClient.ListComplete(ctx, rg)
// 	} else {
// 		output, err = virtualNetworkClient.ListAllComplete(ctx)
// 	}
// 	if err != nil {
// 		return err
// 	}
// 	g.Resources, err = g.createResources(ctx, output)
// 	return err
// }

// NOTE on Virtual Networks and Subnet's:
// Terraform currently provides both a standalone Subnet resource, and allows for Subnets to be defined in-line within the Virtual Network
// resource. At this time you cannot use a Virtual Network with in-line Subnets in conjunction with any Subnet resources.
// Doing so will cause a conflict of Subnet configurations and will overwrite Subnet's.
func (g *VirtualNetworkGenerator) PostConvertHook() error {
	for _, resource := range g.Resources {
		if resource.InstanceInfo.Type != "azurerm_virtual_network" {
			continue
		}
		delete(resource.Item, "subnet")
	}
	return nil
}
