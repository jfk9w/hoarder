package executors

type Logger[L any] interface {
	With(args ...any) L
}
