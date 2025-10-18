package govnilo

import (
	"github.com/HazyCorp/govnilo/cmd/govnilo/cmd"
	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"
)

type (
	Checker   = hazycheck.Checker
	CheckerID = hazycheck.CheckerID
	Sploit    = hazycheck.Sploit
	SploitID  = hazycheck.SploitID
	TraceID   = hzlog.TraceID

	Provider = hazycheck.Connector
)

var (
	Execute = cmd.Execute

	RegisterChecker = hazycheck.RegisterChecker
	RegisterSploit  = hazycheck.RegisterSploit

	InternalError = hazycheck.InternalError

	// Trace ID functions (read-only for developers)
	GetTraceID     = hzlog.GetTraceID
	MustGetTraceID = hzlog.MustGetTraceID
	LogWithTraceID = hzlog.LogWithTraceID
	FormatTraceID  = hzlog.FormatTraceID
	GetLogger      = hzlog.GetLogger
)
