//go:build integration

package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"testing"
)

func TestIntegration_SourceSchemaResource_AddNew(t *testing.T) {
	checkForPendingCisTask(context.Background())
	sourceCloudId := *getSources(1, "")[0].Id

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerIntegrationConfig + `
resource "identitynow_source_schema" "test" {
  name                = "account"
  source_id           = "` + sourceCloudId + `"
  native_object_type  = "User"
  identity_attribute  = "id"
  display_attribute   = "uid"
  hierarchy_attribute = "uid"
  include_permissions = false
  features = ["AUTHENTICATE"]
  configuration       = jsonencode({
    groupMemberAttribute = "member"
  })
  attributes = [
    {
      name           = "id"
      type           = "STRING"
      description    = "Id of the user"
      is_multi       = false
      is_entitlement = false
      is_group       = false
      // schema is a Reference object and must point on real entity in the system. Check API docs
      //schema         = {
      //  id = "yes"
      //  name = "name"
      //}
    }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("identitynow_source_schema.test", "id"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "name", "account"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "source_id", sourceCloudId),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "native_object_type", "User"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "identity_attribute", "id"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "display_attribute", "uid"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "include_permissions", "false"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "features.0", "AUTHENTICATE"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "configuration", "{\"groupMemberAttribute\":\"member\"}"),

					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.name", "id"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.type", "STRING"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.description", "Id of the user"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.is_multi", "false"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.is_entitlement", "false"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.is_group", "false"),
					//resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.schema.type", "CONNECTOR_SCHEMA"),
					//resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.schema.id", "yes"),
					//resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.schema.name", "name"),
				),
			},
			// Update and Read testing
			{
				Config: providerIntegrationConfig + `
resource "identitynow_source_schema" "test" {
  // name cannot be updated (error 400 - base don docs) -> will re-create object
  name                = "account"
  source_id           = "` + sourceCloudId + `"
  native_object_type  = "User"
  identity_attribute  = "id"
  display_attribute   = "uid"
  hierarchy_attribute = "uid"
  include_permissions = true
  features = ["AUTHENTICATE", "ENABLE"]
  configuration       = jsonencode({
    groupMemberAttribute = "id"
  })
  attributes = [
    {
      // name cannot be updated (error 400 - base don docs)
      name           = "id"
      type           = "STRING"
      description    = "ID of the user Upd"
      is_multi       = false
      is_entitlement = false
      is_group       = false
      //schema         = {
      //  id = "yes"
      //  name = "Id"
      //}
    },
    {
      name           = "name"
      type           = "STRING"
      description    = "Name of the user"
      is_multi       = true
      is_entitlement = false
      is_group       = false
      //schema         = {
      //  id = "no"
      //  name = "name"
      //}
    }
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("identitynow_source_schema.test", "id"),
					resource.TestCheckResourceAttrSet("identitynow_source_schema.test", "id"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "name", "account"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "source_id", sourceCloudId),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "native_object_type", "User"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "identity_attribute", "id"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "display_attribute", "uid"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "include_permissions", "true"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "features.0", "AUTHENTICATE"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "features.1", "ENABLE"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "configuration", "{\"groupMemberAttribute\":\"id\"}"),

					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.name", "id"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.type", "STRING"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.description", "ID of the user Upd"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.is_multi", "false"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.is_entitlement", "false"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.is_group", "false"),
					//resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.schema.type", "CONNECTOR_SCHEMA"),
					//resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.schema.id", "yes"),
					//resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.0.schema.name", "Id"),

					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.1.name", "name"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.1.type", "STRING"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.1.description", "Name of the user"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.1.is_multi", "true"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.1.is_entitlement", "false"),
					resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.1.is_group", "false"),
					//resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.1.schema.type", "CONNECTOR_SCHEMA"),
					//resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.1.schema.id", "no"),
					//resource.TestCheckResourceAttr("identitynow_source_schema.test", "attributes.1.schema.name", "name"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
