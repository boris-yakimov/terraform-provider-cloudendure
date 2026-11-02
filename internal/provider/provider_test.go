package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/boris-yakimov/terraform-provider-cloudendure/internal/client"
	"github.com/boris-yakimov/terraform-provider-cloudendure/internal/fakeapi"
)

const (
	testAPIKey    = "TEST-API-KEY"
	testProjectID = "test-project"
)

var testProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"cloudendure": providerserver.NewProtocol6WithError(New("test")()),
}

// newFakeAPI starts a fake CloudEndure API and returns it together with a
// provider block pointing at it.
func newFakeAPI(t *testing.T) (*fakeapi.Server, string) {
	t.Helper()
	srv := fakeapi.New(testAPIKey, testProjectID)
	t.Cleanup(srv.Close)
	cfg := fmt.Sprintf(`
provider "cloudendure" {
  api_key    = %q
  api_url    = %q
  project_id = %q
}
`, testAPIKey, srv.BaseURL(), testProjectID)
	return srv, cfg
}

func TestProviderMissingConfig(t *testing.T) {
	t.Setenv(envAPIKey, "")
	t.Setenv(envProjectID, "")
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{{
			Config:      `data "cloudendure_machine" "m" { name = "x" }`,
			ExpectError: regexp.MustCompile(`Missing CloudEndure API key`),
		}},
	})
}

func TestProviderConfigFromEnv(t *testing.T) {
	srv := fakeapi.New(testAPIKey, testProjectID)
	defer srv.Close()
	srv.AddMachine("env-host")

	t.Setenv(envAPIKey, testAPIKey)
	t.Setenv(envAPIURL, srv.BaseURL())
	t.Setenv(envProjectID, testProjectID)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{{
			Config: `data "cloudendure_machine" "m" { name = "env-host" }`,
			Check:  resource.TestCheckResourceAttr("data.cloudendure_machine.m", "name", "env-host"),
		}},
	})
}

