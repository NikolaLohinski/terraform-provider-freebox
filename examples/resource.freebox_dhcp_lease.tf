resource "freebox_dhcp_lease" "example" {
  mac      = "00:11:22:33:44:55"
  ip       = "192.168.1.100"
  hostname = "my-device"
  comment  = "My device static lease"
}
