package main

import (
	"bytes"
	"strconv"

	"github.com/orisano/gosax"
)

const (
	elementNameNextContinuationToken = "NextContinuationToken"
	elementNameKeyCount              = "KeyCount"
	elementNameIsTruncated           = "IsTruncated"

	elementNameContents = "Contents"

	elementNameKey          = "Key"
	elementNameLastModified = "LastModified"
	elementNameSize         = "Size"
)

type handleEventFunc func(e gosax.Event) error
type handleTextFunc func(b []byte) error

func isSelfClosing(e gosax.Event) bool {
	return bytes.HasSuffix(e.Bytes, []byte{'/', '>'})
}

func buildHandleTextString(dest *string) handleTextFunc {
	return func(b []byte) error {
		*dest = string(b)
		return nil
	}
}

func buildHandleTextUint64(dest *uint64) handleTextFunc {
	return func(b []byte) error {
		n, err := strconv.ParseUint(string(b), 10, 64)
		if err != nil {
			return err
		}
		*dest = n
		return nil
	}
}

func buildHandleTextBool(dest *bool) handleTextFunc {
	return func(b []byte) error {
		v, err := strconv.ParseBool(string(b))
		if err != nil {
			return err
		}
		*dest = v
		return nil
	}
}
