package main

import(
	"os"
	"log"
	"net"
	"time"
	"reflect"
	"net/http"
	"strconv"
	"syscall"
	"os/signal"

	// "rsc.io/letsencrypt"
	"github.com/satori/go.uuid"
	consul "github.com/hashicorp/consul/api"
)

type(
	App struct {
		Bind            string        `env:"BIND"`
		ConsulToken     string        `env:"CONSUL_TOKEN"`
		ConsulService   string        `env:"CONSUL_SERVICE"`
		RenewInterval   time.Duration `env:"RENEW_INTERVAL"`
		ReloadInterval  time.Duration `env:"RELOAD_INTERVAL"`
		consulClient    *consul.Client
		consulServiceID string
	}
)

func (app *App) config() error {
	structType := reflect.TypeOf(*app)
	structValue := reflect.ValueOf(app).Elem()

	stringType := reflect.TypeOf(string(""))
	durationType := reflect.TypeOf(time.Duration(0))

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		envKey := field.Tag.Get("env")
		if envKey != "" {
			envValue := os.Getenv(envKey)

			switch field.Type {
			case stringType:
				structValue.FieldByName(field.Name).SetString(envValue)
			case durationType:
				duration, err := time.ParseDuration(envValue)
				if err != nil {
					return err
				}
				structValue.FieldByName(field.Name).Set(reflect.ValueOf(duration))
			}
		}
	}

	consulConfig := &consul.Config{
		Address:    "127.0.0.1:8500",
		Token:      app.ConsulToken,
		Scheme:     "http",
		HttpClient: http.DefaultClient,
	}

	client, err := consul.NewClient(consulConfig)
	if err != nil {
		return err
	}

	app.consulClient = client

	return nil
}


func (app *App) register() error {
	app.consulServiceID = uuid.NewV4().String()

	checks := consul.AgentServiceChecks{
		&consul.AgentServiceCheck{
			TTL: "5s",
		},
	}

	_, portstr, err := net.SplitHostPort(app.Bind)
	if err != nil {
		return err
	}

	port, err := strconv.Atoi(portstr)
	if err != nil {
		return err
	}

	service := &consul.AgentServiceRegistration{
		ID: app.consulServiceID,
		Name: app.ConsulService,
		Port: port,
		Checks: checks,
	}

	agent := app.consulClient.Agent()
	err = agent.ServiceRegister(service)
	if err != nil {
		return err
	}

	go func() {
		for {
			<- time.After(2 * time.Second)
			err = agent.PassTTL("service:" + app.consulServiceID, "Internal TTL ping")
			if err != nil {
				log.Println(err)
			}
		}
	}()

	go func() {
		signalCh := make(chan os.Signal, 4)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
		<-signalCh
		log.Println("Interruption signal caught, exiting...")
		log.Println("Deregister service with ID:", app.consulServiceID)
		err := agent.ServiceDeregister(app.consulServiceID)
		if err != nil {
			log.Fatal("Deregistration failed. Reason:", err)
		} else {
			log.Println("Deregistration successful. Exiting with 0 exit code.")
			os.Exit(0)
		}
	}()

	return nil
}

func (app *App) renewDomains() error {
	letsconsul := &Letsconsul{}
	letsconsul.Domains = make(map[string]*DomainRecord)

	err := letsconsul.get(app.consulClient, app.ConsulService)
	if err != nil {
		return err
	}

	err = letsconsul.renew(app.RenewInterval)
	if err != nil {
		return err
	}

	return nil
}

// func domainConfirmationHandler (w http.ResponseWriter, r *http.Request) {
// 	log.Fprintf(w, "Hello, TLS!\n")
// }

func (app *App) start() error {
	go func() {
		for {
			err := app.renewDomains()
			if err != nil {
				log.Println(err)
			}

			<- time.After(app.ReloadInterval)
		}
	}()

	log.Println("Application loaded")

	// http.HandleFunc("/", domainConfirmationHandler)

	// var m letsencrypt.Manager
	// if err := m.CacheFile("letsencrypt.cache"); err != nil {
	// 	log.Fatal(err)
	// }

	// return m.Serve()

	<- time.After(2000 * time.Second)
	return nil
}
