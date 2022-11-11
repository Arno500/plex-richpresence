package autoupdate

import (
	"log"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

var Version = ""

func Autoupdate() {
	log.Println("Checking for updates...")
	// Do not update dev version!
	if Version == "dev" {
		return
	}
	var updater = Updater{
		CurrentVersion: Version,
		S3Bucket:       "plex-rich-presence",
		S3ReleaseKey:   "plex-rich-presence_{{OS}}_{{ARCH}}-{{VERSION}}",
		S3Path:         "binaries",
		S3VersionKey:   "VERSION",
		AWSConfig: &aws.Config{
			Region:           aws.String("fr-par"),
			Endpoint:         aws.String("s3.fr-par.scw.cloud"),
			Credentials:      credentials.AnonymousCredentials,
			S3ForcePathStyle: aws.Bool(true),
		},
	}
	if runtime.GOOS == "windows" {
		updater.S3ReleaseKey = updater.S3ReleaseKey + ".exe"
	}
	err := runAutoUpdate(updater)
	if err != nil {
		log.Println("Error while checking for updates: ", err)
	}
}
