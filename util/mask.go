package util

func GenMask(items ...int64) int64 {
	v := int64(0)
	for _, val := range items {
		v |= val
	}
	return v
}

func TestMask(item, mask int64) bool {
	return (item & mask) > 0
}

func MaskAddItem(item, mask int64) int64 {
	return item | mask
}

func MaskDelItem(item, mask int64) int64 {
	return ^item & mask
}
