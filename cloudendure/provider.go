package cloudendure

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// summary:
// tested cleanup of schema, seems to work only with set fields
// tested cookie authentication - still lots of issues (cookiejar problems, etc)
// figured out how to show custom debug outputs and errors in terraform
// merge datasource blueprint read stuff into one function
// move api_key on provider level

// TODO:
// Test machine & blueprint creation with postman
// Check this error -  No Update defined, must set ForceNew on:
// Move URLs on provider level so they dont duplicate in resources and datasources
// implement a resource for changing blueprint
// move hardcoded tokens on provider level to do them only ones while debugging
// debug authentication cookie problems - test now that http client is actually defined only in the provider and than reused
// check response flatten approaches - https://learn.hashicorp.com/tutorials/terraform/provider-complex-read?in=terraform/providers
// check if we need to use both Optional and Computed flags in some places
// reuse common attribute schemas - https://www.youtube.com/watch?v=XlxkqXQCZ4Y
// create API client library for cloudendure - can copy hashicups Client and try to adapt it; separate library that is just imported in the provider
// unit testing

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDENDURE_API_KEY", nil),
				Description: "API Key used for authenticating to Cloudendure",
			},
			"cloudendure_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CLOUDENDURE_URL", nil),
				Description: "Cloudendure API URL",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"cloudendure_blueprint":     resourceBlueprint(),
			"cloudendure_recovery_plan": resourceRecoveryPlan(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"cloudendure_blueprint": dataSourceBlueprint(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

// Authenticate in Cloudendure - assign cookie and xsrf token to HTTP client
// func authCloudEndure(httpClient *http.Client, apiKey string, cloudEndureApiURL string) (cookieList []*http.Cookie, authErr error) {
func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	cloudEndureApiURL := d.Get("cloudendure_url").(string)
	apiKey := d.Get("api_key").(string)

	// TODO: Taking cookie from authenticated session doesnt work for some reason
	//var cookieList []*http.Cookie
	//for i := range cookieList {
	//request.AddCookie(cookieList[i])
	//}

	// TODO: check if the same session is being kept in each request; different sessions may not see the same cookie
	// TODO: figure out how to handle passing of cookie
	//cookieJar, initCookieJarErr := cookiejar.New(nil)
	//if initCookieJarErr != nil {
	//return nil, diag.FromErr(initCookieJarErr)
	//}

	// Init HTTP client
	client := &http.Client{
		Timeout: 20 * time.Second,
		//Jar:     cookieJar,
	}

	// TODO: check why api_key appears empty here
	// Check if CloudEndure API Key provided
	//apiKey, ok := d.Get("api_key").(string)
	//if apiKey == "" || ok == false {
	//diags = append(diags, diag.Diagnostic{
	//Severity: diag.Warning,
	//Summary:  fmt.Sprintf("Error retrieving API Key: %v", apiKey),
	//Detail:   "Unable to authenticate to Cloudendure API",
	//})
	//}

	userApiToken := fmt.Sprintf(`{"userApiToken":"%v"}`, apiKey)

	var requestBody = []byte(userApiToken)

	// TODO: figure out how to print those as INFO message in TF
	fmt.Println("Authenticate into CloudEndure using API Key")

	loginURL := fmt.Sprintf("%s/login", cloudEndureApiURL)
	request, defineRequestErr := http.NewRequest("POST", loginURL, bytes.NewBuffer(requestBody))
	if defineRequestErr != nil {
		return nil, diag.FromErr(defineRequestErr)
	}

	request.Header.Set("Content-Type", "application/json")

	fmt.Println("request URL:", request.URL)
	fmt.Println("request Method:", request.Method)
	fmt.Println("request Headers:", request.Header)
	fmt.Println("")

	response, sendRequestErr := client.Do(request)
	if sendRequestErr != nil {
		return nil, diag.FromErr(sendRequestErr)
	}
	defer response.Body.Close()

	fmt.Println("response Status:", response.Status)
	fmt.Println("response Headers:", response.Header)

	// TODO: cookie problems
	// https://stackoverflow.com/questions/50483866/get-cookie-from-golang-post
	// var cookie []*http.Cookie
	// Get cookies from response :
	//req, _ := http.NewRequest("GET", url, nil)
	//resp, err := client.Do(req) //send request
	//if err != nil {
	//return
	//}
	//cookie = resp.Cookies() //save cookies
	// Create new request using cookies from response

	//req, _ := http.NewRequest("POST", url, nil)
	//for i := range cookie {
	//req.AddCookie(cookie[i])
	//}
	//resp, err := client.Do(req) //send request
	//if err != nil {
	//return
	//}

	responseBody, readResponseBodyErr := ioutil.ReadAll(response.Body)
	if readResponseBodyErr != nil {
		return nil, diag.FromErr(readResponseBodyErr)
	}
	fmt.Println("response Body:", string(responseBody))
	fmt.Println("")

	return client, diags
}
