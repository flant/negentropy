{{- define "vault.env" }}
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