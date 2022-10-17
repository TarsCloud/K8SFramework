package meta

const TarsAbilityLabelPrefix = "tars.io/ability" // 此标签表示 该节点可以被 tars 当做 App节点池使用
const TarsNodeLabel = "tars.io/node"             // 此标签表示 该节点可以被 tars 使用

const (
	TServerAppLabel  = "tars.io/ServerApp"
	TServerNameLabel = "tars.io/ServerName"
	TSubTypeLabel    = "tars.io/SubType"
	TServerIdLabel   = "tars.io/ServerID"
	TTemplateLabel   = "tars.io/Template"

	TTemplateParentLabel = "tars.io/Parent"

	TConfigNameLabel       = "tars.io/ConfigName"
	TConfigVersionLabel    = "tars.io/Version"
	TConfigActivatedLabel  = "tars.io/Activated"
	TConfigPodSeqLabel     = "tars.io/PodSeq"
	TConfigDeletingLabel   = "tars.io/Deleting"
	TConfigDeactivateLabel = "tars.io/Deactivate"

	K8SHostNameLabel = "kubernetes.io/hostname"
)
