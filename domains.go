package main

import(
	"log"
	"time"
	"errors"
	"strconv"
	// "bytes"
	"encoding/json"

	// "rsc.io/letsencrypt"
	consul "github.com/hashicorp/consul/api"
)

type(
	Letsconsul struct {
		Domains map[string]*DomainRecord
	}

	DomainRecord struct {
		Domains   []string  `consul:"domains"`
		Timestamp time.Time `consul:"timestamp"`
		Cert      string    `consul:"cert"`
		Chain     string    `consul:"chain"`
		Fullchain string    `consul:"fullchain"`
	}
)

func (domainRecord *DomainRecord) renew(name string) error {
	//return  errors.New("Failed to renew domain record: " + name +" Reason: " + err)
	log.Println(domainRecord, name)
	return nil
}

func (letsconsul *Letsconsul) renew(renewInterval time.Duration) error {
	// for name, domainRecord := range letsconsul.Domains {
	// 	timestamp, err := time.Parse(time.RFC3339, domainRecord.Timestamp)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	err = domainRecord.renew(name)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	return nil
}

func kvFetch(kv *consul.KV, prefix string, domainRecordName string, key string) ([]byte, error) {
	kvPair, _, err := kv.Get(prefix + "/domains/" + domainRecordName + "/" + key, nil)
	if err != nil {
		return nil, err
	}

	if kvPair == nil {
		return nil, errors.New("Can't fetch '" + key + "' key from '" + domainRecordName + "' domain")
	}

	return kvPair.Value, nil
}

func (domainRecord *DomainRecord) get(kv *consul.KV, prefix string, domainRecordName string) error {
	v, err := kvFetch(kv, prefix, domainRecordName, "domain_list")
	if err != nil {
		return err
	}

	err = json.Unmarshal(v, &domainRecord.Domains)
	if err != nil {
		return err
	}


	v, err = kvFetch(kv, prefix, domainRecordName, "timestamp")
	if err != nil {
		return err
	}

	i, err := strconv.ParseInt(string(v), 10, 64)
	if err != nil {
		return err
	}

	domainRecord.Timestamp = time.Unix(i, 0)

	v, err = kvFetch(kv, prefix, domainRecordName, "cert")
	if err != nil {
		return err
	}

	domainRecord.Cert = string(v)

	v, err = kvFetch(kv, prefix, domainRecordName, "chain")
	if err != nil {
		return err
	}

	domainRecord.Chain = string(v)

	v, err = kvFetch(kv, prefix, domainRecordName, "fullchain")
	if err != nil {
		return err
	}

	domainRecord.Fullchain = string(v)


	return nil
}

func (letsconsul *Letsconsul) get(client *consul.Client, prefix string) error {
	kv := client.KV()

	kvPair, _, err := kv.Get(prefix + "/domains_enabled", nil)
	if err != nil {
		return err
	}

	if kvPair == nil {
		return errors.New("Can't fetch 'domains_enabled' key")
	}

	var domainsEnabled []string = []string{}

	err = json.Unmarshal(kvPair.Value, &domainsEnabled)
	if err != nil {
		return err
	}

	for i := range domainsEnabled {
		if letsconsul.Domains[domainsEnabled[i]] == nil {
			domainRecord := &DomainRecord{}
			err := domainRecord.get(kv, prefix, domainsEnabled[i])
			if err != nil {
				return err
			}
			letsconsul.Domains[domainsEnabled[i]] = domainRecord
		}
	}

	return nil
}
