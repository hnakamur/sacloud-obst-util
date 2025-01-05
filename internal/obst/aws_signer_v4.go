package obst

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	signerv4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	serviceS3 = "s3"

	// https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-header-based-auth.html
	unsginedPayload = "UNSIGNED-PAYLOAD"

	contentSHAKey = "X-Amz-Content-Sha256"
)

// SignS3Request signs a HTTP request with AWS Signature V4
func SignS3Request(ctx context.Context, req *http.Request, region, accessKeyID,
	secretAccessKey string, signingTime time.Time) error {

	// https://github.com/aws/aws-sdk-go-v2/blob/v1.2.0/aws/signer/v4/v4.go
	// https://github.com/aws/aws-sdk-go-v2/blob/v1.2.0/aws/signer/internal/v4/headers.go

	req.Header.Set(contentSHAKey, unsginedPayload)

	cred := aws.Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}
	signer := signerv4.NewSigner(func(o *signerv4.SignerOptions) {
		// https://github.com/aws/aws-sdk-go-v2/blob/v1.3.4/service/s3/api_client.go#L259
		o.DisableURIPathEscaping = true
	})
	if err := signer.SignHTTP(ctx, cred, req, unsginedPayload, serviceS3,
		region, signingTime); err != nil {
		return fmt.Errorf("sign request with aws signer v4: %s", err)
	}
	return nil
}
