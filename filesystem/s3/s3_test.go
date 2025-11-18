// Copyright (C) 2025 ZedCloud Org.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package s3_test

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pluto-org-co/fsio/filesystem/s3"
	"github.com/pluto-org-co/fsio/filesystem/testsuite"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	miniotc "github.com/testcontainers/testcontainers-go/modules/minio"
)

func Test_S3(t *testing.T) {
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

	ctxBucket, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = client.MakeBucket(ctxBucket, bucketName, bucketOptions)
	if !assertions.Nil(err, "failed to create bucket") {
		return
	}

	s3Root := s3.New(client, bucketName, time.Minute)

	t.Run("Testsuite", testsuite.TestFilesystem(t, s3Root))
}
