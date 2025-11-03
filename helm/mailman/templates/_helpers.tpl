{{/*
Expand the name of the chart.
*/}}
{{- define "mailman.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "mailman.fullname" -}}
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
{{- define "mailman.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "mailman.labels" -}}
helm.sh/chart: {{ include "mailman.chart" . }}
{{ include "mailman.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "mailman.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mailman.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "mailman.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "mailman.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create MySQL fullname
*/}}
{{- define "mailman.mysql.fullname" -}}
{{- if .Values.mysql.enabled }}
{{- printf "%s-%s" (include "mailman.fullname" .) "mysql" | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s" .Values.mysql.auth.host | quote }}
{{- end }}
{{- end }}

{{/*
Database secret name
*/}}
{{- define "mailman.database.secretName" -}}
{{- if .Values.secrets.database.existingSecret }}
{{- .Values.secrets.database.existingSecret }}
{{- else if .Values.secrets.database.create }}
{{- .Values.secrets.database.name }}
{{- else }}
{{- printf "%s-%s" (include "mailman.fullname" .) "mysql" | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Return the proper Docker Image Registry Secret Names
*/}}
{{- define "mailman.imagePullSecrets" -}}
{{- include "common.images.pullSecrets" (dict "images" (list .Values.mailman.backend.image .Values.mailman.frontend.image) "global" .Values.global) -}}
{{- end }}

{{/*
Get the backend image name
*/}}
{{- define "mailman.backend.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.mailman.backend.image "global" .Values.global) -}}
{{- end }}

{{/*
Get the frontend image name
*/}}
{{- define "mailman.frontend.image" -}}
{{ include "common.images.image" (dict "imageRoot" .Values.mailman.frontend.image "global" .Values.global) -}}
{{- end }}

{{/*
Validate values
*/}}
{{- define "mailman.validateValues" -}}
{{- $messages := list -}}
{{- if not .Values.mysql.auth.rootPassword -}}
{{- $messages = append $messages "MySQL root password is required" -}}
{{- end -}}
{{- if not .Values.mysql.auth.password -}}
{{- $messages = append $messages "MySQL user password is required" -}}
{{- end -}}
{{- if $messages -}}
{{- printf "\nVALUES VALIDATION:\n%s" (join "\n" $messages) | fail -}}
{{- end -}}
{{- end -}}

{{/*
Render the templates
*/}}
{{- template "mailman.validateValues" . }}