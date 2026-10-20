package geodata

type Asset struct {
	Url  string
	File string
}

type Config struct {
	Cron     string
	Outbound string
	Assets   []*Asset
}
