package controllers

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/caicloud/cyclone/pkg/controller"
	"github.com/caicloud/cyclone/pkg/k8s/clientset"
	"github.com/caicloud/cyclone/pkg/k8s/informers"
	"github.com/caicloud/cyclone/pkg/server/controller/handler/workflowtrigger"
)

// NewWorkflowTriggerController ...
func NewWorkflowTriggerController(client clientset.Interface) *controller.Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	factory := informers.NewSharedInformerFactoryWithOptions(
		client,
		time.Minute*5,
	)

	informer := factory.Cyclone().V1alpha1().WorkflowTriggers().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				return
			}
			log.WithField("name", key).Debug("new WorkflowTrigger observed")
			queue.Add(controller.Event{
				Key:       key,
				EventType: controller.CREATE,
				Object:    obj,
			})
		},
		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err != nil {
				return
			}
			log.WithField("name", key).Debug("WorkflowTrigger update observed")
			queue.Add(controller.Event{
				Key:       key,
				EventType: controller.UPDATE,
				Object:    new,
			})
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				return
			}
			log.WithField("name", key).Debug("deleting WorkflowTrigger")
			queue.Add(controller.Event{
				Key:       key,
				EventType: controller.DELETE,
				Object:    obj,
			})
		},
	})

	fmt.Println("return wft ctrl")
	return controller.NewController(
		"WorkflowTrigger Controller",
		client,
		queue,
		informer,
		&workflowtrigger.Handler{
			CronManager: workflowtrigger.NewTriggerManager(client),
		},
	)
}
