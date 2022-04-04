package main

import (
	"github.com/steambap/captcha"
	"log"
	"strings"
	"sync"
	"time"
)

var begin = time.Now().UnixNano()
var listLock sync.Mutex
var blockLock sync.Mutex

var max int
var keys []int64
var codes []string
var ips []string
var left int
var end int
var exp int64
var nextClean time.Time

var block = make(map[string][2]int64)

func setNextClean() {
	n := cfg.Expire / 2
	if n < 1 {
		n = 1
	}
	nextClean = time.Now().Add(time.Duration(n) * time.Second)
}

func getExpire() int64 {
	return time.Second.Nanoseconds() * int64(cfg.Expire)
}

var expire = getExpire()

func reset() {
	expire = getExpire()
	_max := max
	max = cfg.Max
	if max < 0 {
		max = 1
	}
	if _max != max || keys == nil {
		clean()
	}
}

func opt(opt *captcha.Options) {
	opt.CharPreset = cfg.CharPreset
	opt.BackgroundColor = cfg.getBackground()
	opt.FontDPI = cfg.Dpi
	opt.Palette = cfg.getColors()
	opt.Noise = cfg.Noise
	opt.CurveNumber = cfg.Curve
	opt.TextLength = cfg.Length
}

func clean() {
	setNextClean()
	exp = time.Now().UnixNano() - begin - expire
	listLock.Lock()
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
		if exp > k {
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
	if exp > key || left == 0 {
		return generate(ip)
	}
	if blocked(ip) {
		return -1, nil
	}
	s := -1
	e := end
	b := s
	for i := (e + s) / 2; i != b && i < e && i > s; {
		b = i
		k := keys[i]
		if k < key {
			s = i
		} else if k > key {
			e = i
		} else {
			if ip == ips[i] {
				c := codes[i]
				if c != "" {
					codes[i] = ""
					ips[i] = ""
					left--
					if code != "" {
						if strings.ToLower(code) == strings.ToLower(c) {
							delete(block, ip)
							return 0, nil
						}
						fail(ip)
					}
				}
			}
		}
		i = (e + s) / 2
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
	k := time.Now().UnixNano() - begin
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
