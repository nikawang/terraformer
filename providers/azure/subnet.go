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

type SubnetGenerator struct {
	AzureService
}

// func (az *SubnetGenerator) lisSubnets() ([]network.Subnet, error) {
// 	subscriptionID, resourceGroup, authorizer, resourceManagerEndpoint := az.getClientArgs()
// 	subnetClient := network.NewSubnetsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
// 	subnetClient.Authorizer = authorizer
// 	vnetClient := network.NewVirtualNetworksClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
// 	vnetClient.Authorizer = authorizer
// 	var (
// 		vnetIter   network.VirtualNetworkListResultIterator
// 		subnetIter network.SubnetListResultIterator
// 		err        error
// 	)
// 	ctx := context.Background()
// 	if resourceGroup != "" {
// 		vnetIter, err = vnetClient.ListComplete(ctx, resourceGroup)
// 	} else {
// 		vnetIter, err = vnetClient.ListAllComplete(ctx)
// 	}
// 	if err != nil {
// 		return nil, err
// 	}
// 	var resources []network.Subnet
// 	for vnetIter.NotDone() {
// 		vnet := vnetIter.Value()
// 		vnetID, err := ParseAzureResourceID(*vnet.ID)
// 		if err != nil {
// 			return nil, err
// 		}
// 		subnetIter, err = subnetClient.ListComplete(ctx, vnetID.ResourceGroup, *vnet.Name)
// 		if err != nil {
// 			return nil, err
// 		}
// 		for subnetIter.NotDone() {
// 			item := subnetIter.Value()
// 			resources = append(resources, item)
// 			if err := subnetIter.NextWithContext(ctx); err != nil {
// 				log.Println(err)
// 				return resources, err
// 			}
// 		}
// 		if err := vnetIter.NextWithContext(ctx); err != nil {
// 			log.Println(err)
// 			return resources, err
// 		}
// 	}
// 	return resources, nil
// }

func (az *SubnetGenerator) lisSubnets() ([]network.Subnet, error) {
	subscriptionID, resourceGroup, authorizer, resourceManagerEndpoint := az.getClientArgs()
	subnetClient := network.NewSubnetsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	subnetClient.Authorizer = authorizer
	vnetClient := network.NewVirtualNetworksClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	vnetClient.Authorizer = authorizer
	// var (
	// 	err error
	// )
	ctx := context.Background()
	var resources []network.Subnet

	if resourceGroup != "" {
		// 将 resourceGroup 按逗号分割
		resourceGroups := strings.Split(resourceGroup, ",")
		for _, rgName := range resourceGroups {
			rgName = strings.TrimSpace(rgName)
			log.Default().Println("Subnet Resource Group: ", rgName)
			// 列出该资源组中的虚拟网络
			vnetIter, err := vnetClient.ListComplete(ctx, rgName)
			if err != nil {
				log.Default().Println("subnet error: ", err)
				return nil, err
			}
			for vnetIter.NotDone() {
				vnet := vnetIter.Value()
				vnetID, err := ParseAzureResourceID(*vnet.ID)
				if err != nil {
					return nil, err
				}
				subnetIter, err := subnetClient.ListComplete(ctx, vnetID.ResourceGroup, *vnet.Name)
				if err != nil {
					return nil, err
				}
				for subnetIter.NotDone() {
					item := subnetIter.Value()
					resources = append(resources, item)
					if err := subnetIter.NextWithContext(ctx); err != nil {
						log.Println(err)
						return resources, err
					}
				}
				if err := vnetIter.NextWithContext(ctx); err != nil {
					log.Println(err)
					return resources, err
				}
			}
		}
	} else {
		// 如果 resourceGroup 为空，列出所有订阅中的虚拟网络
		vnetIter, err := vnetClient.ListAllComplete(ctx)
		if err != nil {
			return nil, err
		}
		for vnetIter.NotDone() {
			vnet := vnetIter.Value()
			vnetID, err := ParseAzureResourceID(*vnet.ID)
			if err != nil {
				return nil, err
			}
			subnetIter, err := subnetClient.ListComplete(ctx, vnetID.ResourceGroup, *vnet.Name)
			if err != nil {
				return nil, err
			}
			for subnetIter.NotDone() {
				item := subnetIter.Value()
				resources = append(resources, item)
				if err := subnetIter.NextWithContext(ctx); err != nil {
					log.Println(err)
					return resources, err
				}
			}
			if err := vnetIter.NextWithContext(ctx); err != nil {
				log.Println(err)
				return resources, err
			}
		}
	}
	return resources, nil
}

