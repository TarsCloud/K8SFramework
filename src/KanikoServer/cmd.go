/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	k8sMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	tarsAppsV1beta3 "k8s.tars.io/apps/v1beta3"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kaniko/pkg/buildcontext"
	"github.com/GoogleContainerTools/kaniko/pkg/config"
	"github.com/GoogleContainerTools/kaniko/pkg/constants"
	"github.com/GoogleContainerTools/kaniko/pkg/executor"
	"github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/containerd/containerd/platforms"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	opts                               = &config.KanikoOptions{}
	id                                 = ""
	timageName                         = ""
	timageSnap *tarsAppsV1beta3.TImage = nil
	k8sContext *K8SContext             = nil
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&id, "id", "", "", "")
	RootCmd.PersistentFlags().StringVarP(&timageName, "timage", "", "", ".")
	RootCmd.PersistentFlags().StringVarP(&opts.DockerfilePath, "dockerfile", "f", "Dockerfile", "Path to the dockerfile to be built.")
	RootCmd.PersistentFlags().StringVarP(&opts.SrcContext, "context", "c", "/workspace/", "Path to the dockerfile build context.")
	RootCmd.PersistentFlags().VarP(&opts.Destinations, "destination", "d", "Registry the final image should be pushed to. Set it repeatedly for multiple destinations.")
	RootCmd.PersistentFlags().StringVarP(&opts.SnapshotMode, "snapshotMode", "", "full", "Change the file attributes inspected during snapshotting")
	RootCmd.PersistentFlags().BoolVarP(&opts.SkipTLSVerify, "skip-tls-verify", "", true, "Push to insecure registry ignoring TLS verify")
	RootCmd.PersistentFlags().IntVar(&opts.PushRetry, "push-retry", 3, "Number of retries for the push operation")
	RootCmd.PersistentFlags().StringVarP(&opts.KanikoDir, "kaniko-dir", "", "/kaniko", "Path to the kaniko directory")
	RootCmd.PersistentFlags().BoolVarP(&opts.SingleSnapshot, "single-snapshot", "", false, "Take a single snapshot at the end of the build.")
	RootCmd.PersistentFlags().BoolVarP(&opts.Cache, "cache", "", true, "Use cache when building image")
	RootCmd.PersistentFlags().StringVarP(&opts.CacheDir, "cache-dir", "", "/cache", "Specify a local directory to use as a cache.")
	RootCmd.PersistentFlags().BoolVarP(&opts.Cleanup, "cleanup", "", false, "Clean the filesystem at the end")
	RootCmd.PersistentFlags().DurationVarP(&opts.CacheTTL, "cache-ttl", "", time.Hour*336, "Cache timeout in hours. Defaults to two weeks.")
	RootCmd.PersistentFlags().BoolVarP(&opts.IgnoreVarRun, "ignore-var-run", "", true, "Ignore /var/run directory when taking image snapshot. Set it to false to preserve /var/run/ in destination image.")
	// Default the custom platform flag to our current platform, and validate it.
	if opts.CustomPlatform == "" {
		opts.CustomPlatform = platforms.Format(platforms.Normalize(platforms.DefaultSpec()))
	}
	if _, err := v1.ParsePlatform(opts.CustomPlatform); err != nil {
		logrus.Fatalf("Invalid platform %q: %v", opts.CustomPlatform, err)
	}
	RootCmd.PersistentFlags().BoolVarP(&opts.IgnoreVarRun, "whitelist-var-run", "", true, "Ignore /var/run directory when taking image snapshot. Set it to false to preserve /var/run/ in destination image.")
}

func pushStateOrDie(phase string, message string) {
	timageSnap.Build.Running.Phase = phase
	timageSnap.Build.Running.Message = message
	var err error
	for i := 0; i < 3; i++ {
		timageSnap, err = k8sContext.crdClient.AppsV1beta3().TImages(k8sContext.namespace).Update(context.TODO(), timageSnap, k8sMetaV1.UpdateOptions{})
		if err == nil {
			return
		}
		time.Sleep(time.Duration(300) * time.Millisecond)
	}
	if err != nil {
		exit(fmt.Errorf("update running state failed: %s", err.Error()))
	}
}

