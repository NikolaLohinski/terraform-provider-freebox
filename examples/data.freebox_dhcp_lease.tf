data "freebox_dhcp_lease" "example" {
  mac = "00:11:22:33:44:55"
}

output "ip" {
  value = data.freebox_dhcp_lease.example.ip
}
