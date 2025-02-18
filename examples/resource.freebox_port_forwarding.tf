resource "freebox_port_forwarding" "example" {
  enabled          = true
  ip_protocol      = "udp"
  target_ip        = "192.168.1.255"
  comment          = "This is an example comment"
  source_ip        = "0.0.0.0"
  # Required to set either source_port/target_port as shown
  # to do port mapping, or range_port_start/range_port_end
  # to forward a full range of ports without port forwarding
  source_port      = 443
  target_port      = 8443
}

output "hostname" {
  value = resource.freebox_port_forwarding.example.hostname
}
