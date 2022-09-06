{{- define "globals" -}}
{{- $_ := set . "kafka_name" (pluck .Values.werf.env .Values.kafka.name | first | default .Values.kafka.name._default) -}}
{{- $_ := set . "vault_port" (pluck .Values.werf.env .Values.vault.port | first | default .Values.vault.port._default) -}}
{{- $_ := set . "vault_cluster_port" (pluck .Values.werf.env .Values.vault.cluster_port | first | default .Values.vault.cluster_port._default) -}}
{{- $_ := set . "vault_data_path" (pluck .Values.werf.env .Values.vault.data_path | first | default .Values.vault.data_path._default) -}}
{{- $_ := set . "vault_storage_class" (pluck .Values.werf.env .Values.vault.storage_class | first | default .Values.vault.storage_class._default) -}}
{{- $_ := set . "vault_storage_size" (pluck .Values.werf.env .Values.vault.storage_size | first | default .Values.vault.storage_size._default) -}}
{{- $_ := set . "vault_ha" (pluck .Values.werf.env .Values.vault.ha | first | default .Values.vault.ha._default) -}}
{{- $_ := set . "vault_bucket" (pluck .Values.werf.env .Values.vault.bucket | first | default printf .Values.vault.bucket._default .Values.werf.env) -}}
{{- end -}}

{{- define "vault.env" -}}
- name: BUCKET
  value: {{ .vault_bucket }}
- name: VAULT_LOG_LEVEL
  value: debug
- name: RLIMIT_CORE
  value: "0"
- name: VAULT_SEAL_TYPE
  value: "gcpckms"
- name: VAULT_GCPCKMS_SEAL_KEY_RING
  value: "vault"
- name: VAULT_GCPCKMS_SEAL_CRYPTO_KEY
  value: "vault-unseal"
- name: GOOGLE_PROJECT
  value: "negentropy-dev"
- name: GOOGLE_REGION
  value: "europe"
- name: HOST_IP
  valueFrom:
    fieldRef:
      fieldPath: status.hostIP
- name: POD_IP
  valueFrom:
    fieldRef:
      fieldPath: status.podIP
- name: VAULT_ADDR
  value: {{ printf "http://127.0.0.1:%s" .vault_port | quote }} #change to HTTPS after turning TLS on
- name: VAULT_API_ADDR
  value: {{ printf "http://$(POD_IP):%s" .vault_port | quote }} #change to HTTPS after turning TLS on
- name: VAULT_LOG_FORMAT
  value: "json"
- name: SKIP_CHOWN
  value: "true"
- name: SKIP_SETCAP
  value: "true"
- name: HOSTNAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
{{- if (.vault_ha) -}}
- name: VAULT_K8S_POD_NAME
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: VAULT_K8S_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: VAULT_RAFT_NODE_ID
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: VAULT_CLUSTER_ADDR
  value: {{ printf "http://$(HOSTNAME).%s:%s"  .vault_cluster_port | quote }} #change to HTTPS after turning TLS on
{{- end -}}
{{- end -}}

{{- define "vault.securitycontext" -}}
{{- if eq .Values.werf.env "production" -}}
readOnlyRootFilesystem: true
runAsNonRoot: true
{{- end -}}
capabilities:
  add:
  - IPC_LOCK
{{- end -}}

{{- define "vault.volumemounts" -}}
- name: config
  mountPath: /etc/vault.hcl
  subPath: vault.hcl
  readOnly: true
- name: data
  mountPath: {{ .vault_data_path }}
{{- end -}}

{{- define "vault.volumeclaimtemplate" -}}
volumeClaimTemplates:
- metadata:
    name: data
  spec:
    accessModes: [ "ReadWriteOnce" ]
    resources:
      requests:
        storage: {{ .vault_storage_size }}
    storageClassName: {{ .vault_storage_class }}
{{- end -}}

{{- define "vault.probes" -}}
readinessProbe:
  # Check status; unsealed vault servers return 0
  # The exit code reflects the seal status:
  #   0 - unsealed
  #   1 - error
  #   2 - sealed
  exec:
    command: ["/bin/sh", "-ec", "vault status -tls-skip-verify"]
  failureThreshold: 2
  initialDelaySeconds: 5
  periodSeconds: 5
  successThreshold: 1
  timeoutSeconds: 3
livenessProbe:
  httpGet:
    path: "/v1/sys/health?standbyok=true"
    port: 8200
    scheme: "HTTP" #change to HTTPS after turning TLS on
  failureThreshold: 2
  initialDelaySeconds: 60
  periodSeconds: 5
  successThreshold: 1
  timeoutSeconds: 3
{{- end -}}

{{- define "vault.lifecycle" -}}
lifecycle:
  # Vault container doesn't receive SIGTERM from Kubernetes
  # and after the grace period ends, Kube sends SIGKILL.  This
  # causes issues with graceful shutdowns such as deregistering itself
  # from Consul (zombie services).
  preStop:
    exec:
      command: [
        "/bin/sh", "-c",
        # Adding a sleep here to give the pod eviction a
        # chance to propagate, so requests will not be made
        # to this pod while it's terminating
        "sleep 5 && kill -SIGTERM $(pidof vault)",
      ]
{{- end -}}