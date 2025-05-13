// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package omni

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"slices"

	cosi_res "github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/protobuf"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

const (
	// Error messages
	errUnexpectedProviderType = "Unexpected Data Source Configure Type"
	errCreationFailed         = "Creation Error"
	errUpdateFailed           = "Update Error"
	errDeleteFailed           = "Delete Error"
	errStateError             = "State Error"
	errContextError           = "Context Error"
	errYAMLDecodingError      = "YAML Decoding Error"
)

var _ resource.Resource = &applyYamlResource{}

func NewApplyYamlResource() resource.Resource {
	return &applyYamlResource{}
}

type applyYamlResource struct {
	provider *omniProvider
}

type applyYamlResourceModel struct {
	ID   types.String `tfsdk:"id"`
	Yaml types.String `tfsdk:"yaml"`
}

// ResourceOperationMode defines how the resource operation should behave
type ResourceOperationMode int

const (
	// ModeCreateOnly only allows creation of new resources
	ModeCreateOnly ResourceOperationMode = iota
	// ModeCreateOrUpdate allows both creation and updates
	ModeCreateOrUpdate
)

func (r *applyYamlResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_apply_yaml"
}

func (r *applyYamlResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Applies YAML configuration to the Omni system.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the applied configuration.",
			},
			"yaml": schema.StringAttribute{
				Required:    true,
				Description: "The YAML configuration to apply to Omni.",
			},
		},
	}
}

func (r *applyYamlResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	provider, ok := req.ProviderData.(*omniProvider)
	if !ok {
		resp.Diagnostics.AddError(
			errUnexpectedProviderType,
			fmt.Sprintf("Expected *omniProvider, got: %T", req.ProviderData),
		)
		return
	}

	r.provider = provider
}

func (r *applyYamlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	st := r.provider.client.Omni().State()

	var plan applyYamlResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resources, diags := r.decodeYAMLResources(ctx, plan.Yaml.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var idArr []string

	for _, resource := range resources {
		if err := r.processResource(ctx, st, resource, ModeCreateOnly); err != nil {
			resp.Diagnostics.AddError(errCreationFailed, err.Error())
			return
		}
		idArr = append(idArr, fmt.Sprintf("%s.%s", resource.Metadata().Type(), resource.Metadata().ID()))
	}

	plan.ID = types.StringValue(r.generateResourceId(idArr))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *applyYamlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	st := r.provider.client.Omni().State()

	var tfState, plan applyYamlResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &tfState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateResources, diags := r.decodeYAMLResources(ctx, tfState.Yaml.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	planResources, diags := r.decodeYAMLResources(ctx, plan.Yaml.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var idArr []string

	// Process plan resources and remove matched ones from stateResources
	for _, resource := range planResources {
		// Find and remove matching resource from stateResources if it exists
		for i, stateResource := range stateResources {
			if r.resourcesMatch(stateResource, resource) {
				stateResources = slices.Delete(stateResources, i, i+1)
				break
			}
		}

		if err := r.processResource(ctx, st, resource, ModeCreateOrUpdate); err != nil {
			resp.Diagnostics.AddError(errUpdateFailed, err.Error())
			return
		}
		idArr = append(idArr, fmt.Sprintf("%s.%s", resource.Metadata().Type(), resource.Metadata().ID()))
	}

	// Any remaining resources in stateResources need to be deleted
	for _, stateResource := range stateResources {
		if err := r.deleteResource(ctx, st, stateResource); err != nil {
			resp.Diagnostics.AddError(errDeleteFailed, err.Error())
			return
		}
	}

	plan.ID = types.StringValue(r.generateResourceId(idArr))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *applyYamlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

func (r *applyYamlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	st := r.provider.client.Omni().State()

	var plan applyYamlResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resources, diags := r.decodeYAMLResources(ctx, plan.Yaml.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, resource := range resources {
		if err := r.deleteResource(ctx, st, resource); err != nil {
			resp.Diagnostics.AddError(errDeleteFailed, err.Error())
			return
		}
	}
}

func (r *applyYamlResource) processResource(ctx context.Context, st state.State, resource cosi_res.Resource, mode ResourceOperationMode) error {
	result, err := st.Get(ctx, resource.Metadata())
	if err != nil {
		if state.IsNotFoundError(err) {
			if err := st.Create(ctx, resource); err != nil {
				return fmt.Errorf("failed to create resource '%s' of type '%s': %v",
					resource.Metadata().ID(), resource.Metadata().Type(), err)
			}
			return nil
		}
		return fmt.Errorf("failed to check resource '%s' of type '%s': %v",
			resource.Metadata().ID(), resource.Metadata().Type(), err)
	}

	if result != nil {
		if mode == ModeCreateOnly {
			return fmt.Errorf("resource '%s' of type '%s' already exists",
				resource.Metadata().ID(), resource.Metadata().Type())
		}
		// ModeCreateOrUpdate: update existing resource
		resource.Metadata().SetVersion(result.Metadata().Version())
		if err := st.Update(ctx, resource); err != nil {
			return fmt.Errorf("failed to update resource '%s' of type '%s': %v",
				resource.Metadata().ID(), resource.Metadata().Type(), err)
		}
	}

	return nil
}

func (r *applyYamlResource) decodeYAMLResources(ctx context.Context, yamlInput string) ([]cosi_res.Resource, diag.Diagnostics) {
	var diags diag.Diagnostics
	var resources []cosi_res.Resource
	var yamlResource protobuf.YAMLResource

	decoder := yaml.NewDecoder(bytes.NewReader([]byte(yamlInput)))
	for {
		if err := ctx.Err(); err != nil {
			diags.AddError(errContextError, fmt.Sprintf("Operation cancelled: %v", err))
			return nil, diags
		}

		if err := decoder.Decode(&yamlResource); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			diags.AddError(errYAMLDecodingError, fmt.Sprintf("Failed to decode YAML: %v", err))
			return nil, diags
		}

		resources = append(resources, yamlResource.Resource())
	}

	return resources, diags
}

// Helper functions
func (r *applyYamlResource) generateResourceId(arr []string) string {
	join := strings.Join(arr, "-")
	sum := sha256.Sum256([]byte(join))
	return hex.EncodeToString(sum[:])
}

func (r *applyYamlResource) resourcesMatch(a, b cosi_res.Resource) bool {
	return a.Metadata().ID() == b.Metadata().ID() &&
		a.Metadata().Type() == b.Metadata().Type() &&
		a.Metadata().Namespace() == b.Metadata().Namespace()
}

func (r *applyYamlResource) deleteResource(ctx context.Context, st state.State, resource cosi_res.Resource) error {
	_, err := st.Get(ctx, resource.Metadata())
	if err != nil {
		if state.IsNotFoundError(err) {
			return nil // Resource already deleted
		}
		return fmt.Errorf("failed to check resource '%s' of type '%s': %v",
			resource.Metadata().ID(), resource.Metadata().Type(), err)
	}

	if err := st.Destroy(ctx, resource.Metadata()); err != nil {
		return fmt.Errorf("failed to delete resource '%s' of type '%s': %v",
			resource.Metadata().ID(), resource.Metadata().Type(), err)
	}

	return nil
}
