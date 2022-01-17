{{- define "tserver.deployment" -}}
{{- $lower_app := lower .Values.app -}}
{{- $lower_server := lower .Values.server -}}
{{- $id := (printf "%s-%s" $lower_app $lower_server) -}}

apiVersion: k8s.tars.io/{{ .Chart.AppVersion }}
kind: TServer
metadata:
  name: {{ $id }}
  labels:
    tars.io/ServerApp: {{ .Values.app }}
    tars.io/ServerName: {{ .Values.server }}
    tars.io/SubType: {{ .Values.subtype | default "tars" }}
    tars.io/Template: {{ .Values.template }}
  annotations:
    name: {{ $id }}
spec:
  app: {{ .Values.app }}
  server: {{ .Values.server }}
  subType: tars
  important: 5
  tars:
    template: {{ .Values.template }}
    profile: {{ .Values.profile | quote }}
    asyncThread: {{ .Values.asyncThread | default 3 }}
{{- if .Values.servants }}    
    servants:
{{- range .Values.servants }}
    - name: {{ .name }}
      port: {{ .port }}
      isTars: {{ .isTars }}
      thread: {{ .thread | default 5 }}
      capacity: {{ .capacity | default 100000 }}
      connection: {{ .connection | default 100000 }}
      isTcp: {{ .isTcp }}
      timeout: {{ .timeout | default 60000 }}
{{- end }}
{{- else }}    
    servants: []
{{- end }}
  k8s:
{{- if .Release.IsUpgrade }}  
    replicas: {{ (lookup (printf "k8s.tars.io/%s" $.Chart.AppVersion) "TServer" $.Release.Namespace $id ).spec.k8s.replicas }}
{{- else }}    
    replicas: {{ .Values.replicas | default 1 }}
{{- end }}    
    hostNetwork: {{ .Values.hostNetwork }}
    hostIPC: {{ .Values.hostIPC }}
{{- if .Values.hostPorts }}
    hostPorts:
{{- include "tserver.host-ports" . | indent 6 }}
{{- end}}
{{- if .Values.nodeSelector}}      
    nodeSelector:
{{ toYaml .Values.nodeSelector | indent 6}}
{{- else}}
    nodeSelector: []
{{- end}}      
    abilityAffinity: {{ .Values.abilityAffinity | default "AppOrServerPreferred" }}
    notStacked: {{ .Values.notStacked | default false }}
    daemonSet: {{ .Values.daemonSet | default false }}
    imagePullPolicy: {{ .Values.imagePullPolicy | default "Always" }}
    launcherType: {{ .Values.launcherType | default "background" }}
{{- if .Values.updateStrategy}}      
{{ toYaml .Values.updateStrategy | indent 6}}
{{- else}}
    updateStrategy: 
      rollingUpdate:
        partition: 0
      type: RollingUpdate
 {{- end}}       
    podManagementPolicy: {{ .Values.podManagementPolicy | default "Parallel" }}
    env:
{{ include "tserver.env-vars" . | indent 6 }}
    mounts:
    - name: host-log-dir
      source:
        hostPath:
          path: /usr/local/app/tars/app_log
          type: DirectoryOrCreate
      mountPath: /usr/local/app/tars/app_log
      subPathExpr: $(Namespace)/$(PodName)    
{{- if .Values.mounts}}      
{{ toYaml .Values.mounts | indent 4}}
{{- end}}
  release:
    id: {{ .Values.repo.id | quote }}
    image: {{ .Values.repo.image }}
    secret: {{ .Values.repo.secret }}

---

{{ range .Values.config }}
apiVersion: k8s.tars.io/{{ $.Chart.AppVersion }}
kind: TConfig
metadata:
  name: {{ $id }}-{{ .name | lower | replace "." "-" }}-{{ now | unixEpoch }}
  annotations: 
    helm.sh/resource-policy: keep  
app: {{ $.Values.app }}
server: {{ $.Values.server }}
configName: {{ .name }}
configContent: {{ .content | quote }}
updatePerson: {{ $.Values.user | default "helm" | quote }}
updateReason: {{ $.Values.reason  | default "helm install" | quote }}
activated: true

---

{{- end}}

{{ range .Values.nodeConfig }}
apiVersion: k8s.tars.io/{{ $.Chart.AppVersion }}
kind: TConfig
metadata:
  name: {{ $id }}-{{ .name | lower | replace "." "-" }}-{{ .podSeq }}-{{ now | unixEpoch }}
  annotations: 
    helm.sh/resource-policy: keep  
app: {{ $.Values.app }}
server: {{ $.Values.server }}
podSeq: {{ .podSeq | quote }}
configName: {{ .name }}
configContent: {{ .content | quote }}
updatePerson: {{ $.Values.user | default "helm" | quote }}
updateReason: {{ $.Values.reason  | default "helm install" | quote }}
activated: true

---

{{- end}}

{{- if .Values.appConfig }}
{{ range .Values.appConfig }}
apiVersion: k8s.tars.io/{{ $.Chart.AppVersion }}
kind: TConfig
metadata:
  name: {{ $id }}-{{ .name | lower | replace "." "-" }}-{{ now | unixEpoch }}
  annotations: 
    helm.sh/resource-policy: keep
app: {{ $.Values.app }}
server: ""
configName: {{ .name }}
configContent: {{ .content | quote }}
updatePerson: {{ $.Values.user | default "helm" | quote }}
updateReason: {{ $.Values.reason  | default "helm install" | quote }}
activated: true

---

{{ end }}
{{- end }}

apiVersion: k8s.tars.io/{{ $.Chart.AppVersion }}
kind: TImage
metadata:
  name: {{ $id }}
  labels:
    tars.io/ImageType: server
    tars.io/ServerApp: {{ .Values.app }}
    tars.io/ServerName: {{ .Values.server }}
imageType: server
releases:
- image: {{ .Values.repo.image }}
  secret: {{ .Values.repo.secret }}
  id: {{ .Values.repo.id }}
  createPerson: {{ $.Values.user | default "helm" | quote }}
  mark: {{ $.Values.reason | default "helm install" | quote }}
{{- range $index, $service := (lookup (printf "k8s.tars.io/%s" $.Chart.AppVersion) "TImage" $.Release.Namespace $id ).releases }}  
- image: {{ $service.image }}
  secret: {{ $service.secret }}
  id: {{ $service.id }}
  createPerson: {{ $service.createPerson | default "helm" }}
  mark: {{ $service.mark | default "helm install" | quote }}
  createTime: {{ $service.createTime }}
{{- end}}

{{- end }}
