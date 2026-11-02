# Look up a blueprint by the machine it belongs to...
data "cloudendure_blueprint" "web" {
  machine_id = data.cloudendure_machine.web.id
}

# ...or directly by its ID.
data "cloudendure_blueprint" "by_id" {
  id = "00000000-0000-0000-0000-000000000000"
}

output "web_subnets" {
  value = data.cloudendure_blueprint.web.subnet_ids
}
