package main

import "sync"

var repositoryLock sync.Mutex
var repositoryValue = ""
var repositorySecretValue = ""

func setRepository(repository, secret string) {
	repositoryLock.Lock()
	defer repositoryLock.Unlock()
	repositoryValue = repository
	repositorySecretValue = secret
}

func getRepository() (repository string, secret string) {
	repositoryLock.Lock()
	defer repositoryLock.Unlock()
	return repositoryValue, repositorySecretValue
}

var maxReleaseLock sync.Mutex
var maxReleaseValue int = 120

func setMaxReleases(v int) {
	maxReleaseLock.Lock()
	defer maxReleaseLock.Unlock()
	maxReleaseValue = v
}

func getMaxReleases() int {
	maxReleaseLock.Lock()
	defer maxReleaseLock.Unlock()
	return maxReleaseValue
}

func setTagFormat(v string) {
}

func getTagFormat(v string) {
}

var maxBuildTime sync.Mutex

var executorLock sync.Mutex
var executorImage = ""
var executeSecret = ""

func setExecutor(image string, secret string) {
	executorLock.Lock()
	defer executorLock.Unlock()
	executorImage, executeSecret = image, secret
}

func getExecutor() (image string, secret string) {
	executorLock.Lock()
	defer executorLock.Unlock()
	return executorImage, executeSecret
}
