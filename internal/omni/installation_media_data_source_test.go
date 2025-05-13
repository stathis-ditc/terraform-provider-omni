package omni

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestInstallationMediaDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
data "omni_installation_media" "basic" {
  image_name    = "iso"
  talos_version = "v1.9.5"
}
output "iso_schematic" {
  value = data.omni_installation_media.basic.schematic
}
		  `,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.omni_installation_media.basic", "schematic", "d23d7a50ced403dfc4a9405b2863893f1f3ce4792f601f227f27ccb019c416ca"),
					resource.TestCheckResourceAttr("data.omni_installation_media.basic", "pxe_url", "https://pxe.factory.talos.dev/pxe/d23d7a50ced403dfc4a9405b2863893f1f3ce4792f601f227f27ccb019c416ca/v1.9.5/metal-amd64"),
				),
			},
			{
				Config: providerConfig + `
data "omni_installation_media" "detailed" {
  image_name    = "iso"    # Required
  talos_version = "v1.9.5" # Required
  arch          = "amd64"  # Optional: architecture ('amd64', 'arm64')
  secure_boot   = false    # Optional: enable secure boot
  extra_kernel_args = [    # Optional: extra kernel arguments
    "ip=1.2.3.4::1.2.3.1:255.255.255.0::ens18:off:1.2.3.53"
  ]
  extensions = [ # Optional: extensions
    "docker",
    "k8s"
  ]
  use_siderolink_grpc_tunnel = false # Optional: Use SideroLink GRPC tunnel
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.omni_installation_media.detailed", "schematic", "17745c21ca7335d5d654f09e447d30f3bcdd57dc071252d470199167a1aeb968"),
					resource.TestCheckResourceAttr("data.omni_installation_media.detailed", "pxe_url", "https://pxe.factory.talos.dev/pxe/17745c21ca7335d5d654f09e447d30f3bcdd57dc071252d470199167a1aeb968/v1.9.5/metal-amd64"),
				),
			},
		},
	})

}
