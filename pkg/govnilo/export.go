package govnilo

import (
	"github.com/HazyCorp/govnilo/internal/cmd/cmd"
	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"
)

type (
	Checker   = hazycheck.Checker
	CheckerID = hazycheck.CheckerID
	Sploit    = hazycheck.Sploit
	SploitID  = hazycheck.SploitID

	Provider = hazycheck.Connector
)

var (
	Execute = cmd.Execute

	RegisterChecker     = hazycheck.RegisterChecker
	RegisterSploit      = hazycheck.RegisterSploit
	RegisterConstructor = hazycheck.RegisterConstructor

	InternalError = hazycheck.InternalError

	GetLogger = hzlog.GetLogger
)
