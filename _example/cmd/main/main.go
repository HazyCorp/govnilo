package main

import (
	"github.com/HazyCorp/govnilo/_example/checkers/sleeper"
	"github.com/HazyCorp/govnilo/pkg/govnilo"
)

func main() {
	govnilo.RegisterChecker(sleeper.NewSleeperPutChecker)
	govnilo.RegisterChecker(sleeper.NewSleeperGetChecker)

	govnilo.RegisterConstructor(sleeper.NewSleeperStorage)

	govnilo.Execute()
}
