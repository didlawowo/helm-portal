{{/*
Expand the name of the chart.
*/}}
{{- define "application.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}




{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "application.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Generate labels.
*/}}
{{- define "application.labels" -}}
helm.sh/chart: {{ template "application.chart" . }}
{{ include "application.selectorLabels" . }}
app.kubernetes.io/part-of: {{ template "application.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
{{- end }}
{{- if .Values.commonLabels}}
{{ toYaml .Values.commonLabels }}
{{- end }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "application.selectorLabels" -}}
app.kubernetes.io/name: {{ template "application.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
