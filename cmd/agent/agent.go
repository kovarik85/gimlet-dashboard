package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gimlet-io/gimlet-dashboard/agent"
	"github.com/gimlet-io/gimlet-dashboard/cmd/agent/config"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Warnf("could not load .env file, relying on env vars")
	}

	config, err := config.Environ()
	if err != nil {
		log.Fatalln("main: invalid configuration")
	}

	initLogger(config)
	if log.IsLevelEnabled(log.TraceLevel) {
		log.Traceln(config.String())
	}

	if config.Host == "" {
		panic(fmt.Errorf("please provide the HOST variable"))
	}
	if config.AgentKey == "" {
		panic(fmt.Errorf("please provide the AGENT_KEY variable"))
	}
	if config.Env == "" {
		panic(fmt.Errorf("please provide the ENV variable"))
	}

	envName, namespace, err := parseEnvString(config.Env)
	if err != nil {
		panic(fmt.Errorf("invalid ENV variable. Format is env1=ns1,env2=ns2"))
	}

	if namespace != "" {
		log.Infof("Initializing %s kubeEnv in %s namespace scope", envName, namespace)
	} else {
		log.Infof("Initializing %s kubeEnv in cluster scope", envName)
	}

	k8sConfig, err := k8sConfig(config)
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		panic(err.Error())
	}

	kubeEnv := &agent.KubeEnv{
		Name:      envName,
		Namespace: namespace,
		Client:    clientset,
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	podController := agent.PodController(kubeEnv, config.Host, config.AgentKey)
	deploymentController := agent.DeploymentController(kubeEnv, config.Host, config.AgentKey)
	ingressController := agent.IngressController(kubeEnv, config.Host, config.AgentKey)
	go podController.Run(1, stopCh)
	go deploymentController.Run(1, stopCh)
	go ingressController.Run(1, stopCh)

	go serverCommunication(kubeEnv, config)

	signals := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// This goroutine executes a blocking receive for signals.
	// When it gets one it’ll print it out and then notify the program that it can finish.
	go func() {
		sig := <-signals
		log.Info(sig)
		done <- true
	}()

	log.Info("Initialized")
	<-done
	log.Info("Exiting")
}

func k8sConfig(config config.Config) (*rest.Config, error) {
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Infof("In-cluster-config didn't work (%s), loading from path in KUBECONFIG if set", err.Error())
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", config.KubeConfig)
		if err != nil {
			panic(err.Error())
		}
	}
	return k8sConfig, err
}

// helper function configures the logging.
func initLogger(c config.Config) {
	log.SetReportCaller(true)

	customFormatter := &log.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return "", fmt.Sprintf("[%s:%d]", filename, f.Line)
		},
	}
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)

	if c.Logging.Debug {
		log.SetLevel(log.DebugLevel)
	}
	if c.Logging.Trace {
		log.SetLevel(log.TraceLevel)
	}
}

func parseEnvString(envString string) (string, string, error) {
	if strings.Contains(envString, "=") {
		parts := strings.Split(envString, "=")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("")
		}
		return parts[0], parts[1], nil
	} else {
		return envString, "", nil
	}
}

func serverCommunication(kubeEnv *agent.KubeEnv, config config.Config) {
	for {
		done := make(chan bool)

		events, err := register(config.Host, kubeEnv.Name, kubeEnv.Namespace, config.AgentKey)
		if err != nil {
			log.Errorf("could not connect to Gimlet: %s", err.Error())
			time.Sleep(time.Second * 3)
			continue
		}

		log.Info("Connected to Gimlet")
		go sendState(kubeEnv, config.Host, config.AgentKey)

		go func(events chan map[string]interface{}) {
			for {
				e, more := <-events
				if more {
					log.Debugf("event received: %v", e)
					switch e["action"] {
					case "refetch":
						go sendState(kubeEnv, config.Host, config.AgentKey)
					}
				} else {
					log.Info("event stream closed")
					done <- true
					return
				}
			}
		}(events)

		<-done
		time.Sleep(time.Second * 3)
		log.Info("Disconnected from Gimlet")
	}
}

func sendState(kubeEnv *agent.KubeEnv, gimletHost string, agentKey string) {
	stacks, err := kubeEnv.Services("")
	if err != nil {
		log.Errorf("could not get state from k8s apiServer: %v", err)
		return
	}

	stacksString, err := json.Marshal(stacks)
	if err != nil {
		log.Errorf("could not serialize k8s state: %v", err)
		return
	}

	params := url.Values{}
	params.Add("name", kubeEnv.Name)
	reqUrl := fmt.Sprintf("%s/agent/state?%s", gimletHost, params.Encode())
	req, err := http.NewRequest("POST", reqUrl, bytes.NewBuffer(stacksString))
	if err != nil {
		log.Errorf("could not create http request: %v", err)
		return
	}
	req.Header.Set("Authorization", "BEARER " + agentKey)
	req.Header.Set("Content-Type", "application/json")

	client := httpClient()
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Errorf("could not send k8s state: %d - %v", resp.StatusCode, string(body))
		return
	}

	log.Debug("init state sent")
}

func httpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   20 * time.Second,
			ResponseHeaderTimeout: 20 * time.Second,
			ExpectContinueTimeout: 10 * time.Second,
		},
	}
}
