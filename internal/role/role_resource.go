package role

import (
	"context"
	"fmt"
	"terraform-provider-identitynow/internal/patch"
	"terraform-provider-identitynow/internal/sailpoint/custom"
	"terraform-provider-identitynow/internal/util"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	sailpoint "github.com/sailpoint-oss/golang-sdk/v2"
	sailpoint_v3 "github.com/sailpoint-oss/golang-sdk/v2/api_v3"
)

var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithConfigure   = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

func NewRoleResource() resource.Resource {
	return &roleResource{}
}

type roleResource struct {
	apiClient *sailpoint.APIClient
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*custom.APIClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sailpoint.APIClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.apiClient = client.ApiClient
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The id of the Role. This field must be left null when creating an Role, otherwise a 400 Bad Request error will result.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The human-readable display name of the Role",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A human-readable description of the Role",
				Optional:    true,
			},
			"owner":           util.ResourceReferenceSchema("IDENTITY", true, "The owner of this object."),
			"access_profiles": util.ResourceReferenceSetSchema("ACCESS_PROFILE", false, "Access Profiles granted by the Role"),
			"entitlements":    util.ResourceReferenceSetSchema("ENTITLEMENT", false, "Entitlements granted by the Role"),
			"membership": schema.SingleNestedAttribute{
				Description: "When present, specifies that the Role is to be granted to Identities which either satisfy specific criteria or which are members of a given list of Identities.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "This enum characterizes the type of a Role's membership selector. " +
							"Only the following two are fully supported:\n" +
							"STANDARD: Indicates that Role membership is defined in terms of a criteria expression\n" +
							"IDENTITY_LIST: Indicates that Role membership is conferred on the specific identities listed",
						Required: true,
						Validators: []validator.String{
							stringvalidator.OneOf("STANDARD", "IDENTITY_LIST"),
						},
					},

					"criteria": schema.SingleNestedAttribute{
						Description: "Defines STANDARD type Role membership",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"operation": schema.StringAttribute{
								Description: "An operation",
								Required:    true,
								Validators: []validator.String{
									stringvalidator.OneOf("EQUALS", "NOT_EQUALS", "CONTAINS", "STARTS_WITH", "ENDS_WITH", "AND", "OR"),
								},
							},
							"key": schema.SingleNestedAttribute{
								Description: "Refers to a specific Identity attribute, Account attribute, or Entitlement used in Role membership criteria",
								Optional:    true,
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Description: "Indicates whether the associated criteria represents an expression " +
											"on identity attributes, account attributes, or entitlements, respectively.",
										Required: true,
										Validators: []validator.String{
											stringvalidator.OneOf("IDENTITY", "ACCOUNT", "ENTITLEMENT"),
										},
									},
									"property": schema.StringAttribute{
										Description: "The name of the attribute or entitlement to which the associated criteria applies.",
										Required:    true,
									},
									"source_id": schema.StringAttribute{
										Description: "ID of the Source from which an account attribute or entitlement is drawn. " +
											"Required if type is ACCOUNT or ENTITLEMENT",
										Optional: true,
									},
								},
							},
							"string_value": schema.StringAttribute{
								Description: "String value to test the Identity attribute, Account attribute, or Entitlement specified " +
									"in the key w/r/t the specified operation. If this criteria is a leaf node, that is, if the operation" +
									" is one of EQUALS, NOT_EQUALS, CONTAINS, STARTS_WITH, or ENDS_WITH, this field is required. " +
									"Otherwise, specifying it is an error.",
								Optional: true,
							},
							"children": schema.ListNestedAttribute{
								Description: "Array of child criteria. Required if the operation is AND or OR, otherwise it must be left null. " +
									"A maximum of three levels of criteria are supported, including leaf nodes. " +
									"Additionally, AND nodes can only be children or OR nodes and vice-versa.",
								Optional: true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"operation": schema.StringAttribute{
											Description: "An operation",
											Required:    true,
											Validators: []validator.String{
												stringvalidator.OneOf("EQUALS", "NOT_EQUALS", "CONTAINS", "STARTS_WITH", "ENDS_WITH", "AND", "OR"),
											},
										},
										"key": schema.SingleNestedAttribute{
											Description: "Refers to a specific Identity attribute, Account attribute, or Entitlement used in Role membership criteria",
											Optional:    true,
											Attributes: map[string]schema.Attribute{
												"type": schema.StringAttribute{
													Description: "Indicates whether the associated criteria represents an expression" +
														" on identity attributes, account attributes, or entitlements, respectively.",
													Required: true,
													Validators: []validator.String{
														stringvalidator.OneOf("IDENTITY", "ACCOUNT", "ENTITLEMENT"),
													},
												},
												"property": schema.StringAttribute{
													Description: "The name of the attribute or entitlement to which the associated criteria applies.",
													Required:    true,
												},
												"source_id": schema.StringAttribute{
													Description: "ID of the Source from which an account attribute or entitlement is drawn." +
														" Required if type is ACCOUNT or ENTITLEMENT",
													Optional: true,
												},
											},
										},
										"string_value": schema.StringAttribute{
											Description: "String value to test the Identity attribute, Account attribute, or Entitlement specified " +
												"in the key w/r/t the specified operation. If this criteria is a leaf node, that is, if the operation" +
												" is one of EQUALS, NOT_EQUALS, CONTAINS, STARTS_WITH, or ENDS_WITH, this field is required. " +
												"Otherwise, specifying it is an error.",
											Optional: true,
										},
										"children": schema.ListNestedAttribute{
											Description: "Array of child criteria. Required if the operation is AND or OR, otherwise it must be left null. " +
												"A maximum of three levels of criteria are supported, including leaf nodes. " +
												"Additionally, AND nodes can only be children or OR nodes and vice-versa.",
											Optional: true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"operation": schema.StringAttribute{
														Description: "An operation",
														Required:    true,
														Validators: []validator.String{
															stringvalidator.OneOf("EQUALS", "NOT_EQUALS", "CONTAINS", "STARTS_WITH", "ENDS_WITH", "AND", "OR"),
														},
													},
													"key": schema.SingleNestedAttribute{
														Description: "Refers to a specific Identity attribute, Account attribute, or Entitlement used in Role membership criteria",
														Optional:    true,
														Attributes: map[string]schema.Attribute{
															"type": schema.StringAttribute{
																Description: "Indicates whether the associated criteria represents an expression" +
																	" on identity attributes, account attributes, or entitlements, respectively.",
																Required: true,
																Validators: []validator.String{
																	stringvalidator.OneOf("IDENTITY", "ACCOUNT", "ENTITLEMENT"),
																},
															},
															"property": schema.StringAttribute{
																Description: "The name of the attribute or entitlement to which the associated criteria applies.",
																Required:    true,
															},
															"source_id": schema.StringAttribute{
																Description: "ID of the Source from which an account attribute or entitlement is drawn." +
																	" Required if type is ACCOUNT or ENTITLEMENT",
																Optional: true,
															},
														},
													},
													"string_value": schema.StringAttribute{
														Description: "String value to test the Identity attribute, Account attribute, or Entitlement specified " +
															"in the key w/r/t the specified operation. If this criteria is a leaf node, that is, if the operation" +
															" is one of EQUALS, NOT_EQUALS, CONTAINS, STARTS_WITH, or ENDS_WITH, this field is required. " +
															"Otherwise, specifying it is an error.",
														Optional: true,
													},
												},
											},
										},
									},
								},
							},
						},
					},
					"identities": util.ResourceReferenceSetSchema("IDENTITY", false, "Defines role membership as being exclusive to the specified Identities, when type is IDENTITY_LIST."),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Indicates whether the Role is enabled or not",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"requestable": schema.BoolAttribute{
				Description: "Indicates whether the Role is enabled or not",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"access_request_config":     r.requestConfigSchema("Access request configuration for this object"),
			"revocation_request_config": r.requestConfigSchema("Revocation request configuration for this object."),
			"segments": schema.SetAttribute{
				Description: "List of IDs of segments, if any, to which this Role is assigned.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *roleResource) requestConfigSchema(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: description,
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"comments_required": schema.BoolAttribute{
				Description: "Whether the requester of the containing object must provide comments justifying the request",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"denial_comments_required": schema.BoolAttribute{
				Description: "Whether an approver must provide comments when denying the request",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"approval_schemas": schema.ListNestedAttribute{
				Description: "List describing the steps in approving the request",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"approver_type": schema.StringAttribute{
							Description: "Describes the individual or group that is responsible for an approval step. Values are as follows.\n" +
								"OWNER: Owner of the associated Role\n" +
								"MANAGER: Manager of the Identity making the request\n" +
								"GOVERNANCE_GROUP: A Governance Group, the ID of which is specified by the approverId field",
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("OWNER", "MANAGER", "GOVERNANCE_GROUP"),
							},
						},
						"approver_id": schema.StringAttribute{
							Description: "Id of the specific approver, used only when approverType is GOVERNANCE_GROUP",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {

	var plan roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role := r.convertToAPIModel(&plan, &resp.Diagnostics, ctx)

	tflog.Info(ctx, fmt.Sprintf("Creating role '%s': %s", plan.Name.ValueString(), util.PrettyPrint(role)))
	roleResp, spResp, err := r.apiClient.V3.RolesAPI.CreateRole(ctx).Role(role).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Role",
			"Could not create Role '"+plan.Name.ValueString()+"': "+err.Error()+"\n"+util.GetBody(spResp),
		)
		return
	}

	r.mapToTerraformModel(&plan, roleResp, &resp.Diagnostics, ctx)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state to fully populated data
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleResp, spResp, err := r.apiClient.V3.RolesAPI.GetRole(ctx, state.Id.ValueString()).Execute()
	if spResp.StatusCode == 404 {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Role",
			"Could not read Role '"+state.Name.ValueString()+"': "+err.Error()+"\n"+util.GetBody(spResp),
		)
		return
	}
	r.mapToTerraformModel(&state, roleResp, &resp.Diagnostics, ctx)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newModel := r.convertToAPIModel(&plan, &resp.Diagnostics, ctx)
	if resp.Diagnostics.HasError() {
		return
	}
	oldModel := r.convertToAPIModel(&state, &resp.Diagnostics, ctx)
	if resp.Diagnostics.HasError() {
		return
	}
	jsonPatch := r.generateJsonPatch(&newModel, &oldModel, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, fmt.Sprintf("Updating role '%s' with json patch: %s", state.Id.ValueString(), util.PrettyPrint(jsonPatch)))
	roleResp, spResp, err := r.apiClient.V3.RolesAPI.PatchRole(ctx, plan.Id.ValueString()).JsonPatchOperation(jsonPatch).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Role",
			"Could not update Role '"+plan.Name.ValueString()+"': "+err.Error()+"\n"+util.GetBody(spResp),
		)
		return
	}

	r.mapToTerraformModel(&plan, roleResp, &resp.Diagnostics, ctx)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	spResp, err := r.apiClient.V3.RolesAPI.DeleteRole(ctx, state.Id.ValueString()).Execute()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Role",
			"Could not delete Role '"+state.Name.ValueString()+"': "+err.Error()+"\n"+util.GetBody(spResp),
		)
		return
	}
}

