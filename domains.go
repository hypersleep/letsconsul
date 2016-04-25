package main

import(
	"log"
	"time"

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

func (letsconsul *Letsconsul) get(client *consul.Client, prefix string) error {
	kv := client.KV()

	kvPairs, _, err := kv.Keys(prefix, "", nil)
	if err != nil {
		return err
	}

	for kv := range kvPairs {
		log.Println(kvPairs[kv])
	}

	return nil
}
