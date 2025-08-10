package logging

import "regexp"

var (
    reAuthBearer   = regexp.MustCompile(`(?i)(Authorization: Bearer )([A-Za-z0-9\-\._~\+\/]+=*)`)
    reJSONSecrets  = regexp.MustCompile(`(?i)\"(password|token|secret|api_key)\"\s*:\s*\"[^\"]*\"`)
)

// Redact returns a sanitized copy of the input string.
func Redact(s string) string {
    s = reAuthBearer.ReplaceAllString(s, "$1[REDACTED]")
    s = reJSONSecrets.ReplaceAllString(s, `"$1":"[REDACTED]"`)
    return s
}


