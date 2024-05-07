data "freebox_api_version" "example" {}

output "box_model" {
  value = data.freebox_api_version.example.box_model
}