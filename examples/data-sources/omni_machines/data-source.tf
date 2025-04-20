data "omni_machines" "example" {}

output "connected_machines" {
  description = "List of connected machines"
  value = [for machine in data.omni_machines.example.machines : machine.id if machine.connected]
}
