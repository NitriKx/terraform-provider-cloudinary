# Terraform Provider: Cloudinary

Manages product-environment-level Cloudinary resources (folders) as infrastructure-as-code.

For account-level resources (product environments, access keys, custom policies), see the companion
[cloudinary-provisioning provider](https://github.com/NitriKx/terraform-provider-cloudinary-provisioning).

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

### Option 1: `CLOUDINARY_URL` (recommended)

```bash
export CLOUDINARY_URL="cloudinary://API_KEY:API_SECRET@CLOUD_NAME"
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
}
```

Environment variable fallbacks: `CLOUDINARY_CLOUD_NAME`, `CLOUDINARY_API_KEY`, `CLOUDINARY_API_SECRET`.

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

The `id` and `external_id` attributes expose the folder's Cloudinary-assigned external ID, which can be used in Cedar policy statements (e.g. `resource.ancestor_ids.contains("...")`).

Renaming a folder (changing `path`) is performed as an in-place move — no destroy/recreate.

**Import** (use the folder's `external_id`):
```bash
terraform import cloudinary_folder.assets "<external_id>"
```

---

## Data Sources

### `cloudinary_current_principal`

Returns the identity of the currently authenticated principal (the credentials used to configure
the provider). No API call is made — the values come from the provider configuration.

```hcl
data "cloudinary_current_principal" "me" {}

output "current_principal_id" {
  value = data.cloudinary_current_principal.me.principal_id
}
```

**Attributes:**
- `principal_id` - The unique identifier of the current principal (the API key).
- `principal_type` - The type of the current principal (currently always `"apiKey"`).
- `cloud_name` - The Cloudinary cloud name the provider is configured for.

### `cloudinary_folder`

Reads information about an existing folder by path. Exposes `id`, `external_id`, `name`, and `path`.

```hcl
data "cloudinary_folder" "assets" {
  path = "production/assets"
}

output "folder_external_id" {
  value = data.cloudinary_folder.assets.external_id
}
```

---

## Using both providers together

```hcl
terraform {
  required_providers {
    cloudinary = {
      source  = "NitriKx/cloudinary"
      version = "~> 0.1"
    }
    cloudinaryprovisioning = {
      source  = "NitriKx/cloudinary-provisioning"
      version = "~> 0.1"
    }
  }
}

provider "cloudinary" {}             # reads CLOUDINARY_URL
provider "cloudinaryprovisioning" {} # reads CLOUDINARY_ACCOUNT_URL

data "cloudinaryprovisioning_product_environment" "current" {
  cloud_name = "my-cloud"
}

resource "cloudinary_folder" "assets" {
  path = "my-project/images"
}

resource "cloudinaryprovisioning_access_key" "system" {
  product_environment_id = data.cloudinaryprovisioning_product_environment.current.id
  name                   = "system-key"
}

resource "cloudinaryprovisioning_custom_policy" "restrict" {
  name       = "restrict-to-folder"
  scope_type = "prodenv"
  scope_id   = data.cloudinaryprovisioning_product_environment.current.id

  policy_statement = <<-EOT
    permit(
      principal == Cloudinary::APIKey::"${cloudinaryprovisioning_access_key.system.api_key}",
      action in [Cloudinary::Action::"create", Cloudinary::Action::"read"],
      resource is Cloudinary::Asset
    ) when {
      resource.ancestor_ids.contains("${cloudinary_folder.assets.external_id}")
    };
  EOT
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
