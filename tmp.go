package kiwi

var Mod = TSvcCode(1000)

func MergeSvcCode(svc TSvc, code TCode) TSvcCode {
	return svc*Mod + TSvcCode(code)
}

func SplitSvcCode(sc TSvcCode) (TSvc, TCode) {
	return sc / Mod, TCode(sc % Mod)
}
