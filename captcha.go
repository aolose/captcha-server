package main

import (
	"github.com/steambap/captcha"
	"log"
	"strings"
	"sync"
	"time"
)

var listLock sync.Mutex
var blockLock sync.Mutex
var expire = time.Duration(cfg.Expire) * time.Second
var max = cfg.Max
var keys = make([]int64, max)
var codes = make([]string, max)
var ips = make([]string, max)
var left = 0
var end = 0

var exp int64 = 0

var block = make(map[string][2]int64)

func reset() {
	expire = time.Duration(cfg.Expire) * time.Second
	_max := max
	max = cfg.Max
	if _max != max {
		clean()
	}
}

func opt(opt *captcha.Options) {
	opt.BackgroundColor = cfg.getBackground()
	opt.FontDPI = cfg.Dpi
	opt.Palette = cfg.getColors()
	opt.Noise = cfg.Noise
	opt.CurveNumber = cfg.Curve
	opt.TextLength = cfg.Length
}
func clean() {
	listLock.Lock()
	exp = time.Now().Add(-expire).UnixNano()
	end = 0
	var _q = make([]int64, max)
	var _a = make([]string, max)
	var _i = make([]string, max)
	l := len(keys)
	m := max
	if l < m {
		m = l
	}
	for i := 0; i < m; i++ {
		k := keys[i]
		if k < exp {
			break
		}
		v := codes[i]
		if v == "" {
			continue
		}
		_q[end] = k
		_a[end] = v
		_i[end] = ips[i]
		end++
	}
	keys = _q
	codes = _a
	ips = _i
	left = end
	listLock.Unlock()
}

func fail(ip string) {
	if cfg.Block == 0 {
		return
	}
	blockLock.Lock()
	b, ok := block[ip]
	if ok {
		b[0]++
	} else {
		b = [2]int64{1, 0}
	}
	tm := b[0] - int64(cfg.Block)
	if tm > 0 {
		b[1] = time.Now().Add(time.Duration(cfg.Wait) * (2 << tm)).Unix()
	}
	block[ip] = b
	blockLock.Unlock()
}

func cleanBlock() {
	n := time.Now().Unix()
	blockLock.Lock()
	for k, v := range block {
		if v[1] < n {
			v[0] = v[0] - 1
			block[k] = v
		}
		if v[0] < 1 {
			delete(block, k)
		}
	}
	blockLock.Unlock()
}

func blocked(ip string) bool {
	if b, ok := block[ip]; ok {
		return int(b[0]) >= cfg.Block
	}
	return false
}

func Check(key int64, code, ip string) (int64, *captcha.Data) {
	if key < exp {
		return generate(ip)
	}
	if blocked(ip) {
		return -1, nil
	}
	bf := -1
	for i := end / 2; i != bf && i < end && i > -1; {
		bf = i
		k := keys[i]
		if k < key {
			i++
		} else if k > key {
			i--
		} else {
			if ip == ips[i] {
				c := codes[i]
				if c != "" {
					codes[i] = ""
					ips[i] = ""
					left--
					if code != "" {
						if strings.ToLower(code) == strings.ToLower(c) {
							return 0, nil
						}
						fail(ip)
					}
				}
			}
		}
	}
	return generate(ip)
}

func generate(ip string) (int64, *captcha.Data) {
	if blocked(ip) {
		return -1, nil
	}
	log.Println("generate start", time.Now())
	defer func() { log.Println("generate end", time.Now()) }()
	if left < end/2 {
		clean()
	}
	listLock.Lock()
	if end == max {
		if codes[0] != "" {
			left--
		}
		copy(keys[1:], keys)
		copy(codes[1:], codes)
		copy(ips[1:], ips)
		end--
	}
	k := time.Now().UnixNano()
	v := generateCode()
	keys[end] = k
	codes[end] = v.Text
	ips[end] = ip
	left++
	end++
	listLock.Unlock()
	return k, v
}

func generateCode() *captcha.Data {
	data, _ := captcha.New(150, 50, opt)
	return data
}
