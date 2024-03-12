package util

func ArgsToSlc[T any](args ...T) []T {
	return args
}

func SplitSlc1[T any](slc []any) T {
	return slc[0].(T)
}

func SplitSlc2[T1, T2 any](slc []any) (T1, T2) {
	return slc[0].(T1), slc[1].(T2)
}

func SplitSlc3[T1, T2, T3 any](slc []any) (T1, T2, T3) {
	return slc[0].(T1), slc[1].(T2), slc[2].(T3)
}

func SplitSlc4[T1, T2, T3, T4 any](slc []any) (T1, T2, T3, T4) {
	return slc[0].(T1), slc[1].(T2), slc[2].(T3), slc[3].(T4)
}

func SplitSlc5[T1, T2, T3, T4, T5 any](slc []any) (T1, T2, T3, T4, T5) {
	return slc[0].(T1), slc[1].(T2), slc[2].(T3), slc[3].(T4), slc[4].(T5)
}

func SplitSlc6[T1, T2, T3, T4, T5, T6 any](slc []any) (T1, T2, T3, T4, T5, T6) {
	return slc[0].(T1), slc[1].(T2), slc[2].(T3), slc[3].(T4), slc[4].(T5), slc[5].(T6)
}

func SplitSlc7[T1, T2, T3, T4, T5, T6, T7 any](slc []any) (T1, T2, T3, T4, T5, T6, T7) {
	return slc[0].(T1), slc[1].(T2), slc[2].(T3), slc[3].(T4), slc[4].(T5), slc[5].(T6), slc[6].(T7)
}

func SplitSlc8[T1, T2, T3, T4, T5, T6, T7, T8 any](slc []any) (T1, T2, T3, T4, T5, T6, T7, T8) {
	return slc[0].(T1), slc[1].(T2), slc[2].(T3), slc[3].(T4), slc[4].(T5), slc[5].(T6), slc[6].(T7), slc[7].(T8)
}
