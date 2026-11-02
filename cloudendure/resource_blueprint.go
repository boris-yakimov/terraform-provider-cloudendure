package cloudendure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"terraform-provider-cloudendure/models"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceBlueprint() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBlueprintCreate,
		ReadContext:   resourceBlueprintRead,
		//UpdateContext: resourceBlueprintUpdate,
		DeleteContext: resourceBlueprintDelete,
		//Importer: &schema.ResourceImporter{
		//State: schema.ImportStatePassthrough,
		//},
		Schema: map[string]*schema.Schema{
			"byol_on_dedicated_instance": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "boolean - specifies whether to use byol windows license if dedicated instance tenancy is selected.",
			},
			"dedicated_host_identifier": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - Host identifier",
			},
			"disks": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"throughput": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
					Description: "array of maps - AWS only. Target machine disk properties",
				},
			},
			"force_uefi": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "boolean - weather to force UEFI",
			},
			"iam_role": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - AWS only. Possible values can be fetched from the Region object.",
			},
			"id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "string - Part of HTTP response - id of created resource",
			},
			"instance_type": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - Possible values can be fetched from the Region object, plus special values COPY_ORIGIN or CUSTOM",
			},
			"machine_id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "string - Part of HTTP response - if of created machine object",
			},
			"network_interface": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - network interface to be used",
			},
			"placement_group": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - AWS only. Possible values can be fetched from the Region object.",
			},
			"private_ip_action": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - Valid values: CREATE_NEW; COPY_ORIGIN; CUSTOM_IP; USE_NETWORK_INTERFACE",
			},
			"private_ips": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "array of strings <IP> - List of IP addresses",
			},
			"public_ip_action": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - Valid values: ALLOCATE; DONT_ALLOCATE; AS_SUBNET - Whether to allocate an ephemeral public IP, or not. AS_SUBNET causes CloudEndure to copy this property from the source machine.",
			},
			"recommended_private_ip": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "string - Part of HTTP response - Recommended Priate IP Address",
			},
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "string - Part of HTTP response - Region Object ID",
			},
			"run_after_launch": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "boolean - AWS only. Whether to power on the launched target machine after launch. True by default.",
			},
			"security_group_ids": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "array of strings - AWS only. The security groups that will be applied to the target machine. Possible values can be fetched from the Region object.",
			},
			"static_ip": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string <IP> - Possible values can be fetched from the Region object.",
			},
			"static_ip_action": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - Valid values: EXISTING; DONT_CREATE; CREATE_NEW; IF_IN_ORIGIN",
			},
			"subnet_ids": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "array of strings - AWS only. Configures a subnets in which the instance network interface will take part. Possible values can be fetched from the Region object.",
			},
			"subnets_host_project": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - GCP only. Host project for corss project network subnet.",
			},
			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"value": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Description: "array of maps - AWS only. Key/Value pair tags that will be applied to the target machine.",
			},
			"tenancy": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "SHARED",
				Description: "string - Valid values: SHARED; DEDICATED; HOST",
			},
			"scsi_adapter_type": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - Currently relevant for vCenter cloud only. Possible values can be fetched from the Region object.",
			},
			"machine_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "string - Cloudendure machine name",
			},
			"cpus": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "integer - Number of CPUs per per Target machine; Currently relevant for vCenter cloud only; Max value can be fetched from the maxCpusPerMachine property of the Region object.",
			},
			"mb_ram": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "integer - MB RAM per Target machine; Currently relevant for vCenter cloud only; Max value can be fetched from the maxMbRamPerMachine property of the Region object.",
			},
			"cores_per_cpu": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "integer - Number of CPU cores per CPU in Target machine; Currently relevant for vCenter cloud only.",
			},
			"recommended_instance_type": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				ForceNew:    true,
				Description: "string - Part of HTTP response - Recommended Instance Type",
			},
			"launch_on_instance_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - Instance id for target machine managed by AMS.",
			},
			"security_group_action": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "FROM_POLICY",
				Description: "string - How to assign a security group to the target machine.",
			},
			"compute_location_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - TODO",
			},
			"logical_location_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - vcenter = vmFolder; relates to $ref LogicalLocation",
			},
			"network_adapter_type": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "string - Currently relevant for vCenter cloud only. Possible values can be fetched from the Region object.",
			},
			"use_shared_ram": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "boolean - weather to use shared RAM",
			},
		},
	}
}

