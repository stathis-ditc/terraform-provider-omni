// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package omni

import (
	"context"
	"fmt"

	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/siderolabs/omni/client/pkg/omni/resources"
	"github.com/siderolabs/omni/client/pkg/omni/resources/omni"
)

var _ datasource.DataSource = &machinesDataSource{}

func NewMachinesDataSource() datasource.DataSource {
	return &machinesDataSource{}
}

type machinesDataSource struct {
	provider *omniProvider
}

type MachinesDataSourceModel struct {
	Machines []MachineModel `tfsdk:"machines"`
}

type MachineModel struct {
	ID        types.String `tfsdk:"id"`
	Connected types.Bool   `tfsdk:"connected"`
	Cluster   types.String `tfsdk:"cluster"`
}

func (d *machinesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machines"
}

func (d *machinesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List all machines in the Omni cluster",
		Attributes: map[string]schema.Attribute{
			"machines": schema.ListNestedAttribute{
				MarkdownDescription: "List of all machines",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Machine ID",
							Computed:            true,
						},
						"connected": schema.BoolAttribute{
							MarkdownDescription: "Whether the machine is connected",
							Computed:            true,
						},
						"cluster": schema.StringAttribute{
							MarkdownDescription: "The cluster this machine is assigned to",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *machinesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *machinesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MachinesDataSourceModel

	st := d.provider.client.Omni().State()

	machines, err := safe.StateList[*omni.MachineStatus](ctx, st, omni.NewMachineStatus(resources.DefaultNamespace, "").Metadata())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to Get Machines",
			fmt.Sprintf("Failed to get machines from Omni: %s", err),
		)
		return
	}

	var machinesList []MachineModel
	for item := range machines.All() {
		var clusterName string
		if val, ok := item.Metadata().Labels().Get(omni.LabelCluster); ok {
			clusterName = val
		}

		machinesList = append(machinesList, MachineModel{
			ID:        types.StringValue(item.Metadata().ID()),
			Connected: types.BoolValue(item.TypedSpec().Value.GetConnected()),
			Cluster:   types.StringValue(clusterName),
		})
	}

	data.Machines = machinesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
} 