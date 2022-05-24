package main

import "time"

const UploadDir = "/upload"
const BuildDir = "/build"
const CacheDir = "/cache"

const AutoDeleteServerFileDuration = time.Minute * 60
const AutoDeleteServerBuildDirDuration = time.Minute * 60
const MaximumConcurrencyBuildTask = 5

const (
	ServerAppFormKey    = "ServerApp"
	ServerNameFormKey   = "ServerName"
	ServerTypeFormKey   = "ServerType"
	ServerTagFormKey    = "ServerTag"
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
	BuildPhasePending         = "Pending"
	BuildPhasePreparing       = "Preparing"
	BuildPhaseSubmitting      = "Submitting"
	BuildPhasePrepareBuilding = "Building"
	BuildPhasePreparePushing  = "Pushing"
	BuildPhaseFailed          = "Failed"
	BuildPhaseDone            = "Done"
)

const (
	TImageTypeBase   = "base"
	TImageTypeServer = "server"
	TImageTypeNode   = "node"
)
