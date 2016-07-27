package main

import(
	"time"
	"errors"
	"encoding/json"

	consul "github.com/hashicorp/consul/api"
)

type(
	Letsconsul struct {
		Bind    string
		Domains map[string]*DomainRecord
	}
)

func (letsconsul *Letsconsul) renew(client *consul.Client, consulService string, renewInterval time.Duration) error {
	for domainRecordName, domainRecord := range letsconsul.Domains {
		if domainRecord.Timestamp.Add(renewInterval).Before(time.Now()) {
			err := domainRecord.renew(letsconsul.Bind)
			if err != nil {
				return err
			}

			err = domainRecord.write(client, consulService, domainRecordName)
			if err != nil {
				return err
			}
		}
	}
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
