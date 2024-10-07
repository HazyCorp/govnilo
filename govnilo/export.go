package govnilo

import (
	"github.com/HazyCorp/govnilo/govnilo/cmd/checker/cmd"
	"github.com/HazyCorp/govnilo/govnilo/internal/hazycheck"
)

type (
	Checker   = hazycheck.Checker
	CheckerID = hazycheck.CheckerID
	Sploit    = hazycheck.Sploit
	SploitID  = hazycheck.SploitID

	Provider = hazycheck.Provider
)

var (
	Execute = cmd.Execute

	RegisterChecker = hazycheck.RegisterChecker
	RegisterSploit  = hazycheck.RegisterSploit
)
