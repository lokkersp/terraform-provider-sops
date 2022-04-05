# sops_file_entry_entry Data Source

Read key from a sops-encrypted file on disk.

## Example Usage

```hcl
provider "sops" {}

data "sops_file_entry" "password-value" {
  source_file = "demo-secret.enc.json"
  data_key    = "password"
}
data "sops_file_entry" "db-secret" {
  source_file = "demo-secret.enc.json"
  data_key    = "db"
}

output "root-value-password" {
  # Access the password variable from the map
  value = data.sops_file_entry.demo-secret.map["password"]
}

output "mapped-nested-value" {
  # Access the password variable that is under db via the terraform map of data
  value = data.sops_file_entry.db-secret.map["db.password"]
}

output "nested-json-value" {
  # Access the password variable that is under db via the terraform object
  value = jsondecode(data.sops_file_entry.db-secret.yaml).db.password
}
```

## Argument Reference

* `source_file` - (Required) Path to the encrypted file.
* `data_key` - (Required) Key to read from the encrypted file.
* `input_type` - (Optional) The provider will use the file extension to determine how to unmarshal the data. If your file does not have the usual extension, set this argument to `yaml` or `json` accordingly, or `raw` if the encrypted data is encoded differently.

## Attribute Reference

* `data` - value of the data key in the encrypted file.
* `yaml` - Multi-line string containing the key with value in YAML format.
* `map` - The unmarshalled data as a dictionary. Use dot-separated keys to access nested data.
* `raw` - The entire unencrypted file as a string.

