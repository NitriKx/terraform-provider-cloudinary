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

**Precedence (highest to lowest):** HCL `cloud_name`/`api_key`/`api_secret` → HCL `cloudinary_url` → env `CLOUDINARY_CLOUD_NAME`/`CLOUDINARY_API_KEY`/`CLOUDINARY_API_SECRET` → env `CLOUDINARY_URL`. HCL attributes always win over environment variables, so a provider block with explicit credentials targets the intended environment even when `CLOUDINARY_URL` is set in the shell.

---

## Resources

### `cloudinary_trigger`

Manages a Cloudinary webhook notification trigger. Triggers fire on specific asset events
and POST a payload to the configured URL. Up to 30 triggers per product environment
(the Cloudinary console UI shows only 10).

```hcl
resource "cloudinary_trigger" "on_upload" {
  uri        = "https://example.com/webhooks/cloudinary"
  event_type = "upload"
}
```

**Arguments:**
- `uri` (Required) — The webhook URL.
- `event_type` (Required) — The event that fires the trigger. One of: `upload`, `delete`, `rename`, `move`, `eager`, `explode`, `multi`, `resource_tags_changed`, `resource_context_changed`, `resource_metadata_changed`, `resource_display_name_changed`, `access_control_changed`, `related_assets`, `create_folder`, `delete_folder`, `move_or_rename_asset_folder`, `proof_status_changed`, `error`, `all`.
- `additive` (Optional, default `false`) — When `true`, fires alongside per-asset `notification_url` callbacks.
- `filter` (Optional) — A [JSONLogic](https://jsonlogic.com/) expression (JSON-encoded string) that filters which events fire this trigger.
- `payload_template` (Optional) — A Mustache template object (JSON-encoded string, max 16 KB) that customises the notification payload. Available namespaces: `event`, `asset`, `folder`, `eager`, `error`, `raw`.
- `auth_scheme` (Optional, default `"default"`) — Signature method for verifying webhook payloads. One of: `default`, `legacy_hmac`, `eddsa_v2`.

**Computed attributes:** `id`, `product_environment_id`, `uri_type`, `created_at`, `updated_at`.

**Import** (use the trigger's `id`):
```bash
terraform import cloudinary_trigger.on_upload "<trigger_id>"
```

> **Note on webhook signing key:** The API key used to sign outgoing webhook payloads is
> configured separately on a Cloudinary access key with `dedicated_for = "webhooks"`,
> not on the trigger itself.

---

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

### `cloudinary_trigger`

Reads information about an existing trigger by `id`, or by the combination of `event_type` + `uri`.

```hcl
data "cloudinary_trigger" "existing" {
  event_type = "upload"
  uri        = "https://example.com/webhooks/cloudinary"
}
```

### `cloudinary_triggers`

Returns all webhook notification triggers configured for the product environment.

```hcl
data "cloudinary_triggers" "all" {}

output "trigger_uris" {
  value = [for t in data.cloudinary_triggers.all.triggers : t.uri]
}
```

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
