package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"context"
	"time"

	"github.com/pkg/errors"
	rancher "github.com/rancher/go-rancher/v2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	env            string
	service        string
	rancherURL     string
	dockerImage    string
	upgradeTimeout time.Duration
)

func init() {
	flag.StringVar(&env, "env", "", "Rancher environment")
	flag.StringVar(&service, "service", "", "Service name")
	flag.StringVar(&rancherURL, "rancher-url", "", "Rancher url")
	flag.StringVar(&dockerImage, "docker-image", "", "Docker image")
	flag.DurationVar(&upgradeTimeout, "upgrade-timeout", time.Second*60, "Upgrade timeout (in seconds)")
}

func main() {
	flag.Parse()
	fmt.Printf("Run rancher-deploy %v %v %v\n", version, commit, date)

	if service == "" || rancherURL == "" || dockerImage == "" {
		log.Fatal("service, rancher-url, docker-image are required")
	}

	client, err := rancher.NewRancherClient(&rancher.ClientOpts{
		Url:       rancherURL,
		AccessKey: os.Getenv("RANCHER_ACCESS_KEY"),
		SecretKey: os.Getenv("RANCHER_SECRET_KEY"),
	})

	if err != nil {
		log.Fatal(err)
	}

	deploy := &Deploy{
		Client:      client,
		DockerImage: dockerImage,
		Env:         env,
		Service:     service,
	}

	if err := deploy.Run(); err != nil {
		log.Fatal(err)
	}
}

type Deploy struct {
	// Service could be some-service or some-stack/some-service format. Required!
	Service string

	// DockerImage eg. flyin/screenshot:latest. Required!
	DockerImage string

	// Env is environment name eg. default
	Env string

	Client *rancher.RancherClient
	stack  string
}

func (d *Deploy) Run() error {
	if strings.Contains(d.Service, "/") {
		parts := strings.SplitN(d.Service, "/", 2)
		d.stack = parts[0]
		d.Service = parts[1]
	}

	serviceFilters, err := d.getServiceFilters()
	if err != nil {
		return err
	}

	services, err := d.Client.Service.List(&rancher.ListOpts{Filters: serviceFilters})
	if err != nil {
		return errors.Wrap(err, "couldn't retrieve services list")
	}
	if len(services.Data) <= 0 {
		return fmt.Errorf("service %s not found", d.Service)
	}

	service := services.Data[0]

	upgrade := &rancher.ServiceUpgrade{
		InServiceStrategy: &rancher.InServiceUpgradeStrategy{
			LaunchConfig:           service.LaunchConfig,
			SecondaryLaunchConfigs: service.SecondaryLaunchConfigs,
			StartFirst:             false,
			IntervalMillis:         1000,
			BatchSize:              1,
		},

		ToServiceStrategy: &rancher.ToServiceUpgradeStrategy{},
	}

	_, err = d.Client.Service.ActionUpgrade(&service, upgrade)
	if err != nil {
		return errors.Wrap(err, "couldn't upgrade")
	}

	upgradedService, err := d.WaitUpgrade(service.Id)
	if err != nil {
		return err
	}

	_, err = d.Client.Service.ActionFinishupgrade(upgradedService)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("couldn't upgrader service %s", service.Name))
	}

	return nil
}

func (d *Deploy) WaitUpgrade(serviceId string) (*rancher.Service, error) {
	log.Printf("%s(%s) - Upgrading...\n", d.Service, serviceId)

	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()
	ctx, _ := context.WithTimeout(context.Background(), upgradeTimeout)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout for service %s exceed", d.Service)
		case <-ticker.C:
			s, err := d.Client.Service.ById(serviceId)
			if err != nil {
				return nil, errors.Wrap(err, "couldn't retrieve service by Id")
			}

			if s.State == "upgraded" {
				log.Printf("%s(%s) - Done...\n", d.Service, serviceId)
				return s, nil
			}
		}
	}
}

func (d *Deploy) getServiceFilters() (map[string]interface{}, error) {
	filters := map[string]interface{}{"name": d.Service}

	if d.stack == "" {
		return filters, nil
	}

	stackFilters := map[string]interface{}{"name": d.stack}
	if d.Env != "" {
		stackFilters["env"] = d.Env
	}

	stacks, err := d.Client.Stack.List(&rancher.ListOpts{Filters: stackFilters})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't retrieve stack list")
	}

	if len(stacks.Data) <= 0 {
		return nil, fmt.Errorf("stack %s not found", d.stack)
	}

	filters["stackId"] = stacks.Data[0].Id
	return filters, nil
}
