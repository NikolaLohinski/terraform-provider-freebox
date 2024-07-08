resource "freebox_port_forwarding" "example" {
  enabled          = true
  ip_protocol      = "udp"
  port_range_start = 8000
  port_range_end   = 8000
  target_ip        = "192.168.1.255"
  comment          = "This is an example comment"
  source_ip        = "0.0.0.0"
}

output "task_id" {
  value = resource.freebox_port_forwarding.example.hostname
}
