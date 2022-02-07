resource "google_compute_instance_group" "root-source" {
  for_each = { for i in local.instances : i.name => i if i.root_source_instance_group }
  name     = join("-", [var.prefix, "vault-root-source", each.value.region, each.value.zone_postfix])
  zone     = join("-", [each.value.region, each.value.zone_postfix])

  instances = [
    google_compute_instance.main[each.value.name].self_link,
  ]

  named_port {
    name = "https"
    port = "443"
  }
}

resource "google_compute_backend_service" "root-source" {
  name                  = join("-", [var.prefix, "vault-root-source"])
  load_balancing_scheme = "EXTERNAL"
  port_name             = "https"
  protocol              = "HTTPS"
  health_checks         = [google_compute_health_check.root-source.id]

  dynamic "backend" {
    for_each = { for index, ig in google_compute_instance_group.root-source : index => ig }
    content {
      group = backend.value["id"]
    }
  }
}

resource "google_compute_health_check" "root-source" {
  name                = join("-", [var.prefix, "vault-root-source"])
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

resource "google_compute_managed_ssl_certificate" "root-source" {
  name = join("-", [var.prefix, "vault-root-source"])

  managed {
    domains = ["root-source.negentropy.dev.flant.com."]
  }
}

resource "google_compute_url_map" "root-source" {
  name = join("-", [var.prefix, "vault-root-source"])
  default_service = google_compute_backend_service.root-source.id
}

resource "google_compute_target_https_proxy" "root-source" {
  name             = join("-", [var.prefix, "vault-root-source"])
  url_map          = google_compute_url_map.root-source.id
  ssl_certificates = [google_compute_managed_ssl_certificate.root-source.id]
}

resource "google_compute_global_forwarding_rule" "root-source" {
  name       = join("-", [var.prefix, "vault-root-source"])
  target     = google_compute_target_https_proxy.root-source.id
  port_range = 443
}

resource "google_dns_record_set" "root-source" {
  name         = "root-source.negentropy.dev.flant.com."
  type         = "A"
  ttl          = 300
  managed_zone = "negentropy"
  rrdatas      = [google_compute_global_forwarding_rule.root-source.ip_address]
}

resource "google_compute_instance_group" "auth" {
  for_each = { for i in local.instances : i.name => i if i.auth_instance_group }
  name     = join("-", [var.prefix, "vault-auth", each.value.region, each.value.zone_postfix])
  zone     = join("-", [each.value.region, each.value.zone_postfix])

  instances = [
    google_compute_instance.main[each.value.name].self_link,
  ]

  named_port {
    name = "https"
    port = "443"
  }
}

resource "google_compute_health_check" "auth" {
  name                = join("-", [var.prefix, "vault-auth"])
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

resource "google_compute_backend_service" "auth" {
  name                  = join("-", [var.prefix, "vault-auth"])
  load_balancing_scheme = "EXTERNAL"
  port_name             = "https"
  protocol              = "HTTPS"
  health_checks         = [google_compute_health_check.auth.id]

  dynamic "backend" {
    for_each = { for index, ig in google_compute_instance_group.auth : index => ig }
    content {
      group = backend.value["id"]
    }
  }
}

resource "google_compute_managed_ssl_certificate" "auth" {
  name = join("-", [var.prefix, "vault-auth"])
  managed {
    domains = ["auth.negentropy.dev.flant.com."]
  }
}

resource "google_compute_url_map" "auth" {
  name = join("-", [var.prefix, "vault-auth"])
  default_service = google_compute_backend_service.auth.id
}

resource "google_compute_target_https_proxy" "auth" {
  name             = join("-", [var.prefix, "vault-auth"])
  url_map          = google_compute_url_map.auth.id
  ssl_certificates = [google_compute_managed_ssl_certificate.auth.id]
}

resource "google_compute_global_forwarding_rule" "auth" {
  name       = join("-", [var.prefix, "vault-auth"])
  target     = google_compute_target_https_proxy.auth.id
  port_range = 443
}

resource "google_dns_record_set" "auth" {
  name         = "auth.negentropy.dev.flant.com."
  type         = "A"
  ttl          = 300
  managed_zone = "negentropy"
  rrdatas      = [google_compute_global_forwarding_rule.auth.ip_address]
}
