package envvar

import (
	"fmt"
	"os"
	"strings"
)

// Just used for displaying help.
type envVar struct {
	key, usage string
	mandatory  bool
	value      *string
}

var envVars []envVar

func Usage() string {
	var usage []string
	for _, elmt := range envVars {
		isMandatory := ""
		if elmt.mandatory {
			isMandatory = " (mandatory)"
		}
		usage = append(usage, fmt.Sprintf("  %s\n    \t%s%s", elmt.key, elmt.usage, isMandatory))
	}
	return strings.Join(usage, "\n")
}

func Getenv(key string, usage string) *string {
	value := ""
	envVars = append(envVars, envVar{key: key, usage: usage, value: &value})
	return &value
}

func Parse() {
	for _, v := range envVars {
		*v.value = os.Getenv(v.key)
	}
}
