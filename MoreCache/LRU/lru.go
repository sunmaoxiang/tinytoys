package LRU

import (
	"container/list"
)
// 使用hash_map + 双端链表实现LRU
type Cache struct {
	ll *list.List           
	cache map[string]*list.Element
	maxBytes int64
	nbyte int64
	OnEvicted func(key string, value Value)
}
type entry struct {
	key string
	value Value
}
type Value interface {
	Len() int
}
func New(maxBytes int64, onEvicted func(key string,value Value)) *Cache {
	return &Cache{
		ll: list.New(),
		cache: make(map[string]*list.Element),
		maxBytes: maxBytes,
		nbyte: 0,
		OnEvicted: onEvicted,
	}
}
func (this* Cache)Get(key string) (value Value, ok bool) {
	if ele, ok := this.cache[key]; ok {
		this.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return 
}
func (this *Cache) RemoveOldest() {
	ele := this.ll.Back()
	if ele != nil {
		this.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(this.cache, kv.key)
		this.nbyte -= int64(len(kv.key)) + int64(kv.value.Len())
		if this.OnEvicted != nil {
			this.OnEvicted(kv.key, kv.value)
		}
	}
}
func (this *Cache) Add(key string, value Value) {
	if ele, ok := this.cache[key]; ok {
		this.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		this.nbyte += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := this.ll.PushFront(&entry{key, value})
		this.cache[key] = ele
		this.nbyte += int64(len(key)) + int64(value.Len())
	}
	for this.maxBytes != 0 && this.maxBytes < this.nbyte {
		this.RemoveOldest()
	}
}
func (this* Cache) Len() int {
	return this.ll.Len()
}