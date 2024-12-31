package azure

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-08-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
	"github.com/hashicorp/go-azure-helpers/authentication"
)

type FirewallGenerator struct {
	AzureService
}

// createFirewallResources 获取 Azure Firewall 并同时获取其下三种 Rule Collection
func (g *FirewallGenerator) createFirewallResources(ctx context.Context, rgName string) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource

	subscriptionID := g.Args["config"].(authentication.Config).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(authentication.Config).CustomResourceManagerEndpoint

	firewallClient := network.NewAzureFirewallsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	firewallClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	// 如果需要支持 ListAll，可以使用 ListAllComplete。
	firewallIterator, err := firewallClient.ListComplete(ctx, rgName)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for firewallIterator.NotDone() {
		fw := firewallIterator.Value()
		if fw.ID == nil || fw.Name == nil {
			// 避免空指针
			if err := firewallIterator.NextWithContext(ctx); err != nil {
				log.Println(err)
				return resources, err
			}
			continue
		}

		iObj, _ := ParseAzureResourceID(*fw.ID)
		resourceGroup := iObj.ResourceGroup

		firewallID := fixFWRGChars(*fw.ID)
		tferName := terraformutils.TfSanitize(*fw.Name)

		// 1) 先把主资源 azurerm_firewall 放进列表
		resources = append(resources, terraformutils.NewSimpleResource(
			firewallID,
			resourceGroup+"_"+tferName,
			"azurerm_firewall",
			g.ProviderName,
			[]string{},
		))

		// firewallIterator 前进
		if err := firewallIterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return resources, err
		}
	}

	return resources, nil
}

// createFirewallPolicyResources 获取 Azure Firewall Policy 及其子资源 (Rule Collection Groups)
func (g *FirewallGenerator) createFirewallPolicyResources(ctx context.Context, rgName string) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource

	subscriptionID := g.Args["config"].(authentication.Config).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(authentication.Config).CustomResourceManagerEndpoint

	firewallPolicyClient := network.NewFirewallPoliciesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	firewallPolicyClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	policyIterator, err := firewallPolicyClient.ListComplete(ctx, rgName)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for policyIterator.NotDone() {
		policy := policyIterator.Value()
		if policy.ID == nil || policy.Name == nil {
			if err := policyIterator.NextWithContext(ctx); err != nil {
				log.Println(err)
				return resources, err
			}
			continue
		}

		iObj, _ := ParseAzureResourceID(*policy.ID)
		resourceGroup := iObj.ResourceGroup

		policyID := fixFWRGChars(*policy.ID)
		tferName := terraformutils.TfSanitize(*policy.Name)

		// 1) 主资源 azurerm_firewall_policy
		resources = append(resources, terraformutils.NewSimpleResource(
			policyID,
			resourceGroup+"_"+tferName,
			"azurerm_firewall_policy",
			g.ProviderName,
			[]string{},
		))

		// 2) 获取下属的 Rule Collection Groups
		rcgResources, err := g.createFirewallPolicyRCGResources(ctx, resourceGroup, *policy.Name)
		if err != nil {
			return resources, err
		}
		resources = append(resources, rcgResources...)

		if err := policyIterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return resources, err
		}
	}

	return resources, nil
}

