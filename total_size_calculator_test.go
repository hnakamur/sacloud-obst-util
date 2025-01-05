package main

import (
	_ "embed"
	"encoding/xml"
	"fmt"
	"log"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/orisano/gosax"
)

func TestTotalSizeCalculator(t *testing.T) {
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
		h := newTotalSizeCalculator()
		err := h.handleResponseBody(strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}
		verifyObjCountAndTotalSize(t, input, h.objCount, h.totalSize, h.nextContinuationToken)
	}
}

func generateTestXML(objCount int, continuationToken, nextContinuationToken string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString(`<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
	b.WriteString(`<Name>bucket-name</Name>`)
	b.WriteString(`<Prefix/>`)
	b.WriteString(`<MaxKeys>1000</MaxKeys>`)
	fmt.Fprintf(&b, `<IsTruncated>%v</IsTruncated>`, nextContinuationToken != "")
	b.WriteString(`<FetchOwner>false</FetchOwner>`)
	if continuationToken == "" {
		b.WriteString(`<ContinuationToken/>`)
	} else {
		fmt.Fprintf(&b, `<ContinuationToken>%s</ContinuationToken>`, continuationToken)
	}
	if nextContinuationToken == "" {
		b.WriteString(`<NextContinuationToken/>`)
	} else {
		fmt.Fprintf(&b, `<NextContinuationToken>%s</NextContinuationToken>`, nextContinuationToken)
	}
	fmt.Fprintf(&b, `<KeyCount>%d</KeyCount>`, objCount)
	for i := 0; i < objCount; i++ {
		b.WriteString(`<Contents>`)
		fmt.Fprintf(&b, `<Key>key%06d</Key>`, i+1)
		lastModified := time.Unix(int64(i+1)+time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix(), 0).UTC()
		fmt.Fprintf(&b, `<LastModified>%s</LastModified>`, lastModified.Format(time.RFC3339))
		fmt.Fprintf(&b, `<ETag>&quot;%032x&quot;</ETag>`, uint64(i)+0xabcd000000000001)
		fmt.Fprintf(&b, `<Size>%d</Size>`, i+1)
		b.WriteString(`<StorageClass>STANDARD</StorageClass>`)
		b.WriteString(`</Contents>`)
	}
	b.WriteString(`</ListBucketResult>`)
	return b.String()
}

func verifyObjCountAndTotalSize(t *testing.T, xmlText string, gotObjCount, gotTotalSize uint64, gotNextContinuationToken string) {
	t.Helper()

	type Contents struct {
		Size int `xml:"Size"`
	}

	type ListBucketResult struct {
		XMLName               xml.Name   `xml:"ListBucketResult"`
		NextContinuationToken string     `xml:"NextContinuationToken"`
		Contents              []Contents `xml:"Contents"`
	}

	var result ListBucketResult

	decoder := xml.NewDecoder(strings.NewReader(xmlText))
	err := decoder.Decode(&result)
	if err != nil {
		t.Fatalf("Error decoding XML: %v\n", err)
	}

	totalSize := uint64(0)
	for _, content := range result.Contents {
		totalSize += uint64(content.Size)
	}

	wantObjCount := uint64(len(result.Contents))
	wantTotalSize := totalSize
	wantNextContinuationToken := result.NextContinuationToken

	mismatched := false
	if got, want := gotObjCount, wantObjCount; got != want {
		t.Errorf("objCount mismatch, got=%d, want=%d", got, want)
		mismatched = true
	}
	if got, want := gotTotalSize, wantTotalSize; got != want {
		t.Errorf("totalSize mismatch, got=%d, want=%d", got, want)
		mismatched = true
	}
	if got, want := gotNextContinuationToken, wantNextContinuationToken; got != want {
		t.Errorf("nextContinuationToken mismatch, got=%s, want=%s", got, want)
		mismatched = true
	}
	if mismatched {
		t.Logf("input=\n%s", xmlText)
	}
}

func TestGosaxSelfClosingElement(t *testing.T) {
	xmlData := `<root><element/></root>`
	reader := strings.NewReader(xmlData)

	var gotTypes []uint8
	var gotTexts []string
	r := gosax.NewReader(reader)
	for {
		e, err := r.Event()
		if err != nil {
			log.Fatal(err)
		}
		if e.Type() == gosax.EventEOF {
			break
		}
		gotTypes = append(gotTypes, e.Type())
		gotTexts = append(gotTexts, string(e.Bytes))
	}

	wantTypes := []uint8{gosax.EventStart, gosax.EventStart, gosax.EventEnd}
	wantTexts := []string{`<root>`, `<element/>`, `</root>`}

	if !slices.Equal(gotTypes, wantTypes) {
		t.Errorf("types mismatch, got=%v, want=%v", gotTypes, wantTypes)
	}
	if !slices.Equal(gotTexts, wantTexts) {
		t.Errorf("texts mismatch, got=%v, want=%v", gotTexts, wantTexts)
	}
}
