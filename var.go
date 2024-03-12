package kiwi

import (
	"flag"
	"fmt"
	"github.com/15mga/kiwi/util"
	"os"
	"strconv"
	"strings"

	"github.com/15mga/kiwi/ds"
)

type varItem struct {
	name  string
	usage string
	val   any
}

type Var interface {
	int | int64 | float64 | bool | string
}

var (
	_VarMap = ds.NewKSet[string, *varItem](4, func(item *varItem) string {
		return item.name
	})
)

func AddVar[T Var](name string, def T, usage string) {
	_VarMap.Set(&varItem{
		name:  name,
		val:   def,
		usage: usage,
	})
}

func ParseVar() {
	parseFlag()
	parseEnv()
	m := util.M{}
	for _, v := range _VarMap.Values() {
		m[v.name] = v.val
	}
}

func parseFlag() {
	m := make(map[string]any)
	for _, item := range _VarMap.Values() {
		switch d := item.val.(type) {
		case int:
			var v int
			flag.IntVar(&v, item.name, d, item.usage)
			m[item.name] = &v
		case int64:
			var v int64
			flag.Int64Var(&v, item.name, d, item.usage)
			m[item.name] = &v
		case float64:
			var v float64
			flag.Float64Var(&v, item.name, d, item.usage)
			m[item.name] = &v
		case bool:
			var v bool
			flag.BoolVar(&v, item.name, d, item.usage)
			m[item.name] = &v
		case string:
			var v string
			flag.StringVar(&v, item.name, d, item.usage)
			m[item.name] = &v
		}
	}
	flag.Parse()
	for _, item := range _VarMap.Values() {
		v := m[item.name]
		switch d := v.(type) {
		case *int:
			item.val = *d
		case *int64:
			item.val = *d
		case *float64:
			item.val = *d
		case *bool:
			item.val = *d
		case *string:
			item.val = *d
		}
	}
}

func parseEnv() {
	for _, item := range _VarMap.Values() {
		v, ok := os.LookupEnv(strings.ToUpper(item.name))
		if ok {
			switch item.val.(type) {
			case int:
				i, err := strconv.Atoi(v)
				if err != nil {
					fmt.Println(err.Error())
				}
				item.val = i
			case int64:
				i, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					fmt.Println(err.Error())
				}
				item.val = i
			case float64:
				i, err := strconv.ParseFloat(v, 64)
				if err != nil {
					fmt.Println(err.Error())
				}
				item.val = i
			case bool:
				item.val = strings.ToLower(v) == "true"
			case string:
				item.val = v
			}
		}
	}
}

func GetVar[T Var](name string) (T, bool) {
	o, ok := _VarMap.Get(name)
	if !ok {
		return util.Default[T](), false
	}
	v, ok := o.val.(T)
	return v, ok
}
