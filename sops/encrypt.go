package sops

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	wordwrap "github.com/mitchellh/go-wordwrap"

	mozillasops "go.mozilla.org/sops/v3"
	"go.mozilla.org/sops/v3/age"
	"go.mozilla.org/sops/v3/logging"
	//"go.mozilla.org/sops/v3/azkv"
	"go.mozilla.org/sops/v3/cmd/sops/codes"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	"go.mozilla.org/sops/v3/gcpkms"
	//"go.mozilla.org/sops/v3/hcvault"
	"go.mozilla.org/sops/v3/keys"
	"go.mozilla.org/sops/v3/keyservice"
	"go.mozilla.org/sops/v3/kms"
	"go.mozilla.org/sops/v3/version"
)

var log = logging.NewLogger("SOPS")

type EncryptOpts struct {
	Cipher            mozillasops.Cipher
	InputStore        mozillasops.Store
	OutputStore       mozillasops.Store
	InputPath         string
	KeyServices       []keyservice.KeyServiceClient
	UnencryptedSuffix string
	EncryptedSuffix   string
	UnencryptedRegex  string
	EncryptedRegex    string
	KeyGroups         []mozillasops.KeyGroup
	GroupThreshold    int
}

type fileAlreadyEncryptedError struct{}

func (err *fileAlreadyEncryptedError) Error() string {
	return "File already encrypted"
}

func (err *fileAlreadyEncryptedError) UserError() string {
	message := "The file you have provided contains a top-level entry called " +
		"'sops'. This is generally due to the file already being encrypted. " +
		"SOPS uses a top-level entry called 'sops' to store the metadata " +
		"required to decrypt the file. For this reason, SOPS can not " +
		"encrypt files that already contain such an entry.\n\n" +
		"If this is an unencrypted file, rename the 'sops' entry.\n\n" +
		"If this is an encrypted file and you want to edit it, use the " +
		"editor mode, for example: `sops my_file.yaml`"
	return wordwrap.WrapString(message, 75)
}

func ensureNoMetadata(branch mozillasops.TreeBranch) error {
	for _, b := range branch {
		if b.Key == "sops" {
			return &fileAlreadyEncryptedError{}
		}
	}
	return nil
}

func Encrypt(opts EncryptOpts, fileBytes []byte) (encryptedFile []byte, err error) {

	branches, err := opts.InputStore.LoadPlainFile(fileBytes)
	if err != nil {
		return nil, common.NewExitError(fmt.Sprintf("Error unmarshalling file: %tfSops", err), codes.CouldNotReadInputFile)
	}
	if err := ensureNoMetadata(branches[0]); err != nil {
		return nil, common.NewExitError(err, codes.FileAlreadyEncrypted)
	}
	path, err := filepath.Abs(opts.InputPath)
	if err != nil {
		return nil, err
	}
	tree := mozillasops.Tree{
		Branches: branches,
		Metadata: mozillasops.Metadata{
			KeyGroups:         opts.KeyGroups,
			UnencryptedSuffix: opts.UnencryptedSuffix,
			EncryptedSuffix:   opts.EncryptedSuffix,
			UnencryptedRegex:  opts.UnencryptedRegex,
			EncryptedRegex:    opts.EncryptedRegex,
			Version:           version.Version,
			ShamirThreshold:   opts.GroupThreshold,
		},
		FilePath: path,
	}
	dataKey, errs := tree.GenerateDataKeyWithKeyServices(opts.KeyServices)
	if len(errs) > 0 {
		err = fmt.Errorf("Could not generate data key: %tfSops", errs)
		return nil, err
	}

	err = common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    &tree,
		Cipher:  opts.Cipher,
	})
	if err != nil {
		return nil, err
	}

	encryptedFile, err = opts.OutputStore.EmitEncryptedFile(tree)
	if err != nil {
		return nil, common.NewExitError(fmt.Sprintf("Could not marshal tree: %tfSops", err), codes.ErrorDumpingTree)
	}
	return
}

func LocalKeySvc() (svcs []keyservice.KeyServiceClient) {
	svcs = append(svcs, keyservice.NewLocalClient())
	return
}

func GetKmsConf(d *schema.ResourceData) (KmsConf, error) {
	conf := KmsConf{}
	kmsConf := d.Get("kms").(map[string]interface{})
	arn := kmsConf["arn"]
	if arn == nil {
		return conf, fmt.Errorf("arn is not set")
	}
	conf.ARN = arn.(string)
	profile := kmsConf["profile"]
	if profile == nil {
		return conf, fmt.Errorf("AWS profile is not set")
	}
	conf.Profile = profile.(string)
	return conf, nil
}