func (r *roleResource) convertToAPIModel(model *roleModel, diagnostics *diag.Diagnostics, ctx context.Context) sailpoint_v3.Role {
	owner := sailpoint_v3.OwnerReference{
		Type: util.GetTFStringPointer(model.Owner.Type),
		Id:   util.GetTFStringPointer(model.Owner.Id),
		Name: util.GetTFStringPointer(model.Owner.Name),
	}
	accessProfiles := make([]sailpoint_v3.AccessProfileRef, len(model.AccessProfiles.Elements()))
	for i, element := range model.AccessProfiles.Elements() {
		accessProfile := util.ReferenceModel{}
		diagnostics.Append(tfsdk.ValueAs(ctx, element, &accessProfile)...)
		if diagnostics.HasError() {
			return sailpoint_v3.Role{}
		}
		accessProfiles[i] = sailpoint_v3.AccessProfileRef{
			Type: util.GetTFStringPointer(accessProfile.Type),
			Id:   util.GetTFStringPointer(accessProfile.Id),
			Name: util.GetTFStringPointer(accessProfile.Name),
		}
	}
	entitlements := make([]sailpoint_v3.EntitlementRef, len(model.Entitlements.Elements()))
	for i, element := range model.Entitlements.Elements() {
		entitlement := util.ReferenceModel{}
		diagnostics.Append(tfsdk.ValueAs(ctx, element, &entitlement)...)
		if diagnostics.HasError() {
			return sailpoint_v3.Role{}
		}
		entitlements[i] = sailpoint_v3.EntitlementRef{
			Type: util.GetTFStringPointer(entitlement.Type),
			Id:   util.GetTFStringPointer(entitlement.Id),
			Name: *sailpoint_v3.NewNullableString(util.GetTFStringPointer(entitlement.Name)),
		}
	}

	if diagnostics.HasError() {
		return sailpoint_v3.Role{}
	}
	accessRequestConfig := r.convertToRequestabilityForRole(model.AccessRequestConfig)
	revocationRequestConfig := r.convertToRevocabilityForRole(model.RevocationRequestConfig)
	var segments []string = nil
	if !model.Segments.IsNull() && !model.Segments.IsUnknown() {
		segments = make([]string, len(model.Segments.Elements()))
		for i, segment := range model.Segments.Elements() {
			segments[i] = segment.(basetypes.StringValue).ValueString()
		}
	}

	returnRole := sailpoint_v3.Role{
		Name:                    model.Name.ValueString(),
		Description:             *sailpoint_v3.NewNullableString(util.GetTFStringPointer(model.Description)),
		Owner:                   owner,
		AccessProfiles:          accessProfiles,
		Entitlements:            entitlements,
		Enabled:                 model.Enabled.ValueBoolPointer(),
		Requestable:             model.Requestable.ValueBoolPointer(),
		AccessRequestConfig:     accessRequestConfig,
		RevocationRequestConfig: revocationRequestConfig,
		Segments:                segments,
	}

	if model.Membership != nil && !model.Membership.Type.IsNull() {
		membership := r.convertToMembership(ctx, model.Membership, diagnostics)
		returnRole.Membership = *sailpoint_v3.NewNullableRoleMembershipSelector(membership)
	}

	return returnRole
}

