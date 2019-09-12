// thanks to:
// > https://github.com/256dpi/sentinel
// > https://github.com/stevelacy/go-sentry-kubernetes

package main

import (
	"fmt"
	"os"

	"github.com/getsentry/sentry-go"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/caarlos0/env"
)

type config struct {
	Dns         string `env:"SENTRY_DSN,required"`
	Debug       bool   `env:"SENTRY_DEBUG envDefault: False"`
	ServerName  string `env:"SENTRY_SERVER_NAME"`
	Environment string `env:"SENTRY_ENVIRONMENT"`
	KubeConfig  string `env:"KUBE_CONFIG"`
	KubeMaster  string `env:"KUBE_MASTER"`
	Namespace   string `env:"KUBE_NAMESPACE"`
	ReportAll   bool   `env:"KUBE_ALL_EVENTS envDefault: False"`
}

func main() {
	var err error
	cfg := config{}
	err = env.Parse(&cfg)

	// Parse Config
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}

	// initialize sentry
	err = sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.Dns,
		Debug:       cfg.Debug,
		ServerName:  cfg.ServerName,
		Environment: cfg.Environment,
		Transport:   sentry.NewHTTPSyncTransport(),
		Integrations: func([]sentry.Integration) []sentry.Integration {
			// disable all integrations
			return nil
		},
	})

	if err != nil {
		panic(err)
	}

	// Connect to the cluster
	var config *rest.Config
	if cfg.KubeConfig != "" || cfg.KubeMaster != "" {
		// use provided kube master and config
		config, err = clientcmd.BuildConfigFromFlags(cfg.KubeMaster, cfg.KubeConfig)
		if err != nil {
			panic(err)
		}
	} else {
		// otherwise get in cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
	}

	// check the namespaces
	if cfg.Namespace == "" {
		cfg.Namespace = api.NamespaceAll
	}

	// create client set
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// create list watch
	listWatch := cache.NewListWatchFromClient(
		clientSet.CoreV1().RESTClient(),
		"events",
		cfg.Namespace,
		fields.Everything(),
	)

	// create informer controller
	_, controller := cache.NewInformer(
		listWatch,
		&api.Event{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				process(obj.(*api.Event), cfg)
			},
			UpdateFunc: func(_, obj interface{}) {
				process(obj.(*api.Event), cfg)
			},
		},
	)

	// run controller
	fmt.Printf("sentry-kubernetes-event-watcher is running...")
	controller.Run(nil)
}

func process(event *api.Event, cfg config) {
	// ignore normal events if report all is not set
	if event.Type == api.EventTypeNormal && cfg.ReportAll != true {
		return
	}
	// prepare level
	level := sentry.LevelInfo
	if event.Type == api.EventTypeWarning {
		level = sentry.LevelWarning
	}

	// prepare message
	message := fmt.Sprintf(
		"[%s] %s/%s: %s",
		event.InvolvedObject.Kind,
		event.InvolvedObject.Namespace,
		event.InvolvedObject.Name,
		event.Message,
	)

	// prepare sentry event
	sentryEvent := &sentry.Event{
		Message: message,
		Level:   level,
		Tags: map[string]string{
			"type":      event.Type,
			"reason":    event.Reason,
			"kind":      event.InvolvedObject.Kind,
			"name":      event.InvolvedObject.Name,
			"namespace": event.InvolvedObject.Namespace,
		},
		Extra: map[string]interface{}{
			"event":  event.Name,
			"count":  event.Count,
			"source": event.Source.Component,

			"type":      event.Type,
			"reason":    event.Reason,
			"kind":      event.InvolvedObject.Kind,
			"name":      event.InvolvedObject.Name,
			"namespace": event.InvolvedObject.Namespace,
		},
	}

	// capture event
	sentry.CaptureEvent(sentryEvent)

	if cfg.Debug {
		fmt.Printf("sent event: %s\n", message)
	}

}
