package util

import (
	"encoding/hex"
	"os"
)

func Panic(err error) {
	if err == nil {
		return
	}
	panic(err)
}

func BytesToHex(b []byte) string {
	c := make([]byte, hex.EncodedLen(len(b)))
	hex.Encode(c, b)
	return string(c)
}

func TryReturn(f func() (result interface{}, err error), tryCount ...int) (result interface{}, err error) {
	var maxTry = 3
	if len(tryCount) != 0 && tryCount[0] > 0 {
		maxTry = tryCount[0]
	}
	var try int
	for try < maxTry {
		try++
		result, err = f()
		if err == nil {
			return result, nil
		}
	}
	return nil, err
}

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return value
}