func marshalMachineInput(d *schema.ResourceData) *models.MachineInput {
	var subnetIDs []string
	for _, subnetValue := range d.Get("subnet_ids").([]interface{}) {
		subnetIDs = append(subnetIDs, subnetValue.(string))
	}

	var securityGroupIDs []string
	for _, securityGroupValue := range d.Get("security_group_ids").([]interface{}) {
		securityGroupIDs = append(securityGroupIDs, securityGroupValue.(string))
	}

	return &models.MachineInput{
		// requried
		MachineName:      d.Get("machine_name").(string),
		SubnetIDs:        subnetIDs,
		SecurityGroupIDs: securityGroupIDs,

		// optional
		InstanceType:            d.Get("instance_type").(string),
		ByolOnDedicatedInstance: d.Get("byol_on_dedicated_instance").(bool),
		RunAfterLaunch:          d.Get("run_after_launch").(bool),
		ForceUEFI:               d.Get("force_uefi").(bool),
		UseSharedRam:            d.Get("use_shared_ram").(bool),
		StaticIPAction:          d.Get("static_ip_action").(string),
		PublicIPAction:          d.Get("public_ip_action").(string),
		Cpus:                    d.Get("cpus").(int),
		CoresPerCpu:             d.Get("cores_per_cpu").(int),
		MbRam:                   d.Get("mb_ram").(int),
		SecurityGroupAction:     d.Get("security_group_action").(string),
		Tenancy:                 d.Get("tenancy").(string),
		PrivateIPAction:         d.Get("private_ip_action").(string),
		//Disks:                   d.Get("disks").([]string),
	}
}

func resourceBlueprintCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*http.Client)

	cloudEndureApiURL := "https://console.cloudendure.com/api/latest"
	cloudEndureProjectId := "projects/REPLACE_WITH_PROJECT_ID"

	requestURL := fmt.Sprintf("%s/%s/blueprints", cloudEndureApiURL, cloudEndureProjectId)

	requestBody, jsonMarshalErr := json.Marshal(marshalMachineInput(d))
	if jsonMarshalErr != nil {
		return diag.FromErr(jsonMarshalErr)
	}

	diags = append(diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  fmt.Sprintf("Debug MSG"),
		Detail:   fmt.Sprintf("Request Body: %v", string(requestBody)),
	})

	req, defineRequestErr := http.NewRequest("POST", requestURL, bytes.NewBuffer(requestBody))
	if defineRequestErr != nil {
		return diag.FromErr(defineRequestErr)
	}


	response, sendRequestErr := client.Do(req)
	if sendRequestErr != nil {
		return diag.FromErr(sendRequestErr)
	}
	defer response.Body.Close()

	responseBody, readResponseBodyErr := ioutil.ReadAll(response.Body)
	if readResponseBodyErr != nil {
		fmt.Printf("Unable to read HTTP response body: %s", readResponseBodyErr)
		return diag.FromErr(readResponseBodyErr)
	}

	if response.StatusCode != 200 && response.StatusCode != 201 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("Cloudendure HTTP Response: %v", response.Status),
			Detail:   fmt.Sprintf("Error in create blueprint API request: %v", string(responseBody)),
		})
	}

	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))
	return diags
}

func resourceBlueprintRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	client := meta.(*http.Client)

	cloudEndureApiURL := "https://console.cloudendure.com/api/latest"
	cloudEndureProjectId := "projects/REPLACE_WITH_PROJECT_ID"

	requestURL := fmt.Sprintf("%s/%s/blueprints", cloudEndureApiURL, cloudEndureProjectId)

	req, defineRequestErr := http.NewRequest("GET", requestURL, nil)
	if defineRequestErr != nil {
		return diag.FromErr(defineRequestErr)
	}


	response, sendRequestErr := client.Do(req)
	if sendRequestErr != nil {
		return diag.FromErr(sendRequestErr)
	}
	defer response.Body.Close()

	// Parse JSON into map
	//bluePrintConfigs := make(map[string]interface{}, 0)
	//jsonDecodeErr := json.NewDecoder(response.Body).Decode(&bluePrintConfigs)
	//if jsonDecodeErr != nil {
	//return diag.FromErr(jsonDecodeErr)
	//}

	//// Convert JSON to follow Terraform's expected structure (put JSON map inside an array)
	//var updatedItems []interface{}
	//updatedItems = append(updatedItems, bluePrintConfigs)

	//// Set data source schema values
	//d.Set("machine_id", bluePrintConfigs["machineId"])
	//d.Set("instance_type", bluePrintConfigs["instanceType"])
	//d.Set("security_group_ids", bluePrintConfigs["securityGroupIDs"])
	//d.Set("subnet_ids", bluePrintConfigs["subnetIDs"])

	// SetId sets the ID of the resource. If the value is blank, then the resource is destroyed - always run
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}

// TODO:
func resourceBlueprintUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	return diags
}

// TODO:
func resourceBlueprintDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	return diags
}
