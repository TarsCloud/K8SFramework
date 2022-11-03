package translator

type RunTimeConfig interface {
	GetDefaultNodeImage(namespace string) (image string, secret string)
}
