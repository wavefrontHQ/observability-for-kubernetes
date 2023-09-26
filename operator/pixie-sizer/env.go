package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

func RequireFromEnv[T any](envName string, parse func(string) (T, error), value *T) {
	envValue := os.Getenv(envName)
	if envValue == "" {
		panic(fmt.Sprintf("%s is required, but was not set", envName))
	}
	v, err := parse(envValue)
	if err != nil {
		panic(err)
	}
	*value = v
}

func OptionalFromEnv[T any](envName string, parse func(string) (T, error), value *T) {
	envValue := os.Getenv(envName)
	if envValue == "" {
		return
	}
	v, err := parse(envValue)
	if err != nil {
		panic(err)
	}
	*value = v
}

func ParseInt(s string) (int, error) {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	if i > math.MaxInt {
		return 0, fmt.Errorf("integer out of bounds: max int is %d, but got %d", math.MaxInt, i)
	}
	return int(i), nil
}

func ParseFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func ParseString(s string) (string, error) {
	return s, nil
}
