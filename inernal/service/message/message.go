package message

type service struct {
}

type Config struct{}

func New(c *Config) *service {
	return &service{}
}