func (r *roleResource) getMembershipCriteriaLvl1(ctx context.Context, rootNode roleMembershipCriteriaLvl1) *sailpoint_v3.RoleCriteriaLevel1 {
	// Operation
	criteriaOperation, err := sailpoint_v3.NewRoleCriteriaOperationFromValue(rootNode.Operation.ValueString())
	if err != nil {
		tflog.Error(ctx, "Invalid Role Membership Type: "+err.Error())
		return &sailpoint_v3.RoleCriteriaLevel1{}
	}

	// Key
	var criteriaKey *sailpoint_v3.RoleCriteriaKey = nil
	if rootNode.Key != nil {
		criteriaKey = sailpoint_v3.NewRoleCriteriaKey(
			sailpoint_v3.RoleCriteriaKeyType(rootNode.Key.Type.ValueString()),
			rootNode.Key.Property.ValueString(),
		)
	}

	// Children
	var children []sailpoint_v3.RoleCriteriaLevel2 = nil
	if rootNode.Children != nil {
		children = r.getMembershipCriteriaChildrenLvl2(rootNode)
	}

	return &sailpoint_v3.RoleCriteriaLevel1{
		Operation:   criteriaOperation,
		Key:         *sailpoint_v3.NewNullableRoleCriteriaKey(criteriaKey),
		StringValue: *sailpoint_v3.NewNullableString(util.GetTFStringPointer(rootNode.StringValue)),
		Children:    children,
	}
}

