package sops

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sops2 "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/aes"
)

func resourceSourceFile() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"filename": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"encryption_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"content": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"kms": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"age": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"file_permission": {
				Type:         schema.TypeString,
				Description:  "Permissions to set for the output file",
				Optional:     true,
				ForceNew:     true,
				Default:      "0777",
				ValidateFunc: validateMode,
			},
			"directory_permission": {
				Type:         schema.TypeString,
				Description:  "Permissions to set for directories created",
				Optional:     true,
				ForceNew:     true,
				Default:      "0777",
				ValidateFunc: validateMode,
			},
			"encrypted_regex": {
				Type:        schema.TypeString,
				Description: "A regex pattern denoting the contents in the file to be encrypted",
				Optional:    true,
				ForceNew:    true,
			},
		},
		CreateContext: resourceSopsFileCreate,
		Read:          resourceSopsFileRead,
		Delete:        resourceSopsFileDelete,
	}

}

func resourceSopsFileDelete(d *schema.ResourceData, _ interface{}) error {
	os.Remove(d.Get("filename").(string))
	return nil
}

func resourceLocalFileContent(d *schema.ResourceData) ([]byte, error) {
	if content, sensitiveSpecified := d.GetOk("sensitive_content"); sensitiveSpecified {
		return []byte(content.(string)), nil
	}
	if b64Content, b64Specified := d.GetOk("content_base64"); b64Specified {
		return base64.StdEncoding.DecodeString(b64Content.(string))
	}

	if v, ok := d.GetOk("source"); ok {
		source := v.(string)
		return ioutil.ReadFile(source)
	}

	content := d.Get("content")
	return []byte(content.(string)), nil
}

func sopsEncrypt(d *schema.ResourceData, content []byte, config *EncryptConfig) ([]byte, error) {
	inputStore := GetInputStore(d)
	outputStore := GetOutputStore(d)

	encType := d.Get("encryption_type").(string)
	fmt.Printf("enc type: %s\n", encType)

	groups, err, bytes, err2 := getKeyGroups(d, encType, config)
	if err2 != nil {
		return bytes, err2
	}
	encrypt, err := Encrypt(EncryptOpts{
		Cipher:            aes.NewCipher(),
		InputStore:        inputStore,
		OutputStore:       outputStore,
		InputPath:         d.Get("filename").(string),
		KeyServices:       LocalKeySvc(),
		UnencryptedSuffix: "",
		EncryptedSuffix:   "",
		UnencryptedRegex:  "",
		EncryptedRegex:    d.Get("encrypted_regex").(string),
		KeyGroups:         groups,
		GroupThreshold:    0,
	}, content)

	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	return encrypt, nil
}

func getKeyGroups(d *schema.ResourceData, encType string, config *EncryptConfig) ([]sops2.KeyGroup, error, []byte, error) {
	groups, err := KeyGroups(d, encType, config)
	if err != nil {
		return nil, nil, nil, err
	}
	return groups, err, nil, nil
}

func resourceSopsFileCreate(ctx context.Context, d *schema.ResourceData, i interface{}) diag.Diagnostics {
	providerConfig := i.(*EncryptConfig)
	var diags diag.Diagnostics
	content, err := resourceLocalFileContent(d)
	if err != nil {
		return diag.FromErr(err)
	}
	content, err = sopsEncrypt(d, content, providerConfig)
	if err != nil {
		return diag.FromErr(err)
	}
	//content = encrypt
	destination := d.Get("filename").(string)

	destinationDir := path.Dir(destination)
	if _, err := os.Stat(destinationDir); err != nil {
		dirPerm := d.Get("directory_permission").(string)
		dirMode, _ := strconv.ParseInt(dirPerm, 8, 64)
		if err := os.MkdirAll(destinationDir, os.FileMode(dirMode)); err != nil {
			return diag.FromErr(err)
		}
	}

	filePerm := d.Get("file_permission").(string)
	fileMode, _ := strconv.ParseInt(filePerm, 8, 64)

	if err := ioutil.WriteFile(destination, content, os.FileMode(fileMode)); err != nil {
		return diag.FromErr(err)
	}

	checksum := sha1.Sum(content)
	d.SetId(hex.EncodeToString(checksum[:]))

	return diags

}

func resourceSopsFileRead(d *schema.ResourceData, i interface{}) error {
	// If the output file doesn't exist, mark the resource for creation.
	outputPath := d.Get("filename").(string)
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		d.SetId("")
		return nil
	}

	// Verify that the content of the destination file matches the content we
	// expect. Otherwise, the file might have been modified externally and we
	// must reconcile.
	outputContent, err := ioutil.ReadFile(outputPath)
	if err != nil {
		return err
	}

	outputChecksum := sha1.Sum(outputContent)
	if hex.EncodeToString(outputChecksum[:]) != d.Id() {
		d.SetId("")
		return nil
	}

	return nil
}

func validateMode(i interface{}, k string) (s []string, es []error) {
	v, ok := i.(string)

	if !ok {
		es = append(es, fmt.Errorf("expected type of %s to be string", k))
		return
	}

	if len(v) > 4 || len(v) < 3 {
		es = append(es, fmt.Errorf("bad mode for file - string length should be 3 or 4 digits: %s", v))
	}

	fileMode, err := strconv.ParseInt(v, 8, 64)

	if err != nil || fileMode > 0777 || fileMode < 0 {
		es = append(es, fmt.Errorf("bad mode for file - must be three octal digits: %s", v))
	}

	return
}
