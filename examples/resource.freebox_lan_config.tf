resource "freebox_lan_config" "example" {
  name         = "Freebox"
  name_dns     = "freebox"
  name_mdns    = "freebox"
  name_netbios = "FREEBOX"
}

output "lan_ip" {
  value = freebox_lan_config.example.ip
}
