package internal

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

func Hash(bytes []byte) ([32]byte, error) {
	if len(bytes) < 128 {
		return [32]byte{}, errors.New("file too short")
	}
	leftBound := 0

	if string(bytes[:3]) == "ID3" {
		if bytes[5]&0x10 > 0 {
			leftBound = 20
		} else {
			leftBound = 10
		}
		leftBound += int(bytes[9])
		leftBound += int(bytes[8]) * 128
		leftBound += int(bytes[7]) * 128 * 128
		leftBound += int(bytes[6]) * 128 * 128 * 128
	}

	rightBound := len(bytes) - 1

	lastIndex := len(bytes) - 1

	if string(bytes[lastIndex-128:lastIndex-125]) == "TAG" {
		rightBound = -128
	}

	if leftBound > lastIndex || rightBound > lastIndex {
		return [32]byte{}, fmt.Errorf("invalid file; could not parse")
	}

	hash := sha256.Sum256(bytes[leftBound:rightBound])

	return hash, nil
}
