package main

import "time"

const AbsoluteServerFileSaveDir = "/uploadDir"
const AbsoluteBuildWorkPath = "/buildDir"
const RegistryConfigFile = "/etc/registry-env/registry"
const RegistrySecretFile = "/etc/registry-env/secret"

const AutoDeleteServerFileDuration = time.Minute * 60
const AutoDeleteServerBuildDirDuration = time.Minute * 30
const MaximumConcurrencyBuildTask = 5

const (
	ServerAppFormKey    = "ServerApp"
	ServerNameFormKey   = "ServerName"
	ServerTypeFormKey   = "ServerType"
	ServerFileFormKey   = "ServerFile"
	ServerSecretFormKey = "Secret"

	BaseImageFormKey       = "BaseImage"
	BaseImageSecretFormKey = "BaseImageSecret"

	CreatePersonFormKey = "CreatePerson"
	MarkFormKey         = "Mark"
)
const (
	ServerAppPattern           string = ""
	ServerNamePattern          string = ""
	ServerTypePattern          string = ""
	ImageTagPattern            string = ""
	BaseImageRegistryPattern   string = ""
	BaseImageRepositoryPattern string = ""
	BaseImageTagPattern        string = ""
	BaseImageSecretPattern     string = ""
)
const (
	CppServerType     = "cpp"
	JavaWarServerType = "java-jar"
	JavaJarServerType = "java-war"
	NodejsServerType  = "nodejs"
	GoServerType      = "go"
	PHPServerType     = "php"
	PythonServerType  = "python"
)

const (
	BuildPhasePending          = "Pending"
	BuildPhaseReadingSecret    = "ReadingSecret"
	BuildPhasePrepareFile      = "PrepareFile"
	BuildPhasePullingBaseImage = "PullingBaseImage"
	BuildPhaseBuilding         = "Building"
	BuildPhasePushing          = "Pushing"
	BuildPhaseDone             = "Done"
	BuildPhaseFailed           = "Failed"
)

const (
	TImageTypeBase   = "base"
	TImageTypeServer = "server"
	TImageTypeNode   = "node"
)
