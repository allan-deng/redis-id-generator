package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"github.com/allan-deng/redis-id-generator/internal/router"
	"github.com/allan-deng/redis-id-generator/internal/generator"
	"time"
	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
)

func main() {

	confInit()
	logInit()
	generator.IdGenInit()
	serverRun()
}

func serverRun() {
	r := router.GetRouter()
	ip := viper.GetString("app.ip")
	port := viper.GetInt("app.port")

	addr := fmt.Sprintf("%v:%v", ip, port)

	if err := fasthttp.ListenAndServe(addr, r.Handler); err != nil {
		panic(err)
	}
}

func confInit() {
	configFile := flag.String("config", "", "Path to config file")
	flag.Parse()

	if *configFile == "" {
		*configFile = "./config/conf.toml"
	}

	viper.SetConfigFile(*configFile)

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config file: %w", err))
	}
}

func logInit() {

	logOutput := viper.GetString("log.output")
	level := viper.GetString("log.level")
	logPath := viper.GetString("log.filename")
	logMaxAge := viper.GetInt64("log.max_age")
	logRotationTime := viper.GetInt64("log.rotation_time")

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat:           "2006-01-02 15:04:05",
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		DisableLevelTruncation:    true,
	})

	log.SetReportCaller(true)

	if level == "" {
		log.SetLevel(log.InfoLevel)
	} else {
		switch level {
		case "trace":
			log.SetLevel(log.TraceLevel)
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		case "fatal":
			log.SetLevel(log.FatalLevel)
		}
	}

	if logOutput == "file" {
		if logPath == "" {
			logPath = "log/idgen.log"
		}

		dir := filepath.Dir(logPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				log.Fatalf("failed to create log directory: %v", err)
			}
		}

		writer, err := rotatelogs.New(
			logPath+".%Y%m%d-%H00",
			rotatelogs.WithLinkName(logPath),
			rotatelogs.WithMaxAge(time.Duration(logMaxAge*24)*time.Hour),
			rotatelogs.WithRotationTime(time.Duration(logRotationTime)*time.Hour),
		)

		if err != nil {
			panic(fmt.Errorf("failed to init log: %w", err))
		}

		log.SetOutput(writer)
	}

	log.Debugf("log init succ.")
}
