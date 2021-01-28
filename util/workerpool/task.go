package workerpool

type Task interface {
	Exec() (interface{}, error)
	CallBack(interface{}) error
}
