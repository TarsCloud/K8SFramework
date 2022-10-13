package runner

type Runner interface {
	Init() error
	Start(chan struct{})
}
