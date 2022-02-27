package models

type Msg int

const (
	Welcome Msg = iota
	Ok
	Reserved
	InvalidWorker
	Failed
	MalformedCommand
	InvalidWorkerCommand
	Unauthorized
)

func (d Msg) String() string {
	return [...]string{"WELCOME", "OK", "RESERVED", "INVALID WORKER", "FAILED", "MALFORMED COMMAND", "INVALID WORKER COMMAND", "UNAUTHORIZED"}[d]
}
