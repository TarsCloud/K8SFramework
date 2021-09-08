{{- define "tserver.host-ports" -}}
{{- if .Values.hostPorts -}}
{{ range .Values.hostPorts }}
- nameRef: {{ .nameRef }}
  port: {{ .port }}
{{- end }}
{{- end }}
{{- end }}
