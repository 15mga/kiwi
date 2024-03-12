package loader

import (
	"github.com/15mga/kiwi"
	"io"
	"net/http"
	"os"

	"github.com/15mga/kiwi/util"
)

const (
	LocalLoader = "local"
	HttpLoader  = "http"
)

var (
	_Default *Loader
)

func SetDefaultLoader(defaultType string) {
	_Default = NewLoader(defaultType)
}

func BindParser(assetType string, marshaller util.BytesToAnyErr) {
	_Default.BindParser(assetType, marshaller)
}

func BindLoader(loaderType string, loader util.StrToBytesErr) {
	_Default.BindLoader(loaderType, loader)
}

func DefaultLoad(assetType string, action util.FnStrAny, paths ...string) {
	_Default.DefaultLoad(assetType, action, paths...)
}

func Load(assetType string, loaderType string, action util.FnStrAny, paths ...string) {
	_Default.Load(assetType, loaderType, action, paths...)
}

func NewLoader(defaultType string) *Loader {
	l := &Loader{
		defaultType:        defaultType,
		assetTypeToParser:  make(map[string]util.BytesToAnyErr),
		loaderTypeToLoader: make(map[string]util.StrToBytesErr),
	}
	l.BindLoader(LocalLoader, func(path string) ([]byte, *util.Err) {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, util.NewErr(util.EcIo, util.M{
				"path":  path,
				"error": err.Error(),
			})
		}
		return bytes, nil
	})
	l.BindLoader(HttpLoader, func(path string) ([]byte, *util.Err) {
		res, err := http.Get(path)
		if err != nil {
			return nil, util.NewErr(util.EcIo, util.M{
				"path":  path,
				"error": err.Error(),
			})
		}
		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, util.NewErr(util.EcIo, util.M{
				"path":  path,
				"error": err.Error(),
			})
		}
		return bytes, nil
	})
	return l
}

type Loader struct {
	defaultType        string
	assetTypeToParser  map[string]util.BytesToAnyErr
	loaderTypeToLoader map[string]util.StrToBytesErr
}

func (l *Loader) BindParser(assetType string, marshaller util.BytesToAnyErr) {
	l.assetTypeToParser[assetType] = marshaller
}

func (l *Loader) BindLoader(loaderType string, loader util.StrToBytesErr) {
	l.loaderTypeToLoader[loaderType] = loader
}

func (l *Loader) DefaultLoad(assetType string, action util.FnStrAny, paths ...string) {
	l.Load(assetType, l.defaultType, action, paths...)
}

func (l *Loader) Load(assetType string, loaderType string, action util.FnStrAny, paths ...string) {
	marshaller, ok := l.assetTypeToParser[assetType]
	if !ok {
		kiwi.Error2(util.EcNotExist, util.M{
			"asset type": assetType,
		})
		return
	}
	loader, ok := l.loaderTypeToLoader[loaderType]
	if !ok {
		kiwi.Error2(util.EcNotExist, util.M{
			"loader type": loaderType,
		})
		return
	}
	for _, path := range paths {
		bytes, e := loader(path)
		if e != nil {
			kiwi.Error(e)
			continue
		}
		o, e := marshaller(bytes)
		if e != nil {
			kiwi.Error(e)
			continue
		}
		action(path, o)
	}
}
