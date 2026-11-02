# End-to-end example: configure the blueprints of two replicated machines and
# group them into a recovery plan.
#
#   export CLOUDENDURE_API_KEY="..."
#   export CLOUDENDURE_PROJECT_ID="..."
#   terraform init && terraform apply

terraform {
  required_version = ">= 1.5"

  required_providers {
    cloudendure = {
      source = "boris-yakimov/cloudendure"
    }
  }
}

provider "cloudendure" {}

variable "machines" {
  description = "Source machine names, in launch order."
  type        = list(string)
  default     = ["db-01", "web-01"]
}

variable "subnet_ids" {
  type = list(string)
}

variable "security_group_ids" {
  type = list(string)
}

data "cloudendure_machine" "this" {
  for_each = toset(var.machines)
  name     = each.value
}

resource "cloudendure_blueprint" "this" {
  for_each = data.cloudendure_machine.this

  machine_id         = each.value.id
  instance_type      = "COPY_ORIGIN"
  subnet_ids         = var.subnet_ids
  security_group_ids = var.security_group_ids
  private_ip_action  = "CREATE_NEW"
  public_ip_action   = "DONT_ALLOCATE"

  tags = {
    ManagedBy = "terraform"
    Source    = each.key
  }
}

resource "cloudendure_recovery_plan" "this" {
  name = "terraform-managed"

  steps = [
    for name in var.machines : {
      name        = name
      machine_ids = [cloudendure_blueprint.this[name].machine_id]
    }
  ]
}

output "blueprint_ids" {
  value = { for k, bp in cloudendure_blueprint.this : k => bp.id }
}
