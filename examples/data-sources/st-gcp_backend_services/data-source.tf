data "st-gcp_backend_services" "def" {
  name = "backend-service-name"

  tags = {
    env = "test"
    app = "crond"
  }
}
