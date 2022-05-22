package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tfzk/terraform-provider-zookeeper/internal/client"
)

func resourceSeqZNode() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSeqZNodeCreate,
		ReadContext:   resourceSeqZNodeRead,
		UpdateContext: resourceSeqZNodeUpdate,
		DeleteContext: resourceSeqZNodeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceSeqZNodeImport,
		},
		Schema: map[string]*schema.Schema{
			"path_prefix": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "Absolute path to the Sequential ZNode to create. " +
					"ZooKeeper will append a monotonically increasing counter to the end of path. " +
					"This counter is unique to the parent znode. " +
					"The counter has a format of `%010d` (10 digits with `0` padding)." +
					"Example: `<path-prefix>0000000001`.",
			},
			"data": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"data_base64"},
				Description: "Content to store in the ZNode, as a UTF-8 string. " +
					"Mutually exclusive with `data_base64`.",
			},
			"data_base64": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"data"},
				Description: "Content to store in the ZNode, as Base64 encoded bytes. " +
					"Mutually exclusive with `data`.",
			},
			"path": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"stat": statSchema(),
		},
		Description: "Manages the lifecycle of a " +
			"[ZooKeeper ZNode](https://zookeeper.apache.org/doc/current/zookeeperProgrammers.html#sc_zkDataModel_znodes). " +
			"This resource manages **Persistent Sequential ZNodes**. " +
			"The data can be provided either as UTF-8 string, or as Base64 encoded bytes. " +
			"The ability to create ZNodes is determined by ZooKeeper ACL.",
	}
}

func resourceSeqZNodeCreate(_ context.Context, rscData *schema.ResourceData, prvClient interface{}) diag.Diagnostics {
	zkClient := prvClient.(*client.Client)

	znodePathPrefix := rscData.Get("path_prefix").(string)

	dataBytes, err := getDataBytesFromResourceData(rscData)
	if err != nil {
		return diag.FromErr(err)
	}

	znode, err := zkClient.CreateSequential(znodePathPrefix, dataBytes)
	if err != nil {
		return diag.Errorf("Failed to create Sequential ZNode '%s': %v", znodePathPrefix, err)
	}

	// Terraform will use the ZNode.Path as unique identifier for this Resource
	rscData.SetId(znode.Path)
	rscData.MarkNewResource()

	return setAttributesFromZNode(rscData, znode, diag.Diagnostics{})
}

func resourceSeqZNodeRead(ctx context.Context, rscData *schema.ResourceData, prvClient interface{}) diag.Diagnostics {
	return resourceZNodeRead(ctx, rscData, prvClient)
}

func resourceSeqZNodeUpdate(ctx context.Context, rscData *schema.ResourceData, prvClient interface{}) diag.Diagnostics {
	return resourceZNodeUpdate(ctx, rscData, prvClient)
}

func resourceSeqZNodeDelete(ctx context.Context, rscData *schema.ResourceData, prvClient interface{}) diag.Diagnostics {
	return resourceZNodeDelete(ctx, rscData, prvClient)
}

func resourceSeqZNodeImport(_ context.Context, rscData *schema.ResourceData, prvClient interface{}) ([]*schema.ResourceData, error) {
	// Re-create the original `path_prefix` for the imported `sequential_znode`,
	// by removing the sequential suffix from the `id` (i.e. `path`)
	if err := rscData.Set("path_prefix", client.RemoveSequentialSuffix(rscData.Id())); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{rscData}, nil
}