func (r *roleResource) getMembershipCriteriaChildrenLvl2(rootNode roleMembershipCriteriaLvl1) []sailpoint_v3.RoleCriteriaLevel2 {
	childrenLvl2 := make([]sailpoint_v3.RoleCriteriaLevel2, len(rootNode.Children))
	for i, lvl2Child := range rootNode.Children {
		childrenLvl2[i] = r.getMembershipCriteriaLvl2(lvl2Child)
	}
	return childrenLvl2
}

func (r *roleResource) getMembershipCriteriaLvl2(rootNode roleMembershipCriteriaLvl2) sailpoint_v3.RoleCriteriaLevel2 {
	// Operation
	criteriaOperation, _ := sailpoint_v3.NewRoleCriteriaOperationFromValue(rootNode.Operation.ValueString())

	// Key
	var criteriaKey *sailpoint_v3.RoleCriteriaKey = nil
	if rootNode.Key != nil {
		criteriaKey = sailpoint_v3.NewRoleCriteriaKey(
			sailpoint_v3.RoleCriteriaKeyType(rootNode.Key.Type.ValueString()),
			rootNode.Key.Property.ValueString(),
		)
	}

	// Children
	var children []sailpoint_v3.RoleCriteriaLevel3 = nil
	if rootNode.Children != nil {
		children = r.getMembershipCriteriaChildrenLvl3(rootNode)
	}

	return sailpoint_v3.RoleCriteriaLevel2{
		Operation:   criteriaOperation,
		Key:         *sailpoint_v3.NewNullableRoleCriteriaKey(criteriaKey),
		StringValue: *sailpoint_v3.NewNullableString(util.GetTFStringPointer(rootNode.StringValue)),
		Children:    children,
	}
}

