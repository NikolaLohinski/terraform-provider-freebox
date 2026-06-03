data "freebox_system_info" "example" {}

output "firmware_version" {
  value = data.freebox_system_info.example.firmware_version
}

output "temperatures" {
  value = {
    cpu_m  = data.freebox_system_info.example.temp_cpum
    cpu_b  = data.freebox_system_info.example.temp_cpub
    switch = data.freebox_system_info.example.temp_sw
  }
}
