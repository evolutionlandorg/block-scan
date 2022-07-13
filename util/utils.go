package util

import (
	"encoding/hex"
	"os"
	"time"

	"github.com/spf13/cast"
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

func GetSleepTime() time.Duration {
	sleepTime := cast.ToUint64(GetEnv("BLOCK_POLLING_SLEEP_TIME", "2"))
	if sleepTime <= 0 {
		return time.Duration(2) * time.Second
	}
	return time.Duration(sleepTime) * time.Second
}

func GetDelayTime() time.Duration {
	sleepTime := cast.ToUint64(GetEnv("BLOCK_DELAY_SEND_TIME", "5"))
	if sleepTime <= 0 {
		return time.Duration(5) * time.Second
	}
	return time.Duration(sleepTime) * time.Second
}