func (r *roleResource) getMembershipCriteriaChildrenLvl3(rootNode roleMembershipCriteriaLvl2) []sailpoint_v3.RoleCriteriaLevel3 {
	childrenLvl3 := make([]sailpoint_v3.RoleCriteriaLevel3, len(rootNode.Children))
	for i, lvl3Child := range rootNode.Children {
		childrenLvl3[i] = r.getMembershipCriteriaLvl3(lvl3Child)
	}
	return childrenLvl3
}

func (r *roleResource) getMembershipCriteriaLvl3(rootNode roleMembershipCriteriaLvl3) sailpoint_v3.RoleCriteriaLevel3 {
	// Operation
	criteriaOperation, _ := sailpoint_v3.NewRoleCriteriaOperationFromValue(rootNode.Operation.ValueString())

	// Key
	var criteriaKey *sailpoint_v3.RoleCriteriaKey = nil
	if rootNode.Key != nil {
		criteriaKey = sailpoint_v3.NewRoleCriteriaKey(
			sailpoint_v3.RoleCriteriaKeyType(rootNode.Key.Type.ValueString()),
			rootNode.Key.Property.ValueString(),
		)
	}

	return sailpoint_v3.RoleCriteriaLevel3{
		Operation:   criteriaOperation,
		Key:         *sailpoint_v3.NewNullableRoleCriteriaKey(criteriaKey),
		StringValue: util.GetTFStringPointer(rootNode.StringValue),
	}
}

func (r *roleResource) convertToMembership(ctx context.Context, mMembership *roleMembership, diagnostics *diag.Diagnostics) *sailpoint_v3.RoleMembershipSelector {
	if mMembership == nil {
		return nil
	}
	memType, err := sailpoint_v3.NewRoleMembershipSelectorTypeFromValue(mMembership.Type.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid Role Membership Type", err.Error())
		return nil
	}
	var criteria *sailpoint_v3.RoleCriteriaLevel1 = nil
	if mCriteria := mMembership.Criteria; mCriteria != nil {
		criteria = r.getMembershipCriteriaLvl1(ctx, *mCriteria)
	}

	identities := make([]sailpoint_v3.RoleMembershipIdentity, len(mMembership.Identities))
	for i, identity := range mMembership.Identities {
		dtoType, err := sailpoint_v3.NewDtoTypeFromValue(identity.Type.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid Role Membership Identity Type", err.Error())
			return nil
		}
		identities[i] = sailpoint_v3.RoleMembershipIdentity{
			Type: dtoType,
			Id:   util.GetTFStringPointer(identity.Id),
			Name: *sailpoint_v3.NewNullableString(util.GetTFStringPointer(identity.Name)),
		}
	}

	return &sailpoint_v3.RoleMembershipSelector{
		Type:       memType,
		Criteria:   *sailpoint_v3.NewNullableRoleCriteriaLevel1(criteria),
		Identities: identities,
	}
}

func (r *roleResource) convertToRequestabilityForRole(reqConfig *requestConfig) *sailpoint_v3.RequestabilityForRole {
	if reqConfig == nil {
		return &sailpoint_v3.RequestabilityForRole{
			CommentsRequired:       *sailpoint_v3.NewNullableBool(sailpoint_v3.PtrBool(false)),
			DenialCommentsRequired: *sailpoint_v3.NewNullableBool(sailpoint_v3.PtrBool(false)),
		}
	}
	appSchemas := make([]sailpoint_v3.ApprovalSchemeForRole, len(reqConfig.ApprovalSchemas))
	for i, appSchema := range reqConfig.ApprovalSchemas {
		appSchemas[i] = sailpoint_v3.ApprovalSchemeForRole{
			ApproverType: util.GetTFStringPointer(appSchema.ApproverType),
			ApproverId:   *sailpoint_v3.NewNullableString(util.GetTFStringPointer(appSchema.ApproverId)),
		}
	}
	return &sailpoint_v3.RequestabilityForRole{
		CommentsRequired:       *sailpoint_v3.NewNullableBool(reqConfig.CommentsRequired.ValueBoolPointer()),
		DenialCommentsRequired: *sailpoint_v3.NewNullableBool(reqConfig.DenialCommentsRequired.ValueBoolPointer()),
		ApprovalSchemes:        appSchemas,
	}
}

