package cloudendure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBlueprint() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceBlueprintRead,
		Schema: map[string]*schema.Schema{
			"blueprint_id": &schema.Schema{
				Type:        schema.TypeString,
				Description: "string - Cloudendure Blueprint ID to search by",
				Required:    true,
			},
			"machine_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "string - Part of HTTP response - if of created machine object",
			},
			"instance_type": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "string - Possible values can be fetched from the Region object, plus special values COPY_ORIGIN or CUSTOM",
			},
			"security_group_ids": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "array of strings - AWS only. The security groups that will be applied to the target machine. Possible values can be fetched from the Region object.",
			},
			"subnet_ids": &schema.Schema{
				Type:        schema.TypeList,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "array of strings - AWS only. Configures a subnets in which the instance network interface will take part. Possible values can be fetched from the Region object.",
			},
		},
	}
}

// Cloudendure Blueprint - datasource
func dataSourceBlueprintRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	//TODO: Remove this when refactored
	// Define Cloudendure URLs

	//cloudEndureApiURL, cloudEndureUrlOk := d.Get("cloudendure_url").(string)
	//if cloudEndureApiURL == "" || cloudEndureUrlOk == false {
	//diags = append(diags, diag.Diagnostic{
	//Severity: diag.Warning,
	//Summary:  fmt.Sprintf("Error retrieving CloudEndure URL from Terraform, current value set to: [%v]", cloudEndureApiURL),
	//Detail:   "Cloudendure URL not provided",
	//})
	//}

	//cloudEndureApiURL := d.Get("cloudendure_url").(string)
	//if cloudEndureApiURL == "" {
	//diags = append(diags, diag.Diagnostic{
	//Severity: diag.Warning,
	//Summary:  fmt.Sprintf("Error retrieving CloudEndure URL from Terraform, current value set to: [%v]", cloudEndureApiURL),
	//Detail:   "Cloudendure URL not provided",
	//})
	//}

	cloudEndureApiURL := "https://console.cloudendure.com/api/latest"
	cloudEndureProjectId := "projects/REPLACE_WITH_PROJECT_ID"
	//cloudEndureBluePrintId := "f320947e-1555-4cee-9128-58a6cc4dd99c"
	cloudEndureBluePrintId := d.Get("blueprint_id").(string)

	requestURL := fmt.Sprintf("%s/%s/blueprints/%s", cloudEndureApiURL, cloudEndureProjectId, cloudEndureBluePrintId)

	fmt.Printf("Get Cloudendure Blueprint with ID: [%s]\n", cloudEndureBluePrintId)

	req, defineRequestErr := http.NewRequest("GET", requestURL, nil)
	if defineRequestErr != nil {
		return diag.FromErr(defineRequestErr)
	}

	// TODO: http client was inited in provider - verify that htis actually works
	//client := &http.Client{}
	client := meta.(*http.Client)

	fmt.Println("request URL:", req.URL)
	fmt.Println("request Method:", req.Method)
	fmt.Println("request Headers:", req.Header)
	fmt.Println("")

	response, sendRequestErr := client.Do(req)
	if sendRequestErr != nil {
		return diag.FromErr(sendRequestErr)
	}
	defer response.Body.Close()

	fmt.Println("response Status:", response.Status)
	fmt.Println("response Headers:", response.Header)

	// Used for Debugging HTTP body contents
	//responseBody, readResponseBodyErr := ioutil.ReadAll(response.Body)
	//if readResponseBodyErr != nil {
	//fmt.Printf("Unable to read HTTP response body: %s", readResponseBodyErr)
	//return readResponseBodyErr
	//}
	//fmt.Println("response Body:", string(responseBody))

	// Parse JSON into map
	bluePrintConfigs := make(map[string]interface{}, 0)
	jsonDecodeErr := json.NewDecoder(response.Body).Decode(&bluePrintConfigs)
	if jsonDecodeErr != nil {
		return diag.FromErr(jsonDecodeErr)
	}

	// Convert JSON to follow Terraform's expected structure (put JSON map inside an array)
	var updatedItems []interface{}
	updatedItems = append(updatedItems, bluePrintConfigs)

	// Set data source schema values
	d.Set("machine_id", bluePrintConfigs["machineId"])
	d.Set("instance_type", bluePrintConfigs["instanceType"])
	d.Set("security_group_ids", bluePrintConfigs["securityGroupIDs"])
	d.Set("subnet_ids", bluePrintConfigs["subnetIDs"])

	// SetId sets the ID of the resource. If the value is blank, then the resource is destroyed - always run
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	// Add check if resource doesn't exist to set ID to blank
	//if resourceDoesntExist {
	//d.SetId("")
	//return
	//}

	return diags
}
