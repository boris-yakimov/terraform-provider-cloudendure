data "cloudendure_machine" "web" {
  name = "web-01"
}

resource "cloudendure_blueprint" "web" {
  machine_id         = data.cloudendure_machine.web.id
  instance_type      = "COPY_ORIGIN"
  subnet_ids         = ["subnet-0123456789abcdef0"]
  security_group_ids = ["sg-0123456789abcdef0"]
  private_ip_action  = "CREATE_NEW"
  public_ip_action   = "DONT_ALLOCATE"
  run_after_launch   = true

  tags = {
    Environment = "dr"
  }

  disks = [
    { name = "/dev/xvda", type = "SSD" },
  ]
}
