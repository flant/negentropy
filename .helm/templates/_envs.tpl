{{- define "globals" }}
{{- $_ := set . "kafka_name" (pluck .Values.werf.env .Values.kafka.name | first | default .Values.kafka.name._default) }}
{{- $_ := set . "secrets_path" (pluck .Values.werf.env .Values.secrets_path | first | default .Values.secrets_path._default) }}
{{- $_ := set . "vault_port" (pluck .Values.werf.env .Values.vault_port | first | default .Values.vault_port._default) }}
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
  value: {{ printf "%s/ca.crt" .secrets_path }}
- name: NEGENTROPY_KAFKA_SSL_CLIENT_PRIVATE_KEY_PATH
  value: {{ printf "%s/user.key" .secrets_path }}
- name: NEGENTROPY_KAFKA_SSL_CLIENT_CERTIFICATE_PATH
  value: {{ printf "%s/user.crt" .secrets_path }}
- name: NEGENTROPY_OIDC_URL
  value: ""
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
{{- end -}}

{{- define "vault.volumes" -}}
- name: config
  configMap:
    name: vault-auth
    defaultMode: 0644
- name: kafka-secrets
  secret:
    secretName: vault
{{- end -}}
