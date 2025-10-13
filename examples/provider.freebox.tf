terraform {
  required_providers {
    freebox = {
      source = "registry.terraform.io/NikolaLohinski/freebox"
    }
  }
}
provider "freebox" {
  endpoint    = "http://mafreebox.freebox.fr"
  api_version = "v10"
}