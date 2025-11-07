package govnilo

import (
	"github.com/HazyCorp/govnilo/internal/cmd/cmd"
	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"
)

type (
	Checker   = hazycheck.Checker
	CheckerID = hazycheck.CheckerID

	Provider = hazycheck.Connector
)

var (
	Execute = cmd.Execute

	RegisterChecker     = hazycheck.RegisterChecker
	RegisterConstructor = hazycheck.RegisterConstructor

	InternalError = hazycheck.InternalError

	GetLogger = hzlog.GetLogger
)
