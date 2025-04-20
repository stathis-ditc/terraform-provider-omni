# (Unofficial) Terraform Provider for SideroLabs Omni

This is an unofficial Terraform provider for SideroLabs Omni, allowing you to manage and query Omni resources through Terraform.

DISCLAIMER: The project is under development and is NOT an official SideroLabs release. It has not been tested to production
environments, only in development. Use at your own risk as it comes with no warranty. Thank you.

## Requirements

- Terraform >= 1.5, OpenTofu >= 1.9.0
- Go >= 1.20 (for development)
- SideroLabs Omni already installed. 
- Service account created in Omni 

## Installation

### Using Terraform Registry

This provider is under development and currntly it is not avialble in any registry. To use it;

1. Run `go install .`
2. Find the `GOBIN` path where Go installs your binaries. If the GOBIN go environment variable is not set, use the default path `/Users/<username>/go/bin`
3. Create in your home directory a new file `.terraformrc` and add the following contents
```
provider_installation {

  dev_overrides {
      "ditc/omni" = "<PATH from step 2>"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```
4. Add the provider as follows

```hcl
terraform {
  required_providers {
    omni = {
      source = "ditc/-omni"
    }
  }
}

provider "omni" {
  endpoint           = "https://your-omni-instance.example.com"
  service_account_key = "your-base64-encoded-service-account-key"
}
```
5. Since this is a development override, do not run init as it is not going to work. Run `plan` and `apply` commands directly

## Local Development

For local development, you can build and install the provider locally:

```bash
go build -o terraform-provider-omni
```

## Features

Currently, the provider supports the following features:

### Data Sources

- `omni_machines` - List all machines in the Omni cluster

Example:

```hcl
data "omni_machines" "example" {}

output "connected_machines" {
  description = "List of connected machines"
  value = [for machine in data.omni_machines.example.machines : machine.id if machine.connected]
}
```

## Provider Configuration

| Name | Description | Type | Required |
|------|-------------|------|----------|
| `endpoint` | The Omni API endpoint URL | `string` | Yes |
| `service_account_key` | The base64-encoded service account key | `string` | Yes |

## License

This project is licensed under the Mozilla Public License 2.0. See the [LICENSE](LICENSE) file for details.
