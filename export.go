package govnilo

import (
	"github.com/HazyCorp/govnilo/cmd/checker/cmd"
	"github.com/HazyCorp/govnilo/pkg/hazycheck"
)

type Checker = hazycheck.Checker

type Sploit = hazycheck.Sploit

type Provider = hazycheck.Provider

var Execute = cmd.Execute

var RegisterChecker = hazycheck.RegisterChecker

var RegisterSploit = hazycheck.RegisterSploit
