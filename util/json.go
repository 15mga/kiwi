package util

import jsoniter "github.com/json-iterator/go"

var _JsonConf = jsoniter.Config{
	UseNumber:   true,
	EscapeHTML:  true,
	SortMapKeys: true,
}.Froze()

func SetJsonConf(conf jsoniter.Config) {
	_JsonConf = conf.Froze()
}

func JsonConf() jsoniter.API {
	return _JsonConf
}

func JsonMarshal(o any) ([]byte, *Err) {
	bytes, err := _JsonConf.Marshal(o)
	if err != nil {
		return nil, WrapErr(EcMarshallErr, err)
	}
	return bytes, nil
}

func JsonMarshalIndent(o any, prefix, indent string) ([]byte, *Err) {
	bytes, err := _JsonConf.MarshalIndent(o, prefix, indent)
	if err != nil {
		return nil, WrapErr(EcMarshallErr, err)
	}
	return bytes, nil
}

func JsonUnmarshal(bytes []byte, o any) *Err {
	err := _JsonConf.Unmarshal(bytes, o)
	if err != nil {
		return WrapErr(EcUnmarshallErr, err)
	}
	return nil
}
