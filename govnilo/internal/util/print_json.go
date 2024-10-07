package util

import (
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"
)

func PrintJson(obj interface{}) {
	if err := WriteJsonTo(obj, os.Stdout); err != nil {
		panic(err)
	}
}

func WriteJsonTo(obj interface{}, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(obj); err != nil {
		return errors.Wrap(err, "cannot encode object to json")
	}

	return nil
}
