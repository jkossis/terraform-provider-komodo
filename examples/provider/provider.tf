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
