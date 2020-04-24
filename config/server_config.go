package config

import "fmt"

var (
	DefaultServerConfig = ServerConfig{
		Hostname:  "127.0.0.1",
		Listen:    "0.0.0.0",
		Port:      7900,
		WebPort:   7903,
		Threads:   4,
		ZKServers: nil,
		ErrorLog:  "./gobeansdb.log",
		AccessLog: "",
		StaticDir: "./",
	}
)

type ServerConfig struct {
	Hostname    string   `yaml:",omitempty"`
	ZKPath      string   `yaml:",omitempty"` // root path in zk
	ZKServers   []string `yaml:",omitempty"` // e.g. "zk1:2181,zk2:2181"
	Listen      string   `yaml:",omitempty"` // ip
	Port        int      `yaml:",omitempty"`
	WebPort     int      `yaml:",omitempty"`
	Threads     int      `yaml:",omitempty"` // NumCPU
	ErrorLog    string   `yaml:",omitempty"`
	AccessLog   string   `yaml:",omitempty"`
	AnalysisLog string   `yaml:",omitempty"`
	StaticDir   string   `yaml:",omitempty"` // directory for static files, e.g. *.html

}

func (c *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Hostname, c.Port)
}
