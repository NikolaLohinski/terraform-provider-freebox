resource "freebox_port_forwarding" "example" {
  enabled          = true
  ip_protocol      = "udp"
  target_ip        = "192.168.1.255"
  comment          = "This is an example comment"
  source_ip        = "0.0.0.0"
  port_range_start = 32443
  port_range_end   = 32443
  target_port      = 443
}

output "hostname" {
  value = resource.freebox_port_forwarding.example.hostname
}
