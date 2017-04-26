// Copyright 2017 Ross Peoples
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"unsafe"
)

// ZeroByte represents a single zero byte in a byte slice.
var ZeroByte = []byte{0}

var suffixForKeysUsingPrefix = "*_MYDIS_WITHPREFIX"
var suffixForLocks = "*_MYDIS_LOCK"

// MapStringToMapBytes converts a map[string]string to map[string][]byte.
func MapStringToMapBytes(h map[string]string) map[string][]byte {
	m := map[string][]byte{}
	if h == nil {
		return m
	}
	for k, v := range h {
		m[k] = StringToBytes(v)
	}
	return m
}

// MapBoolToMapBytes converts a map[string]string to map[string][]byte.
func MapBoolToMapBytes(h map[string]bool) map[string][]byte {
	m := map[string][]byte{}
	if h == nil {
		return m
	}
	for k, v := range h {
		m[k] = NewValue(v).b
	}
	return m
}

// MapIntToMapBytes converts a map[string]string to map[string][]byte.
func MapIntToMapBytes(h map[string]int64) map[string][]byte {
	m := map[string][]byte{}
	if h == nil {
		return m
	}
	for k, v := range h {
		m[k] = NewValue(v).b
	}
	return m
}

// MapFloatToMapBytes converts a map[string]string to map[string][]byte.
func MapFloatToMapBytes(h map[string]float64) map[string][]byte {
	m := map[string][]byte{}
	if h == nil {
		return m
	}
	for k, v := range h {
		m[k] = NewValue(v).b
	}
	return m
}

// MapValueToMapBytes converts a map[string]Value to map[string][]byte.
func MapValueToMapBytes(h map[string]Value) map[string][]byte {
	m := map[string][]byte{}
	if h == nil {
		return m
	}
	for k, v := range h {
		m[k] = v.b
	}
	return m
}

// BytesToString efficiently converts a byte slice to a string without allocating any additional memory.
func BytesToString(b []byte) string {
	p := unsafe.Pointer(&b)
	return *(*string)(p)
}

// StringToBytes efficiently converts a string to a byte slice without allocating any additional memory.
func StringToBytes(s string) []byte {
	p := unsafe.Pointer(&s)
	return *(*[]byte)(p)
}
