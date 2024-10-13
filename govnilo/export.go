package govnilo

import (
	"github.com/HazyCorp/govnilo/govnilo/internal/cmd/govnilo/cmd"
	"github.com/HazyCorp/govnilo/govnilo/internal/hazycheck"
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

	RegisterChecker = hazycheck.RegisterChecker
	RegisterSploit  = hazycheck.RegisterSploit
)
