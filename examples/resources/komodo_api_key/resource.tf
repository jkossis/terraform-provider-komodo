terraform {
  required_providers {
    komodo = {
      source = "jkossis/komodo"
    }
  }
}

provider "komodo" {
  # endpoint, username, and password may be set here or with
  # KOMODO_ENDPOINT, KOMODO_USERNAME, and KOMODO_PASSWORD.
}

# Create an API key that never expires
resource "komodo_api_key" "example" {
  name    = "my-api-key"
  expires = 0
}

# Output the credentials
output "api_key" {
  value = komodo_api_key.example.key
}

output "api_secret" {
  value     = komodo_api_key.example.secret
  sensitive = true
}

output "user_id" {
  value = komodo_api_key.example.user_id
}

output "created_at" {
  value = komodo_api_key.example.created_at
}
