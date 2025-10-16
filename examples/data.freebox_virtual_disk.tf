data "freebox_virtual_disk" "example" {
  path = "/Freebox/VMs/virtual-disk.qcow2"
}

output "disk_type" {
  value = data.freebox_virtual_disk.example.type # "qcow2"
}

output "sizes" {
  value = {
    virtual : data.freebox_virtual_disk.example.virtual_size,
    physical : data.freebox_virtual_disk.example.actual_size,
  }
}
