data "freebox_dhcp_leases" "all" {}

output "leases" {
  value = data.freebox_dhcp_leases.all.leases
}
