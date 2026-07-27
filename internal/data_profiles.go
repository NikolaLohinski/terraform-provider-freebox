package internal

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var (
	_ datasource.DataSource = &profilesDataSource{}
)

func NewProfilesDataSource() datasource.DataSource {
	return &profilesDataSource{}
}

// profilesDataSource defines the resource implementation.
type profilesDataSource struct {
	client client.Client
}

type ProfilesModel struct {
	Profiles types.List `tfsdk:"profiles"`
}

type ProfileModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Icon types.String `tfsdk:"icon"`
}

func (o ProfileModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.Int64Type,
		"name": types.StringType,
		"icon": types.StringType,
	}
}

func (o ProfileModel) FromClientType(profile freeboxTypes.Profile) basetypes.ObjectValue {
	return basetypes.NewObjectValueMust(o.AttrTypes(), map[string]attr.Value{
		"id":   basetypes.NewInt64Value(profile.ID),
		"name": basetypes.NewStringValue(profile.Name),
		"icon": basetypes.NewStringValue(profile.Icon),
	})
}

func (v *profilesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_profiles"
}

func (v *profilesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Get the list of Freebox profiles (as seen under Freebox OS network control / parental control settings).",
		Attributes: map[string]schema.Attribute{
			"profiles": schema.ListAttribute{
				Computed:            true,
				MarkdownDescription: "List of profiles",
				ElementType: types.ObjectType{
					AttrTypes: ProfileModel{}.AttrTypes(),
				},
			},
		},
	}
}

func (v *profilesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	v.client = client
}

func (v *profilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model ProfilesModel

	if diags := req.Config.Get(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	profiles, err := v.client.ListProfiles(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to list profiles",
			err.Error(),
		)
		return
	}

	elements := make([]attr.Value, len(profiles))
	for i, profile := range profiles {
		elements[i] = ProfileModel{}.FromClientType(profile)
	}

	var diags diag.Diagnostics

	model.Profiles, diags = basetypes.NewListValueFrom(ctx, types.ObjectType{
		AttrTypes: ProfileModel{}.AttrTypes(),
	}, elements)

	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if diags := resp.State.Set(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
}
