terraform {
  required_providers {
    freebox = {
      source = "terraform.registry.io/nikolalohinski/freebox"
    }
  }
}
provider "freebox" {
  endpoint    = "http://mafreebox.freebox.fr"
  api_version = "v10"
}