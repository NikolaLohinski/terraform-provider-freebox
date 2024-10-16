resource "freebox_remote_file" "example" {
  source_url = "https://example.com/file.txt"
  destination_path = "/Freebox/VMs/file.txt"
  checksum = "sha256:0a0a9f2a6772942557ab5347d9b0e6b8"
}

output "task_id" {
  value = resource.freebox_remote_file.example.task_id
}
