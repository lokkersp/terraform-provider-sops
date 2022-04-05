package sops

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceFileKey() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceFileKeyRead,

		Schema: map[string]*schema.Schema{
			"input_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"source_file": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"data_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"age_key_file": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"data": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"yaml": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"map": {
				Type:      schema.TypeMap,
				Computed:  true,
				Sensitive: true,
			},
			"raw": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func dataSourceFileKeyRead(d *schema.ResourceData, meta interface{}) error {
	sourceFile := d.Get("source_file").(string)
	content, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		return err
	}
	if ageKeyFile := d.Get("age_key_file").(string); ageKeyFile != "" {
		envVarName := "SOPS_AGE_KEY_FILE"
		err := os.Setenv(envVarName, ageKeyFile)
		if err != nil {
			log.Errorf("fail to set environment variable %s.Error is %s", envVarName, err)
		}
	}
	var format string
	if inputType := d.Get("input_type").(string); inputType != "" {
		format = inputType
	} else {
		switch ext := path.Ext(sourceFile); ext {
		case ".json":
			format = "json"
		case ".yaml", ".yml":
			format = "yaml"
		case ".env":
			format = "dotenv"
		case ".ini":
			format = "ini"
		default:
			return fmt.Errorf("Don't know how to decode file with extension %s, set input_type to json, yaml or raw as appropriate", ext)
		}
	}

	if err := validateInputType(format); err != nil {
		return err
	}
	dataKey := d.Get("data_key").(string)
	return readDataKey(content, format, dataKey, d)
}
