package singleflight

import (
	"sync"
)

// 设计一个布隆过滤器防止缓存穿透
const mod = 64 * 101
const N = 101
const bitlen = 64

var table [N]uint64

func BloomHash(key string, i int) uint64 {
	p := []uint64{107, 177, 10009, 100007, 100000007}
	var ret uint64 = 0
	for i := 0; i < len(key); i++ {
		ret += p[i] * uint64(key[i])
		p[i] *= p[i]
		ret %= mod
		p[i] %= mod
	}
	return ret
}

func BloomAdd(str string) {
	for i := 0; i < 5; i++ {
		num := BloomHash(str, i)
		table[num/bitlen] |= (1 << (num % bitlen))
	}
}

func BloomHav(str string) bool {
	for i := 0; i < 5; i++ {
		num := BloomHash(str, i)
		if table[num/bitlen]&(1<<(num%bitlen)) == 0 {
			return false
		}
	}
	return true
}

// call表示正在进行中，或已经结束的请求。使用sync.WaitGroup锁避免重入
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// 针对相同的key无论DO被调用多少次，fn都只会调用一次，等待fn调用结束了，返回返回值或错误！这是防止热点数据造成的缓存击穿
// 同时对于存在的数据会放到bloom过滤器中，用来防止恶意大量访问数据中没有的存储
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	//if BloomHav(key) {
	//	return nil, fmt.Errorf("数据没有%s，已被bloom过滤器发现", key)
	//}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	} else {
		//BloomAdd(key)
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()
	c.val, c.err = fn()
	c.wg.Done()
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
	return c.val, c.err
}