func GetAgeConf(d *schema.ResourceData) (string, error) {
	ageConf := d.Get("age").(map[string]interface{})
	ageKey := ageConf["key"]
	log.Debugf("ageKey:%tfSops", ageKey)
	if ageKey == nil {
		return "", fmt.Errorf("age key is not set")
	}
	return ageKey.(string), nil
}

func GetEncryptionKey(d *schema.ResourceData, encType string) (interface{}, error) {
	switch encType {
	case "kms":
		kmsConf, err := GetKmsConf(d)
		if err != nil {
			return nil, err
		}
		return kmsConf, nil
	case "age":
		ageConf, err := GetAgeConf(d)
		if err != nil {
			return nil, err
		}
		return ageConf, nil
	}
	return nil, fmt.Errorf("failed to recognize encType:%tfSops", encType)
}

func KeyGroups(d *schema.ResourceData, encType string, config *EncryptConfig) ([]mozillasops.KeyGroup, error) {
	//var pgpKeys []keys.MasterKey
	//var azkvKeys []keys.MasterKey
	//var hcVaultMkKeys []keys.MasterKey
	//var cloudKmsKeys []keys.MasterKey
	var kmsKeys []keys.MasterKey
	var ageMasterKeys []keys.MasterKey
	//kmsEncryptionContext := kms.ParseKMSContext(c.String("encryption-context"))
	//if c.String("encryption-context") != "" && kmsEncryptionContext == nil {
	//  return nil, common.NewExitError("Invalid KMS encryption context format", codes.ErrorInvalidKMSEncryptionContextFormat)
	//}
	if "kms" == encType {

		resourceKmsConf, err := GetKmsConf(d)
		if err != nil {
			log.Errorf("fail to set kms from resource:%s", d.Id())
			if config.Kms.IsConfigured() {
				resourceKmsConf = config.Kms
			} else {
				return nil, err
			}
		}
		//todo support encryption context
		for _, k := range kms.MasterKeysFromArnString(resourceKmsConf.ARN, nil, resourceKmsConf.Profile) {
			kmsKeys = append(kmsKeys, k)
		}
	}

	if "gcpkms" == encType {
		gcpkmsConf := d.Get("gcpkms").(map[string]interface{})
		resourceIDs := gcpkmsConf["ids"].(string)

		for _, k := range gcpkms.MasterKeysFromResourceIDString(resourceIDs) {
			kmsKeys = append(kmsKeys, k)
		}
	}

	if "age" == encType {
		ageConf, err := GetAgeConf(d)
		if err != nil {
			log.Errorf("fail to set age key")
			if len(config.Age) > 0 {
				ageConf = config.Age
			} else {
				return nil, err
			}
		}
		ageKeys, err := age.MasterKeysFromRecipients(ageConf)
		if err != nil {
			return nil, err
		}
		for _, k := range ageKeys {
			ageMasterKeys = append(ageMasterKeys, k)
		}
	}

	if "mix" == encType {
		kmsConf, err := GetKmsConf(d)
		if err != nil {
			log.Errorf("fail to set kms from resource:%s\n", d.Id())
			if config.Kms.IsConfigured() {
				log.Infof("will use kms config from provider\n")
				kmsConf = config.Kms
			} else {
				log.Errorf("KMS isn't configured at all.\n")
				return nil, err
			}
		}
		//todo support encryption context
		for _, k := range kms.MasterKeysFromArnString(kmsConf.ARN, nil, kmsConf.Profile) {
			kmsKeys = append(kmsKeys, k)
		}
		ageConf, err := GetAgeConf(d)
		if err != nil {
			log.Errorf("fail to set age key")
			if len(config.Age) > 0 {
				log.Infof("will use age config from provider\n")
				ageConf = config.Age
			} else {
				log.Errorf("Age isn't configured at all.\n")
				return nil, err
			}
		}
		ageKeys, err := age.MasterKeysFromRecipients(ageConf)
		if err != nil {
			return nil, err
		}
		for _, k := range ageKeys {
			ageMasterKeys = append(ageMasterKeys, k)
		}
	}
	var group mozillasops.KeyGroup
	//group = append(group, azkvKeys...)
	//group = append(group, pgpKeys...)
	//group = append(group, hcVaultMkKeys...)
	//group = append(group, cloudKmsKeys...)
	group = append(group, ageMasterKeys...)
	group = append(group, kmsKeys...)
	log.Debugf("Master keys available:  %+v", group)
	return []mozillasops.KeyGroup{group}, nil
}
