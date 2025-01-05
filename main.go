package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/alecthomas/kong"
	"github.com/hnakamur/sacloud-obst-util/internal/obst"
	apiclient "github.com/sacloud/api-client-go"
	"github.com/sacloud/api-client-go/profile"
)

type Context struct {
	Debug bool
}

type SummaryCmd struct {
	Profile  string `required:"" help:"sacloud profile name."`
	Bucket   string `arg:"" name:"bucket" help:"Buckets to check."`
	Endpoint string `default:"s3.isk01.sakurastorage.jp" help:"object storage endpoint domain"`
	Region   string `default:"jp-north-1" help:"object storage region"`
}

func (s *SummaryCmd) Run(ctx *Context) error {
	log.Printf("summary command, debug=%v, bucket=%v", ctx.Debug, s.Bucket)
	prof, err := loadProfile(s.Profile)
	if err != nil {
		return err
	}

	httpClient := &http.Client{}
	continuationToken := ""
	handler := newTotalSizeCalculator()
	apiCallCount := 0
	for {
		err = obst.ListObjectsV2(context.Background(), httpClient, s.Bucket,
			s.Endpoint, s.Region, prof.AccessToken, prof.AccessTokenSecret,
			continuationToken, handler.handleResponseBody)
		if err != nil {
			return err
		}

		if handler.nextContinuationToken == "" {
			break
		}
		continuationToken = handler.nextContinuationToken

		apiCallCount++
		if ctx.Debug && apiCallCount%100 == 0 {
			log.Printf("DEBUG: current apiCallCount=%d, totalSize=%d, objCount=%d", apiCallCount, handler.totalSize, handler.objCount)
		}
	}

	log.Printf("final: apiCallCount=%d, totalSize=%d, objCount=%d", apiCallCount, handler.totalSize, handler.objCount)
	return nil
}

func loadProfile(profileName string) (*profile.ConfigValue, error) {
	opts, err := apiclient.OptionsFromProfile(profileName)
	if err != nil {
		return nil, err
	}
	return opts.ProfileConfigValue(), nil
}

var cli struct {
	Debug bool `help:"Enable debug mode."`

	Summary SummaryCmd `cmd:"" help:"Show total object size and count."`
	Version VersionCmd `cmd:"" help:"Show version and exit."`
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{
		Debug: cli.Debug,
	})
	ctx.FatalIfErrorf(err)
}

type VersionCmd struct{}

func (v *VersionCmd) Run(ctx *Context) error {
	fmt.Println(version())
	return nil
}

func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}
	return info.Main.Version
}
