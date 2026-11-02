data "cloudendure_machine" "web" {
  name = "web-01"
}

output "web_replication_status" {
  value = data.cloudendure_machine.web.replication_status
}
