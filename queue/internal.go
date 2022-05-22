package queue

type Queue[VALUE any] interface {
	Add(value VALUE) error
	Delete(value VALUE)
	Len() int
	BlockPop() VALUE
	Pop() (VALUE, error)
}
