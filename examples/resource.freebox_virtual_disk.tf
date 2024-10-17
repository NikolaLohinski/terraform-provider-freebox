resource "freebox_virtual_disk" "example" {
  path         = "/Freebox/VMs/disk.qcow2"
  type         = "qcow2"
  virtual_size = 10 * 1024 * 1024 * 1024  # 10 GB
}

output "task_id" {
  value = resource.freebox_virtual_disk.example.size_on_disk
}
