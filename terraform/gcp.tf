resource "google_compute_network" "vpc" {
  name                    = "${var.project_name}-vpc"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  name          = "${var.project_name}-subnet"
  network       = google_compute_network.vpc.id
  region        = var.gcp_region
  ip_cidr_range = "10.0.0.0/24"
}

resource "google_compute_firewall" "ssh" {
  name    = "${var.project_name}-ssh"
  network = google_compute_network.vpc.id

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["${var.project_name}-vm"]
}

resource "google_compute_firewall" "app" {
  name    = "${var.project_name}-app"
  network = google_compute_network.vpc.id

  allow {
    protocol = "tcp"
    ports    = ["8080", "3333"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["${var.project_name}-vm"]
}

resource "google_compute_address" "vm" {
  name   = "${var.project_name}-ip"
  region = var.gcp_region
}

data "google_compute_image" "ubuntu" {
  family  = "ubuntu-2204-lts"
  project = "ubuntu-os-cloud"
}

resource "tls_private_key" "ssh" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

locals {
  # The free-tier VM has 1 GB of RAM and a quarter of a core. Adding swap is what
  # lets the deploy build the web image on it instead of being OOM-killed. The
  # marker file at the end tells the deploy when the VM is ready, so it does not
  # race the first-boot install.
  startup_script = <<-EOT
    #!/bin/bash
    set -euo pipefail

    if [ ! -f /swapfile ]; then
      fallocate -l 2G /swapfile
      chmod 600 /swapfile
      mkswap /swapfile
      swapon /swapfile
      echo '/swapfile none swap sw 0 0' >> /etc/fstab
    fi

    if ! command -v docker >/dev/null 2>&1; then
      while fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1; do sleep 5; done
      apt-get update
      apt-get install -y ca-certificates curl git
      curl -fsSL https://get.docker.com | sh
    fi

    if id ubuntu >/dev/null 2>&1; then
      usermod -aG docker ubuntu
    fi

    touch /var/lib/stash-vm-ready
  EOT
}

resource "google_compute_instance" "vm" {
  name         = "${var.project_name}-vm"
  machine_type = var.instance_type
  zone         = var.gcp_zone
  tags         = ["${var.project_name}-vm"]

  boot_disk {
    initialize_params {
      image = data.google_compute_image.ubuntu.self_link
      size  = var.boot_disk_size_gb
      type  = "pd-standard"
    }
  }

  network_interface {
    network    = google_compute_network.vpc.id
    subnetwork = google_compute_subnetwork.subnet.id

    access_config {
      nat_ip = google_compute_address.vm.address
    }
  }

  metadata = {
    ssh-keys = "ubuntu:${tls_private_key.ssh.public_key_openssh}"
  }

  metadata_startup_script = local.startup_script
}
