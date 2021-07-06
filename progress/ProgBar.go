package progress

import (
	"fmt"
	"strings"
	"time"
)

type Progressbar struct {
	Total   int
	current int
	last    int
	tick    *time.Ticker
}

const (
	maxbars  int = 100
	interval     = 500 * time.Millisecond
)

func (p *Progressbar) Print() {
	if p.last != p.current {
		p.last = p.current
		bars := p.calcBars(p.current) //算长度
		spaces := maxbars - bars - 1
		percent := 100 * (float32(p.current) / float32(p.Total))
		builder := strings.Builder{}
		for i := 0; i < bars; i++ {
			builder.WriteRune('=')
		}
		builder.WriteRune('>')
		for i := 0; i <= spaces; i++ {
			builder.WriteRune(' ')
		}
		fmt.Printf(" \r[%s] %3.2f%% (%d/%d)", builder.String(), percent, p.current, p.Total)
	}
	if p.current >= p.Total {
		fmt.Print("\n")
		p.tick.Stop()
	}
}

func (p *Progressbar) calcBars(portion int) int {
	if portion == 0 {
		return portion
	}
	return int(float32(maxbars) / (float32(p.Total) / float32(portion)))
}

func (p *Progressbar) Run() {
	go func() {
		p.tick = time.NewTicker(interval) //定时发送tick tock
		for range p.tick.C {
			p.Print()
		}
	}()
}

func (p *Progressbar) Current(v int) {
	p.current = v
}
func (p *Progressbar) Add(v int) {
	p.current += v
}
