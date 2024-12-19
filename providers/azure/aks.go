package azure

import (
	"context"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-03-01/containerservice"
	"github.com/Azure/go-autorest/autorest"
	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
	"github.com/hashicorp/go-azure-helpers/authentication"
)

type AKSGenerator struct {
	AzureService
}

func (g AKSGenerator) createResources(ctx context.Context, iterator containerservice.ManagedClusterListResultIterator) ([]terraformutils.Resource, error) {
	var resources []terraformutils.Resource
	for iterator.NotDone() {
		cluster := iterator.Value()
		tferName := terraformutils.TfSanitize(*cluster.Name)

		iObj, _ := ParseAzureResourceID(*cluster.ID)
		resourceGroup := iObj.ResourceGroup
		aksID := fixRGChars(*cluster.ID)
		// 添加 AKS 集群资源
		resources = append(resources, terraformutils.NewSimpleResource(
			aksID,
			resourceGroup+"_"+tferName,
			"azurerm_kubernetes_cluster",
			g.ProviderName,
			[]string{}))

		// 创建节点池客户端
		agentPoolClient := containerservice.NewAgentPoolsClientWithBaseURI(g.Args["config"].(authentication.Config).CustomResourceManagerEndpoint, g.Args["config"].(authentication.Config).SubscriptionID)
		agentPoolClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

		// 列出 AKS 集群的节点池
		agentPoolIterator, err := agentPoolClient.ListComplete(ctx, resourceGroup, *cluster.Name)
		if err != nil {
			log.Println(err)
			return resources, err
		}
		for agentPoolIterator.NotDone() {
			agentPool := agentPoolIterator.Value()

			// 使用正确的资源 ID 格式
			agentPoolID := fixRGChars(*agentPool.ID)

			// 检查并修正资源 ID 中的大小写
			// agentPoolID = fixRGChars(agentPoolID)
			// log.Println("agentPoolID: ", agentPoolID)

			resources = append(resources, terraformutils.NewSimpleResource(
				agentPoolID,
				resourceGroup+"_"+tferName+"_"+*agentPool.Name,
				"azurerm_kubernetes_cluster_node_pool",
				g.ProviderName,
				[]string{}))
			if err := agentPoolIterator.NextWithContext(ctx); err != nil {
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

// 新增一个函数来修正资源 ID 的大小写
func fixRGChars(id string) string {
	// 将 resourcegroups 替换为 resourceGroups
	id = strings.Replace(id, "/resourcegroups/", "/resourceGroups/", -1)
	// 其他必要的替换
	return id
}

func (g *AKSGenerator) InitResources() error {
	ctx := context.Background()
	subscriptionID := g.Args["config"].(authentication.Config).SubscriptionID
	resourceManagerEndpoint := g.Args["config"].(authentication.Config).CustomResourceManagerEndpoint
	clusterClient := containerservice.NewManagedClustersClientWithBaseURI(resourceManagerEndpoint, subscriptionID)
	clusterClient.Authorizer = g.Args["authorizer"].(autorest.Authorizer)

	var (
		iterator containerservice.ManagedClusterListResultIterator
		err      error
	)

	if rg := g.Args["resource_group"].(string); rg != "" {
		resourceGroups := strings.Split(rg, ",")
		for _, rgName := range resourceGroups {
			rgName = strings.TrimSpace(rgName)
			iterator, err = clusterClient.ListByResourceGroupComplete(ctx, rgName)
			if err != nil {
				return err
			}
			resources, err := g.createResources(ctx, iterator)
			if err != nil {
				return err
			}
			g.Resources = append(g.Resources, resources...)
		}
		// iterator, err = clusterClient.ListByResourceGroupComplete(ctx, rg)
	} else {
		iterator, err = clusterClient.ListComplete(ctx)
		if err != nil {
			return err
		}
		g.Resources, err = g.createResources(ctx, iterator)
		return err
	}
	return nil

}
