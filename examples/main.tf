terraform {
  required_providers {
    pshdns = {
      versions = ["0.2"]
      source   = "dopsdigital.com/infra/pshdns"
    }
  }
}

provider "pshdns" {
  username   = "root"
  password   = "qq"
  ssh_server = "localhost"
}

resource "pshdns" "cname" {
  record_name    = "lofasz"
  record_type    = "CNAME"
  zone_name      = "lofasz"
  hostname_alias = "lofaszkaka.com"
}
