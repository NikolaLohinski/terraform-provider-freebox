data "freebox_lan_interfaces" "example" {}

output "interfaces" {
  value = data.freebox_lan_interfaces.example.interfaces
}
