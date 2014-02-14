package lsp

const (
	doread = iota
	dowrite
	doclose
	docloseconn
	doconnid
	doconnect
	epochtimer
	receivemsg
)

type request struct {
	op     int
	val    interface{}
	replyc chan *retType
}

type retType struct {
	val interface{}
	err error
}
