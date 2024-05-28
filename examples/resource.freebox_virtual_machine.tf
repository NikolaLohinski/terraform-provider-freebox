resource "freebox_virtual_machine" "example" {
  name       = "vm"
  vcpus      = 1
  memory     = 300
  disk_path  = "Freebox/VMs/debian.qcow2"
  disk_type  = "qcow2"
  os         = "debian"
}
