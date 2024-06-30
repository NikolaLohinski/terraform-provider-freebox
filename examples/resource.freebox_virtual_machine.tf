resource "freebox_virtual_machine" "example" {
  name       = "vm"
  vcpus      = 1
  memory     = 300
  disk_path  = "Freebox/VMs/debian.qcow2"
  disk_type  = "qcow2"
  os         = "ubuntu"
  timeouts  = {
    kill       = "15s"
    networking = "30s"
  }
}

output "ipv4" {
  value = one(resource.freebox_virtual_machine.example.networking[*].ipv4)
}
