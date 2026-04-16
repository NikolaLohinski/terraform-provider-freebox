resource "freebox_vpn_user" "example" {
  login       = "myuser"
  password    = "s3cr3t"
  description = "Example VPN user"
}

output "ovpn_config" {
  value     = freebox_vpn_user.example.ovpn_config
  sensitive = true
}
