package controllers

import (
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/caicloud/cyclone/pkg/controller"
	"github.com/caicloud/cyclone/pkg/k8s/clientset"
	"github.com/caicloud/cyclone/pkg/k8s/informers"
	"github.com/caicloud/cyclone/pkg/workflow/common"
	wfctl "github.com/caicloud/cyclone/pkg/workflow/controller"
	handlers "github.com/caicloud/cyclone/pkg/workflow/controller/handlers/workflowrun"
	"github.com/caicloud/cyclone/pkg/workflow/workflowrun"
)

// NewWorkflowRunController ...
func NewWorkflowRunController(client clientset.Interface) *controller.Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	factory := informers.NewSharedInformerFactory(
		client,
		common.ResyncPeriod,
	)

	informer := factory.Cyclone().V1alpha1().WorkflowRuns().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				return
			}
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
			queue.Add(controller.Event{
				Key:       key,
				EventType: controller.DELETE,
				Object:    obj,
			})
		},
	})

	return controller.NewController(
		"WorkflowRun Controller",
		client,
		queue,
		informer,
		&handlers.Handler{
			Client:           client,
			TimeoutProcessor: workflowrun.NewTimeoutProcessor(client),
			GCProcessor:      workflowrun.NewGCProcessor(client, wfctl.Config.GC.Enabled),
			LimitedQueues:    workflowrun.NewLimitedQueues(client, wfctl.Config.Limits.MaxWorkflowRuns),
		},
	)
}
