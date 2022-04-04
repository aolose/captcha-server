package main

import (
	"flag"
	"gopkg.in/yaml.v2"
	"image/color"
	"log"
	"os"
	"time"
)

var config = "./cfg.yml"

func init() {
	flag.StringVar(
		&config,
		"config",
		config,
		"config file")

}

type Cfg struct {
	Expire      int
	Max         int
	Wait        int
	Conns       int
	Block       int
	Font        string
	Addr        string
	Backend     string
	Width       float64
	Height      float64
	Length      int
	Noise       float64
	Curve       int
	Dpi         float64
	Background  uint32
	Colors      []uint32
	ForwardHost bool `yaml:"forward_host"`
}

func getColor(c uint32) color.RGBA {
	n := c >> 8
	b := uint8(c - (n << 8))
	c = n
	n = c >> 8
	g := uint8(c - (n << 8))
	c = n
	n = c >> 8
	r := uint8(c - (n << 8))
	c = n
	n = c >> 8
	a := uint8(c - (n << 8))
	if a == 0 {
		a = 0xff
	}
	return color.RGBA{R: r, G: g, B: b, A: a}
}

func (c *Cfg) getColors() color.Palette {
	if c.Colors == nil {
		return nil
	}
	cc := make([]color.Color, len(c.Colors))
	for i, u := range c.Colors {
		cc[i] = getColor(u)
	}
	return cc
}

func (c *Cfg) getBackground() color.Color {
	return getColor(c.Background)
}

var cfg = Cfg{
	ForwardHost: false,
	Addr:        "localhost:9001",
	Backend:     "",
	Conns:       0,
	Expire:      30,
	Max:         1e4,
	Wait:        30,
	Block:       6,
	Font:        "",
	Width:       150,
	Height:      50,
	Length:      4,
	Noise:       1,
	Curve:       1,
	Background:  0xffffff,
	Dpi:         70,
	Colors:      nil,
}

func loadCfg() {
	_addr := cfg.Addr
	d, e := os.ReadFile(config)
	if e == nil {
		e = yaml.Unmarshal(d, &cfg)
	}
	if e != nil {
		log.Printf("[error] %v\n", e)
	}
	loadFont()
	reset()
	if server == nil {
		go start()
	} else if _addr != cfg.Addr {
		stop()
		time.Sleep(time.Second)
		go start()
	}
}

func main() {
	go func() {
		for {
			cleanBlock()
			time.Sleep(time.Duration(cfg.Wait/2) * time.Second)
		}
	}()
	for {
		loadCfg()
		time.Sleep(10 * time.Second)
	}
}
