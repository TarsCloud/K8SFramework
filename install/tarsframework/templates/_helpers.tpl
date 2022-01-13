{{- define "ImagePullSecret" }}
{{- printf "{\"auths\": {\"%s\": {\"auth\": \"%s\"}}}" .Values.repository.url (printf "%s:%s" .Values.repository.user .Values.repository.password | b64enc) | b64enc }}
{{- end }}

{{- define "TImageMerger"}}
{{- $releases := (lookup (printf "k8s.tars.io/%s" .version) "TImage" .namespace .name).releases}}
{{- if $releases }}
{{- range $release :=$releases}}
{{  printf "- id: %s" $release.id }}
{{- range $k,$v :=$release }}
{{- if ne $k "id" }}
{{- if ne $k "mark" }}
{{  printf "  %s: %s" $k $v }}
{{- else }}
{{  printf "  mark: |%s" (toString $v |nindent 4) }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
