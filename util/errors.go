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

import "errors"

var (
	// ErrKeyNotFound indicates the given key was not found.
	ErrKeyNotFound = errors.New("Key not found")
	// ErrKeyLocked menas that the key cannot be modified, as it's locked by another process.
	ErrKeyLocked = errors.New("Key is locked")
	// ErrInvalidKey signals that the given key name is invalid.
	ErrInvalidKey = errors.New("Invalid key name")
	// ErrTypeMismatch signals that the type of value being requested is unexpected.
	ErrTypeMismatch = errors.New("Type mismatch")
	// ErrListEmpty signals that the List is empty.
	ErrListEmpty = errors.New("List is empty")
	// ErrListIndexOutOfRange signals that the given index is out of range of the list.
	ErrListIndexOutOfRange = errors.New("Index out of range")
	// ErrHashFieldNotFound signals that the hash does not have the given field.
	ErrHashFieldNotFound = errors.New("Hash field does not exist")
)
