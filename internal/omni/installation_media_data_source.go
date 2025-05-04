// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package omni

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/siderolabs/omni/client/api/omni/management"
	"github.com/siderolabs/omni/client/pkg/omni/resources"
	"github.com/siderolabs/omni/client/pkg/omni/resources/omni"
)

var _ datasource.DataSource = &installationMediaDataSource{}

func NewInstallationMediaDataSource() datasource.DataSource {
	return &installationMediaDataSource{}
}

type installationMediaDataSource struct {
	provider *omniProvider
}

type installationMediaDataSourceModel struct {
	ID                   types.String   `tfsdk:"id"`
	Arch                 types.String   `tfsdk:"arch"`
	TalosVersion         types.String   `tfsdk:"talos_version"`
	ImageName            types.String   `tfsdk:"image_name"`
	Extensions           []types.String `tfsdk:"extensions"`
	ExtraKernelArgs      []types.String `tfsdk:"extra_kernel_args"`
	SecureBoot           types.Bool     `tfsdk:"secure_boot"`
	SiderolinkGRPCTunnel types.Bool     `tfsdk:"use_siderolink_grpc_tunnel"`

	Schematic types.String `tfsdk:"schematic"`
	PXEUrl    types.String `tfsdk:"pxe_url"`
}

func (d *installationMediaDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_installation_media"
}

func (d *installationMediaDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Generate the schematic and pxe url of the installation media",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID",
				Computed:            true,
			},
			"arch": schema.StringAttribute{
				MarkdownDescription: "Architecture ('amd64' or 'arm64')",
				Optional:            true,
			},
			"extensions": schema.ListAttribute{
				MarkdownDescription: "Extensions",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"extra_kernel_args": schema.ListAttribute{
				MarkdownDescription: "Extra kernel arguments",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"image_name": schema.StringAttribute{
				MarkdownDescription: "Image name",
				Required:            true,
			},
			"secure_boot": schema.BoolAttribute{
				MarkdownDescription: "Enable secure boot",
				Optional:            true,
			},
			"talos_version": schema.StringAttribute{
				MarkdownDescription: "Talos version",
				Required:            true,
			},
			"use_siderolink_grpc_tunnel": schema.BoolAttribute{
				MarkdownDescription: "Use Siderolink GRPC tunnel",
				Optional:            true,
			},
			"schematic": schema.StringAttribute{
				MarkdownDescription: "Generated schematic",
				Computed:            true,
			},
			"pxe_url": schema.StringAttribute{
				MarkdownDescription: "Generated pxe url",
				Computed:            true,
			},
		},
	}
}

func (d *installationMediaDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	provider, ok := req.ProviderData.(*omniProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *omniProvider, got: %T", req.ProviderData),
		)
		return
	}

	d.provider = provider
}

func (d *installationMediaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data installationMediaDataSourceModel

	// Get the attribute values and assign them to the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	// Setup default values if they are not set
	if data.Arch.IsNull() || data.Arch.ValueString() == "" {
		data.Arch = types.StringValue("amd64")
	}

	st := d.provider.client.Omni().State()

	options := []state.ListOption{
		state.WithIDQuery(resource.IDRegexpMatch(regexp.MustCompile(buildRegex(data)))),
	}

	image, err := safe.StateList[*omni.InstallationMedia](ctx, st, omni.NewInstallationMedia(resources.EphemeralNamespace, "").Metadata(), options...)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Get Image",
			fmt.Sprintf("Failed to get Image from Omni: %s", err),
		)
		return
	}

	if image.Len() == 0 {
		resp.Diagnostics.AddError(
			"No image found",
			fmt.Sprintf("No image found with the criteria given: %s", err),
		)
		return
	}

	extraKernelArgs := make([]string, 0, len(data.ExtraKernelArgs))
	for _, v := range data.ExtraKernelArgs {
		extraKernelArgs = append(extraKernelArgs, v.ValueString())
	}

	extentions := make([]string, 0, len(data.Extensions))
	for _, v := range data.Extensions {
		extentions = append(extentions, v.ValueString())
	}

	grpcTunnel := management.CreateSchematicRequest_AUTO

	if !data.SiderolinkGRPCTunnel.IsNull() {
		switch data.SiderolinkGRPCTunnel.ValueBool() {
		case true:
			grpcTunnel = management.CreateSchematicRequest_ENABLED
		case false:
			grpcTunnel = management.CreateSchematicRequest_DISABLED
		}
	}

	schematic, err := d.provider.client.Management().CreateSchematic(ctx, &management.CreateSchematicRequest{
		MetaValues:               map[uint32]string{},
		ExtraKernelArgs:          extraKernelArgs,
		Extensions:               extentions,
		MediaId:                  image.Get(0).Metadata().ID(),
		SecureBoot:               data.SecureBoot.ValueBool(),
		TalosVersion:             data.TalosVersion.ValueString(),
		SiderolinkGrpcTunnelMode: grpcTunnel,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Scematic generation failed",
			fmt.Sprintf("Schematic generation failed: %s", err),
		)
		return
	}

	data.Schematic = types.StringValue(schematic.SchematicId)
	data.PXEUrl = types.StringValue(schematic.PxeUrl)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Helper function to create a regex that is used to filter for the image
func buildRegex(data installationMediaDataSourceModel) string {
	var regexBuilder strings.Builder

	if data.ImageName.ValueString() == "iso" {
		regexBuilder.WriteString(".*")
		regexBuilder.WriteString(strings.ToLower(data.Arch.ValueString()))
		regexBuilder.WriteString("\\.iso$")
		return regexBuilder.String()
	}

	regexBuilder.WriteString(".*")
	regexBuilder.WriteString(strings.ToLower(data.ImageName.ValueString()))
	regexBuilder.WriteString(".*")
	regexBuilder.WriteString(strings.ToLower(data.Arch.ValueString()))
	regexBuilder.WriteString(".*")

	return regexBuilder.String()
}
