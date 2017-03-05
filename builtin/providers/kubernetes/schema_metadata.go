package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
)

var metadataSchema = &schema.Schema{
	Type:        schema.TypeList,
	Description: "Standard object's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata",
	Required:    true,
	MaxItems:    1,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"annotations": {
				Type:         schema.TypeMap,
				Description:  "An unstructured key value map stored with the object that may be used to store arbitrary metadata. More info: http://kubernetes.io/docs/user-guide/annotations",
				Optional:     true,
				ValidateFunc: validateAnnotations,
			},
			"generate_name": {
				Type:          schema.TypeString,
				Description:   "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency",
				Optional:      true,
				ForceNew:      true,
				ValidateFunc:  validateGenerateName,
				ConflictsWith: []string{"metadata.name"},
			},
			"generation": {
				Type:        schema.TypeInt,
				Description: "A sequence number representing a specific generation of the desired state.",
				Computed:    true,
			},
			"labels": {
				Type:         schema.TypeMap,
				Description:  "Map of string keys and values that can be used to organize and categorize (scope and select) objects. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels",
				Optional:     true,
				ValidateFunc: validateLabels,
			},
			"name": {
				Type:          schema.TypeString,
				Description:   "Name of the object, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ValidateFunc:  validateName,
				ConflictsWith: []string{"metadata.generate_name"},
			},
			"namespace": {
				Type:        schema.TypeString,
				Description: "Namespace defines the space within which name must be unique.",
				Optional:    true,
				ForceNew:    true,
				Default:     "default",
			},
			"resource_version": {
				Type:        schema.TypeString,
				Description: "An opaque value that represents the internal version of this object that can be used by clients to determine when objects have changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency",
				Computed:    true,
			},
			"self_link": {
				Type:        schema.TypeString,
				Description: "A URL representing this object.",
				Computed:    true,
			},
			"uid": {
				Type:        schema.TypeString,
				Description: "The unique in time and space value for this object. More info: http://kubernetes.io/docs/user-guide/identifiers#uids",
				Computed:    true,
			},
		},
	},
}
