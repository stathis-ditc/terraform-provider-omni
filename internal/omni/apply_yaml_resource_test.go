package omni

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestApplyYamlResource(t *testing.T) {
	r := &applyYamlResource{}

	testApplyYamlId := "MachineClasses.omni.sidero.dev.test-apply-yaml"
	testApplyYaml2Id := "MachineClasses.omni.sidero.dev.test-apply-yaml-2"
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create new resource
				Config: providerConfig + `
resource "omni_apply_yaml" "test-apply" {
  yaml = <<-EOT
metadata:
    namespace: default
    type: MachineClasses.omni.sidero.dev
    id: test-apply-yaml
spec:
    matchlabels:
        - omni.sidero.dev/platform = test-apply-yaml
    autoprovision: null
EOT
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("omni_apply_yaml.test-apply", "yaml", "metadata:\n    namespace: default\n    type: MachineClasses.omni.sidero.dev\n    id: test-apply-yaml\nspec:\n    matchlabels:\n        - omni.sidero.dev/platform = test-apply-yaml\n    autoprovision: null\n"),
					resource.TestCheckResourceAttrSet("omni_apply_yaml.test-apply", "id"),
					resource.TestCheckResourceAttr("omni_apply_yaml.test-apply", "id", r.generateResourceId([]string{testApplyYamlId})),
				),
			},
			{
				// Update existing
				Config: providerConfig + `
resource "omni_apply_yaml" "test-apply" {
  yaml = <<-EOT
metadata:
    namespace: default
    type: MachineClasses.omni.sidero.dev
    id: test-apply-yaml
spec:
    matchlabels:
        - omni.sidero.dev/platform = test-apply-yaml
    autoprovision: null
---
metadata:
    namespace: default
    type: MachineClasses.omni.sidero.dev
    id: test-apply-yaml-2
spec:
    matchlabels:
        - omni.sidero.dev/platform = test-apply-yaml-2
    autoprovision: null
EOT
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("omni_apply_yaml.test-apply", "yaml", "metadata:\n    namespace: default\n    type: MachineClasses.omni.sidero.dev\n    id: test-apply-yaml\nspec:\n    matchlabels:\n        - omni.sidero.dev/platform = test-apply-yaml\n    autoprovision: null\n---\nmetadata:\n    namespace: default\n    type: MachineClasses.omni.sidero.dev\n    id: test-apply-yaml-2\nspec:\n    matchlabels:\n        - omni.sidero.dev/platform = test-apply-yaml-2\n    autoprovision: null\n"),
					resource.TestCheckResourceAttr("omni_apply_yaml.test-apply", "id", r.generateResourceId([]string{testApplyYamlId, testApplyYaml2Id})),
				),
			},
			{
				// Update existing by removing one resource
				Config: providerConfig + `
resource "omni_apply_yaml" "test-apply" {
  yaml = <<-EOT
metadata:
    namespace: default
    type: MachineClasses.omni.sidero.dev
    id: test-apply-yaml-2
spec:
    matchlabels:
        - omni.sidero.dev/platform = test-apply-yaml-2
    autoprovision: null
EOT
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("omni_apply_yaml.test-apply", "yaml", "metadata:\n    namespace: default\n    type: MachineClasses.omni.sidero.dev\n    id: test-apply-yaml-2\nspec:\n    matchlabels:\n        - omni.sidero.dev/platform = test-apply-yaml-2\n    autoprovision: null\n"),
					resource.TestCheckResourceAttr("omni_apply_yaml.test-apply", "id", r.generateResourceId([]string{testApplyYaml2Id})),
				),
			},
		},
	})
}

func TestDecodeYAMLResources_Valid(t *testing.T) {
	r := &applyYamlResource{}
	ctx := context.Background()

	inputMap := map[string]interface{}{
		"metadata": map[string]interface{}{
			"type": "MachineClasses.omni.sidero.dev",
			"id":   "test-id",
		},
		"spec": map[string]interface{}{},
	}
	buf, err := yaml.Marshal(inputMap)
	require.NoError(t, err)

	// wrap in a YAML stream
	input := "---\n" + string(buf)

	resources, diags := r.decodeYAMLResources(ctx, input)
	if diags.HasError() {
		t.Logf("Diagnostics: %+v", diags)
	}
	require.False(t, diags.HasError())
	require.Len(t, resources, 1)

	md := resources[0].Metadata()
	require.Equal(t, "MachineClasses.omni.sidero.dev", md.Type())
	require.Equal(t, "test-id", md.ID())
}

func TestGenerateResourceId(t *testing.T) {
	r := &applyYamlResource{}

	id1 := r.generateResourceId([]string{"type1.id1", "type2.id2"})
	id2 := r.generateResourceId([]string{"type1.id1", "type2.id2"})
	id3 := r.generateResourceId([]string{"type2.id2", "type1.id1"})

	require.Equal(t, id1, id2, "IDs with same order should be equal")
	require.NotEqual(t, id1, id3, "IDs with different order should differ")
}
