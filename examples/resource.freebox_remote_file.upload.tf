resource "freebox_remote_file" "upload" {
  source_local_file = "/path/to/local/file.txt"
  destination_path  = "/Freebox/VMs/file.txt"
  checksum          = "sha256:0a0a9f2a6772942557ab5347d9b0e6b8"
}