func TestMachineDataSource(t *testing.T) {
	srv, provider := newFakeAPI(t)
	id := srv.AddMachine("web-1")
	srv.AddMachine("dup")
	srv.AddMachine("dup")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      provider + `data "cloudendure_machine" "m" {}`,
				ExpectError: regexp.MustCompile(`Exactly one of these attributes must be configured`),
			},
			{
				Config: provider + `
data "cloudendure_machine" "by_name" { name = "web-1" }
data "cloudendure_machine" "by_id"   { id = data.cloudendure_machine.by_name.id }
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudendure_machine.by_name", "id", id),
					resource.TestCheckResourceAttr("data.cloudendure_machine.by_name", "os", "linux"),
					resource.TestCheckResourceAttr("data.cloudendure_machine.by_id", "name", "web-1"),
					resource.TestCheckResourceAttr("data.cloudendure_machine.by_id", "replication_status", "STARTED"),
				),
			},
			{
				Config:      provider + `data "cloudendure_machine" "m" { name = "dup" }`,
				ExpectError: regexp.MustCompile(`multiple machines named "dup"`),
			},
			{
				Config:      provider + `data "cloudendure_machine" "m" { name = "missing" }`,
				ExpectError: regexp.MustCompile(`no machine named "missing"`),
			},
		},
	})
}

func TestBlueprintResource(t *testing.T) {
	srv, provider := newFakeAPI(t)
	machineID := srv.AddMachine("app-1")
	const addr = "cloudendure_blueprint.bp"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: provider + fmt.Sprintf(`
resource "cloudendure_blueprint" "bp" {
  machine_id         = %q
  instance_type      = "COPY_ORIGIN"
  subnet_ids         = ["subnet-1"]
  security_group_ids = ["sg-1", "sg-2"]
  public_ip_action   = "DONT_ALLOCATE"
  run_after_launch   = false
  tags = {
    Environment = "test"
  }
  disks = [{ name = "c:0", type = "SSD" }]
}
`, machineID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(addr, "id"),
					resource.TestCheckResourceAttr(addr, "machine_name", "app-1"),
					resource.TestCheckResourceAttr(addr, "region", "region-0001"),
					resource.TestCheckResourceAttr(addr, "security_group_ids.#", "2"),
					resource.TestCheckResourceAttr(addr, "run_after_launch", "false"),
					resource.TestCheckResourceAttr(addr, "tags.Environment", "test"),
					resource.TestCheckResourceAttr(addr, "disks.0.type", "SSD"),
				),
			},
			{
				Config: provider + fmt.Sprintf(`
resource "cloudendure_blueprint" "bp" {
  machine_id         = %q
  instance_type      = "m5.large"
  subnet_ids         = ["subnet-2"]
  security_group_ids = ["sg-1"]
  public_ip_action   = "DONT_ALLOCATE"
  run_after_launch   = true
  tags = {
    Environment = "prod"
  }
  disks = [{ name = "c:0", type = "SSD" }]
}
`, machineID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "instance_type", "m5.large"),
					resource.TestCheckResourceAttr(addr, "subnet_ids.0", "subnet-2"),
					resource.TestCheckResourceAttr(addr, "security_group_ids.#", "1"),
					resource.TestCheckResourceAttr(addr, "run_after_launch", "true"),
					resource.TestCheckResourceAttr(addr, "tags.Environment", "prod"),
				),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: func(*terraform.State) error {
			// Blueprints cannot be deleted; it must still exist in the API.
			if srv.BlueprintCount() != 1 {
				return fmt.Errorf("expected blueprint to remain in CloudEndure, found %d", srv.BlueprintCount())
			}
			return nil
		},
	})
}

func TestBlueprintResourceAdoptsExisting(t *testing.T) {
	srv, provider := newFakeAPI(t)
	machineID := srv.AddMachine("legacy-1")
	existingID := srv.AddBlueprint(client.Blueprint{MachineID: machineID, InstanceType: "t3.small", Region: "r1"})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{{
			Config: provider + fmt.Sprintf(`
resource "cloudendure_blueprint" "bp" {
  machine_id    = %q
  instance_type = "t3.large"
}
`, machineID),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("cloudendure_blueprint.bp", "id", existingID),
				resource.TestCheckResourceAttr("cloudendure_blueprint.bp", "instance_type", "t3.large"),
				resource.TestCheckResourceAttr("cloudendure_blueprint.bp", "region", "r1"),
			),
		}},
	})
}

func TestBlueprintResourceValidation(t *testing.T) {
	_, provider := newFakeAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{{
			Config: provider + `
resource "cloudendure_blueprint" "bp" {
  machine_id = "m"
  tenancy    = "SOMETIMES"
}
`,
			ExpectError: regexp.MustCompile(`Invalid Attribute Value Match`),
		}},
	})
}

func TestBlueprintDataSource(t *testing.T) {
	srv, provider := newFakeAPI(t)
	machineID := srv.AddMachine("db-1")
	subnets := []string{"subnet-a", "subnet-b"}
	bpID := srv.AddBlueprint(client.Blueprint{MachineID: machineID, InstanceType: "r5.xlarge", SubnetIDs: &subnets})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{{
			Config: provider + fmt.Sprintf(`
data "cloudendure_blueprint" "by_machine" { machine_id = %q }
data "cloudendure_blueprint" "by_id"      { id = %q }
`, machineID, bpID),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.cloudendure_blueprint.by_machine", "id", bpID),
				resource.TestCheckResourceAttr("data.cloudendure_blueprint.by_machine", "instance_type", "r5.xlarge"),
				resource.TestCheckResourceAttr("data.cloudendure_blueprint.by_id", "machine_id", machineID),
				resource.TestCheckResourceAttr("data.cloudendure_blueprint.by_id", "subnet_ids.#", "2"),
			),
		}},
	})
}

func TestRecoveryPlanResource(t *testing.T) {
	srv, provider := newFakeAPI(t)
	db := srv.AddMachine("db")
	app := srv.AddMachine("app")
	const addr = "cloudendure_recovery_plan.rp"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: provider + fmt.Sprintf(`
resource "cloudendure_recovery_plan" "rp" {
  name = "plan-a"
  steps = [
    { name = "databases", machine_ids = [%q] },
    { name = "apps", machine_ids = [%q], wait_before = "300" },
  ]
}
`, db, app),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(addr, "id"),
					resource.TestCheckResourceAttr(addr, "steps.#", "2"),
					resource.TestCheckResourceAttr(addr, "steps.1.wait_before", "300"),
				),
			},
			{
				Config: provider + fmt.Sprintf(`
resource "cloudendure_recovery_plan" "rp" {
  name  = "plan-b"
  steps = [{ machine_ids = [%q, %q] }]
}
`, db, app),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "name", "plan-b"),
					resource.TestCheckResourceAttr(addr, "steps.#", "1"),
					resource.TestCheckResourceAttr(addr, "steps.0.machine_ids.#", "2"),
					resource.TestCheckNoResourceAttr(addr, "steps.0.name"),
				),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: func(*terraform.State) error {
			if srv.RecoveryPlanCount() != 0 {
				return fmt.Errorf("expected recovery plans to be deleted, found %d", srv.RecoveryPlanCount())
			}
			return nil
		},
	})
}

func TestRecoveryPlanRemovedOutsideTerraform(t *testing.T) {
	srv, provider := newFakeAPI(t)
	m := srv.AddMachine("m")
	cfg := provider + fmt.Sprintf(`
resource "cloudendure_recovery_plan" "rp" {
  name  = "plan"
  steps = [{ machine_ids = [%q] }]
}
`, m)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg},
			{
				// Delete the plan behind Terraform's back: it must be recreated.
				PreConfig:          srv.DeleteAllRecoveryPlans,
				Config:             cfg,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestDeprecationMessageOnAllSchemas(t *testing.T) {
	ctx := t.Context()
	p := New("test")()

	var provResp fwprovider.SchemaResponse
	p.Schema(ctx, fwprovider.SchemaRequest{}, &provResp)
	if provResp.Schema.DeprecationMessage == "" {
		t.Error("provider schema is missing a deprecation message")
	}

	for _, f := range p.Resources(ctx) {
		r := f()
		var meta fwresource.MetadataResponse
		r.Metadata(ctx, fwresource.MetadataRequest{ProviderTypeName: "cloudendure"}, &meta)
		var resp fwresource.SchemaResponse
		r.Schema(ctx, fwresource.SchemaRequest{}, &resp)
		if resp.Schema.DeprecationMessage == "" {
			t.Errorf("resource %s is missing a deprecation message", meta.TypeName)
		}
	}

	for _, f := range p.DataSources(ctx) {
		d := f()
		var meta datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "cloudendure"}, &meta)
		var resp datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &resp)
		if resp.Schema.DeprecationMessage == "" {
			t.Errorf("data source %s is missing a deprecation message", meta.TypeName)
		}
	}
}
