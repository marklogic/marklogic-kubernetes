{{/*
Expand the name of the chart.
*/}}
{{- define "marklogic.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* 
newFullname is the name used after 1.1.x release, in an effort to make the release name shorter.
*/}}
{{- define "marklogic.newFullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}


{{/* 
oldFullname is the name used before 1.1.x release
*/}}
{{- define "marklogic.oldFullname" -}}
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

{{- define "marklogic.shouldUseNewName" -}}
{{- if .Release.IsInstall -}}
{{- true }}
{{- else }}
{{- if eq .Values.useLegacyHostnames true -}}
{{- false }}
{{- else }}
{{- true }}
{{- end }}
{{- end }}
{{- end }}

{{- define "marklogic.checkUpgradeError" -}}
{{- if and .Release.IsUpgrade (ne .Values.useLegacyHostnames true) -}}
{{- $stsName := trim (include "marklogic.oldFullname" .) -}}
{{- if .Values.fullnameOverride  -}}
{{- $stsName := trim .Values.fullnameOverride -}}
{{- end }}
{{- $sts := lookup "apps/v1" "StatefulSet" .Release.Namespace $stsName }}
{{- if $sts }}
{{- $labels := $sts.metadata.labels }}
{{- $chartVersionFull := get $labels "helm.sh/chart" }}
{{- if $chartVersionFull }}
{{- $chartVersionWithDot := trimPrefix "marklogic-" $chartVersionFull }}
{{- $chartVersionString := $chartVersionWithDot | replace "." "" }}
{{- $chartVersionDigit := int $chartVersionString }}
{{- if lt $chartVersionDigit 110 -}}
{{- $errorMessage := printf "A new algorithm for generating hostnames was introduced in version 1.1.0. When upgrading from version %s to version %s, the \"useLegacyHostnames\" setting must be set to true to prevent the StatefulSet from being recreated. Please add the following to the values file and attempt the upgrade again: \n\nuseLegacyHostnames: true\n" $chartVersionWithDot .Chart.Version }}
{{- fail $errorMessage }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}

{{/*
{{- end }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
To surrport the upgrade from 1.0.x to 1.1.x, we keep the old name when doing upgrade from 1.0.x.
For the new install, we use the new name, which is the release name.
*/}}
{{- define "marklogic.fullname" -}}
{{- if eq (include "marklogic.shouldUseNewName" .) "true" -}}
{{- include "marklogic.newFullname" . }}
{{- else }}
{{- include "marklogic.oldFullname" . }}
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
{{- if eq (include "marklogic.shouldUseNewName" .) "true" -}}
{{- include "marklogic.newFullname" . }}
{{- else }}
{{- printf "%s-headless" (include "marklogic.oldFullname" .) }}
{{- end }}
{{- end }}
{{/*
{{- end}}


{{/*
Create cluster service name for statefulset
*/}}
{{- define "marklogic.clusterServiceName" -}}
{{- if eq (include "marklogic.shouldUseNewName" .) "true" -}}
{{- include "marklogic.newFullname" . }}-cluster
{{- else }}
{{- include "marklogic.oldFullname" . }}
{{- end }}
{{- end }}
{{/*
{{- end}}


{{/*
Create URL for headless service 
*/}}
{{- define "marklogic.headlessURL" -}}
{{- printf "%s.%s.svc.%s" (include "marklogic.headlessServiceName" .) .Release.Namespace .Values.clusterDomain }}
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
Get the name for secret that is used for auth and managed by the Chart.
*/}}
{{- define "marklogic.authSecretName" -}}
{{- printf "%s-admin" (include "marklogic.fullname" .) }}
{{- end }}

{{/*
Get the secret name to mount to statefulSet.
Use the auth.secretName value if set, otherwise use the name from marklogic.authSecretName.
*/}}
{{- define "marklogic.authSecretNameToMount" -}}
{{- if .Values.auth.secretName }}
{{- .Values.auth.secretName }}
{{- else }}
{{- include "marklogic.authSecretName" . }}
{{- end }}
{{- end }}

{{/*
Fully qualified domain name
*/}}
{{- define "marklogic.fqdn" -}}
{{- printf "%s-0.%s.%s.svc.%s" (include "marklogic.fullname" .) (include "marklogic.headlessServiceName" .) .Release.Namespace .Values.clusterDomain }}
{{- end}}

{{/*
Validate values file
*/}}
{{- define "marklogic.checkInputError" -}}
{{- $fqdn := include "marklogic.fqdn" . }}
{{- if and (gt (len $fqdn) 64) (not .Values.allowLongHostnames) }}
{{- $errorMessage := printf "%s%s%s" "The FQDN: " $fqdn " is longer than 64. Please use a shorter release name and try again. MarkLogic App Server does not support turning on SSL with FQDN over 64 characters. If you still want to install with an FQDN longer than 64 characters, you can override this restriction by setting allowLongHostnames: true in your Helm values file." }}
{{- fail $errorMessage }}
{{- end }}
{{- end }}

{{/*
Name to distinguish marklogic image whether root or rootless
*/}}
{{- define "marklogic.imageType" -}}
{{- if .Values.image.tag | contains "rootless" }}
{{- printf "rootless" }}
{{- else }}
{{- printf "root" }}
{{- end }}
{{- end }}

