package meta

const (
	ResourceOutControlReason = "OutControl"

	ResourceDeleteReason = "DeleteError"

	ResourceGetReason = "GetError"
)

const (
	// ResourceOutControlError = "kind namespace/name already exists but not managed by namespace/name"
	ResourceOutControlError = "%s %s/%s already exists but not managed by %s/%s"

	// ResourceExistError = "kind namespace/name already exists"
	ResourceExistError = "%s %s/%s already exists"

	// ResourceNotExistError = "kind namespace/name already exists"
	ResourceNotExistError = "%s %s/%s not exists"

	// FiledImmutableError = "kind resource \"filed\" is immutable"
	FiledImmutableError = "%s resource filed \"%s\" is immutable"

	// ResourceDeleteError = "delete kind namespace/name err: errMsg"
	ResourceDeleteError = "delete %s %s/%s err: %s"

	// ResourceDeleteCollectionError ResourceDeleteError = "deleteCollection kind selector(labelSelector) err: errMsg"
	ResourceDeleteCollectionError = "deleteCollection %s selector(%s) err: %s"

	//ResourceGetError = "get kind namespace/name err: errMsg"
	ResourceGetError = "get %s %s/%s error: %s"

	//ResourceCreateError = "create kind namespace/name err: errMsg"
	ResourceCreateError = "create %s %s/%s error: %s"

	//ResourceUpdateError = "update kind namespace/name err: errMsg"
	ResourceUpdateError = "update %s %s/%s error: %s"

	//ResourcePatchError = "patch kind namespace/name err: errMsg"
	ResourcePatchError = "patch %s %s/%s error: %s"

	//ResourceSelectorError = "selector namespace/kind err: errMsg"
	ResourceSelectorError = "selector %s/%s error: %s"

	//ResourceInvalidError = "kind resource is invalid : errMsg"
	ResourceInvalidError = "%s resource is invalid : %s"

	//ShouldNotHappenError = "kind resource is invalid : errMsg"
	ShouldNotHappenError = "should not happen : %s"
)
