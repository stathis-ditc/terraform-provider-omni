// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package omni

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/siderolabs/omni/client/pkg/client"
	"github.com/siderolabs/omni/client/pkg/version"
)

// Ensure omniProvider satisfies the provider.Provider interface.
var _ provider.Provider = &omniProvider{}

type omniProvider struct {
	client *client.Client
}

type omniProviderModel struct {
	Endpoint          types.String `tfsdk:"endpoint"`
	ServiceAccountKey types.String `tfsdk:"service_account_key"`
}

func (p *omniProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "omni"
}

func (p *omniProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The Omni's API endpoint",
				Required:            true,
			},
			"service_account_key": schema.StringAttribute{
				MarkdownDescription: "The generated base64 key of the service account created in Omni",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *omniProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	version.Name = "terraform-provider-omni"
	version.SHA = "build SHA"
	version.Tag = "v0.1.0"

	var config omniProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validations
	if config.Endpoint.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Endpoint Configuration",
			"The endpoint must be set for Omni provider",
		)
		return
	}

	if config.ServiceAccountKey.IsNull() {
		resp.Diagnostics.AddError(
			"Missing Service Account Key Configuration",
			"The service_account_key must be set for Omni provider",
		)
		return
	}

	// Create Omni client
	omniClient, err := client.New(
		config.Endpoint.ValueString(),
		client.WithServiceAccount(config.ServiceAccountKey.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Create Omni Client",
			fmt.Sprintf("Failed to create Omni client: %s", err),
		)
		return
	}

	p.client = omniClient

	// Make the client available to resources and data sources
	resp.ResourceData = p
	resp.DataSourceData = p
}

func (p *omniProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *omniProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *omniProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewMachinesDataSource,
		NewInstallationMediaDataSource,
	}
}

func (p *omniProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func (p *omniProvider) Shutdown(ctx context.Context) error {
	if p.client != nil {
		return p.client.Close()
	}

	return nil
}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &omniProvider{}
	}
}
