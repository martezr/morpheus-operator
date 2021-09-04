/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"log"

	"github.com/gomorpheus/morpheus-go-sdk"
	infrastructurev1 "github.com/martezr/morpheus-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const morpheusFinalizer = "vsphereinstance.morpheusoperator.morpheusdata.com"

// VsphereInstanceReconciler reconciles a VsphereInstance object
type VsphereInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=infrastructure.morpheusdata.com,resources=vsphereinstance,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.morpheusdata.com,resources=vsphereinstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.morpheusdata.com,resources=vsphereinstances/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *VsphereInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	vsphere := &infrastructurev1.VsphereInstance{}
	err := r.Get(ctx, req.NamespacedName, vsphere)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Println("Memcached resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Println(err, "Failed to get Memcached")
		return ctrl.Result{}, err
	}

	key := types.NamespacedName{Namespace: req.Namespace, Name: "morpheus-credentials"}
	secret := &corev1.Secret{}
	err = r.Get(context.TODO(), key, secret)
	if err != nil {
		log.Println(err)
	}
	secretData := make(map[string]string)
	for k, v := range secret.Data {
		secretData[k] = string(v)
	}

	client := morpheus.NewClient(secretData["url"])
	client.SetUsernameAndPassword(secretData["username"], secretData["password"])
	resp, err := client.Login()
	if err != nil {
		fmt.Println("LOGIN ERROR: ", err)
	}
	fmt.Println("LOGIN RESPONSE:", resp)

	// Check if the Morpheus vSphere instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isMemcachedMarkedToBeDeleted := vsphere.GetDeletionTimestamp() != nil
	if isMemcachedMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(vsphere, morpheusFinalizer) {
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeVsphereInstance(client, vsphere); err != nil {
				return ctrl.Result{}, err
			}

			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(vsphere, morpheusFinalizer)
			err := r.Update(ctx, vsphere)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(vsphere, morpheusFinalizer) {
		controllerutil.AddFinalizer(vsphere, morpheusFinalizer)
		err = r.Update(ctx, vsphere)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Find by name, then get by ID
	instanceResponse, err := client.ListInstances(&morpheus.Request{
		QueryParams: map[string]string{
			"name": vsphere.Name,
		},
	})
	if err != nil {
		log.Println(err)
	}
	listResult := instanceResponse.Result.(*morpheus.ListInstancesResult)
	instanceCount := len(*listResult.Instances)
	if instanceCount == 0 {
		instancePayload := map[string]interface{}{
			"name": req.Name,
			"type": vsphere.Spec.InstanceTypeCode,
			"site": map[string]interface{}{
				"id": vsphere.Spec.GroupID,
			},
			"plan": map[string]interface{}{
				"id": vsphere.Spec.PlanID,
			},
			"layout": map[string]interface{}{
				"id": vsphere.Spec.InstanceTypeLayout,
			},
		}
		instancePayload["instanceContext"] = vsphere.Spec.Environment
		config := make(map[string]interface{})
		customOptions := make(map[string]interface{})
		config["resourcePoolId"] = vsphere.Spec.ResourcePoolID
		for key, value := range vsphere.Spec.CustomOptions {
			customOptions[key] = value
		}
		config["customOptions"] = customOptions
		var networkInterfaces []map[string]interface{}
		nic := make(map[string]interface{})
		nic["network"] = map[string]interface{}{
			"id": fmt.Sprintf("network-%d", vsphere.Spec.NetworkID),
		}
		networkInterfaces = append(networkInterfaces, nic)

		var storageVolumes []map[string]interface{}
		volume := make(map[string]interface{})
		volume["rootVolume"] = true
		volume["name"] = "root"
		volume["size"] = "20"
		volume["datastoreId"] = 1
		volume["storageType"] = 1
		volume["id"] = -1
		storageVolumes = append(storageVolumes, volume)

		payload := map[string]interface{}{
			"zoneId":   vsphere.Spec.CloudID,
			"instance": instancePayload,
			"config":   config,
		}
		payload["networkInterfaces"] = networkInterfaces
		payload["volumes"] = storageVolumes

		instanceRequest := &morpheus.Request{Body: payload}
		//slcB, _ := json.Marshal(instanceRequest.Body)
		//log.Printf("API JSON REQUEST: %s", string(slcB))
		instanceResponse, err := client.CreateInstance(instanceRequest)
		//log.Printf("API REQUEST: %s", instanceResponse) // debug
		if err != nil {
			log.Printf("API FAILURE: %s - %s", instanceResponse, err)
			log.Println(err)
		}
		//log.Printf("API RESPONSE: %s", instanceResponse)
		result := instanceResponse.Result.(*morpheus.CreateInstanceResult)
		instance := result.Instance
		vsphere.Status.MorpheusID = int(instance.ID)
		for {
			status, err := PollInstanceStatus(client, int(instance.ID))
			if err != nil {
				log.Println(err)
			}
			vsphere.Status.State = status
			statusErr := r.Status().Update(ctx, vsphere)
			if statusErr != nil {
				log.Println(statusErr, "Failed to update vSphere status")
				return ctrl.Result{}, statusErr
			}
			if status != "provisioning" {
				fmt.Printf("Status: %s", status)
				break
			}
			time.Sleep(30 * time.Second)
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VsphereInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1.VsphereInstance{}).
		Complete(r)
}

func (r *VsphereInstanceReconciler) finalizeVsphereInstance(client *morpheus.Client, v *infrastructurev1.VsphereInstance) error {
	req := &morpheus.Request{
		QueryParams: map[string]string{},
	}
	//if USE_FORCE {
	//	req.QueryParams["force"] = "true"
	//}
	var instanceResponse *morpheus.Response
	var instanceDeleteResponse *morpheus.Response

	log.Println(v.Name)
	// Find by name, then get by ID
	instanceResponse, err := client.ListInstances(&morpheus.Request{
		QueryParams: map[string]string{
			"name": v.Name,
		},
	})
	if err != nil {
		log.Println(err)
	}
	listResult := instanceResponse.Result.(*morpheus.ListInstancesResult)
	instanceCount := len(*listResult.Instances)
	if instanceCount > 0 {
		firstRecord := (*listResult.Instances)[0]
		instanceId := firstRecord.ID

		instanceDeleteResponse, err = client.DeleteInstance(instanceId, req)
		if err != nil {
			if instanceDeleteResponse != nil && instanceDeleteResponse.StatusCode == 404 {
				log.Printf("API 404: %s - %s", instanceDeleteResponse, err)
				log.Println(err)
			} else {
				log.Printf("API FAILURE: %s - %s", instanceDeleteResponse, err)
				log.Println(err)
			}
		}
		log.Printf("API RESPONSE: %s", instanceDeleteResponse)
	}
	log.Println("Successfully finalized memcached")
	return nil
}

func PollInstanceStatus(client *morpheus.Client, instanceID int) (status string, err error) {
	req := &morpheus.Request{
		QueryParams: map[string]string{},
	}
	instanceDetails, err := client.GetInstance(int64(instanceID), req)
	if err != nil {
		return "failed", err
	}
	result := instanceDetails.Result.(*morpheus.GetInstanceResult)
	instance := result.Instance
	return instance.Status, nil
}