// createFirewallPolicyRCGResources 封装获取 azurerm_firewall_policy_rule_collection_group 的逻辑
func (g *FirewallGenerator) createFirewallPolicyRCGResources(
	ctx context.Context,
	rgName, policyName string,
) ([]terraformutils.Resource, error) {

	var resources []terraformutils.Resource

	subscriptionID := g.Args["config"].(authentication.Config).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(authentication.Config).CustomResourceManagerEndpoint

	rcgClient := network.NewFirewallPolicyRuleCollectionGroupsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	rcgClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	// ListByFirewallPolicy: 根据 resourceGroupName + firewallPolicyName 列出所有 rule collection group
	rcgIterator, err := rcgClient.ListComplete(ctx, rgName, policyName)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for rcgIterator.NotDone() {
		rcg := rcgIterator.Value()
		if rcg.ID == nil || rcg.Name == nil {
			if err := rcgIterator.NextWithContext(ctx); err != nil {
				log.Println(err)
				return resources, err
			}
			continue
		}

		rcgID := fixFWRGChars(*rcg.ID)
		rcgName := terraformutils.TfSanitize(*rcg.Name)

		// log.Default().Println("rcgName: ", rcgName)
		// rcgJSON, err := json.MarshalIndent(rcg, "", "  ")
		// if err != nil {
		// 	fmt.Printf("Error marshaling RCG to JSON: %v\n", err)
		// } else {
		// 	fmt.Println(string(rcgJSON))
		// }
		// rcgProps := rcg.FirewallPolicyRuleCollectionGroupProperties
		// if rcgProps != nil && rcgProps.RuleCollections != nil {
		// 	for _, baseRC := range *rcgProps.RuleCollections {
		// 		// 1) 判断是否为 NAT Rule Collection
		// 		if natCol, ok := baseRC.AsFirewallPolicyNatRuleCollection(); ok {
		// 			// 这里就能拿到 natCol.Name / natCol.RuleCollectionType / natCol.Rules
		// 			if natCol.Name != nil {
		// 				fmt.Printf("  [NAT] RuleCollection Name=%s\n", *natCol.Name)
		// 			}

		// 			// 如果想记录 NAT Collection 到 TF 资源：自行调用 terraformutils.NewSimpleResource(...)

		// 			// 2) 判断是否为 Filter Rule Collection（包含 Network Rule / Application Rule）
		// 		} else if filterCol, ok := baseRC.AsFirewallPolicyFilterRuleCollection(); ok && filterCol != nil {
		// 			if filterCol.Name != nil {
		// 				fmt.Printf("  [Filter Rule Collection] Name=%s\n", *filterCol.Name)
		// 			}
		// 			if filterCol.Rules != nil {
		// 				// 遍历每条 Filter Rule
		// 				for _, r := range *filterCol.Rules {
		// 					// 再将“基础规则”转成 FirewallPolicyFilterRule
		// 					if fltRule, ok := r.AsRule(); ok && fltRule != nil {
		// 						fmt.Println("	[Rule] Name=", *fltRule.Name)
		// 					}
		// 				}
		// 			}
		// 		} else {
		// 			// 未匹配到 NAT / Filter，可能是其他类型（或 nil）
		// 			fmt.Printf("  [UNKNOWN] One rule collection is neither NAT nor Filter.\n")
		// 		}
		// 	}
		// }

		// 将 RCG 本身作为资源添加到 Terraformer 资源列表
		resources = append(resources, terraformutils.NewSimpleResource(
			rcgID,
			rgName+"_"+policyName+"_"+rcgName,
			"azurerm_firewall_policy_rule_collection_group",
			g.ProviderName,
			[]string{},
		))

		if err := rcgIterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

// InitResources 收集 Firewall & Firewall Policy 及其子资源
func (g *FirewallGenerator) InitResources() error {
	ctx := context.Background()

	var allResources []terraformutils.Resource

	// 支持多个逗号分隔的 RG
	if rg := g.Args["resource_group"].(string); rg != "" {
		resourceGroups := strings.Split(rg, ",")
		for _, rgName := range resourceGroups {
			rgName = strings.TrimSpace(rgName)

			firewallRes, err := g.createFirewallResources(ctx, rgName)
			if err != nil {
				return err
			}
			allResources = append(allResources, firewallRes...)

			policyRes, err := g.createFirewallPolicyResources(ctx, rgName)
			if err != nil {
				return err
			}
			allResources = append(allResources, policyRes...)
		}
	} else {
		// 未指定 resource_group 时，可一次性列出整个订阅下的所有资源
		firewallRes, err := g.createFirewallResourcesAll(ctx)
		if err != nil {
			return err
		}
		allResources = append(allResources, firewallRes...)

		policyRes, err := g.createFirewallPolicyResourcesAll(ctx)
		if err != nil {
			return err
		}
		allResources = append(allResources, policyRes...)
	}

	g.Resources = append(g.Resources, allResources...)
	return nil
}

// createFirewallResourcesAll 一次性列出订阅下所有 Firewall，并同时遍历其三种 Rule Collection
func (g *FirewallGenerator) createFirewallResourcesAll(ctx context.Context) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource

	subscriptionID := g.Args["config"].(authentication.Config).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(authentication.Config).CustomResourceManagerEndpoint
	firewallClient := network.NewAzureFirewallsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	firewallClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	firewallIterator, err := firewallClient.ListAllComplete(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for firewallIterator.NotDone() {
		fw := firewallIterator.Value()
		if fw.ID == nil || fw.Name == nil {
			if err := firewallIterator.NextWithContext(ctx); err != nil {
				log.Println(err)
				return resources, err
			}
			continue
		}

		iObj, _ := ParseAzureResourceID(*fw.ID)
		resourceGroup := iObj.ResourceGroup

		firewallID := fixFWRGChars(*fw.ID)
		tferName := terraformutils.TfSanitize(*fw.Name)

		// azurerm_firewall
		resources = append(resources, terraformutils.NewSimpleResource(
			firewallID,
			resourceGroup+"_"+tferName,
			"azurerm_firewall",
			g.ProviderName,
			[]string{},
		))

		// 拿出 sub-resources: Network / NAT / Application Rule Collections
		if fw.NetworkRuleCollections != nil {
			for _, netRuleCol := range *fw.NetworkRuleCollections {
				if netRuleCol.Name == nil {
					continue
				}
				netRuleID := fmt.Sprintf("%s|networkRuleCollections|%s", firewallID, *netRuleCol.Name)
				resources = append(resources, terraformutils.NewSimpleResource(
					netRuleID,
					resourceGroup+"_"+tferName+"_"+terraformutils.TfSanitize(*netRuleCol.Name),
					"azurerm_firewall_network_rule_collection",
					g.ProviderName,
					[]string{},
				))
			}
		}
		if fw.NatRuleCollections != nil {
			for _, natRuleCol := range *fw.NatRuleCollections {
				if natRuleCol.Name == nil {
					continue
				}
				natRuleID := fmt.Sprintf("%s|natRuleCollections|%s", firewallID, *natRuleCol.Name)
				resources = append(resources, terraformutils.NewSimpleResource(
					natRuleID,
					resourceGroup+"_"+tferName+"_"+terraformutils.TfSanitize(*natRuleCol.Name),
					"azurerm_firewall_nat_rule_collection",
					g.ProviderName,
					[]string{},
				))
			}
		}
		if fw.ApplicationRuleCollections != nil {
			for _, appRuleCol := range *fw.ApplicationRuleCollections {
				if appRuleCol.Name == nil {
					continue
				}
				appRuleID := fmt.Sprintf("%s|applicationRuleCollections|%s", firewallID, *appRuleCol.Name)
				resources = append(resources, terraformutils.NewSimpleResource(
					appRuleID,
					resourceGroup+"_"+tferName+"_"+terraformutils.TfSanitize(*appRuleCol.Name),
					"azurerm_firewall_application_rule_collection",
					g.ProviderName,
					[]string{},
				))
			}
		}

		if err := firewallIterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

// createFirewallPolicyResourcesAll 一次性列出订阅下所有 Firewall Policy，并同时列出其 Rule Collection Groups
func (g *FirewallGenerator) createFirewallPolicyResourcesAll(ctx context.Context) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource

	subscriptionID := g.Args["config"].(authentication.Config).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(authentication.Config).CustomResourceManagerEndpoint
	firewallPolicyClient := network.NewFirewallPoliciesClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	firewallPolicyClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	policyIterator, err := firewallPolicyClient.ListAllComplete(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	for policyIterator.NotDone() {
		policy := policyIterator.Value()
		if policy.ID == nil || policy.Name == nil {
			if err := policyIterator.NextWithContext(ctx); err != nil {
				log.Println(err)
				return resources, err
			}
			continue
		}

		iObj, _ := ParseAzureResourceID(*policy.ID)
		resourceGroup := iObj.ResourceGroup

		policyID := fixFWRGChars(*policy.ID)
		tferName := terraformutils.TfSanitize(*policy.Name)

		// azurerm_firewall_policy
		resources = append(resources, terraformutils.NewSimpleResource(
			policyID,
			resourceGroup+"_"+tferName,
			"azurerm_firewall_policy",
			g.ProviderName,
			[]string{},
		))

		// 获取 rule collection group
		rcgClient := network.NewFirewallPolicyRuleCollectionGroupsClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
		rcgClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

		rcgIterator, err := rcgClient.ListComplete(ctx, resourceGroup, *policy.Name)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		for rcgIterator.NotDone() {
			rcg := rcgIterator.Value()
			if rcg.ID == nil || rcg.Name == nil {
				if err := rcgIterator.NextWithContext(ctx); err != nil {
					log.Println(err)
					return resources, err
				}
				continue
			}

			rcgID := fixFWRGChars(*rcg.ID)
			rcgName := terraformutils.TfSanitize(*rcg.Name)

			resources = append(resources, terraformutils.NewSimpleResource(
				rcgID,
				resourceGroup+"_"+*policy.Name+"_"+rcgName,
				"azurerm_firewall_policy_rule_collection_group",
				g.ProviderName,
				[]string{},
			))

			if err := rcgIterator.NextWithContext(ctx); err != nil {
				log.Println(err)
				return resources, err
			}
		}

		if err := policyIterator.NextWithContext(ctx); err != nil {
			log.Println(err)
			return resources, err
		}
	}
	return resources, nil
}

// fixFWRGChars 修正资源 ID 中大小写或其他字符串问题，避免后续处理时不一致
func fixFWRGChars(id string) string {
	// 将 /resourcegroups/ 替换为 /resourceGroups/
	id = strings.Replace(id, "/resourcegroups/", "/resourceGroups/", -1)
	// 如果有其他替换需求可以继续补充
	return id
}
