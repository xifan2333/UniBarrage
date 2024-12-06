package utils

import (
	crand "crypto/rand"
)

// @#@ 加密字符串 @#@
// @#@ GenerateRandomKey 生成一个指定长度的随机密钥 @#@
func GenerateRandomKey(length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := crand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func IntSliceToByteSlice(data []int) []byte {
	res := make([]byte, len(data))
	for i, v := range data {
		res[i] = byte(v)
	}
	return res
}
