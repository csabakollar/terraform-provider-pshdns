terraform {
  required_providers {
    pshdns = {
      source   = "csabakollar/pshdns"
    }
  }
}

provider "pshdns" {
  username   = "root"
  password   = "qq"
  ssh_server = "localhost"
  ssh_port   = "3333"
}

resource "pshdns" "cname" {
  record_name    = "lofasz"
  record_type    = "CNAME"
  zone_name      = "lofasz"
  hostname_alias = "lofaszkaka.com"
}
