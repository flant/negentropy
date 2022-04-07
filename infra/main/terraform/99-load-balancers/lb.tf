data "google_compute_instance" "vault-root-source-1" {
  name = "${local.prefix}-vault-root-source-1"
  zone = "europe-west1-c"
}

data "google_compute_instance" "vault-root-source-2" {
  name = "${local.prefix}-vault-root-source-2"
  zone = "europe-west2-a"
}

data "google_compute_instance" "vault-root-source-3" {
  name = "${local.prefix}-vault-root-source-3"
  zone = "europe-west3-b"
}

resource "google_compute_instance_group" "vault-root-source-1" {
  name = "${local.prefix}-vault-root-source-1"
  zone = "europe-west1-c"

  instances = [
    data.google_compute_instance.vault-root-source-1.self_link
  ]

  named_port {
    name = "https"
    port = "8200"
  }
}

resource "google_compute_instance_group" "vault-root-source-2" {
  name = "${local.prefix}-vault-root-source-2"
  zone = "europe-west2-a"

  instances = [
    data.google_compute_instance.vault-root-source-2.self_link
  ]

  named_port {
    name = "https"
    port = "8200"
  }
}

resource "google_compute_instance_group" "vault-root-source-3" {
  name = "${local.prefix}-vault-root-source-3"
  zone = "europe-west3-b"

  instances = [
    data.google_compute_instance.vault-root-source-3.self_link
  ]

  named_port {
    name = "https"
    port = "8200"
  }
}

resource "google_compute_health_check" "vault-root-source" {
  name                = "${local.prefix}-vault-root-source"
  timeout_sec         = 5
  check_interval_sec  = 5
  healthy_threshold   = 2
  unhealthy_threshold = 3
  https_health_check {
    port         = "8200"
    request_path = "/v1/sys/health"
#    port_name          = "https"
#    port_specification = "USE_NAMED_PORT"
  }
}

resource "google_compute_backend_service" "vault-root-source" {
  name                  = "${local.prefix}-vault-root-source"
  load_balancing_scheme = "EXTERNAL"
  port_name             = "https"
  protocol              = "HTTPS"
  health_checks         = [google_compute_health_check.vault-root-source.id]

  backend {
    group = google_compute_instance_group.vault-root-source-1.self_link
  }
  backend {
    group = google_compute_instance_group.vault-root-source-2.self_link
  }
  backend {
    group = google_compute_instance_group.vault-root-source-3.self_link
  }
}

resource "google_compute_managed_ssl_certificate" "vault-root-source" {
  name = "${local.prefix}-vault-root-source"

  managed {
    domains = ["${var.vault_root_source_loadbalancer_domain}."]
  }
}

resource "google_compute_url_map" "vault-root-source" {
  name = "${local.prefix}-vault-root-source"
  default_service = google_compute_backend_service.vault-root-source.id
}

resource "google_compute_target_https_proxy" "vault-root-source" {
  name             = "${local.prefix}-vault-root-source"
  url_map          = google_compute_url_map.vault-root-source.id
  ssl_certificates = [google_compute_managed_ssl_certificate.vault-root-source.id]
}

resource "google_compute_global_forwarding_rule" "vault-root-source" {
  name       = "${local.prefix}-vault-root-source"
  target     = google_compute_target_https_proxy.vault-root-source.id
  port_range = 443
}

resource "google_dns_record_set" "vault-root-source" {
  name         = "${var.vault_root_source_loadbalancer_domain}."
  type         = "A"
  ttl          = 300
  managed_zone = "negentropy"
  rrdatas      = [google_compute_global_forwarding_rule.vault-root-source.ip_address]
}

data "google_compute_instance" "vault-auth-ew3a1" {
  name = "${local.prefix}-vault-auth-ew3a1"
  zone = "europe-west3-a"
}

resource "google_compute_instance_group" "vault-auth-ew3a1" {
  name = "${local.prefix}-vault-auth-europe-west3-a"
  zone = "europe-west3-a"

  instances = [
    data.google_compute_instance.vault-auth-ew3a1.self_link
  ]

  named_port {
    name = "https"
    port = "443"
  }
}

resource "google_compute_health_check" "vault-auth" {
  name                = "${local.prefix}-vault-auth"
  timeout_sec         = 5
  check_interval_sec  = 5
  healthy_threshold   = 2
  unhealthy_threshold = 3
  https_health_check {
    port         = "443"
    request_path = "/v1/sys/health"
#    port_name          = "https"
#    port_specification = "USE_NAMED_PORT"
  }
}

resource "google_compute_backend_service" "vault-auth" {
  name                  = "${local.prefix}-vault-auth"
  load_balancing_scheme = "EXTERNAL"
  port_name             = "https"
  protocol              = "HTTPS"
  health_checks         = [google_compute_health_check.vault-auth.id]

  backend {
    group = google_compute_instance_group.vault-auth-ew3a1.self_link
  }
}

resource "google_compute_managed_ssl_certificate" "vault-auth" {
  name = "${local.prefix}-vault-auth"
  managed {
    domains = ["${var.vault_auth_loadbalancer_domain}."]
  }
}

resource "google_compute_url_map" "vault-auth" {
  name = "${local.prefix}-vault-auth"
  default_service = google_compute_backend_service.vault-auth.id
}

resource "google_compute_target_https_proxy" "vault-auth" {
  name             = "${local.prefix}-vault-auth"
  url_map          = google_compute_url_map.vault-auth.id
  ssl_certificates = [google_compute_managed_ssl_certificate.vault-auth.id]
}

resource "google_compute_global_forwarding_rule" "vault-auth" {
  name       = "${local.prefix}-vault-auth"
  target     = google_compute_target_https_proxy.vault-auth.id
  port_range = 443
}

resource "google_dns_record_set" "vault-auth" {
  name         = "${var.vault_auth_loadbalancer_domain}."
  type         = "A"
  ttl          = 300
  managed_zone = "negentropy"
  rrdatas      = [google_compute_global_forwarding_rule.vault-auth.ip_address]
}