func (r *roleResource) convertToRevocabilityForRole(reqConfig *requestConfig) *sailpoint_v3.RevocabilityForRole {
	if reqConfig == nil {
		return &sailpoint_v3.RevocabilityForRole{}
	}
	appSchemas := make([]sailpoint_v3.ApprovalSchemeForRole, len(reqConfig.ApprovalSchemas))
	for i, appSchema := range reqConfig.ApprovalSchemas {
		appSchemas[i] = sailpoint_v3.ApprovalSchemeForRole{
			ApproverType: util.GetTFStringPointer(appSchema.ApproverType),
			ApproverId:   *sailpoint_v3.NewNullableString(util.GetTFStringPointer(appSchema.ApproverId)),
		}
	}
	return &sailpoint_v3.RevocabilityForRole{
		CommentsRequired:       *sailpoint_v3.NewNullableBool(reqConfig.CommentsRequired.ValueBoolPointer()),
		DenialCommentsRequired: *sailpoint_v3.NewNullableBool(reqConfig.DenialCommentsRequired.ValueBoolPointer()),
		ApprovalSchemes:        appSchemas,
	}
}

func (r *roleResource) mapToTerraformModel(model *roleModel, role *sailpoint_v3.Role, diagnostics *diag.Diagnostics, ctx context.Context) {
	model.Id = types.StringPointerValue(role.Id)
	model.Name = types.StringValue(role.Name)
	model.Description = types.StringPointerValue(role.Description.Get())
	model.Owner = *util.NewPointerReferenceModel(role.Owner.Type, role.Owner.Id, role.Owner.Name)
	if len(role.AccessProfiles) > 0 {
		accessProfiles := make([]util.ReferenceModel, len(role.AccessProfiles))
		for i, accessProfile := range role.AccessProfiles {
			accessProfiles[i] = *util.NewPointerReferenceModel(accessProfile.Type, accessProfile.Id, accessProfile.Name)
		}
		diagnostics.Append(util.ConvertReferenceModelToMap(ctx, accessProfiles, &model.AccessProfiles)...)
		if diagnostics.HasError() {
			return
		}
	}
	if len(role.Entitlements) > 0 {
		entitlements := make([]util.ReferenceModel, len(role.Entitlements))
		for i, entitlement := range role.Entitlements {
			entitlements[i] = *util.NewPointerReferenceModel(entitlement.Type, entitlement.Id, entitlement.Name.Get())
		}
		diagnostics.Append(util.ConvertReferenceModelToMap(ctx, entitlements, &model.Entitlements)...)
		if diagnostics.HasError() {
			return
		}
	}

	model.Membership = r.mapToMembership(role.Membership.Get())
	model.Enabled = types.BoolPointerValue(role.Enabled)
	model.Requestable = types.BoolPointerValue(role.Requestable)
	model.AccessRequestConfig = r.mapToAccessRequestConfig(role.AccessRequestConfig)
	model.RevocationRequestConfig = r.mapToRevocationRequestConfig(role.RevocationRequestConfig)
	if len(role.Segments) > 0 {
		segments := make([]attr.Value, len(role.Segments))
		for i, segment := range role.Segments {
			segments[i] = types.StringValue(segment)
		}
		model.Segments = types.SetValueMust(types.StringType, segments)
	} else {
		if role.Segments == nil {
			model.Segments = types.SetNull(types.StringType)
		} else {
			segments := make([]attr.Value, len(role.Segments))
			model.Segments = types.SetValueMust(types.StringType, segments)
		}
	}
}

func (r *roleResource) mapToAccessRequestConfig(config *sailpoint_v3.RequestabilityForRole) *requestConfig {
	// Ensure config is never nil
	if config == nil {
		return &requestConfig{
			CommentsRequired:       types.BoolPointerValue(nil), // Allow Terraform to treat it as unknown
			DenialCommentsRequired: types.BoolPointerValue(nil),
			ApprovalSchemas:        []approvalSchemas{}, // Always return an empty slice, not nil
		}
	}

	// Avoid nil pointer dereference when checking booleans
	commentsRequired := config.CommentsRequired.Get()
	denialCommentsRequired := config.DenialCommentsRequired.Get()

	if (commentsRequired == nil || *commentsRequired == false) &&
		(denialCommentsRequired == nil || *denialCommentsRequired == false) &&
		len(config.ApprovalSchemes) == 0 {
		return &requestConfig{
			CommentsRequired:       types.BoolPointerValue(nil), // Ensure empty object instead of null
			DenialCommentsRequired: types.BoolPointerValue(nil),
			ApprovalSchemas:        []approvalSchemas{}, // Return empty slice instead of nil
		}
	}

	// Ensure ApprovalSchemas is always a valid slice
	schemas := make([]approvalSchemas, len(config.ApprovalSchemes))
	for i, appSchema := range config.ApprovalSchemes {
		schemas[i] = approvalSchemas{
			ApproverType: types.StringPointerValue(appSchema.ApproverType),
			ApproverId:   types.StringPointerValue(appSchema.ApproverId.Get()),
		}
	}

	return &requestConfig{
		CommentsRequired:       types.BoolPointerValue(commentsRequired),
		DenialCommentsRequired: types.BoolPointerValue(denialCommentsRequired),
		ApprovalSchemas:        schemas,
	}
}

