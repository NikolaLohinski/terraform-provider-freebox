data "freebox_virtual_machine_distributions" "example" {}

output "distributions" {
  value = data.freebox_virtual_machine_distributions.example.distributions
}
