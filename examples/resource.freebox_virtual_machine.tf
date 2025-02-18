resource "freebox_virtual_machine" "example" {
  name       = "vm"
  vcpus      = 1
  memory     = 4000
  disk_path  = "RHODES/VMs/terraform-provider-freebox-ubuntu-22.04-aarch64.qcow2"
  disk_type  = "qcow2"
  os         = "ubuntu"
  timeouts  = {
    kill = "15s"
  }
}

output "ip" {
  value = one(resource.freebox_virtual_machine.example.networking[*].ipv4)
}
