module assets_base {
  source = "../base"

  cloud_provider = "aws"
  ingress_kind   = "haproxy-router"

  tectonic_admin_email             = "${var.tectonic_admin_email}"
  tectonic_admin_password          = "${var.tectonic_admin_password}"
  tectonic_admin_ssh_key           = "${var.tectonic_admin_ssh_key}"
  tectonic_base_domain             = "${var.tectonic_base_domain}"
  tectonic_cluster_cidr            = "${var.tectonic_cluster_cidr}"
  tectonic_cluster_id              = "${var.tectonic_cluster_id}"
  tectonic_cluster_name            = "${var.tectonic_cluster_name}"
  tectonic_container_images        = "${var.tectonic_container_images}"
  tectonic_container_linux_channel = "${var.tectonic_container_linux_channel}"
  tectonic_container_linux_version = "${var.tectonic_container_linux_version}"
  tectonic_image_re                = "${var.tectonic_image_re}"
  tectonic_kubelet_debug_config    = "${var.tectonic_kubelet_debug_config}"
  tectonic_license_path            = "${var.tectonic_license_path}"
  tectonic_networking              = "${var.tectonic_networking}"
  tectonic_platform                = "${var.tectonic_platform}"
  tectonic_pull_secret_path        = "${var.tectonic_pull_secret_path}"
  tectonic_service_cidr            = "${var.tectonic_service_cidr}"
  tectonic_update_channel          = "${var.tectonic_update_channel}"
  tectonic_versions                = "${var.tectonic_versions}"
  aws_region                       = "${var.tectonic_aws_region}"
  aws_az                           = "${data.aws_availability_zones.azs.names[0]}"
  aws_ami                          = "${coalesce(var.tectonic_aws_ec2_ami_override, module.ami.id)}"
  aws_worker_ign_config            = "${file("worker.ign")}"
  replicas                         = "${var.tectonic_worker_count}"
  mao_provider                     = "aws"
}

provider "aws" {
  region  = "${var.tectonic_aws_region}"
  profile = "${var.tectonic_aws_profile}"
  version = "1.8.0"

  assume_role {
    role_arn     = "${var.tectonic_aws_installer_role}"
    session_name = "TECTONIC_INSTALLER_${var.tectonic_cluster_name}"
  }
}

// TODO(enxebre): consider to deploy machineSet per az
// https://github.com/kubernetes-sigs/cluster-api/issues/46
data "aws_availability_zones" "azs" {}

module "ami" {
  source = "../../../modules/aws/ami"

  region          = "${var.tectonic_aws_region}"
  release_channel = "${var.tectonic_container_linux_channel}"
  release_version = "${var.tectonic_container_linux_version}"
}
