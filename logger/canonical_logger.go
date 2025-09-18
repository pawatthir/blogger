package logger

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log/slog"
	"strings"
	"time"
)

var canonicalLogTemplate *template.Template

type Level int

const (
	Debug Level = 1 << iota
	Info
	Warn
	Error
)

type CanonicalLog struct {
	Transport string
	Traffic   string
	Method    string
	Status    int
	Path      string
	Duration  time.Duration
	Message   string
	Level     slog.Level
}

type ExceptionError struct {
	Code            int
	GlobalMessage   string
	DebugMessage    string
	APIStatusCode   int
	ErrFields       map[string]interface{}
	OverrideLogLevel bool
	Level           string
	StackErrors     []StackError
}

type StackError struct {
	Kind    string
	Message string
	Stack   string
}

func (e *ExceptionError) Error() string {
	return e.DebugMessage
}

func CompileCanonicalLogTemplate() {
	logTemplate := "[{{.Transport}}][{{.Traffic}}] {{.Method}} {{.Status}} {{.Path}} {{.Duration}} - {{.Message}}"
	compiled, err := template.New("log_template").Parse(logTemplate)
	if err != nil {
		panic(err)
	}
	canonicalLogTemplate = compiled
}

func GetCanonicalLogTemplate() (*template.Template, error) {
	if canonicalLogTemplate != nil {
		return canonicalLogTemplate, nil
	}
	return nil, errors.New("canonicalLogTemplate is nil")
}

func CanonicalLogger(ctx context.Context, slogger slog.Logger, level Level, request []byte, response []byte, err error, canonicalLog CanonicalLog, metadata []any) {
	logKey := canonicalLog.Path
	var reqFields []any

	var jsonObj map[string]interface{}
	if unmarshalErr := json.Unmarshal(request, &jsonObj); unmarshalErr != nil {
		reqFields = append(reqFields, slog.String("request", string(request)))
	} else {
		reqFields = append(reqFields, slog.Any("request", jsonObj))
	}

	shouldSanitize := Sanitize(logKey)
	if shouldSanitize {
		reqFields = []any{slog.String("request", "REDACTED")}
	}

	var respFields []any
	if err != nil {
		level = Error
		cErr, ok := err.(*ExceptionError)
		if ok && cErr != nil {
			if cErr.StackErrors != nil {
				stackTrace := GetStackField(cErr.StackErrors)
				stackTraceParts := strings.Split(stackTrace.Stack, "\n\t")
				if len(stackTraceParts) > 6 {
					stackTrace.Stack = strings.Join(stackTraceParts[:6], "\n\t")
				}
				respFields = append(respFields, slog.Group("error",
					slog.String("kind", stackTrace.Kind),
					slog.String("message", stackTrace.Message),
					slog.String("stack", stackTrace.Stack),
				))
			}
			respFields = append(respFields, slog.Group("response",
				slog.Int("status_code", cErr.APIStatusCode),
				slog.Any("data", nil),
				slog.Group("error",
					slog.Int("code", cErr.Code),
					slog.String("message", cErr.GlobalMessage),
					slog.String("debug_message", cErr.DebugMessage),
					slog.Any("details", cErr.ErrFields),
				)))
			canonicalLog.Message = cErr.DebugMessage
		} else {
			var jsonObj map[string]interface{}
			if err := json.Unmarshal(response, &jsonObj); err != nil {
				respFields = append(respFields, slog.String("response", string(response)))
			} else {
				respFields = append(respFields, slog.Any("response", jsonObj))
			}
		}
	} else {
		level = Info
		var jsonObj map[string]interface{}
		if err := json.Unmarshal(response, &jsonObj); err != nil {
			respFields = append(respFields, slog.String("response", string(response)))
		} else {
			respFields = append(respFields, slog.Any("response", jsonObj))
		}
	}

	if shouldSanitize {
		respFields = []any{slog.String("response", "REDACTED")}
	}

	var mdFields []any
	mdFields = append(mdFields,
		slog.String("logger_name", "canonical"),
		slog.Group("md", metadata...),
	)

	var logMsgBuilder strings.Builder
	var logMsg string
	logTmpl, logTmplErr := GetCanonicalLogTemplate()
	if logTmplErr != nil {
		logMsg = "failed to get canonical log template"
	} else {
		executeErr := logTmpl.Execute(&logMsgBuilder, canonicalLog)
		if executeErr != nil {
			logMsg = "failed to execute canonical log template"
		} else {
			logMsg = logMsgBuilder.String()
		}
	}

	fields := append(reqFields, respFields...)
	fields = append(fields, mdFields...)

	switch level {
	case Debug:
		slogger.DebugContext(ctx, logMsg, fields...)
	case Info:
		slogger.InfoContext(ctx, logMsg, fields...)
	case Warn:
		slogger.WarnContext(ctx, logMsg, fields...)
	case Error:
		slogger.ErrorContext(ctx, logMsg, fields...)
	default:
		slogger.ErrorContext(ctx, logMsg, fields...)
	}
}

var DenyPatterns = []string{
	"login",
	"refresh-token",
	"verify-otp",
	"password",
	"token",
	"secret",
	"key",
}

func Sanitize(logKey string) bool {
	logKeyLower := strings.ToLower(logKey)
	for _, denyPattern := range DenyPatterns {
		if strings.Contains(logKeyLower, denyPattern) {
			return true
		}
	}
	return false
}

func GetStackField(stackErrors []StackError) StackError {
	if len(stackErrors) > 0 {
		return stackErrors[0]
	}
	return StackError{
		Kind:    "unknown",
		Message: "no stack information available",
		Stack:   "",
	}
}