func (az *SubnetGenerator) AppendSubnet(subnet *network.Subnet) {
	// get vnet name from azure subnet id
	parts := strings.Split(*subnet.ID, "/")
	resourceGroup := parts[4]
	vnetName := parts[8]
	az.AppendSimpleResource(*subnet.ID, resourceGroup+"_"+vnetName+"_"+*subnet.Name, "azurerm_subnet")
}

func (az *SubnetGenerator) appendRouteTable(subnet *network.Subnet) {
	parts := strings.Split(*subnet.ID, "/")
	resourceGroup := parts[4]
	vnetName := parts[8]
	if props := subnet.SubnetPropertiesFormat; props != nil {
		if prop := props.RouteTable; prop != nil {
			az.appendSimpleAssociation(
				*subnet.ID, resourceGroup+"_"+vnetName+"_"+*subnet.Name, prop.Name,
				"azurerm_subnet_route_table_association",
				map[string]string{
					"subnet_id":      *subnet.ID,
					"route_table_id": *prop.ID,
				})
		}
	}
}

func (az *SubnetGenerator) appendNetworkSecurityGroupAssociation(subnet *network.Subnet) {
	parts := strings.Split(*subnet.ID, "/")
	resourceGroup := parts[4]
	vnetName := parts[8]
	if props := subnet.SubnetPropertiesFormat; props != nil {
		if prop := props.NetworkSecurityGroup; prop != nil {
			az.appendSimpleAssociation(
				*subnet.ID, resourceGroup+"_"+vnetName+"_"+*subnet.Name, prop.Name,
				"azurerm_subnet_network_security_group_association",
				map[string]string{
					"subnet_id":                 *subnet.ID,
					"network_security_group_id": *prop.ID,
				})
		}
	}
}

func (az *SubnetGenerator) appendNatGateway(subnet *network.Subnet) {
	parts := strings.Split(*subnet.ID, "/")
	resourceGroup := parts[4]
	vnetName := parts[8]
	if props := subnet.SubnetPropertiesFormat; props != nil {
		if prop := props.NatGateway; prop != nil {
			az.appendSimpleAssociation(
				*subnet.ID, resourceGroup+"_"+vnetName+"_"+*subnet.Name, nil,
				"azurerm_subnet_nat_gateway_association",
				map[string]string{
					"subnet_id":      *subnet.ID,
					"nat_gateway_id": *prop.ID,
				})
		}
	}
}

func (az *SubnetGenerator) appendServiceEndpointPolicies() error {
	subscriptionID, resourceGroup, authorizer, resourceManagerEndpoint := az.getClientArgs()
	client := network.NewServiceEndpointPoliciesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	client.Authorizer = authorizer
	var (
		iterator network.ServiceEndpointPolicyListResultIterator
		err      error
	)
	ctx := context.Background()
	if resourceGroup != "" {
		resourceGroups := strings.Split(resourceGroup, ",")
		for _, rgName := range resourceGroups {
			iterator, err = client.ListByResourceGroupComplete(ctx, rgName)
			if err != nil {
				return err
			}
			for iterator.NotDone() {
				item := iterator.Value()
				az.AppendSimpleResource(*item.ID, rgName+"_"+*item.Name, "azurerm_subnet_service_endpoint_policy")
				if err := iterator.NextWithContext(ctx); err != nil {
					log.Println(err)
					return err
				}
			}
		}
		// iterator, err = client.ListByResourceGroupComplete(ctx, resourceGroup)
	} else {
		iterator, err = client.ListComplete(ctx)
		for iterator.NotDone() {
			item := iterator.Value()
			az.AppendSimpleResource(*item.ID, *item.Name, "azurerm_subnet_service_endpoint_storage_policy")
			if err := iterator.NextWithContext(ctx); err != nil {
				log.Println(err)
				return err
			}
		}
		return err
	}
	if err != nil {
		return err
	}

	return nil
}

func (az *SubnetGenerator) InitResources() error {

	subnets, err := az.lisSubnets()
	if err != nil {
		return err
	}
	for _, subnet := range subnets {
		az.AppendSubnet(&subnet)
		az.appendRouteTable(&subnet)
		az.appendNetworkSecurityGroupAssociation(&subnet)
		az.appendNatGateway(&subnet)
	}
	if err := az.appendServiceEndpointPolicies(); err != nil {
		return err
	}
	return nil
}

func (az *SubnetGenerator) PostConvertHook() error {
	for _, resource := range az.Resources {
		if resource.InstanceInfo.Type != "azurerm_subnet" {
			continue
		}
		delete(resource.Item, "address_prefix")
	}
	return nil
}