func (r *roleResource) mapToRevocationRequestConfig(config *sailpoint_v3.RevocabilityForRole) *requestConfig {
	if config == nil {
		return &requestConfig{
			CommentsRequired:       types.BoolValue(false),
			DenialCommentsRequired: types.BoolValue(false),
			ApprovalSchemas:        []approvalSchemas{}, // Empty slice, not nil
		}
	}

	// Ensure approval schemas is always an empty slice if no data is present
	schemas := make([]approvalSchemas, len(config.ApprovalSchemes))
	for i, appSchema := range config.ApprovalSchemes {
		schemas[i] = approvalSchemas{
			ApproverType: types.StringPointerValue(appSchema.ApproverType),
			ApproverId:   types.StringPointerValue(appSchema.ApproverId.Get()),
		}
	}

	return &requestConfig{
		CommentsRequired:       types.BoolPointerValue(config.CommentsRequired.Get()),
		DenialCommentsRequired: types.BoolPointerValue(config.DenialCommentsRequired.Get()),
		ApprovalSchemas:        schemas,
	}
}

func (r *roleResource) mapToMembershipCriteriaLvl1(rootNode sailpoint_v3.RoleCriteriaLevel1) *roleMembershipCriteriaLvl1 {
	// Operation
	criteriaOperation := types.StringValue(string(*rootNode.Operation))

	// Key
	var criteriaKey *roleCriteriaKey = nil
	if rootNode.Key.Get() != nil {
		criteriaKey = &roleCriteriaKey{
			Type:     types.StringValue(string(rootNode.Key.Get().Type)),
			Property: types.StringValue(rootNode.Key.Get().Property),
			SourceId: types.StringPointerValue(rootNode.Key.Get().SourceId.Get()),
		}
	}

	// Children
	var children []roleMembershipCriteriaLvl2 = nil
	if rootNode.Children != nil {
		children = r.mapToMembershipCriteriaChildrenLvl2(rootNode)
	}

	return &roleMembershipCriteriaLvl1{
		Operation:   criteriaOperation,
		Key:         criteriaKey,
		StringValue: types.StringPointerValue(rootNode.StringValue.Get()),
		Children:    children,
	}
}

func (r *roleResource) mapToMembershipCriteriaChildrenLvl2(rootNode sailpoint_v3.RoleCriteriaLevel1) []roleMembershipCriteriaLvl2 {
	childrenLvl2 := make([]roleMembershipCriteriaLvl2, len(rootNode.Children))
	for i, lvl2Child := range rootNode.Children {
		childrenLvl2[i] = r.mapToMembershipCriteriaLvl2(lvl2Child)
	}
	return childrenLvl2
}

func (r *roleResource) mapToMembershipCriteriaLvl2(rootNode sailpoint_v3.RoleCriteriaLevel2) roleMembershipCriteriaLvl2 {
	// Operation
	criteriaOperation := types.StringValue(string(*rootNode.Operation))

	// Key
	var criteriaKey *roleCriteriaKey = nil
	if rootNode.Key.Get() != nil {
		criteriaKey = &roleCriteriaKey{
			Type:     types.StringValue(string(rootNode.Key.Get().Type)),
			Property: types.StringValue(rootNode.Key.Get().Property),
			SourceId: types.StringPointerValue(rootNode.Key.Get().SourceId.Get()),
		}
	}

	// Children
	var children []roleMembershipCriteriaLvl3 = nil
	if rootNode.Children != nil {
		children = r.mapToMembershipCriteriaChildrenLvl3(rootNode)
	}

	return roleMembershipCriteriaLvl2{
		Operation:   criteriaOperation,
		Key:         criteriaKey,
		StringValue: types.StringPointerValue(rootNode.StringValue.Get()),
		Children:    children,
	}
}

