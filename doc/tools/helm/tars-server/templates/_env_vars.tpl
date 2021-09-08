{{- define "tserver.env-vars" -}}
- name: Namespace
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
- name: PodName
  valueFrom:
    fieldRef:
      fieldPath: metadata.name
- name: PodIP
  valueFrom:
    fieldRef:
      fieldPath: status.podIP
- name: ServerApp
  valueFrom:
    fieldRef:
      fieldPath: metadata.labels['tars.io/ServerApp']
{{- if .Values.envVars -}}
{{ range .Values.envVars }}
- name: {{ .name }}
  value: {{ .value | quote }}
{{- end }}
{{- end }}
{{- end }}
