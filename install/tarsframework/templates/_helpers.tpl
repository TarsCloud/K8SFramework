{{- define "ImagePullSecret"}}
{{- printf "{\"auths\": {\"%s\": {\"auth\": \"%s\"}}}" .Values.repository.url (printf "%s:%s" .Values.repository.user .Values.repository.password | b64enc) | b64enc }}
{{- end}}
