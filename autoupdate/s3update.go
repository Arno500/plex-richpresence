package autoupdate

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	version "github.com/hashicorp/go-version"
)

// Heavily copied from https://github.com/heetch/s3update with a few modifications for logging and customization

type Updater struct {
	// CurrentVersion represents the current binary version.
	// This is generally set at the compilation time with -ldflags "-X main.Version=42"
	// See the README for additional information
	CurrentVersion string
	// S3Bucket represents the S3 bucket containing the different files used by s3update.
	S3Bucket string
	// S3Region represents the S3 region you want to work in.
	S3Region string
	// S3ReleaseKey represents the raw key on S3 to download new versions.
	// The value can be something like `cli/releases/cli-{{OS}}-{{ARCH}}`
	S3ReleaseKey string
	// A path where to check for binaries
	S3Path string
	// S3VersionKey represents the key on S3 to download the current version
	S3VersionKey string
	// AWSCredentials represents the config to use to connect to s3
	AWSCredentials *credentials.Credentials
	// Manual full AWSConfig
	AWSConfig *aws.Config
}

// validate ensures every required fields is correctly set. Otherwise and error is returned.
func (u Updater) validate() error {
	if u.CurrentVersion == "" {
		return fmt.Errorf("no version set")
	}

	if u.S3Bucket == "" {
		return fmt.Errorf("no bucket set")
	}

	if u.AWSConfig != nil {
		return nil
	}

	if u.S3Region == "" {
		return fmt.Errorf("no s3 region")
	}

	if u.S3ReleaseKey == "" {
		return fmt.Errorf("no s3ReleaseKey set")
	}

	if u.S3VersionKey == "" {
		return fmt.Errorf("no s3VersionKey set")
	}

	return nil
}

// AutoUpdate runs synchronously a verification to ensure the binary is up-to-date.
// If a new version gets released, the download will happen automatically
// It's possible to bypass this mechanism by setting the S3UPDATE_DISABLED environment variable.
func AutoUpdate(u Updater) error {
	if os.Getenv("S3UPDATE_DISABLED") != "" {
		log.Println("[AutoUpdater] Autoupdate disabled")
		return nil
	}

	if err := u.validate(); err != nil {
		log.Printf("[AutoUpdater] %s - Skipping auto update\n", err.Error())
		return err
	}

	return runAutoUpdate(u)
}

// generateS3ReleaseKey dynamically builds the S3 key depending on the os and architecture.
func generateS3ReleaseKey(path string, version string) string {
	path = strings.Replace(path, "{{OS}}", runtime.GOOS, -1)
	path = strings.Replace(path, "{{ARCH}}", runtime.GOARCH, -1)
	path = strings.Replace(path, "{{VERSION}}", version, -1)

	return path
}

func runAutoUpdate(u Updater) error {
	localVersion, err := version.NewVersion(u.CurrentVersion)
	if err != nil || localVersion == nil {
		return err
	}

	var svc *s3.S3
	s3Session, err := session.NewSession()
	if err != nil {
		return err
	}

	if u.AWSConfig != nil {
		svc = s3.New(s3Session, u.AWSConfig)
	} else {
		svc = s3.New(s3Session, &aws.Config{
			Region:      aws.String(u.S3Region),
			Credentials: u.AWSCredentials,
		})
	}

	resp, err := svc.GetObject(&s3.GetObjectInput{Bucket: aws.String(u.S3Bucket), Key: aws.String(u.S3VersionKey)})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	remoteVersion, err := version.NewVersion(string(b))
	if err != nil {
		return err
	}

	log.Printf("[AutoUpdater] Local Version %s - Remote Version: %s\n", localVersion.Original(), remoteVersion.Original())
	if remoteVersion.GreaterThan(localVersion) {
		fmt.Printf("[AutoUpdater] Version outdated... \n")
		var s3Key string
		if u.S3Path == "" {
			s3Key = generateS3ReleaseKey(u.S3ReleaseKey, remoteVersion.Original())
		} else {
			s3Key = fmt.Sprintf("%s/%s", u.S3Path, generateS3ReleaseKey(u.S3ReleaseKey, remoteVersion.Original()))
		}
		resp, err := svc.GetObject(&s3.GetObjectInput{Bucket: aws.String(u.S3Bucket), Key: aws.String(s3Key)})
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		destExec, err := os.Executable()
		if err != nil {
			return err
		}

		dest := filepath.Dir(destExec) + "/" + generateS3ReleaseKey(u.S3ReleaseKey, remoteVersion.Original())

		// Move the old version to a backup path that we can recover from
		// in case the upgrade fails
		destBackup := destExec + ".bak"
		if _, err := os.Stat(destExec); err == nil {
			os.Rename(destExec, destBackup)
		}

		// Use the same flags that ioutil.WriteFile uses
		f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			os.Rename(destBackup, dest)
			return err
		}
		defer f.Close()

		log.Printf("[AutoUpdater] Downloading new version to %s\n", dest)
		if _, err := io.Copy(f, resp.Body); err != nil {
			os.Rename(destBackup, dest)
			return err
		}
		// The file must be closed already so we can execute it in the next step
		f.Close()

		// Removing backup
		os.Remove(destBackup)

		log.Printf("[AutoUpdater] Updated with success to version %s\nRestarting application...\n", remoteVersion.Original())

		// The update completed, we can now restart the application without requiring any user action.

		if runtime.GOOS == "windows" {
			err = exec.Command(dest).Start()
		} else {
			err = syscall.Exec(dest, os.Args, os.Environ())
		}
		if err != nil {
			return err
		}

		os.Exit(0)
	}

	return nil
}
