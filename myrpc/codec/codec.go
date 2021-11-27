package codec

import "io"

type Header struct {
	ServiceMethod string // format:"Service.Method"
	Seq           uint64 // Seq是请求序号，某个请求的ID，用来区分不同的请求
	Error         string
}

// 把Codec设计为接口是为了实现不同的Codec实例，例如json和自己整的编码方式
type Codec interface {
	io.Closer
	ReadHeader(*Header) error         // 从io流中读头
	ReadBody(interface{}) error       // 从io流中读体
	Write(*Header, interface{}) error // 写入io流 ： 头、体
}

type NewCodecFunc func(closer io.ReadWriteCloser) Codec // 这类函数的功能是可以将有io能力的类型包装成Codec从而有获取信息和发送信息的能力

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json" // not implemeted
)

var NewCodecFuncMap map[Type]NewCodecFunc // 一个map传入什么类型的编码获得特定的封装函数
// const -> var -> init
func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
