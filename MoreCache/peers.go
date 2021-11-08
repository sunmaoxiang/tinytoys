package MoreCache

import pb "MoreCache/protobuf_info"
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool) 
}

type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}


