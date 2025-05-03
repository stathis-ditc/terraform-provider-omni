# Example 1: Basic usage with required fields
data "omni_installation_media" "basic" {
  image_name    = "iso"
  talos_version = "v1.9.5"
}

# Example 2: Advanced usage with all optional fields
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
}

# Output the results
output "iso_schematic" {
  value = data.omni_installation_media.basic.schematic
}

output "iso_pxe" {
  value = data.omni_installation_media.detailed.pxe_url
}


