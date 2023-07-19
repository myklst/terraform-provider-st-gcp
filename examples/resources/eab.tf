terraform {
  required_providers {
    # acme = {
    #   source  = "vancluever/acme"
    #   version = "2.15.1"
    # }
    st-gcp = {
      source  = "myklst/st-gcp"
      version = "0.1.1"
    }
  }
}

provider "st-gcp" {
  project     = "xxxx"
  credentials = file("projectid-6f5d9ed9a85d.json")
}

resource "st-gcp_acme_eab" "eab" {
  credentials_json = file("projectid-6f5d9ed9a85d.json")
}

output "eab_key_id" {
  description = "The eab account of key id"
  value       = st-gcp_eab.eab.key_id
}

output "eab_hmac_base64" {
  description = "The eab account of hmac_base64"
  value       = st-gcp_eab.eab.hmac_base64
}
