package config

// 监听secret变化并执行后续的动作配置, 配置文件示例见config.yml定义
type WatchConfig struct {
	Watch   Watch     `yaml:"watch"`
	Actions []*Action `yaml:"actions"`
}

type Watch struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type Action struct {
	Name     string   `yaml:"name"`
	Strategy string   `yaml:"strategy"`
	Selector Selector `yaml:"selector"`
	URL      string   `yaml:"url"`
	Header   string   `yaml:"header"`
}

type Selector struct {
	Namespace string `yaml:"namespace"`
	Labels    string `yaml:"labels"`
}
