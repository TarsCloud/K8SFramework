package main

import "sync"

var registryLock sync.Mutex
var registryValue = ""
var registrySecretValue = ""

func setRegistry(registry, secret string) {
	registryLock.Lock()
	defer registryLock.Unlock()
	registryValue = registry
	registrySecretValue = secret
}

func getRegistry() (registry string, secret string) {
	registryLock.Lock()
	defer registryLock.Unlock()
	return registryValue, registrySecretValue
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
