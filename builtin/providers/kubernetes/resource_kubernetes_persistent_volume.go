package kubernetes

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/kubernetes/pkg/api/errors"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_5"
)

func resourceKubernetesPersistentVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesPersistentVolumeCreate,
		Read:   resourceKubernetesPersistentVolumeRead,
		Exists: resourceKubernetesPersistentVolumeExists,
		Update: resourceKubernetesPersistentVolumeUpdate,
		Delete: resourceKubernetesPersistentVolumeDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema,
			"spec": {
				Type: schema.TypeList,
			},
		},
	}
}

func resourceKubernetesPersistentVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	volume := api.PersistentVolume{
		ObjectMeta: metadata,
		// TODO
	}
	log.Printf("[INFO] Creating new persistent volume: %#v", volume)
	out, err := conn.CoreV1().PersistentVolumes().Create(&volume)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new persistent volume: %#v", out)

	stateConf := &resource.StateChangeConf{
		Target:  []string{"Available"},
		Pending: []string{"Pending"},
		Timeout: 5 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.CoreV1().PersistentVolumes().Get(metadata.Name)
			if err != nil {
				log.Printf("[ERROR] Received error: %#v", err)
				return out, "Error", err
			}

			statusPhase := fmt.Sprintf("%v", out.Status.Phase)
			log.Printf("[DEBUG] Persistent volume %s status received: %#v", out.Name, statusPhase)
			return out, statusPhase, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}
	log.Printf("[INFO] Persistent volume %s deleted", name)

	d.SetId(out.Name)

	return resourceKubernetesPersistentVolumeRead(d, meta)
}

func resourceKubernetesPersistentVolumeRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Reading persistent volume %s", name)
	volume, err := conn.CoreV1().PersistentVolumes().Get(name)
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received persistent volume: %#v", volume)
	err = d.Set("metadata", flattenMetadata(volume.ObjectMeta))
	if err != nil {
		return err
	}

	// TODO

	return nil
}

func resourceKubernetesPersistentVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	// This is necessary in case the name is generated
	metadata.Name = d.Id()

	volume := api.PersistentVolume{
		ObjectMeta: metadata,
		// TODO
	}
	log.Printf("[INFO] Updating persistent volume: %#v", volume)
	out, err := conn.CoreV1().PersistentVolumes().Update(&volume)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted updated persistent volume: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesPersistentVolumeRead(d, meta)
}

func resourceKubernetesPersistentVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Deleting persistent volume: %#v", name)
	err := conn.CoreV1().PersistentVolumes().Delete(name, &api.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Persistent volume %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesPersistentVolumeExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	name := d.Id()
	log.Printf("[INFO] Checking persistent volume %s", name)
	_, err := conn.CoreV1().PersistentVolumes().Get(name)
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
