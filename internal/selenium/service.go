package selenium

import (
	"fmt"
	"net"

	"github.com/go-faster/errors"
	"github.com/jfk9w-go/based"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"github.com/tebeka/selenium/firefox"
)

type ServiceParams struct {
	Config Config `validate:"required"`
}

type Service struct {
	service   *selenium.Service
	caps      selenium.Capabilities
	urlPrefix string
}

func NewService(params ServiceParams) (*Service, error) {
	if err := based.Validate(params); err != nil {
		return nil, err
	}

	opts := []selenium.ServiceOption{
		// selenium.StartFrameBuffer(),
	}

	caps := selenium.Capabilities{}
	switch params.Config.Browser {
	case "firefox":
		driverPath, err := params.Config.getDriverPath("geckodriver")
		if err != nil {
			return nil, errors.Wrap(err, "find geckodriver")
		}

		opts = append(opts, selenium.GeckoDriver(driverPath))
		caps.AddFirefox(firefox.Capabilities{
			Args:   params.Config.Args,
			Binary: params.Config.Binary,
		})

	case "chrome", "chromium":
		driverPath, err := params.Config.getDriverPath("chromedriver")
		if err != nil {
			return nil, errors.Wrap(err, "find chromedriver")
		}

		opts = append(opts, selenium.ChromeDriver(driverPath))
		caps.AddChrome(chrome.Capabilities{
			Args: params.Config.Args,
			Path: params.Config.Binary,
		})

	default:
		return nil, errors.New("unsupported browser")
	}

	port, err := getFreePort()
	if err != nil {
		return nil, errors.Wrap(err, "find free port")
	}

	service, err := selenium.NewSeleniumService(params.Config.Jar, port, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "init selenium service")
	}

	return &Service{
		service:   service,
		caps:      caps,
		urlPrefix: fmt.Sprintf("http://localhost:%d/wd/hub", port),
	}, nil
}

func (f *Service) Capabilities() selenium.Capabilities {
	return f.caps
}

func (f *Service) URLPrefix() string {
	return f.urlPrefix
}

func (f *Service) Stop() {
	_ = f.service.Stop()
}

func getFreePort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}

	return
}
