package meta

const KubernetesSystemAccountPrefix = "system:serviceaccount:kube-system:"
const TarsControllerAccountPrefix = "system:serviceaccount:kube-system:"

const TPodReadinessGate = "tars.io/active"
const NodeServantName = "nodeobj"
const NodeServantPort = 19385

const FixedTTreeResourceName = "tars-tree"
const FixedTFrameworkConfigResourceName = "tars-framework"

const TStorageClassName = "tars-storage-class"
const THostBindPlaceholder = "delay-bind"

const MaxTServerName = 59

type LauncherType string

const (
	Foreground LauncherType = "foreground"
	Background LauncherType = "background"
)

const ServiceImagePlaceholder = " "
