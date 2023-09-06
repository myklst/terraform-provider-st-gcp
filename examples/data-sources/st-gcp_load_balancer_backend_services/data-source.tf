terraform {
  required_providers {
    st-gcp = {
      source  = "myklst/st-gcp"
      version = "~> 0.1"
    }
  }
}

provider "st-gcp" {}

data "st-gcp_load_balancer_backend_services" "def" {
  name = "backend-service-name"

  tags = {
    env = "test"
    app = "crond"
  }
}
