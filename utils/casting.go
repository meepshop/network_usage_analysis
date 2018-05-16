package utils

import (
	"math"
	"strconv"
)

func Round(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return float64(int((v*pow)+0.5)) / pow
}

func StrToInt(s string) (int, error) {

	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return i, nil
}

func StrToFloat(s string) (float64, error) {

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return f, nil
}

// PInt convert int to pointer of int
func PInt(in int) *int {
	tmp := in
	return &tmp
}

// PInt32 convert int32 to pointer of int32
func PInt32(in int32) *int32 {
	tmp := in
	return &tmp
}

// PInt64 convert int64 to pointer of int64
func PInt64(in int64) *int64 {
	tmp := in
	return &tmp
}

// PFloat64 convert float64 to pointer of float64
func PFloat64(in float64) *float64 {
	tmp := in
	return &tmp
}

// PStr convert string to pointer of string
func PStr(in string) *string {
	tmp := in
	return &tmp
}
