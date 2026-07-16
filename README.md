# Terraform Provider for Komodo

This is a Terraform provider for managing [Komodo](https://komo.do/) resources via the Komodo API.

## Table of Contents

- [Requirements](#requirements)
- [Quick Start](#quick-start)
- [Using the Provider](#using-the-provider)
- [Resources](#resources)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)
- [Developing the Provider](#developing-the-provider)

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (for development)
- A Komodo instance with API access

## Quick Start

### 1. Build the Provider

```bash
# Clone the repository
git clone <your-repo-url>
cd terraform-provider-komodo

# Build the provider
go build -o terraform-provider-komodo
```

### 2. Install the Provider Locally

For local development, use Terraform's [development overrides](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers).

Create or edit `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "jkossis/komodo" = "/path/to/terraform-provider-komodo"
  }

  direct {}
}
```

Replace `/path/to/terraform-provider-komodo` with the actual path to your repository directory.

### 3. Configure Your Environment

Set your Komodo credentials:

```bash
export KOMODO_ENDPOINT="https://your-komodo-server.com"
export KOMODO_USERNAME="your-username"
export KOMODO_PASSWORD="your-password"
```

### 4. Create Your First Configuration

Create a file named `main.tf`:

```hcl
terraform {
  required_providers {
    komodo = {
      source = "jkossis/komodo"
    }
  }
}

provider "komodo" {
  # endpoint, username, and password will be read from environment variables
}

resource "komodo_api_key" "example" {
  name = "my-api-key"
}

output "api_key" {
  value = komodo_api_key.example.key
}

output "api_secret" {
  value     = komodo_api_key.example.secret
  sensitive = true
}
```

### 5. Run Terraform

```bash
# Initialize Terraform
terraform init

# Plan the changes
terraform plan

# Apply the configuration
terraform apply

# When you're done, destroy the resources
terraform destroy
```

## Using the Provider

### Configuration

The provider requires three effective configuration values:

- `endpoint` - The URL of your Komodo API endpoint
- `username` - Your Komodo username
- `password` - Your Komodo password

The `endpoint`, `username`, and `password` provider attributes are optional because each can be supplied through its matching environment variable. Configuration values take precedence over environment variables.
An explicitly empty or unknown configuration value is an error and does not fall back to its environment variable.

#### 1. In the provider block:

```hcl
provider "komodo" {
  endpoint = "https://your-komodo-server.com"
  username = "your-username"
  password = "your-password"
}
```

#### 2. Via environment variables:

```bash
export KOMODO_ENDPOINT="https://your-komodo-server.com"
export KOMODO_USERNAME="your-username"
export KOMODO_PASSWORD="your-password"
```

#### 3. Mixed approach:

```hcl
provider "komodo" {
  endpoint = "https://your-komodo-server.com"
  # username and password will be read from environment variables
}
```

### Resources

#### `komodo_api_key`

Manages a Komodo API key.

**Example Usage:**

```hcl
# Create an API key
resource "komodo_api_key" "example" {
  name    = "my-api-key"
  expires = 0  # 0 means never expires
}

# Output the credentials
output "api_key" {
  value = komodo_api_key.example.key
}

output "api_secret" {
  value     = komodo_api_key.example.secret
  sensitive = true
}
```

**Schema:**

- `name` (Required, String) - A human-friendly name for the API key.
- `expires` (Optional, Int64) - Expiration timestamp in milliseconds since epoch. Use 0 for no expiration. Default: `0`

**Computed Attributes:**

- `key` (String) - The API key identifier
- `secret` (String, Sensitive) - The API key secret (only available on creation)
- `user_id` (String) - The ID of the user who owns this key
- `created_at` (Int64) - Creation timestamp in milliseconds since epoch

**Important Notes:**
- The secret is only returned when the key is created. It's not available after creation, so store it securely.
- API keys are immutable. Changing the `name` or `expires` attributes will force the resource to be destroyed and recreated with a new key and secret.

## Examples

Check out the [examples](./examples/) directory for more configurations:

- [Basic Provider Configuration](./examples/provider/provider.tf)
- [API Key Resource Examples](./examples/resources/komodo_api_key/resource.tf)

## Troubleshooting

### "Provider not found" error

If you see an error like "provider registry.terraform.io/jkossis/komodo not found", make sure:

1. You've set up the dev overrides in `~/.terraformrc` correctly
2. The path in dev overrides points to the directory containing the built binary
3. You've built the provider binary (`go build -o terraform-provider-komodo`)

### Authentication errors

If you see authentication errors:

1. Verify your Komodo credentials are correct
2. Check that the endpoint URL is correct (including protocol)
3. Ensure your user has permission to create API keys

### Connection errors

If you see connection errors:

1. Verify your Komodo instance is running and accessible
2. Check that the endpoint URL is correct
3. Check Komodo logs for any API errors

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

### Testing

To run the full suite of acceptance tests, you'll need a running Komodo instance.

Set up your test environment:

```bash
export TF_ACC=1
export KOMODO_ENDPOINT="https://your-komodo-server.com"
export KOMODO_USERNAME="your-username"
export KOMODO_PASSWORD="your-password"
```

Acceptance tests are skipped when `TF_ACC` is unset. When `TF_ACC=1`, `KOMODO_ENDPOINT`, `KOMODO_USERNAME`, and `KOMODO_PASSWORD` must all be set.

Then run the tests:

```bash
make testacc
```

## License

This provider is published under the MPL-2.0 license.
