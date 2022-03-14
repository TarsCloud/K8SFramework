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

{{/* Allow KubeVersion to be overridden. */}}
{{- define "tars.kubeVersion" }}
  {{- default .Capabilities.KubeVersion.Version .Values.kubeVersionOverride -}}
{{- end }}

{{/* Get Ingress API Version */}}
{{- define "tars.ingress.apiVersion" }}
  {{- if and (.Capabilities.APIVersions.Has "networking.k8s.io/v1") (semverCompare ">= 1.19-0" (include "tars.kubeVersion" .)) }}
      {{- print "networking.k8s.io/v1" }}
  {{- else if .Capabilities.APIVersions.Has "networking.k8s.io/v1beta1" }}
    {{- print "networking.k8s.io/v1beta1" }}
  {{- else }}
    {{- print "extensions/v1beta1" }}
  {{- end }}
{{- end }}

{{/* Check Ingress stability */}}
{{- define "tars.ingress.isStable" }}
  {{- eq (include "tars.ingress.apiVersion" .) "networking.k8s.io/v1" }}
{{- end }}

{{/* Check Ingress supports pathType */}}
{{/* pathType was added to networking.k8s.io/v1beta1 in Kubernetes 1.18 */}}
{{- define "tars.ingress.supportsPathType" }}
  {{- or (eq (include "tars.ingress.isStable" .) "true") (and (eq (include "tars.ingress.apiVersion" .) "networking.k8s.io/v1beta1") (semverCompare ">= 1.18-0" (include "tars.kubeVersion" .))) }}
{{- end }}
