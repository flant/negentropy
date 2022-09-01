{{- define "globals" -}}
{{- $_ := set . "kafka_name" (pluck .Values.werf.env .Values.kafka.name | first | default .Values.kafka.name._default) -}}
{{- $_ := set . "vault_port" (pluck .Values.werf.env .Values.vault.port | first | default .Values.vault.port._default) -}}
{{- $_ := set . "vault_cluster_port" (pluck .Values.werf.env .Values.vault.cluster_port | first | default .Values.vault.cluster_port._default) -}}
{{- $_ := set . "vault_data_path" (pluck .Values.werf.env .Values.vault.data_path | first | default .Values.vault.data_path._default) -}}
{{- $_ := set . "vault_storage_class" (pluck .Values.werf.env .Values.vault.storage_class | first | default .Values.vault.storage_class._default) -}}
{{- $_ := set . "vault_storage_size" (pluck .Values.werf.env .Values.vault.storage_size | first | default .Values.vault.storage_size._default) -}}
{{- $_ := set . "vault_ha" (pluck .Values.werf.env .Values.vault.ha | first | default .Values.vault.ha._default) -}}
{{- end -}}

{{- define "vault.env" -}}
- name: VAULT_LOG_LEVEL
  value: debug
- name: RLIMIT_CORE
  value: "0"
- name: HOST_IP
  valueFrom:
    fieldRef:
      fieldPath: status.hostIP
- name: POD_IP
  valueFrom:
    fieldRef:
      fieldPath: status.podIP
- name: VAULT_ADDR
  value: {{ printf "%s://127.0.0.1:%s" .vault_scheme .vault_port | quote }}
- name: VAULT_API_ADDR
  value: {{ printf "%s://$(POD_IP):%s" .vault_scheme .vault_port | quote }}
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
  value: {{ printf "https://$(HOSTNAME).%s:%s"  .vault_cluster_port | quote }}
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

{{- define "vault.volumes" -}}
- name: config
  configMap:
    name: vault-auth
    defaultMode: 0644
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

#{{- define "vault.probes" -}}
#to do probes https://github.com/hashicorp/vault-helm/blob/9efd98a30f9d13ff003b91dd445339f9d99c424a/templates/server-statefulset.yaml
#{{- end -}}

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