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
  eab_account_expires_days = 60
}

output "eab_key_id" {
  description = "The eab account of key id"
  value       = st-gcp_acme_eab.eab.key_id
}

output "eab_hmac_base64" {
  description = "The eab account of hmac_base64"
  value       = st-gcp_acme_eab.eab.hmac_base64
}
