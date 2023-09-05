terraform {
  required_providers {
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

}

output "eab_key_id" {
  description = "The eab key id"
  value       = st-gcp_acme_eab.eab.key_id
}

output "eab_hmac_base64" {
  description = "The eab credential with hmac_base64 format"
  value       = st-gcp_acme_eab.eab.hmac_base64
}