func (r *roleResource) mapToMembershipCriteriaChildrenLvl3(rootNode sailpoint_v3.RoleCriteriaLevel2) []roleMembershipCriteriaLvl3 {
	childrenLvl3 := make([]roleMembershipCriteriaLvl3, len(rootNode.Children))
	for i, lvl3Child := range rootNode.Children {
		childrenLvl3[i] = r.mapToMembershipCriteriaLvl3(lvl3Child)
	}
	return childrenLvl3
}

func (r *roleResource) mapToMembershipCriteriaLvl3(rootNode sailpoint_v3.RoleCriteriaLevel3) roleMembershipCriteriaLvl3 {
	// Operation
	criteriaOperation := types.StringValue(string(*rootNode.Operation))

	// Key
	var criteriaKey *roleCriteriaKey = nil
	if rootNode.Key.Get() != nil {
		criteriaKey = &roleCriteriaKey{
			Type:     types.StringValue(string(rootNode.Key.Get().Type)),
			Property: types.StringValue(rootNode.Key.Get().Property),
			SourceId: types.StringPointerValue(rootNode.Key.Get().SourceId.Get()),
		}
	}

	return roleMembershipCriteriaLvl3{
		Operation:   criteriaOperation,
		Key:         criteriaKey,
		StringValue: types.StringValue(*rootNode.StringValue),
	}
}

func (r *roleResource) mapToMembership(membership *sailpoint_v3.RoleMembershipSelector) *roleMembership {
	if membership == nil {
		return nil
	}
	// Criteria Type
	criteriaType := types.StringValue(string(*membership.Type))

	// Criteria
	var criteria *roleMembershipCriteriaLvl1 = nil
	if membership.Criteria.Get() != nil {
		membershipCriteria := membership.Criteria.Get()
		criteria = r.mapToMembershipCriteriaLvl1(*membershipCriteria)
	}

	// Identities
	var identities []util.ReferenceModel = nil
	if len(membership.Identities) > 0 {
		identities = make([]util.ReferenceModel, len(membership.Identities))
		for i, identity := range membership.Identities {
			identities[i] = util.ReferenceModel{
				Type: types.StringValue(string(*identity.Type)),
				Id:   types.StringPointerValue(identity.Id),
				Name: types.StringPointerValue(identity.Name.Get()),
			}
		}
	}

	return &roleMembership{
		Type:       criteriaType,
		Criteria:   criteria,
		Identities: identities,
	}
}

func (r *roleResource) generateJsonPatch(newModel *sailpoint_v3.Role, oldModel *sailpoint_v3.Role, diagnostics *diag.Diagnostics) []sailpoint_v3.JsonPatchOperation {
	jsonPatch, err := patch.NewRolePatchBuilder(newModel, oldModel).GenerateJsonPatch()
	if err != nil {
		diagnostics.AddError(
			"Error Generating Update Patch",
			"Could not generate update patch for Role '"+oldModel.Name+"': "+err.Error(),
		)
		return nil
	}
	v3JsonPatch, err := patch.ConvertPatchOperationFromBetaToV3(jsonPatch)
	if err != nil {
		diagnostics.AddError(
			"Error Generating Update Patch",
			"Could not convert patch to V3 for Role '"+oldModel.Name+"': "+err.Error(),
		)
		return nil
	}
	return v3JsonPatch
}

func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	// Check for errors from the passthrough function
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch nested attributes from the state
	var ownerId, ownerName, ownerType string

	// Read the nested attributes safely
	resp.State.GetAttribute(ctx, path.Root("owner").AtName("id"), &ownerId)

	resp.State.GetAttribute(ctx, path.Root("owner").AtName("name"), &ownerName)

	resp.State.GetAttribute(ctx, path.Root("owner").AtName("type"), &ownerType)

	// Create a new ReferenceModel for the owner
	updatedOwner := util.ReferenceModel{
		Type: types.StringValue(ownerType),
		Id:   types.StringValue(ownerId),
		Name: types.StringValue(ownerName),
	}

	// Set the updated owner back into the state
	err := resp.State.SetAttribute(ctx, path.Root("owner"), updatedOwner)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting updated owner",
			fmt.Sprintf("Failed to set updated owner attribute: %s", err),
		)
	}
}
