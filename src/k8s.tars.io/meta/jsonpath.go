package meta

type JsonPatchOperator string

const (
	JsonPatchAdd     JsonPatchOperator = "add"
	JsonPatchRemove  JsonPatchOperator = "remove"
	JsonPatchReplace JsonPatchOperator = "replace"
)

type JsonPatchItem struct {
	OP    JsonPatchOperator `json:"op"`
	Path  string            `json:"path"`
	Value interface{}       `json:"value,omitempty"`
}
type JsonPatch []JsonPatchItem
