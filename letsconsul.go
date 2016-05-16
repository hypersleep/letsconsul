package main

import(
	"fmt"
	"net"
	"time"
	"crypto"
	"crypto/ecdsa"
	"errors"
	"strconv"
	"encoding/json"

	"github.com/xenolf/lego/acme"
	consul "github.com/hashicorp/consul/api"
)

type(
	Letsconsul struct {
		Bind    string
		Domains map[string]*DomainRecord
	}

	DomainRecord struct {
		Domains   []string  `consul:"domains"`
		Email     string    `consul:"email"`
		Timestamp time.Time `consul:"timestamp"`
		Cert      string    `consul:"cert"`
		Chain     string    `consul:"chain"`
		Fullchain string    `consul:"fullchain"`
		key   *ecdsa.PrivateKey
		Reg   *acme.RegistrationResource
	}
)

func (domainRecord *DomainRecord) GetEmail() string { return domainRecord.Email }
func (domainRecord *DomainRecord) GetRegistration() *acme.RegistrationResource { return domainRecord.Reg }
func (domainRecord *DomainRecord) GetPrivateKey() crypto.PrivateKey { return domainRecord.key }

func (domainRecord *DomainRecord) write(client *consul.Client, consulService string, domainRecordName string) error {
	kv := client.KV()

	timestamp := domainRecord.Timestamp.Unix()
	timestampStr := strconv.Itoa(int(timestamp))

	p := &consul.KVPair {
		Key: consulService + "/domains/" + domainRecordName + "/timestamp",
		Value: []byte(timestampStr),
	}
	_, err := kv.Put(p, nil)
	if err != nil {
		return err
	}

	p = &consul.KVPair {
		Key: consulService + "/domains/" + domainRecordName + "/cert",
		Value: []byte(domainRecord.Cert),
	}
	_, err = kv.Put(p, nil)
	if err != nil {
		return err
	}

	p = &consul.KVPair {
		Key: consulService + "/domains/" + domainRecordName + "/chain",
		Value: []byte(domainRecord.Chain),
	}
	_, err = kv.Put(p, nil)
	if err != nil {
		return err
	}

	return nil
}

func (domainRecord *DomainRecord) renew(bind string) error {
	const letsEncryptURL = "https://acme-v01.api.letsencrypt.org/directory"

	// create acme client

	client, err := acme.NewClient(letsEncryptURL, domainRecord, acme.EC256)
	if err != nil {
		return err
	}

	// try to register
	// reg, err := client.Register()
	// if err != nil {
	// 	return err
	// }

	// domainRecord.Reg = reg

	// // agree with TOS
	// err = client.AgreeToTOS()
	// if err != nil {
	// 	return err
	// }

	host, port, err := net.SplitHostPort(bind)
	if err != nil {
		return err
	}

	httpProvider := acme.NewHTTPProviderServer(host, port)

	err = client.SetChallengeProvider(acme.TLSSNI01, httpProvider)
	if err != nil {
		return err
	}

	client.ExcludeChallenges([]acme.Challenge{acme.TLSSNI01})

	acmeCert, errmap := client.ObtainCertificate(domainRecord.Domains, true, nil)
	if len(errmap) > 0 {
		err = fmt.Errorf("%v", errmap)
		return err
	}

	domainRecord.Cert = string(acmeCert.Certificate)
	domainRecord.Chain = string(acmeCert.PrivateKey)

	domainRecord.Timestamp = time.Now()

	return nil
}

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

	v, err = kvFetch(kv, prefix, domainRecordName, "email")
	if err != nil {
		return err
	}

	domainRecord.Email = string(v)

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
