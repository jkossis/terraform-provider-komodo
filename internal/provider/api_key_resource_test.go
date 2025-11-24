// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccApiKeyResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApiKeyResourceConfig_basic("test-key-basic"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("komodo_api_key.test", "name", "test-key-basic"),
					resource.TestCheckResourceAttrSet("komodo_api_key.test", "key"),
					resource.TestCheckResourceAttrSet("komodo_api_key.test", "secret"),
					resource.TestCheckResourceAttrSet("komodo_api_key.test", "user_id"),
					resource.TestCheckResourceAttrSet("komodo_api_key.test", "created_at"),
					resource.TestCheckResourceAttr("komodo_api_key.test", "expires", "0"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccApiKeyResource_withExpiration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create key with expiration
			{
				Config: testAccApiKeyResourceConfig_withExpiration("test-key-expiring", 1893456000000),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("komodo_api_key.test", "name", "test-key-expiring"),
					resource.TestCheckResourceAttr("komodo_api_key.test", "expires", "1893456000000"),
					resource.TestCheckResourceAttrSet("komodo_api_key.test", "key"),
					resource.TestCheckResourceAttrSet("komodo_api_key.test", "secret"),
				),
			},
		},
	})
}

func TestAccApiKeyResource_multipleKeys(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple keys
			{
				Config: testAccApiKeyResourceConfig_multiple("test-key-1", "test-key-2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check first key
					resource.TestCheckResourceAttr("komodo_api_key.test1", "name", "test-key-1"),
					resource.TestCheckResourceAttrSet("komodo_api_key.test1", "key"),
					resource.TestCheckResourceAttrSet("komodo_api_key.test1", "secret"),
					// Check second key
					resource.TestCheckResourceAttr("komodo_api_key.test2", "name", "test-key-2"),
					resource.TestCheckResourceAttrSet("komodo_api_key.test2", "key"),
					resource.TestCheckResourceAttrSet("komodo_api_key.test2", "secret"),
				),
			},
		},
	})
}

func TestAccApiKeyResource_secretNotAvailableAfterCreation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create key
			{
				Config: testAccApiKeyResourceConfig_basic("test-key-secret"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("komodo_api_key.test", "secret"),
				),
			},
			// Re-apply - secret should still be in state
			{
				Config: testAccApiKeyResourceConfig_basic("test-key-secret"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("komodo_api_key.test", "secret"),
				),
			},
		},
	})
}

func TestAccApiKeyResource_import(t *testing.T) {
	var keyID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create the key and capture its ID
			{
				Config: testAccApiKeyResourceConfig_basic("test-key-import"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("komodo_api_key.test", "name", "test-key-import"),
					resource.TestCheckResourceAttrSet("komodo_api_key.test", "key"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["komodo_api_key.test"]
						if !ok {
							return fmt.Errorf("resource not found")
						}
						keyID = rs.Primary.Attributes["key"]
						return nil
					},
				),
			},
			// Import the same resource by ID
			{
				Config:                            testAccApiKeyResourceConfig_basic("test-key-import"),
				ResourceName:                      "komodo_api_key.test",
				ImportState:                       true,
				ImportStateVerify:                 true,
				ImportStateId:                     keyID,
				ImportStateVerifyIdentifierAttribute: "key",
				// Secret is only available on creation
				ImportStateVerifyIgnore: []string{"secret"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return keyID, nil
				},
			},
		},
	})
}

func TestAccApiKeyResource_differentExpirations(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create keys with different expirations
			{
				Config: testAccApiKeyResourceConfig_differentExpirations(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Key that never expires
					resource.TestCheckResourceAttr("komodo_api_key.never_expires", "expires", "0"),
					// Key with specific expiration
					resource.TestCheckResourceAttr("komodo_api_key.expires_later", "expires", "1893456000000"),
				),
			},
		},
	})
}

// Test configuration functions

func testAccApiKeyResourceConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "komodo_api_key" "test" {
  name = %[1]q
}
`, name)
}

func testAccApiKeyResourceConfig_withExpiration(name string, expires int64) string {
	return fmt.Sprintf(`
resource "komodo_api_key" "test" {
  name    = %[1]q
  expires = %[2]d
}
`, name, expires)
}

func testAccApiKeyResourceConfig_multiple(name1, name2 string) string {
	return fmt.Sprintf(`
resource "komodo_api_key" "test1" {
  name = %[1]q
}

resource "komodo_api_key" "test2" {
  name = %[2]q
}
`, name1, name2)
}

func testAccApiKeyResourceConfig_differentExpirations() string {
	return `
resource "komodo_api_key" "never_expires" {
  name    = "never-expires-key"
  expires = 0
}

resource "komodo_api_key" "expires_later" {
  name    = "expires-later-key"
  expires = 1893456000000
}
`
}
