package main

import (
	_ "github.com/HazyCorp/govnilo/_example/checkers/sleeper"
	_ "github.com/HazyCorp/govnilo/_example/sploits/example"
	"github.com/HazyCorp/govnilo/pkg/govnilo"
)

func main() {
	govnilo.Execute()
}
