package myrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"myrpc/codec"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c // 用来识别特定的rpc服务数据，比如说bios读取扇区前512B的时候最后两个字节就是特定的魔数

type Option struct {
	MagicNumber int        // MagicNumber marks this's a myrpc request
	CodeType    codec.Type // client may choose differrent Codec to encode body // 比如说可能是我定义的GobCodeC结构体
}

// 默认的操作的魔数和编解码类型
var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodeType:    codec.GobType,
}

/*
| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|
*/

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DafaultServer = NewServer()

func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
		}
		go server.ServeConn(conn) // 多协程模式，实现异步，处理连接。
	}
}
func Accept(lis net.Listener) {
	DafaultServer.Accept(lis)
}

func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option
	// Option{MagicNumber: xxx, CodecType: xxx} json格式的操作方式结构体编码
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodeType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodeType)
	}
	// 把conn用专门的函数包装成Codec然后调用专门的处理Codec函数进行处理
	server.serveCodec(f(conn)) // 把连接包装成Codec
}

var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex) // make sure to send a complete response
	wg := new(sync.WaitGroup)  // wait until all request are handled
	for {
		// header{ServiceMethod ...} 读取头部分
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}

			req.h.Error = err.Error()
			// 出错了就直接告诉客户端，进行sendResponse
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		// 处理请求
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

// ⚠️ 处理请求是并发的，回复请求的报文必须是逐个发送的，所以设计了一个sending锁
type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}
func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// 目前先假设request argv是一个string ，也就是body是一个普通的字符串
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}
func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	// 简单的echo
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())                                   // 打印请求头和请求体的具体值
	req.replyv = reflect.ValueOf(fmt.Sprintf("myrpc resp %d", req.h.Seq)) // 返回就返回这个请求的序号
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)       // 传入sending是为了保证send消息是互斥的

}
