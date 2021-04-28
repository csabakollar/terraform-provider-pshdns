# Terraform PowerSHell DNS Provider

This repository was forked from [for a Terraform Windows DNS Provider, which you can use to create DNS records in Microsoft Windows DNS.](https://github.com/PortOfPortland/terraform-provider-windns) and slightly modified. We had problems connecting via psh sessions to a windows server from linux, so decided to install OpenSSH Daemon on the windows server. The rest of the provider more or less the same as the original.

# Using the Provider

### Example

```hcl
# configure the provider
# username + password - used to build a powershell credential
# server - the server we'll create a WinRM session into to perform the DNS operations
# usessl - whether or not to use HTTPS for our WinRM session (by default port TCP/5986)
variable "username" {
  type = "string"
}

variable "password" {
  type = "string"
}

provider "pshdns" {
  server = "mydc.mydomain.com"
  username = "${var.username}"
  password = "${var.password}"
  ssh_server = "<windows server, which has opensshd installed and running>
  ssh_port = "<port where the opensshd is running>
  dns_server = "<windows server, which hosts the zones you want to manager records in>"
}

#create an a record
resource "pshdns" "dns" {
  record_name = "testentry1"
  record_type = "A"
  zone_name = "mydomain.com"
  ipv4address = "192.168.1.5"
}

#create a cname record
resource "pshdns" "dnscname" {
  record_name = "testcname1"
  record_type = "CNAME"
  zone_name = "mydomain.com"
  hostnamealias = "myhost1.mydomain.com"
}
```
