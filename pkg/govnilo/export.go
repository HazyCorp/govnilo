package govnilo

import (
	"github.com/HazyCorp/govnilo/cmd/govnilo/cmd"
	"github.com/HazyCorp/govnilo/hazycheck"
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
