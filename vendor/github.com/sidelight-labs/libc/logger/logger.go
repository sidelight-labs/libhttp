package logger

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"runtime"
)

const TestingEnvironmentVariable = "TESTING"

func Log(message string) {
	if _, testing := os.LookupEnv(TestingEnvironmentVariable); !testing {
		pc, _, line, ok := runtime.Caller(1)
		details := runtime.FuncForPC(pc)
		if ok && details != nil {
			fmt.Printf("%s[%d]: %s\n", details.Name(), line, message)
		}
	}
}

func Wrap(err error, message string) error {
	pc, _, line, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		err = errors.Wrap(err, fmt.Sprintf("%s[%d]: %s", details.Name(), line, message))
	}
	return err
}
