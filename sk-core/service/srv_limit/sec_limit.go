package srv_limit

// SecLimit 每秒限制
// 实现了 sk-app 的 TimeLimit interface
type SecLimit struct {
	count   int   // 次数
	preTime int64 // 上一次记录时间
}

// Count 当前秒的访问次数
func (p *SecLimit) Count(nowTime int64) (curCount int) {
	if p.preTime != nowTime {
		p.count = 1
		p.preTime = nowTime
		curCount = p.count
		return
	}

	p.count++
	curCount = p.count
	return
}

func (p *SecLimit) Check(nowTime int64) int {
	if p.preTime != nowTime {
		return 0
	}
	return p.count
}
