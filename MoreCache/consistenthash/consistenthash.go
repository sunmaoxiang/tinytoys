// 一致性hash算法，来预防除了缓存宕机、清空引起的缓存雪崩
// 缓存雪崩：缓存基本全部失效导致数据库的压力突然增大

package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32


type Map struct {
	hash Hash
	replicas int
	keys []int
	hashMap map[int]string
}

// New create a Map instance
func New(replicas int, fn Hash) *Map {
	m := &Map {
		replicas: replicas,
		hash: fn,
		hashMap: make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}
// 把主机加入hash中
func (m *Map) Add(key ...string) {
	for _,key := range key {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key) ))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// 获取hash中与制定键最近的项
func (m *Map) Get(key string) string  {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	// Binary search for appropriate replica.
	// 0   1   2    3   4  (x -> 0)
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	return m.hashMap[ m.keys[ idx % len(m.keys)  ]  ]
}