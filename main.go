package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/kevin1sMe/secret-wehbook/config"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// 解析kubeconfig路径
	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	yamlconfig := flag.String("config", "", "absolute path to the config file")
	flag.Parse()

	// 从yamlconfig中获取 Watch 配置，使用yaml库解析
	yconfig := config.WatchConfig{}
	yamlFile, err := os.ReadFile(*yamlconfig)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(yamlFile, &yconfig)
	if err != nil {
		panic(err)
	}
	log.Info().Msgf("config: %v", yconfig)

	// 从环境变量解析
	secretNS := yconfig.Watch.Namespace
	secretName := yconfig.Watch.Name

	// 构建配置
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// 创建Clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// 创建Informer工厂并指定命名空间
	factory := informers.NewSharedInformerFactoryWithOptions(clientset, time.Hour*12, informers.WithNamespace(secretNS))
	secretInformer := factory.Core().V1().Secrets().Informer()

	// 添加事件处理程序
	secretInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			// 转换对象为 Secret 类型
			secret, ok := obj.(*corev1.Secret)
			if !ok {
				// 如果转换失败，返回 false 表示不处理该事件
				return false
			}
			// 如果 Secret 名称匹配，返回 true 表示处理该事件
			return secret.Name == secretName
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				log.Info().Msgf("Secret [%s] added", secretName)
				Handle(clientset, yconfig.Actions, obj.(*corev1.Secret))
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				log.Info().Msgf("Secret [%s] update", secretName)
				Handle(clientset, yconfig.Actions, newObj.(*corev1.Secret))
			},
			DeleteFunc: func(obj interface{}) {
				log.Warn().Msgf("Secret [%s] deleted", secretName)
			},
		},
	})

	// 启动Informer
	stopCh := make(chan struct{})
	defer close(stopCh)
	go factory.Start(stopCh)

	log.Info().Msgf("Waiting for events...")
	// 等待缓存同步
	if !cache.WaitForCacheSync(stopCh, secretInformer.HasSynced) {
		panic("Failed to sync cache")
	}

	// 阻塞主线程以保持程序运行
	<-stopCh
}

// Handle根据配置中的不同 action.strategy 执行不同的操作
func Handle(clientset *kubernetes.Clientset, actions []*config.Action, secret *corev1.Secret) {
	for _, action := range actions {
		switch action.Strategy {
		case "RestartDeploy":
			// 假设我们想要更新的Deployment名称和命名空间
			deployments, err := clientset.AppsV1().Deployments(action.Selector.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: action.Selector.Labels,
			})
			if err != nil {
				log.Info().Msgf("List deployments error: %v", err)
				return
			}

			if len(deployments.Items) == 0 {
				log.Info().Msgf("No deployments found for selector [%s]", action.Selector.Labels)
				return
			}

			for _, deployment := range deployments.Items {
				// patch 触发pod重建
				patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"labels":{"secret-version":"%s"}}}}}`, secret.ResourceVersion)
				patchBytes := []byte(patch)

				// 应用patch
				_, err = clientset.AppsV1().Deployments(action.Selector.Namespace).Patch(context.Background(), deployment.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
				if err != nil {
					panic(err)
				}

				log.Info().Msgf("Deployment [%s] rollout restarted.", deployment.Name)
			}
		case "Webhook":
			// URL      string   `yaml:"url"`
			// Header   string   `yaml:"header"`
			// 根据配置中的URL和Header发送HTTP请求
			httpClient := &http.Client{}
			// 将secret的data转换为json格式并写入请求体
			secretData, err := json.Marshal(secret.Data)
			if err != nil {
				log.Error().Msgf("Marshal secret data error: %v", err)
				return
			}
			req, err := http.NewRequest("POST", action.URL, io.NopCloser(strings.NewReader(string(secretData))))
			if err != nil {
				log.Error().Msgf("Create request error: %v", err)
				return
			}

			req.Header.Set("Authorization", action.Header)
			req.Header.Set("Content-Type", "application/json")
			// 解析Header配置并添加到请求中
			headers := strings.Split(action.Header, ",")
			for _, header := range headers {
				parts := strings.SplitN(strings.TrimSpace(header), ":", 2)
				log.Debug().Msgf("Header: [%v]=>[%v]", parts[0], parts[1])
				if len(parts) == 2 {
					req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				}
			}
			_, err = httpClient.Do(req)
			if err != nil {
				log.Error().Msgf("Create request error: %v", err)
				return
			}
			log.Info().Msgf("Webhook request sent successfully")
		}
	}
}
