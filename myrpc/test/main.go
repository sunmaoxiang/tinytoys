package main

import (
	"log"
	"reflect"
	"strings"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	typ := reflect.TypeOf(&wg)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		argv := make([]string, 0, method.Type.NumIn())
		returns := make([]string, 0, method.Type.NumOut())
		// j 从 1开始，第0个入参是wg自己
		for j := 1; j < method.Type.NumIn(); j++ {
			argv = append(argv, method.Type.In(j).Name())
		}
		for j := 0; j < method.Type.NumOut(); j++ {
			returns = append(returns, method.Type.Out(j).Name())
		}
		a := typ.Elem().Name()
		log.Printf("func (w *%s) %s(%s) %s",
			a, method.Name, strings.Join(argv, ","), strings.Join(returns, ","))
	}

}
