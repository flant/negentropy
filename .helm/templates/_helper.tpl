{{- define "vault.envs" }}
{{- $vault_port := pluck .Values.werf.env .Values.vault_port | first | default .Values.vault_port._default }}
env:
  - name: VAULT_LOG_LEVEL
    value: debug
  - name: RLIMIT_CORE
    value: "0"
{{- end }}
{{- define "vault.securitycontext" }}
readOnlyRootFilesystem: true
runAsNonRoot: true
capabilities:
  add:
    - IPC_LOCK
{{- end }}