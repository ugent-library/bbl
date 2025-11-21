package bbl

// https://medium.com/@ancilartech/the-shocking-truth-behind-go-errors-and-how-to-fix-them-like-a-pro-602247a215cd

// import (
// 	"fmt"
// 	"runtime"
// )

// func Trace(err error) error {
// 	pcs := make([]uintptr, 1)
// 	runtime.Callers(2, pcs)
// 	fn := runtime.FuncForPC(pcs[0])
// 	file, line := fn.FileLine(pcs[0])
// 	return fmt.Errorf("%s: %w (%s at line %d)", fn.Name(), err, file, line)
// }
