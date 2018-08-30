variable "manifest_names" {
  default = [
    "01-tectonic-namespace.yaml",
    "02-ingress-namespace.yaml",
    "03-openshift-web-console-namespace.yaml",
    "app-version-kind.yaml",
    "app-version-tectonic-network.yaml",
    "app-version-tnc.yaml",
    "kube-apiserver-secret.yaml",
    "kube-cloud-config.yaml",
    "kube-controller-manager-secret.yaml",
    "node-config-kind.yaml",
    "openshift-apiserver-secret.yaml",
    "cluster-apiserver-secret.yaml",
    "pull.json",
    "tectonic-network-operator.yaml",
    "tectonic-node-controller-operator.yaml",
    "tnc-tls-secret.yaml",
    "app-version-mao.yaml",
    "machine-api-operator.yaml",
    "ign-config.yaml",
  ]
}

# Self-hosted manifests (resources/generated/manifests/)
data "template_file" "manifest_file_list" {
  count    = "${length(var.manifest_names)}"
  template = "${file("${path.module}/resources/manifests/${var.manifest_names[count.index]}")}"

  vars {
    tectonic_network_operator_image = "${var.container_images["tectonic_network_operator"]}"
    tnc_operator_image              = "${var.container_images["tnc_operator"]}"

    cloud_provider_config = "${var.cloud_provider_config}"

    root_ca_cert             = "${base64encode(var.root_ca_cert_pem)}"
    aggregator_ca_cert       = "${base64encode(var.aggregator_ca_cert_pem)}"
    aggregator_ca_key        = "${base64encode(var.aggregator_ca_key_pem)}"
    kube_ca_cert             = "${base64encode(var.kube_ca_cert_pem)}"
    kube_ca_key              = "${base64encode(var.kube_ca_key_pem)}"
    service_serving_ca_cert  = "${base64encode(var.service_serving_ca_cert_pem)}"
    service_serving_ca_key   = "${base64encode(var.service_serving_ca_key_pem)}"
    apiserver_key            = "${base64encode(var.apiserver_key_pem)}"
    apiserver_cert           = "${base64encode(var.apiserver_cert_pem)}"
    openshift_apiserver_key  = "${base64encode(var.openshift_apiserver_key_pem)}"
    openshift_apiserver_cert = "${base64encode(var.openshift_apiserver_cert_pem)}"
    apiserver_proxy_key      = "${base64encode(var.apiserver_proxy_key_pem)}"
    apiserver_proxy_cert     = "${base64encode(var.apiserver_proxy_cert_pem)}"
    clusterapi_ca_cert       = "${base64encode(var.clusterapi_ca_cert_pem)}"
    clusterapi_ca_key        = "${base64encode(var.clusterapi_ca_key_pem)}"
    oidc_ca_cert             = "${base64encode(var.oidc_ca_cert)}"
    pull_secret              = "${base64encode(file(var.pull_secret_path))}"
    serviceaccount_pub       = "${base64encode(var.service_account_public_key_pem)}"
    serviceaccount_key       = "${base64encode(var.service_account_private_key_pem)}"
    kube_dns_service_ip      = "${cidrhost(var.service_cidr, 10)}"

    openshift_loopback_kubeconfig = "${base64encode(data.template_file.kubeconfig.rendered)}"

    etcd_ca_cert     = "${base64encode(var.etcd_ca_cert_pem)}"
    etcd_client_cert = "${base64encode(var.etcd_client_cert_pem)}"
    etcd_client_key  = "${base64encode(var.etcd_client_key_pem)}"

    tnc_tls_cert = "${base64encode(var.tnc_cert_pem)}"
    tnc_tls_key  = "${base64encode(var.tnc_key_pem)}"

    worker_ign_config = "${base64encode(var.worker_ign_config)}"
  }
}

# Ignition entry for every bootkube manifest
# Drops them in /opt/tectonic/manifests/<path>
data "ignition_file" "manifest_file_list" {
  count      = "${length(var.manifest_names)}"
  filesystem = "root"
  mode       = "0644"

  path = "/opt/tectonic/manifests/${var.manifest_names[count.index]}"

  content {
    content = "${data.template_file.manifest_file_list.*.rendered[count.index]}"
  }
}

# Log the generated manifest files to disk for debugging and user visibility
# Dest: ./generated/manifests/<path>
resource "local_file" "manifest_files" {
  count    = "${length(var.manifest_names)}"
  filename = "./generated/manifests/${var.manifest_names[count.index]}"
  content  = "${data.template_file.manifest_file_list.*.rendered[count.index]}"
}

# mao config
# Self-hosted manifests (resources/generated/manifests/)
data "template_file" "mao_config" {
  template = "${file("${path.module}/resources/manifests/mao-config-${var.mao_provider}.yaml")}"

  vars {
    replicas     = "${var.replicas}"
    cluster_name = "${var.cluster_name}"

    aws_region        = "${var.aws_region}"
    aws_az            = "${var.aws_az}"
    aws_ami           = "${var.aws_ami}"
    cluster_id        = "${var.cluster_id}"
    worker_ign_config = "${base64encode(var.worker_ign_config)}"

    libvirt_uri = "${var.libvirt_uri}"
  }
}

data "ignition_file" "mao_config" {
  filesystem = "root"
  mode       = "0644"

  path = "/opt/tectonic/manifests/mao-config.yaml"

  content {
    content = "${data.template_file.mao_config.rendered}"
  }
}

resource "local_file" "mao_config" {
  filename = "./generated/manifests/mao-config.yaml"
  content  = "${data.template_file.mao_config.rendered}"
}
