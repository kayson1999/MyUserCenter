package util

import (
	"sync"
	"time"
)

// ── 雪花算法（Snowflake）ID 生成器 ──
//
// ID 结构（64 位）：
//   1 bit 符号位 | 41 bit 时间戳 | 10 bit 机器ID | 12 bit 序列号
//
// - 时间戳精度：毫秒，可用约 69 年
// - 机器 ID：0~1023，支持 1024 个节点
// - 序列号：0~4095，每毫秒可生成 4096 个 ID

const (
	// 自定义纪元：2025-01-01 00:00:00 UTC（毫秒）
	snowflakeEpoch int64 = 1735689600000

	// 各部分占用的位数
	nodeBits     = 10
	sequenceBits = 12

	// 最大值掩码
	maxNodeID   = -1 ^ (-1 << nodeBits)     // 1023
	maxSequence = -1 ^ (-1 << sequenceBits) // 4095

	// 位移量
	nodeShift      = sequenceBits            // 12
	timestampShift = nodeBits + sequenceBits // 22
)

// Snowflake 雪花算法 ID 生成器
type Snowflake struct {
	mu       sync.Mutex
	nodeID   int64
	sequence int64
	lastTime int64
}

var defaultNode *Snowflake

// InitSnowflake 初始化雪花算法节点
// nodeID 范围：0~1023
func InitSnowflake(nodeID int64) {
	if nodeID < 0 || nodeID > int64(maxNodeID) {
		nodeID = 0
	}
	defaultNode = &Snowflake{
		nodeID: nodeID,
	}
}

// GenerateID 生成一个雪花算法 ID
func GenerateID() int64 {
	if defaultNode == nil {
		InitSnowflake(0)
	}
	return defaultNode.Generate()
}

// Generate 生成一个唯一 ID
func (s *Snowflake) Generate() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()

	if now == s.lastTime {
		s.sequence = (s.sequence + 1) & int64(maxSequence)
		if s.sequence == 0 {
			// 当前毫秒序列号用尽，等待下一毫秒
			for now <= s.lastTime {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}

	s.lastTime = now

	id := ((now - snowflakeEpoch) << timestampShift) |
		(s.nodeID << nodeShift) |
		s.sequence

	return id
}
