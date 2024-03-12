package loader

import (
	"fmt"
	"github.com/15mga/kiwi"
	"path"
	"strings"

	"github.com/15mga/kiwi/util"
	"github.com/spf13/viper"
)

const (
	ConfLocalLoader = "local"
	ConfPathSep     = "|"
)

type ConfLoader func(path string, v *viper.Viper) *util.Err

var (
	_TypeToLoader   = make(map[string]ConfLoader)
	_ConfPathParser = func(path string) (string, string, *util.Err) {
		ss := strings.Split(path, ConfPathSep)
		if len(ss) != 2 {
			return "", "", util.NewErr(util.EcParamsErr, util.M{
				"path": path,
			})
		}
		return ss[0], ss[1], nil
	}
	_ConfRoot = util.WorkDir()
)

func init() {
	SetConfLoader(ConfLocalLoader, confLocalLoader)
}

func SetConfRoot(p string) {
	_ConfRoot = p
}

func SetConfPathParser(parser util.StrToStr2Err) {
	_ConfPathParser = parser
}

func LoadConf(conf any, paths ...string) {
	l := len(paths)
	if l == 0 {
		kiwi.Warn(util.NewErr(util.EcParamsErr, nil))
		return
	}
	vpr := viper.New()
	vpr.SetConfigType("yaml")
	for i := 0; i < l; i++ {
		p := paths[i]
		loaderType, filePath, err := _ConfPathParser(p)
		if err != nil {
			fmt.Printf("parse path %s fail %s\n", p, err.Error())
			continue
		}
		loader, ok := _TypeToLoader[loaderType]
		if !ok {
			kiwi.Error2(util.EcNotExist, util.M{
				"loader type": loaderType,
			})
			fmt.Printf("not exist type %s\n", loaderType)
			continue
		}
		err = loader(filePath, vpr)
		if err == nil {
			if i < l-1 {
				for k, v := range vpr.AllSettings() {
					vpr.SetDefault(k, v)
				}
			}
		} else {
			fmt.Printf("load %s fail %s\n", p, err.Error())
		}
	}
	err := vpr.Unmarshal(conf)
	if err != nil {
		fmt.Printf("unmarshal fail %s\n", err.Error())
	}
}

func SetConfLoader(typ string, loader ConfLoader) {
	_TypeToLoader[typ] = loader
}

func GetConfLoader(typ string) ConfLoader {
	return _TypeToLoader[typ]
}

func confLocalLoader(p string, v *viper.Viper) *util.Err {
	v.AddConfigPath(_ConfRoot)
	_, fn := path.Split(p)
	ext := path.Ext(p)[1:]
	switch ext {
	case "yml":
		ext = "yaml"
	}
	n := fn[0 : len(fn)-len(ext)]
	v.SetConfigName(n)
	v.SetConfigType(ext)
	err := v.ReadInConfig()
	if err != nil {
		return util.NewErr(util.EcParamsErr, util.M{
			"error": err.Error(),
			"path":  p,
		})
	}
	return nil
}

func ConvertConfLocalPath(paths ...string) []string {
	for i, p := range paths {
		paths[i] = ConfLocalLoader + ConfPathSep + p
	}
	return paths
}
