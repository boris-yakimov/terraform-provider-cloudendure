# Terraform Provider for CloudEndure

A proof-of-concept Terraform provider for [CloudEndure](https://docs.cloudendure.com/)
Disaster Recovery / Migration, built with the
[Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework).

> **Status: archived proof of concept.** AWS has discontinued CloudEndure
> Disaster Recovery outside the China regions (see the
> [AWS DRS FAQ](https://docs.aws.amazon.com/drs/latest/userguide/FAQ.html)) in
> favour of AWS Elastic Disaster Recovery. The provider is tested against an
> in-memory fake of the CloudEndure API, not against the live service.

## Features

| Type        | Name                        | Notes |
| ----------- | --------------------------- | ----- |
| Resource    | `cloudendure_blueprint`     | Configure a machine's launch blueprint. Adopts the existing blueprint on create; delete only removes it from state (the API cannot delete blueprints). Importable. |
| Resource    | `cloudendure_recovery_plan` | Full CRUD. Importable. |
| Data source | `cloudendure_blueprint`     | Look up by `id` or `machine_id`. |
| Data source | `cloudendure_machine`       | Look up by `id` or source machine `name`. |

Full reference documentation is in [`docs/`](docs/); examples are in [`examples/`](examples/).

## Usage

```hcl
terraform {
  required_providers {
    cloudendure = {
      source = "boris-yakimov/cloudendure"
    }
  }
}

# Or set CLOUDENDURE_API_KEY / CLOUDENDURE_PROJECT_ID / CLOUDENDURE_API_URL.
provider "cloudendure" {
  api_key    = var.cloudendure_api_key
  project_id = var.cloudendure_project_id
}

data "cloudendure_machine" "web" {
  name = "web-01"
}

resource "cloudendure_blueprint" "web" {
  machine_id         = data.cloudendure_machine.web.id
  instance_type      = "COPY_ORIGIN"
  subnet_ids         = ["subnet-0123456789abcdef0"]
  security_group_ids = ["sg-0123456789abcdef0"]
}
```

The provider is not published to the Terraform Registry. To try it locally:

```shell
make install   # installs into ~/.terraform.d/plugins/registry.terraform.io/boris-yakimov/cloudendure/...
cd examples/complete && terraform init && terraform plan
```

## Development

Requirements: Go (see `go.mod`), Terraform >= 1.5, and optionally
[golangci-lint](https://golangci-lint.run/) v2.

```shell
make build      # build ./terraform-provider-cloudendure
make test       # unit tests; runs real terraform plan/apply against a fake API
make lint       # golangci-lint
make generate   # format examples and regenerate docs/ (tfplugindocs)
```

Layout:

```
main.go                  provider server entrypoint
internal/client          CloudEndure API client (auth, machines, blueprints, recovery plans)
internal/client/API.md   notes on the CloudEndure API
internal/fakeapi         in-memory fake API used by tests
internal/provider        provider, resources and data sources
examples/, docs/         tfplugindocs examples and generated docs
```

### Debugging

```shell
make debug   # or: dlv exec --headless ./terraform-provider-cloudendure -- -debug
```

The provider prints a `TF_REATTACH_PROVIDERS=...` line; export it in the shell
where you run `terraform`. Use `TF_LOG=DEBUG` to see provider logs.

## Known limitations

- The JSON shape of recovery plan steps (`{"steps": {"items": [...]}}`) and the
  `wait_before` format come from the published Swagger definition and were not
  verified against the live API.
- Removing an optional blueprint attribute from the configuration does not reset
  it in CloudEndure; the last applied value is kept.
- There is no acceptance test suite against a real CloudEndure account.
