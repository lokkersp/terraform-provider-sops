# sops Provider

A Terraform plugin for using files encrypted with [Mozilla sops](https://github.com/mozilla/sops).

!> To prevent plaintext secrets from being written to disk, you *must* use a secure remote state backend. See the [official docs](https://www.terraform.io/docs/state/sensitive-data.html) on _Sensitive Data in State_ for more information.

## Example Usage

```hcl
provider "sops" {}

data "sops_file" "demo-secret" {
  source_file = "demo-secret.enc.json"
}

output "db-password" {
  # Access the password variable that is under db via the terraform map of data
  value = data.sops_file.demo-secret.data["db.password"]
}

resource "sops_file" "secret_data" {
  encryption_type = local.encrypted_input__type // "age" or "kms"
  content         = local.sensitive_output // the content to encrypt
  filename        = local.sensitive_output_file // the filename to write to
  age             = local.encrypted_output__config__age  // the age configuration
  kms             = local.encrypted_output__config__kms // the kms configuration
}
```
