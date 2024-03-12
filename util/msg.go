package util

import "google.golang.org/protobuf/reflect/protoreflect"

type IMsg interface {
	ProtoReflect() protoreflect.Message
}
