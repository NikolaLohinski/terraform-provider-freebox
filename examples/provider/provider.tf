terraform {
  required_providers {
    freebox = {
      source = "terraform.registry.io/nikolalohinski/freebox"
    }
  }
}
provider "freebox" {}