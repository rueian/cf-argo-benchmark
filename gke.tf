variable "gcp_project_id" {
  type = string
}

variable "gcp_region" {
  type = string
  default = "asia-east1"
}

variable "gcp_zone" {
  type = string
  default = "asia-east1-a"
}

variable "resource_prefix" {
  type = string
  default = "argo-test"
}

provider "google-beta" {
  project = var.gcp_project_id
  region  = var.gcp_region
  zone    = var.gcp_zone
}

resource "google_compute_network" "argo_test" {
  provider = google-beta

  name = var.resource_prefix
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "argo_test_1" {
  provider = google-beta

  name    = "${var.resource_prefix}-1"
  region  = var.gcp_region
  network = google_compute_network.argo_test.self_link

  ip_cidr_range      = "10.110.0.0/20"
  secondary_ip_range = [
    {
      range_name     = "${var.resource_prefix}-1-pod"
      ip_cidr_range  = "10.112.0.0/20"
    },
    {
      range_name     = "${var.resource_prefix}-1-service"
      ip_cidr_range  = "10.114.0.0/20"
    }
  ]
}

resource "google_container_cluster" "argo_test" {
  provider = google-beta

  name                     = var.resource_prefix
  initial_node_count       = 1
  remove_default_node_pool = true

  network    = google_compute_network.argo_test.self_link
  subnetwork = google_compute_subnetwork.argo_test_1.self_link

  release_channel {
    channel = "REGULAR"
  }

  ip_allocation_policy {
    cluster_secondary_range_name  = "${var.resource_prefix}-1-pod"
    services_secondary_range_name = "${var.resource_prefix}-1-service"
  }

  maintenance_policy {
    daily_maintenance_window {
      start_time = "20:00"
    }
  }
}

locals {
  pools = {
    metric = {
      initial_node_count = 1,
      max_node_count     = 1,
      machine_type       = "g1-small"
    }
    egress = {
      initial_node_count = 0,
      max_node_count     = 1,
      machine_type       = "n1-highcpu-4"
    }
    backend = {
      initial_node_count = 0,
      max_node_count     = 20,
      machine_type       = "n1-highcpu-4"
    }
    client = {
      initial_node_count = 0,
      max_node_count     = 20,
      machine_type       = "n1-highcpu-4"
    }
    tunnel = {
      initial_node_count = 0,
      max_node_count     = 20,
      machine_type       = "n1-highcpu-4"
    }
  }
}

resource "google_container_node_pool" "argo_test_pool" {
  provider = google-beta
  for_each = local.pools

  name               = each.key
  cluster            = google_container_cluster.argo_test.name
  initial_node_count = each.value.initial_node_count

  autoscaling {
    max_node_count = each.value.max_node_count
    min_node_count = 0
  }

  node_config {
    disk_size_gb = 10
    machine_type = each.value.machine_type

    metadata = {
      "disable-legacy-endpoints" = "true"
    }

    oauth_scopes = [
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
      "https://www.googleapis.com/auth/service.management.readonly",
      "https://www.googleapis.com/auth/servicecontrol",
      "https://www.googleapis.com/auth/trace.append",
    ]

    preemptible = true
  }
}