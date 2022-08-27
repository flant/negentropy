{{- define "globals" }}
{{- $_ := set . "kafka_name" (pluck .Values.werf.env .Values.kafka.name | first | default .Values.kafka.name._default) }}
{{- $_ := set . "secrets_path" (pluck .Values.werf.env .Values.secrets_path | first | default .Values.secrets_path._default) }}
{{- $_ := set . "ca_path" (pluck .Values.werf.env .Values.ca_path | first | default .Values.ca_path._default) }}
{{- $_ := set . "vault_port" (pluck .Values.werf.env .Values.vault_port | first | default .Values.vault_port._default) }}
{{- $_ := set . "oidc_url" (pluck .Values.werf.env .Values.oidc_url | first | default .Values.oidc_url._default) }}

{{- end }}

{{- define "vault.env" -}}
- name: VAULT_LOG_LEVEL
  value: debug
- name: RLIMIT_CORE
  value: "0"
- name: NEGENTROPY_KAFKA_ENDPOINTS
  value: {{ printf "%s-kafka-bootstrap" .kafka_name }}
- name: NEGENTROPY_KAFKA_USE_SSL
  value: "true"
- name: NEGENTROPY_KAFKA_SSL_CA_PATH
  value: {{ printf "%s/ca.crt" .ca_path }}
- name: NEGENTROPY_KAFKA_SSL_CLIENT_PRIVATE_KEY_PATH
  value: {{ printf "%s/user.key" .secrets_path }}
- name: NEGENTROPY_KAFKA_SSL_CLIENT_CERTIFICATE_PATH
  value: {{ printf "%s/user.crt" .secrets_path }}
- name: NEGENTROPY_OIDC_URL
  value: "{{ .oidc_url }}"
{{- end -}}

{{- define "vault.securitycontext" -}}
readOnlyRootFilesystem: true
# runAsNonRoot: true
capabilities:
  add:
  - IPC_LOCK
{{- end -}}

{{- define "vault.volumemounts" -}}
- name: config
  mountPath: /etc/vault.hcl
  subPath: vault.hcl
  readOnly: true
- name: kafka-secrets
  mountPath: {{ .secrets_path }}
- name: kafka-ca
  mountPath: {{ .ca_path }}
- name: tmp
  mountPath: /tmp/vault
{{- end -}}

{{- define "vault.volumes" -}}
- name: config
  configMap:
    name: vault-auth
    defaultMode: 0644
- name: kafka-secrets
  secret:
    secretName: vault
- name: kafka-ca
  secret:
    secretName: {{ printf "%s-cluster-ca-cert" .kafka_name }}
- emptyDir: {}
  name: tmp    
{{- end -}}
