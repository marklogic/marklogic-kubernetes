{{/*
Expand the name of the chart.
*/}}
{{- define "marklogic.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "marklogic.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "marklogic.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create headless service name for statefulset
*/}}
{{- define "marklogic.headlessServiceName" -}}
{{- printf "%s-headless" (include "marklogic.fullname" .) }}
{{- end}}


{{/*
Create URL for headless service 
*/}}
{{- define "marklogic.headlessURL" -}}
{{- printf "%s.%s.svc.cluster.local" (include "marklogic.headlessServiceName" .) .Release.Namespace }}
{{- end}}

{{/*
Common labels
*/}}
{{- define "marklogic.labels" -}}
helm.sh/chart: {{ include "marklogic.chart" . }}
{{ include "marklogic.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "marklogic.selectorLabels" -}}
app.kubernetes.io/name: {{ include "marklogic.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "marklogic.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "marklogic.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Fully qualified domain name
*/}}
{{- define "marklogic.fqdn" -}}
{{- printf "%s-0.%s.%s.svc.cluster.local" (include "marklogic.fullname" .) (include "marklogic.headlessServiceName" .) .Release.Namespace }}
{{- end}}