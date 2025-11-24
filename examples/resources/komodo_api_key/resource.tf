terraform {
  required_providers {
    komodo = {
      source = "jkossis/komodo"
    }
  }
}

provider "komodo" {
  server   = "https://your-komodo-server.com"
  username = "your-username"
  password = "your-password"
}

# Create an API key that never expires
resource "komodo_api_key" "example" {
  name    = "my-api-key"
  expires = 0
}

# Create an API key with expiration
resource "komodo_api_key" "expiring" {
  name    = "expiring-key"
  expires = 1735689600000 # Timestamp in milliseconds
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
