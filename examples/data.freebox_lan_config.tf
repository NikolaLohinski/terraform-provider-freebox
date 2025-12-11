data "freebox_lan_config" "example" {}

output "mode" {
  value = data.freebox_lan_config.example.mode # "bridge" or "router"
}

output "ip" {
  value = data.freebox_lan_config.example.ip
}
