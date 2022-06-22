package util

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestGetSleepTime(t *testing.T) {
	tests := []struct {
		name string
		want time.Duration
	}{
		{"default1", func() time.Duration {
			os.Setenv("BLOCK_POLLING_SLEEP_TIME", "10")
			return time.Second * 10
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSleepTime(); got != tt.want {
				t.Errorf("GetSleepTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDelayTime(t *testing.T) {
	tests := []struct {
		name string
		want time.Duration
	}{
		{"default1", func() time.Duration {
			os.Setenv("BLOCK_DELAY_SEND_TIME", "10")
			return time.Second * 10
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDelayTime(); got != tt.want {
				t.Errorf("GetDelayTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	type args struct {
		key          string
		defaultValue string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"1", args{
			key:          "a",
			defaultValue: "1",
		}, "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEnv(tt.args.key, tt.args.defaultValue); got != tt.want {
				t.Errorf("GetEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTryReturn(t *testing.T) {
	var count int
	result, err := TryReturn(func() (result interface{}, err error) {
		count++
		if count <= 9 {
			return nil, errors.New("error")
		}
		return count, nil
	}, 10)
	if err != nil {
		t.Fatalf("try request failed: %v", err)
	}
	if result != count {
		t.Fatalf("%v != %v", result, count)
	}
	if result.(int) != 10 {
		t.Fatal("result != 10")
	}
}
