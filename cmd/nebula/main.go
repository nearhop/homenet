package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	screen "screen"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula"
	"github.com/slackhq/nebula/config"
	"github.com/slackhq/nebula/util"
)

// A version string that can be set with
//
//     -ldflags "-X main.Build=SOMEVERSION"
//
// at compile-time.
var Build string

func main() {
	configTest := flag.Bool("test", false, "Test the config and print the end result. Non zero exit indicates a faulty config")
	printVersion := flag.Bool("version", false, "Print version")
	printUsage := flag.Bool("help", false, "Print command line usage")
	resetconfig := flag.Bool("reset", false, "Factory reset the config")

	flag.Parse()

	if *printVersion {
		fmt.Printf("Version: %s\n", Build)
		os.Exit(0)
	}

	if *printUsage {
		flag.Usage()
		os.Exit(0)
	}

	isroot := nebula.IsRootUser()
	if !isroot {
		fmt.Println("It looks like you are running as non-root / non-sudo-user. Nearhopd will not work properly without sudo user / root access. Nearhop app will continue to run. Please ignore if you are already running as root / sudo user")
	}
	if *resetconfig {
		fmt.Println("Removing the config")
		os.RemoveAll(nebula.GetLogsFileDir())
		os.RemoveAll(nebula.GetConfigFileDir())
		os.RemoveAll(nebula.GetNearhopDir())
		os.Exit(0)
	}

	c := config.NewC()
	//configPath := nebula.GetConfigFileDir() + nebula.GetConfigFileName()
	configPath := nebula.GetConfigFileDir()
	amLighthouse := false
	err := c.Load(configPath)
	if err != nil {
		fmt.Printf("failed to load config: %s", err)
		amLighthouse = c.GetBool("lighthouse.am_lighthouse", false)
	}
	l := logrus.New()
	if amLighthouse {
		path := "logs/lighthouse.log"
		/*Log rotation correlation function
		`Withlinkname 'establishes a soft connection for the latest logs
		`Withrotationtime 'sets the time of log splitting, and how often to split
		Only one of withmaxage and withrotationcount can be set
		`Withmaxage 'sets the maximum save time before cleaning the file
		`Withrotationcount 'sets the maximum number of files to be saved before cleaning
		*/
		//The following configuration logs rotate a new file every 1 hour, keep the log files of the last 24 hours, and automatically clean up the surplus.
		writer, _ := rotatelogs.New(
			path+".%Y%m%d%H%M",
			rotatelogs.WithLinkName(path),
			rotatelogs.WithMaxAge(time.Duration(24)*time.Hour),
			rotatelogs.WithRotationTime(time.Duration(1)*time.Hour),
		)
		l.SetOutput(writer)
	} else {
		path := nebula.GetLogsFileDir() + "logs.txt"
		writer, _ := rotatelogs.New(
			path+".%Y%m%d%H%M",
			rotatelogs.WithLinkName(path),
			rotatelogs.WithMaxAge(time.Duration(15)*time.Minute),
			rotatelogs.WithRotationTime(time.Duration(15)*time.Minute),
		)
		l.SetOutput(writer)
		m, _ := screen.NewMainWindow()
		if m != nil {
			var onboarded bool

			_, err := os.Stat(configPath)
			if err != nil {
				onboarded = false
			} else {
				err = c.Load(configPath)
				onboarded = true
			}
			var status_err string
			status_err = ""
			if err != nil {
				status_err = err.Error()
			}
			ctx, _ := context.WithCancel(context.Background())
			ma := NewMainActivity(l, Build, c, m, onboarded, configPath)
			go ma.Run(ctx, status_err)

			werr := m.StartMainWindow(onboarded, c, ma.getCert(), status_err, func(cmd screen.CommandType, a []byte, length int) ([]byte, error) {
				return ma.processCommands(cmd, a, length)
			})
			if werr != nil {
				ma.setStatus("Error while initing Main Window. Closing")
			}
			ma.stop()
			// Exit
			os.Exit(0)
		}
	}

	ctrl, _, err := nebula.Main(c, *configTest, Build, l, nil)

	switch v := err.(type) {
	case util.ContextualError:
		v.Log(l)
		os.Exit(1)
	case error:
		l.WithError(err).Error("Failed to start")
		os.Exit(1)
	}

	if !*configTest {
		ctrl.Start()
		ctrl.ShutdownBlock()
	}

	os.Exit(0)
}
