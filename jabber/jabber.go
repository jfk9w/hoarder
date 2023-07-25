package jabber

type Config struct {
	Jid      string `yaml:"jid"`
	Password string `yaml:"password"`
}

type Client struct {
}
