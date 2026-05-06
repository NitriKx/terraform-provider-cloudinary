# Terraform Provider for Cloudinary

A Terraform provider (also compatible with OpenTofu) for managing [Cloudinary](https://cloudinary.com) resources as infrastructure-as-code.

This provider targets the Cloudinary Admin and Provisioning APIs, and is designed to serve as the foundation for a Crossplane provider.

---

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0 or [OpenTofu](https://opentofu.org/docs/intro/install/) >= 1.0
- Go >= 1.22 (to build from source)

---

## Installation

### Terraform / OpenTofu Registry

```hcl
terraform {
  required_providers {
    cloudinary = {
      source  = "NitriKx/cloudinary"
      version = "~> 0.1"
    }
  }
}
```

Run `terraform init` (or `tofu init`) to download the provider.

### OpenTofu: direct from GitHub (no registry required)

Add a `provider_installation` block to your OpenTofu CLI config (`~/.tofurc`):

```hcl
provider_installation {
  direct {
    include = ["registry.terraform.io/NitriKx/cloudinary"]
  }
}
```

### Local development build

```bash
task install
```

This builds the binary and installs it to `~/.terraform.d/plugins/registry.terraform.io/NitriKx/cloudinary/0.1.0/<os>_<arch>/`.

---

## Authentication

### Option 1: Cloudinary URL (recommended)

Set the `CLOUDINARY_URL` environment variable:

```bash
export CLOUDINARY_URL="cloudinary://API_KEY:API_SECRET@CLOUD_NAME"
export CLOUDINARY_ACCOUNT_ID="your_account_id"  # Required for provisioning resources
```

```hcl
provider "cloudinary" {}
```

### Option 2: Explicit credentials

```hcl
provider "cloudinary" {
  cloud_name = var.cloudinary_cloud_name
  api_key    = var.cloudinary_api_key
  api_secret = var.cloudinary_api_secret
  account_id = var.cloudinary_account_id
}
```

Or via environment variables:

```bash
export CLOUDINARY_CLOUD_NAME="my-cloud"
export CLOUDINARY_API_KEY="123456789"
export CLOUDINARY_API_SECRET="abcdef..."
export CLOUDINARY_ACCOUNT_ID="acc_..."
```

> `account_id` is required for `cloudinary_product_environment`, `cloudinary_product_environment_access_key`, and `cloudinary_custom_policy`.

---

## Resources

### `cloudinary_folder`

Manages a folder in your Cloudinary account.

```hcl
resource "cloudinary_folder" "assets" {
  path = "production/assets"
}

resource "cloudinary_folder" "nested" {
  path = "production/assets/images"
}
```

**Import:**
```bash
terraform import cloudinary_folder.assets "production/assets"
```

---

### `cloudinary_product_environment`

Manages a Cloudinary product environment (sub-account).

```hcl
resource "cloudinary_product_environment" "staging" {
  name    = "staging"
  enabled = true

  custom_attributes = {
    team        = "platform"
    environment = "staging"
  }
}
```

**Import:**
```bash
terraform import cloudinary_product_environment.staging "sub_abc123"
```

---

### `cloudinary_product_environment_access_key`

Manages an API access key for a product environment.

> The `api_secret` is only available immediately after creation and is preserved in Terraform state. Treat it as a sensitive value.

```hcl
resource "cloudinary_product_environment_access_key" "api" {
  product_environment_id = cloudinary_product_environment.staging.id
  name                   = "terraform-managed"
  enabled                = true
}

output "api_key" {
  value = cloudinary_product_environment_access_key.api.api_key
}

output "api_secret" {
  value     = cloudinary_product_environment_access_key.api.api_secret
  sensitive = true
}
```

**Import:**
```bash
terraform import cloudinary_product_environment_access_key.api "sub_abc123/123456789"
```

---

### `cloudinary_custom_policy`

Manages a custom permissions policy. Use `jsonencode()` to build the `policy_statement`.

```hcl
# Account-scoped policy
resource "cloudinary_custom_policy" "readonly" {
  name        = "account-readonly"
  description = "Read-only access to the entire account"
  scope_type  = "account"
  enabled     = true

  policy_statement = jsonencode({
    effect   = "allow"
    action   = ["read:*"]
    resource = ["*"]
  })
}

# Product environment-scoped policy
resource "cloudinary_custom_policy" "staging_policy" {
  name       = "staging-readonly"
  scope_type = "product_environment"
  scope_id   = cloudinary_product_environment.staging.id
  enabled    = true

  policy_statement = jsonencode({
    effect   = "allow"
    action   = ["read:*"]
    resource = ["*"]
  })
}
```

**Import:**
```bash
terraform import cloudinary_custom_policy.readonly "policy_abc123"
```

---

## Data Sources

### `data.cloudinary_folder`

Reads information about an existing folder.

```hcl
data "cloudinary_folder" "assets" {
  path = "production/assets"
}

output "folder_name" {
  value = data.cloudinary_folder.assets.name
}
```

---

### `data.cloudinary_product_environment`

Reads information about an existing product environment.

```hcl
data "cloudinary_product_environment" "production" {
  id = "sub_abc123"
}
```

---

## Development

### Building

```bash
task build
```

### Running tests

Unit tests (no credentials needed):

```bash
task test
```

Acceptance tests (requires real Cloudinary credentials):

```bash
export CLOUDINARY_URL="cloudinary://KEY:SECRET@CLOUD"
export CLOUDINARY_ACCOUNT_ID="acc_..."
task testacc
```

### Generating documentation

```bash
task docs
```

### Linting

```bash
task lint
```

---

## License

[Apache 2.0](LICENSE)
