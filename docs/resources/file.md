# sops_file Resource

Create a sops-encrypted file on disk.

## Example Usage
Provider configuration:
```hcl
provider "sops" {
  // age configuration
  age = {
    key = "~/.config/sops/age/keys.txt" //path to the key file
  }
  // GCP KMS configuration
  gcpkms = {
    ids = "projects/XXX/locations/XXX/keyRings/XXX/cryptoKeys/XXX" //takes a comma separated list of GCP KMS resource IDs
  }
  // AWS KMS configuration
  kms = {
    profile = "default"
    arn     = "arn:aws:kms:<region>:<account>:key/<kms_resource_id>"
  }
}
// or
provider "sops" {}
```

```hcl
resource "sops_file" "secret_data" {
  encryption_type = local.encrypted_input__type // "age" or "kms"
  content         = local.sensitive_output // the content to encrypt
  filename        = local.sensitive_output_file // the filename to write to
  age             = local.encrypted_output__config__age  // the age configuration
  kms             = local.encrypted_output__config__kms // the kms configuration
}
```

## Argument Reference
* `encryption_type` - (Required) The type of encryption to use.
* `content` - (Required) The content to encrypt.
* `filename` - (Required) Path to the encrypted file
* `age` - (Optional) Age configuration
* `gcpkms` - (Optional) GCP KMS configuration
* `kms` - (Optional) AWS KMS configuration
