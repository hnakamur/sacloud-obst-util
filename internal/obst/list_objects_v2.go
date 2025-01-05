package obst

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func ListObjectsV2(ctx context.Context, client *http.Client,
	bucketName, s3Endpoint, s3Region, accessKeyID, secretAccessKey,
	continuationToken string,
	handleResponseBody func(body io.Reader) error) error {

	// https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html

	var query string
	if continuationToken != "" {
		query = "list-type=2&continuation-token=" + url.QueryEscape(continuationToken)
	} else {
		query = "list-type=2"
	}
	u := &url.URL{
		Scheme:   "https",
		Host:     bucketName + "." + s3Endpoint,
		Path:     "/",
		RawQuery: query,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("create request to validate object storage setting: %s", err)
	}
	if err := SignS3Request(ctx, req, s3Region, accessKeyID, secretAccessKey, time.Now()); err != nil {
		return fmt.Errorf("failed to sign request: %s", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request to validate object storage setting: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return discardRestOfResponseBody(err, resp.Body)
	}

	const debug = false
	if debug {
		var b bytes.Buffer
		if err := handleResponseBody(io.TeeReader(resp.Body, &b)); err != nil {
			return discardRestOfResponseBody(fmt.Errorf("failed to handle response body: %s", err), resp.Body)
		}
		err = discardRestOfResponseBody(nil, resp.Body)
		fmt.Printf("response body:\n%s\n", b.String())
		return err
	} else {
		if err := handleResponseBody(resp.Body); err != nil {
			return discardRestOfResponseBody(fmt.Errorf("failed to handle response body: %s", err), resp.Body)
		}
		return discardRestOfResponseBody(nil, resp.Body)
	}
}

func discardRestOfResponseBody(err error, body io.Reader) error {
	if _, err2 := io.Copy(io.Discard, body); err2 != nil {
		if err == nil {
			return err2
		}
		return errors.Join(err, fmt.Errorf("failed to discard error response body: %s", err2))
	}

	return err
}
