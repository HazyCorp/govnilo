package hazycheck

import (
	"github.com/HazyCorp/govnilo/internal/registrar"
)

func RegisterConstructor(constructor any) {
	registrar.Register(constructor)
}
