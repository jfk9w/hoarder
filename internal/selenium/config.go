package selenium

import "os/exec"

type Config struct {
	Jar     string   `yaml:"jar" doc:"Путь к JAR-файлу selenium-server-standalone."`
	Driver  string   `yaml:"driver,omitempty" doc:"Путь к драйверу. Если пустой, будет выполнен поиск в $PATH."`
	Browser string   `yaml:"browser,omitempty" enum:"chrome,chromium,firefox" default:"firefox" doc:"Браузер (browserName в Selenium)."`
	Binary  string   `yaml:"binary,omitempty" doc:"Путь к исполняемому файлу браузера."`
	Args    []string `yaml:"args,omitempty" doc:"Аргументы для запуска браузера." default:"[--headless]"`
}

func (cfg Config) getDriverPath(defaultPath string) (string, error) {
	driverPath := cfg.Driver
	if driverPath == "" {
		driverPath = defaultPath
	}

	return exec.LookPath(driverPath)
}
