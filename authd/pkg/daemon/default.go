package daemon

// Can be set at compile time.
var DefaultConfDirectory = "/etc/flant/negentropy/authd-conf.d"

var DefaultConfig = &Config{
	ConfDirectory: DefaultConfDirectory,
}

func NewDefaultAuthd() *Authd {
	return &Authd{
		Config: DefaultConfig,
	}
}
