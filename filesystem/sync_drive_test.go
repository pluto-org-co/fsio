package filesystem_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pluto-org-co/fsio/filesystem"
	"github.com/pluto-org-co/fsio/filesystem/googledrive"
	"github.com/pluto-org-co/fsio/filesystem/s3"
	"github.com/pluto-org-co/fsio/googleutils"
	"github.com/pluto-org-co/fsio/googleutils/creds"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	miniotc "github.com/testcontainers/testcontainers-go/modules/minio"
	"golang.org/x/oauth2/jwt"
)

func Test_SyncDrive(t *testing.T) {
	const MaxSyncFiles = 10

	var syncOptions = []filesystem.SyncOption{
		filesystem.WithSyncOptionMaxFiles(5),
	}

	t.Run("Succeed", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Can't run this test as root")
			return
		}

		src := googledrive.New(googledrive.Config{
			JWTLoader: func() (config *jwt.Config) {
				config = creds.NewConfiguration(
					t, googleutils.Scopes...,
				)
				config.Subject = creds.UserEmail()
				return config
			},
			CurrentAccount: true,
			SharedDrive:    true,
			OtherUsers:     true,
		})

		t.Run("Sync", func(t *testing.T) {
			assertions := assert.New(t)

			ctxTc, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			minioC, err := miniotc.Run(
				ctxTc,
				"minio/minio:RELEASE.2025-09-07T16-13-09Z-cpuv1",
			)
			if !assertions.Nil(err, "failed to start minio") {
				return
			}
			defer func() {
				err := testcontainers.TerminateContainer(minioC)
				if err != nil {
					log.Printf("failed to terminate container: %s", err)
				}
			}()

			endpoint, err := minioC.Container.PortEndpoint(ctxTc, "9000", "")
			if !assertions.Nil(err, "failed to get port endpoint") {
				return
			}
			t.Logf("Endpoint: %v", endpoint)

			client, err := minio.New(
				endpoint,
				&minio.Options{
					Creds:           credentials.NewStaticV4(minioC.Username, minioC.Password, ""),
					TrailingHeaders: true,
				},
			)
			if !assertions.Nil(err, "failed to create minio client") {
				return
			}

			var isOnline bool
			for range 10 {
				if client.IsOnline() {
					isOnline = true
					break
				}
				time.Sleep(time.Second)
			}
			if !assertions.True(isOnline, "server is not online") {
				return
			}

			const bucketName = "test-bucket"
			var bucketOptions = minio.MakeBucketOptions{
				Region: "US",
			}

			ctxBucket, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			err = client.MakeBucket(ctxBucket, bucketName, bucketOptions)
			if !assertions.Nil(err, "failed to create bucket") {
				return
			}

			dst := s3.New(client, bucketName, time.Minute)

			now := time.Now()

			ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
			defer cancel()

			filesystem.Sync(ctx, dst, src, syncOptions...)
			firstTook := time.Since(now)
			t.Run("Second Time", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				now := time.Now()
				filesystem.Sync(ctx, dst, src, syncOptions...)
				secondTook := time.Since(now)

				if !assertions.Less(secondTook, firstTook, "second sync should be faster") {
					return
				}
			})
		})
		t.Run("SyncWorkers", func(t *testing.T) {
			assertions := assert.New(t)

			ctxTc, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			minioC, err := miniotc.Run(
				ctxTc,
				"minio/minio:RELEASE.2025-09-07T16-13-09Z-cpuv1",
			)
			if !assertions.Nil(err, "failed to start minio") {
				return
			}
			defer func() {
				err := testcontainers.TerminateContainer(minioC)
				if err != nil {
					log.Printf("failed to terminate container: %s", err)
				}
			}()

			endpoint, err := minioC.Container.PortEndpoint(ctxTc, "9000", "")
			if !assertions.Nil(err, "failed to get port endpoint") {
				return
			}
			t.Logf("Endpoint: %v", endpoint)

			client, err := minio.New(
				endpoint,
				&minio.Options{
					Creds:           credentials.NewStaticV4(minioC.Username, minioC.Password, ""),
					TrailingHeaders: true,
				},
			)
			if !assertions.Nil(err, "failed to create minio client") {
				return
			}

			var isOnline bool
			for range 10 {
				if client.IsOnline() {
					isOnline = true
					break
				}
				time.Sleep(time.Second)
			}
			if !assertions.True(isOnline, "server is not online") {
				return
			}

			const bucketName = "test-bucket"
			var bucketOptions = minio.MakeBucketOptions{
				Region: "US",
			}

			ctxBucket, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			err = client.MakeBucket(ctxBucket, bucketName, bucketOptions)
			if !assertions.Nil(err, "failed to create bucket") {
				return
			}

			dst := s3.New(client, bucketName, time.Minute)

			now := time.Now()

			ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
			defer cancel()

			filesystem.SyncWorkers(100, ctx, dst, src, syncOptions...)

			firstTook := time.Since(now)
			t.Logf("First attempt took: %v", firstTook)
			t.Run("Second Time", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				now := time.Now()
				filesystem.SyncWorkers(100, ctx, dst, src, syncOptions...)
				secondTook := time.Since(now)
				t.Logf("Second attempt took: %v", secondTook)

				if !assertions.Less(secondTook, firstTook, "second sync should be faster") {
					return
				}
			})
			t.Run("Iterating files", func(t *testing.T) {
				assertions := assert.New(t)

				ctx, cancel := context.WithTimeout(context.TODO(), time.Minute)
				defer cancel()

				var count int
				for entry := range dst.Files(ctx) {
					count++
					t.Logf("- Location: %s", entry.Location())
				}
				assertions.NotZero(count, "expecting at least one value synced")
			})
		})
	})
}
