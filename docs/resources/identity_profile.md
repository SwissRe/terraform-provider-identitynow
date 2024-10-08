---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "identitynow_identity_profile Resource - terraform-provider-identitynow"
subcategory: ""
description: |-
  
---

# identitynow_identity_profile (Resource)



## Example Usage

```terraform
resource "identitynow_identity_profile" "demo_identity_profile" {
  name        = "DEMO Terraform Identity Profile"
  description = "DEMO Terraform Identity Profile"
  authoritative_source = {
    id = identitynow_source.demo_source.id
  }
  owner = {
    id   = data.identitynow_identity.default_owner.id
    name = data.identitynow_identity.default_owner.name
  }
  identity_attribute_config = {
    enabled = false
    attribute_transforms = [
      {
        identity_attribute_name = "uid"
        transform_definition = {
          type = "accountAttribute"
          attributes = jsonencode({
            sourceName    = identitynow_source.demo_source.name
            attributeName = "uid"
            sourceId      = identitynow_source.demo_source.id
          })
        }
      },
      {
        identity_attribute_name = "email"
        transform_definition = {
          type = "accountAttribute"
          attributes = jsonencode({
            sourceName    = identitynow_source.demo_source.name
            attributeName = "uid"
            sourceId      = identitynow_source.demo_source.id
          })
        }
      },

      {
        identity_attribute_name = "lastname"
        transform_definition = {
          type = "accountAttribute"
          attributes = jsonencode({
            sourceName    = identitynow_source.demo_source.name
            attributeName = "uid"
            sourceId      = identitynow_source.demo_source.id
          })
        }
      },
      {
        identity_attribute_name = "firstname"
        transform_definition = {
          type = "reference"
          attributes = jsonencode({
            input = {
              attributes = {
                attributeName = "firstName"
                sourceName    = identitynow_source.demo_source.name
                sourceId      = identitynow_source.demo_source.id
              }
              type = "accountAttribute"
            }
            id = identitynow_transform.demo_transform.name
          })
        }
      }
    ]
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `authoritative_source` (Attributes) The authoritative source for this Identity Profile (see [below for nested schema](#nestedatt--authoritative_source))
- `name` (String)

### Optional

- `description` (String) The description of the Identity Profile
- `identity_attribute_config` (Attributes) (see [below for nested schema](#nestedatt--identity_attribute_config))
- `identity_exception_report_reference` (Attributes) (see [below for nested schema](#nestedatt--identity_exception_report_reference))
- `owner` (Attributes) The owner of the Identity Profile (see [below for nested schema](#nestedatt--owner))
- `priority` (Number) The priority for an Identity Profile

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedatt--authoritative_source"></a>
### Nested Schema for `authoritative_source`

Optional:

- `id` (String)
- `name` (String)
- `type` (String)


<a id="nestedatt--identity_attribute_config"></a>
### Nested Schema for `identity_attribute_config`

Optional:

- `attribute_transforms` (Attributes List) (see [below for nested schema](#nestedatt--identity_attribute_config--attribute_transforms))
- `enabled` (Boolean) If the profile or mapping is enabled

<a id="nestedatt--identity_attribute_config--attribute_transforms"></a>
### Nested Schema for `identity_attribute_config.attribute_transforms`

Required:

- `identity_attribute_name` (String) Name of the identity attribute

Optional:

- `transform_definition` (Attributes) The seaspray transformation definition (see [below for nested schema](#nestedatt--identity_attribute_config--attribute_transforms--transform_definition))

<a id="nestedatt--identity_attribute_config--attribute_transforms--transform_definition"></a>
### Nested Schema for `identity_attribute_config.attribute_transforms.transform_definition`

Required:

- `type` (String) The type of the transform definition

Optional:

- `attributes` (String) Arbitrary key-value pairs to store any metadata for the object




<a id="nestedatt--identity_exception_report_reference"></a>
### Nested Schema for `identity_exception_report_reference`

Optional:

- `report_name` (String) The name of the report

Read-Only:

- `task_result_id` (String) The id of the task result


<a id="nestedatt--owner"></a>
### Nested Schema for `owner`

Optional:

- `id` (String)
- `name` (String)
- `type` (String)
