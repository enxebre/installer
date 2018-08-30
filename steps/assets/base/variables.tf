# We use the same variables as the platform-specific step, to keep from going insane. Here
# is where we can define variables that the steps can pass directly
variable "cloud_provider" {
  type = "string"
}

variable "ingress_kind" {
  type = "string"
}

# machine api operator config
variable "aws_region" {
  type    = "string"
  default = ""
}

variable "aws_az" {
  type    = "string"
  default = ""
}

variable "aws_ami" {
  type    = "string"
  default = ""
}

variable "aws_worker_ign_config" {
  type    = "string"
  default = ""
}

variable "replicas" {
  type    = "string"
  default = ""
}

variable "libvirt_uri" {
  type    = "string"
  default = ""
}

variable "mao_provider" {
  type = "string"
}
