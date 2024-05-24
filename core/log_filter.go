package core

import "github.com/15mga/kiwi"

var (
	_LogExcludeMap    = make(map[kiwi.TSvcCode]struct{})
	_LogExcludeMsgMap = make(map[kiwi.TSvcCode]struct{})
)

func ExcludeLog(svc kiwi.TSvc, codes ...kiwi.TCode) {
	for _, code := range codes {
		_LogExcludeMap[kiwi.MergeSvcCode(svc, code)] = struct{}{}
	}
}

func IsExcludeLog(svc kiwi.TSvc, code kiwi.TCode) bool {
	_, ok := _LogExcludeMap[kiwi.MergeSvcCode(svc, code)]
	return ok
}

func ExcludeMsg(svc kiwi.TSvc, codes ...kiwi.TCode) {
	for _, code := range codes {
		_LogExcludeMsgMap[kiwi.MergeSvcCode(svc, code)] = struct{}{}
	}
}

func IsExcludeMsg(svc kiwi.TSvc, code kiwi.TCode) bool {
	_, ok := _LogExcludeMsgMap[kiwi.MergeSvcCode(svc, code)]
	return ok
}
