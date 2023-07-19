# Google Cloud Platform ACME External Account Binding

To create EAB credential for ACME protocol.

## Example

```hcl
terraform {
  required_providers {
    acme = {
      source  = "vancluever/acme"
      version = "~> 2.15"
    }
    st-gcp = {
      source  = "myklst/st-gcp"
      version = "~> 0.1"
    }
  }
}

provider "st-gcp" {
  project = "xxxx"
}

resource "st-gcp_acme_eab" "cred" {
  credentials_json = file("projectid-6f5d9ed9a85d.json")
}

output "eab_key_id" {
  description = "The key_id of EAB"
  value       = st-gcp_acme_eab.cred.key_id
}

output "eab_hmac_base64" {
  description = "The hmac_base64 of EAB"
  value       = st-gcp_acme_eab.cred.hmac_base64
}
```