// RootCmd is the tarskaniko command that is run
var RootCmd = &cobra.Command{
	Use: "tarskaniko",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Use == "tarskaniko" {
			resolveEnvironmentBuildArgs(opts.BuildArgs, os.Getenv)

			if len(opts.Destinations) == 0 {
				exit(fmt.Errorf("empty --destination param"))
			}

			if err := cacheFlagsValid(); err != nil {
				exit(fmt.Errorf("error --cache param"))
			}

			if err := resolveSourceContext(); err != nil {
				exit(fmt.Errorf("error resolving source context"))
			}

			if err := resolveDockerfilePath(); err != nil {
				exit(fmt.Errorf("error resolving dockerfile path"))
			}

			// Update ignored paths
			if opts.IgnoreVarRun {
				// /var/run is a special case. It's common to mount in /var/run/docker.sock
				// or something similar which leads to a special mount on the /var/run/docker.sock
				// file itself, but the directory to exist in the image with no way to tell if it came
				// from the base image or not.
				logrus.Trace("Adding /var/run to default ignore list")
				util.AddToDefaultIgnoreList(util.IgnoreListEntry{
					Path:            "/var/run",
					PrefixMatchOnly: false,
				})
			}
			for _, p := range opts.IgnorePaths {
				util.AddToDefaultIgnoreList(util.IgnoreListEntry{
					Path:            p,
					PrefixMatchOnly: false,
				})
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		k8sContext, err = CreateK8SContext("", "")
		if err != nil {
			exit(fmt.Errorf("create k8s context error: %s", err.Error()))
		}

		timageSnap, err = k8sContext.crdClient.AppsV1beta3().TImages(k8sContext.namespace).Get(context.TODO(), timageName, k8sMetaV1.GetOptions{})
		if err != nil {
			exit(fmt.Errorf("get timage %s/%s error: %s", k8sContext.namespace, timageName, err.Error()))
		}

		if timageSnap.Build == nil || timageSnap.Build.Running == nil || timageSnap.Build.Running.ID != id {
			exit(fmt.Errorf("build id missing or changed"))
		}

		pushStateOrDie(BuildPhasePrepareBuilding, "building image")

		if err := resolveRelativePaths(); err != nil {
			exit(errors.Wrap(err, "error resolving relative paths to absolute paths"))
		}

		if err := os.Chdir("/"); err != nil {
			exit(errors.Wrap(err, "error changing to root dir"))
		}

		image, err := executor.DoBuild(opts)
		if err != nil {
			exit(err)
		}

		pushStateOrDie(BuildPhasePreparePushing, "pushing image")
		if err := executor.DoPush(image, opts); err != nil {
			exit(err)
		}

		return
	},
}

// checkKanikoDir will check whether the executor is operating in the default '/kaniko' directory,
// conducting the relevant operations if it is not
func checkKanikoDir(dir string) error {
	if dir != constants.DefaultKanikoPath {

		if err := os.MkdirAll(dir, os.ModeDir); err != nil {
			return err
		}

		if err := os.Rename(constants.DefaultKanikoPath, dir); err != nil {
			return err
		}
	}
	return nil
}

// cacheFlagsValid makes sure the flags passed in related to caching are valid
func cacheFlagsValid() error {
	if !opts.Cache {
		return nil
	}
	// If --cache=true and --no-push=true, then cache repo must be provided
	// since cache can't be inferred from destination
	if opts.CacheRepo == "" && opts.NoPush {
		return errors.New("if using cache with --no-push, specify cache repo with --cache-repo")
	}
	return nil
}

// resolveDockerfilePath resolves the Dockerfile path to an absolute path
func resolveDockerfilePath() error {
	if isURL(opts.DockerfilePath) {
		return nil
	}
	if util.FilepathExists(opts.DockerfilePath) {
		abs, err := filepath.Abs(opts.DockerfilePath)
		if err != nil {
			return errors.Wrap(err, "getting absolute path for dockerfile")
		}
		opts.DockerfilePath = abs
		return copyDockerfile()
	}
	// Otherwise, check if the path relative to the build context exists
	if util.FilepathExists(filepath.Join(opts.SrcContext, opts.DockerfilePath)) {
		abs, err := filepath.Abs(filepath.Join(opts.SrcContext, opts.DockerfilePath))
		if err != nil {
			return errors.Wrap(err, "getting absolute path for src context/dockerfile path")
		}
		opts.DockerfilePath = abs
		return copyDockerfile()
	}
	return errors.New("please provide a valid path to a Dockerfile within the build context with --dockerfile")
}

// resolveEnvironmentBuildArgs replace build args without value by the same named environment variable
func resolveEnvironmentBuildArgs(arguments []string, resolver func(string) string) {
	for index, argument := range arguments {
		i := strings.Index(argument, "=")
		if i < 0 {
			value := resolver(argument)
			arguments[index] = fmt.Sprintf("%s=%s", argument, value)
		}
	}
}

// copy Dockerfile to /kaniko/Dockerfile so that if it's specified in the .dockerignore
// it won't be copied into the image
func copyDockerfile() error {
	if _, err := util.CopyFile(opts.DockerfilePath, config.DockerfilePath, util.FileContext{}, util.DoNotChangeUID, util.DoNotChangeGID); err != nil {
		return errors.Wrap(err, "copying dockerfile")
	}
	dockerignorePath := opts.DockerfilePath + ".dockerignore"
	if util.FilepathExists(dockerignorePath) {
		if _, err := util.CopyFile(dockerignorePath, config.DockerfilePath+".dockerignore", util.FileContext{}, util.DoNotChangeUID, util.DoNotChangeGID); err != nil {
			return errors.Wrap(err, "copying Dockerfile.dockerignore")
		}
	}
	opts.DockerfilePath = config.DockerfilePath
	return nil
}

// resolveSourceContext unpacks the source context if it is a tar in a bucket or in kaniko container
// it resets srcContext to be the path to the unpacked build context within the image
func resolveSourceContext() error {
	if opts.SrcContext == "" && opts.Bucket == "" {
		return errors.New("please specify a path to the build context with the --context flag or a bucket with the --bucket flag")
	}
	if opts.SrcContext != "" && !strings.Contains(opts.SrcContext, "://") {
		return nil
	}
	if opts.Bucket != "" {
		if !strings.Contains(opts.Bucket, "://") {
			// if no prefix use Google Cloud Storage as default for backwards compatibility
			opts.SrcContext = constants.GCSBuildContextPrefix + opts.Bucket
		} else {
			opts.SrcContext = opts.Bucket
		}
	}
	contextExecutor, err := buildcontext.GetBuildContext(opts.SrcContext, buildcontext.BuildOptions{
		GitBranch:            opts.Git.Branch,
		GitSingleBranch:      opts.Git.SingleBranch,
		GitRecurseSubmodules: opts.Git.RecurseSubmodules,
	})
	if err != nil {
		return err
	}
	logrus.Debugf("Getting source context from %s", opts.SrcContext)
	opts.SrcContext, err = contextExecutor.UnpackTarFromBuildContext()
	if err != nil {
		return err
	}

	logrus.Debugf("Build context located at %s", opts.SrcContext)
	return nil
}

func resolveRelativePaths() error {
	optsPaths := []*string{
		&opts.DockerfilePath,
		&opts.SrcContext,
		&opts.CacheDir,
		&opts.TarPath,
		&opts.DigestFile,
		&opts.ImageNameDigestFile,
		&opts.ImageNameTagDigestFile,
	}

	for _, p := range optsPaths {
		if path := *p; shdSkip(path) {
			logrus.Debugf("Skip resolving path %s", path)
			continue
		}

		// Resolve relative path to absolute path
		var err error
		relp := *p // save original relative path
		if *p, err = filepath.Abs(*p); err != nil {
			return errors.Wrapf(err, "Couldn't resolve relative path %s to an absolute path", *p)
		}
		logrus.Debugf("Resolved relative path %s to %s", relp, *p)
	}
	return nil
}

//exits with the given error and exit code
func exit(err error) {
	fmt.Println(err)
	const StatusFile = "/kaniko/status"
	_ = ioutil.WriteFile(StatusFile, []byte(err.Error()), 0666)
	os.Exit(-1)
}

func isURL(path string) bool {
	if match, _ := regexp.MatchString("^https?://", path); match {
		return true
	}
	return false
}

func shdSkip(path string) bool {
	return path == "" || isURL(path) || filepath.IsAbs(path)
}
