package sops

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"go.mozilla.org/sops/v3/decrypt"
	"gopkg.in/yaml.v2"

	"github.com/lokkersp/terraform-provider-sops/sops/internal/dotenv"
	"github.com/lokkersp/terraform-provider-sops/sops/internal/ini"
)

// readData consolidates the logic of extracting the from the various input methods and setting it on the ResourceData
func readData(content []byte, format string, d *schema.ResourceData) error {
	cleartext, err := decrypt.Data(content, format)
	if err != nil {
		return err
	}

	// Set output attribute for raw content
	err = d.Set("raw", string(cleartext))
	if err != nil {
		return err
	}

	// Set output attribute for content as a map (only for json and yaml)
	var data map[string]interface{}
	switch format {
	case "json":
		err = json.Unmarshal(cleartext, &data)
	case "yaml":
		err = yaml.Unmarshal(cleartext, &data)
	case "dotenv":
		err = dotenv.Unmarshal(cleartext, &data)
	case "ini":
		err = ini.Unmarshal(cleartext, &data)
	}
	if err != nil {
		return err
	}

	err = d.Set("data", flatten(data))
	if err != nil {
		return err
	}

	d.SetId("-")
	return nil
}

// readData consolidates the logic of extracting the from the various input methods and setting it on the ResourceData
func readDataKey(content []byte, format string, key string, d *schema.ResourceData) error {
	cleartext, err := decrypt.Data(content, format)
	if err != nil {
		return fmt.Errorf("fail to decrypt,format is %s:%s", format, err)
	}

	// Set output attribute for raw content
	err = d.Set("raw", string(cleartext))
	if err != nil {
		return fmt.Errorf("can't set raw,%s", err)
	}

	// Set output attribute for content as a map (only for json and yaml)
	var data map[string]interface{}
	switch format {
	case "json":
		err = json.Unmarshal(cleartext, &data)
	case "yaml":
		err = yaml.Unmarshal(cleartext, &data)
	case "dotenv":
		err = dotenv.Unmarshal(cleartext, &data)
	case "ini":
		err = ini.Unmarshal(cleartext, &data)
	}
	if err != nil {
		return fmt.Errorf("evaluated format is %s:%s", err, format)
	}

	flData := flatten(data)

	err = d.Set("data", flData[key])
	if err != nil {
		return err
	}
	err, value := flattenFromKey(data, key)
	out, err := yaml.Marshal(map[string]interface{}{key: data[key]})
	if err != nil {
		return err
	}
	err = d.Set("map", value)
	err = d.Set("yaml", string(out))
	if err != nil {
		return err
	}
	d.SetId("-")
	return nil
}
