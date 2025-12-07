resource "freebox_virtual_disk" "example" {
  path         = "/Freebox/VMs/disk.qcow2"
  virtual_size = 10 * 1024 * 1024 * 1024 # 10 GB
  resize_from  = "/Freebox/debian.qcow2"
}

output "size_on_disk" {
  value = resource.freebox_virtual_disk.example.size_on_disk
}

output "disk_type" {
  value = resource.freebox_virtual_disk.example.type # "qcow2"
}
