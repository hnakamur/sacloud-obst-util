package main

import (
	_ "embed"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

func TestObjectLister(t *testing.T) {
	type Contents struct {
		Key          string `xml:"Key"`
		LastModified string `xml:"LastModified"`
		Size         uint64 `xml:"Size"`
	}

	testCases := []struct {
		objCount              int
		continuationToken     string
		nextContinuationToken string
	}{
		{objCount: 2, continuationToken: "", nextContinuationToken: ""},
		{objCount: 1000, continuationToken: "", nextContinuationToken: "token1"},
		{objCount: 1000, continuationToken: "token1", nextContinuationToken: "token2"},
		{objCount: 500, continuationToken: "token2", nextContinuationToken: ""},
	}
	for _, tc := range testCases {
		input := generateTestXML(tc.objCount, tc.continuationToken, tc.nextContinuationToken)
		var gotContentsList []Contents
		h := newObjectsLister(func(c listBucketsResultContents) (discardsRest bool, err error) {
			gotContentsList = append(gotContentsList, Contents{
				Key:          c.key,
				LastModified: c.lastModified,
				Size:         c.size,
			})
			return false, nil
		})
		err := h.handleResponseBody(strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}

		type ListBucketResult struct {
			XMLName               xml.Name   `xml:"ListBucketResult"`
			NextContinuationToken string     `xml:"NextContinuationToken"`
			Contents              []Contents `xml:"Contents"`
		}

		var result ListBucketResult

		decoder := xml.NewDecoder(strings.NewReader(input))
		err = decoder.Decode(&result)
		if err != nil {
			t.Fatalf("Error decoding XML: %v\n", err)
		}

		mismatched := false
		if got, want := gotContentsList, result.Contents; !reflect.DeepEqual(got, want) {
			t.Errorf("contents mismatch, got=%+v, want=%+v", got, want)
			mismatched = true
		}
		if mismatched {
			t.Logf("input=\n%s", input)
		}

	}
}
