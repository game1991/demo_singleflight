package utils

import (
	"fmt"
	"runtime"
)

func GetCallers(r interface{}) string {
	callers := ""
	for i := 0; true; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		callers = callers + fmt.Sprintf("%v:%v\n", file, line)
	}

	return callers
}
