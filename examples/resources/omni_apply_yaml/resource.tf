resource "omni_apply_yaml" "raw_yaml" {
  yaml = <<-EOT
metadata:
    namespace: default
    type: MachineClasses.omni.sidero.dev
    id: aws
spec:
    matchlabels:
        - omni.sidero.dev/platform = sth
    autoprovision: null
EOT
}

resource "omni_apply_yaml" "encode_yaml" {
  yaml = yamlencode({
    metadata = {
      namespace = "default"
      type      = "MachineClasses.omni.sidero.dev"
      id        = "aws11"
    }
    spec = {
      matchlabels = [
        "omni.sidero.dev/platform = sth"
      ]
      autoprovision = null
    }
  })
}

resource "omni_apply_yaml" "apply_from_yaml_file" {
  yaml = file("/path/to/file.yaml")
}