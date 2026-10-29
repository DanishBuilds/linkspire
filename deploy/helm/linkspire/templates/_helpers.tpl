{{/* Common naming + labels helpers. */}}
{{- define "linkspire.fullname" -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "linkspire.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "linkspire.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/* Chart-wide labels (no per-service component). */}}
{{- define "linkspire.commonLabels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | quote }}
app.kubernetes.io/part-of: linkspire
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end -}}

{{/* Resolve the image reference for a service dict passed as "svc". */}}
{{- define "linkspire.image" -}}
{{- $root := .root -}}
{{- $svc := .svc -}}
{{- $tag := $root.Values.global.image.tag | default $root.Chart.AppVersion -}}
{{- printf "%s/%s:%s" $root.Values.global.image.registry $svc.image $tag -}}
{{- end -}}
