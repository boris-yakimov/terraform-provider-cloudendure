data "cloudendure_machine" "db" {
  name = "db-01"
}

data "cloudendure_machine" "web" {
  name = "web-01"
}

resource "cloudendure_recovery_plan" "app" {
  name = "app-recovery"

  steps = [
    {
      name        = "databases"
      machine_ids = [data.cloudendure_machine.db.id]
    },
    {
      name        = "web"
      machine_ids = [data.cloudendure_machine.web.id]
      wait_before = "300"
    },
  ]
}
