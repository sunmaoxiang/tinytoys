package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	conn io.ReadWriteCloser // 从这里读
	buf  *bufio.Writer      // 对conn的一个buffer封装
	dec  *gob.Decoder       // dec.Decode()是从conn中读取数据并进行解码成为特定的类型
	enc  *gob.Encoder       // enc.Encode(数据)是将数据进行编码写入buf中
}

//var _ Codec = (*GobCodec)(nil)
func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn) // 使用buffer IO， 减少阻塞
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn), // 会从conn中读取消息进行decode，并写入一个专门的结构体中
		enc:  gob.NewEncoder(buf),  // encode的结果会自动放到buf里
	}
}

func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h) // 将conn读取的下一个数据作为解码器解码的数据来源，并将解码后的数据写入h中
}
func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush() // 确保buf中剩余数据被写出
		if err != nil {
			_ = c.Close()
		}
	}()
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}
	return nil
}
func (c *GobCodec) Close() error {
	return c.conn.Close()
}
