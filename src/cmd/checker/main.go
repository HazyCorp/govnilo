package main

import (
	"github.com/HazyCorp/checker/cmd/checker/cmd"
	_ "github.com/HazyCorp/checker/services/example"
)

func main() {
	cmd.Execute()
}
